// Copyright (c) 2019-2022 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

// Component interactions with the DomainInfoStore.
//
//  ┌─────────────────────────────────────────┐
//  │                                         │
//  │         DNS packet snooping             │
//  │                               callbacks │
//  └─┬┬──────────────────────────────────▲▲──┘
// (1)││                               (9)││
//    ││                                  ││
//    ││                                  ││
//    ││                                  ││
//  ┌─▼▼──────────────────────────────────┴┴──┐  (5)    ┌───────────────┐      ┌──────────────┐
//  │ MsgChan                     GetDomainIps│◄────────┤               │      │              │
//  │                                         │◄────────┤               │ (6)  │              │
//  │            DomainInfoStore              │         │ IPSetsManager ├─────►│    IPSets    │
//  │                                         ├────────►│               │      │              │
//  │            HandleUpdates UpdatesApplied ├────────►│OnDomainChange │      │ Apply        │
//  └───────┬───────────▲─────────────▲───────┘  (4)    └───────────────┘      └──▲───────────┘
//       (2)│        (3)│          (8)│                                        (7)│
//          │           │             │                                           │
//          │           │             └───────────────────────────────────────────┼────────┐
//          │           │                                                         │        │
//  ┌───────▼───────────┴─────────────────────────────────────────────────────────┴────────┴──┐
//  │UpdatesReadyChan                                                                         │
//  │                                   Dataplane loop                                        │
//  │                                                                                         │
//  └─────────────────────────────────────────────────────────────────────────────────────────┘
//
//  (1) Snooped packets are sent to the DomainInfoStore on the MsgChannel.
//      DomainInfoStore parses message, updates cache, stores set of changed names
//  (2) DomainInfoStore sends "update ready" tick to the dataplane.
//  (3) Dataplane loop calls back into DomainInfoStore to handle the current set of updates.
//  (4) During HandleUpdates(), the DomainInfoStore calls into the registered handlers (all IPSetsManagers) about each
//      domain that has been impacted (via OnDomainChange).
//  (5) and (6) During OnDomainChange(), the handler calls back into the DomainInfoStore for updated domain IPs and then
//      programs IP set dataplanes.
//  (7) Dataplane loop calls through to the IP set dataplanes to apply changes.
//  (8) Dataplane loop calls UpdatesApplied on DomainInfoStore to notify that the last set of handled updates have now
//      been applied to the IP set dataplanes.
//  (9) DomainInfoStore invokes callbacks associated with the DNS messages that were just applied to the Ip set
//      dataplanes.
//
// Notes:
// - (1) and (2) are channel based communications. The UpdatesReadyChan has capacity 1, so potentially multiple
//   DNS packets may be handled for only a single UpdatesReady tick.
// - Steps (3)-(9) are all callback based
// - In steps (3) - (6), if there are no changes required for the dataplane, the callbacks will be invoked immediately
// - Callbacks (3) and (8) can occur from the main (long-lived) dataplane loop, or from the ephemeral loops that the
//   dataplane starts up specifically for ipset updates while other updates (iptables) are happening. It is not
//   possible for both sets of updates to be happening at the same time, and these two callbacks will never be invoked
//   at the same time.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/collector"
	fc "github.com/projectcalico/calico/felix/config"
	"github.com/projectcalico/calico/felix/ip"
	"github.com/projectcalico/calico/felix/proto"
	"github.com/projectcalico/calico/libcalico-go/lib/set"
)

var (
	prometheusInvalidPacketsInCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "felix_dns_invalid_packets_in",
		Help: "Count of the number of invalid DNS request packets seen",
	})

	prometheusNonQueryPacketsInCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "felix_dns_non_query_packets_in",
		Help: "Count of the number of non-query DNS packets seen",
	})

	prometheusReqPacketsInCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "felix_dns_req_packets_in",
		Help: "Count of the number of DNS request packets seen",
	})

	prometheusRespPacketsInCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "felix_dns_resp_packets_in",
		Help: "Count of the number of DNS response packets seen",
	})
)

func init() {
	prometheus.MustRegister(prometheusInvalidPacketsInCount)
	prometheus.MustRegister(prometheusNonQueryPacketsInCount)
	prometheus.MustRegister(prometheusRespPacketsInCount)
	prometheus.MustRegister(prometheusReqPacketsInCount)
}

// The data that we hold for each value in a name -> value mapping.  A value can be an IP, or
// another name.  The values themselves are held as the keys of the nameData.values map.
type valueData struct {
	// When the validity of this value expires.
	expiryTime time.Time
	// Timer used to notify when the value expires.
	timer *time.Timer
	// Whether the value is another name, as opposed to being an IP.
	isName bool
}

// The data that we hold for each name.
type nameData struct {
	// Known values for this name.  Map keys are the actual values (i.e. IPs or lowercase CNAME names),
	// and valueData is as above.
	values map[string]*valueData
	// Names that we should notify a "change of information" for, and whose cached IP list
	// should be invalidated, when the info for _this_ name changes.
	namesToNotify set.Set
	// The revision sent to the dataplane associated with the creation or update of this nameData.
	revision uint64
}

type ipData struct {
	// The set of nameData entries that directly contain the IP.  We don't need to work our way backwards through the
	// chain because it is actually the namesToNotify that we are interested in which is propagated all the way
	// along the CNAME->A chain.
	nameDatas set.Set
}

type dnsExchangeKey struct {
	clientIP string
	dnsID    uint16
}

type DataWithTimestamp struct {
	Data []byte
	// We use 0 here to mean "invalid" or "unknown", as a 0 value would mean 1970,
	// which will not occur in practice during Calico's active lifetime.
	Timestamp uint64
	// Optional callback for notification that the dataplane updates associated with this DNS data are programmed.
	Callback func()
}

type DomainInfoStore struct {
	// Handlers that we need update.
	handlers []DomainInfoChangeHandler

	// Channel that we write to when we want DNS response capture to stop.
	stopChannel chan struct{}

	// Channel on which we receive captured DNS responses (beginning with the IP header).
	msgChannel chan DataWithTimestamp

	// Channel used to send trigger notifications to the dataplane that there are updates that can be applied to the
	// DomainInfoChangeHandlers.
	updatesReady chan struct{}

	// Stores for the information that we glean from DNS responses.  Note: IPs are held here as
	// strings, and also passed to the ipsets manager as strings.
	mutex    sync.RWMutex
	mappings map[string]*nameData

	// Store for reverse DNS lookups.
	reverse map[[16]byte]*ipData

	// Wildcard domain names that consumers are interested in (i.e. have called GetDomainIPs
	// for).
	wildcards map[string]*regexp.Regexp

	// Cache for "what are the IPs for <domain>?".  We have this to halve our processing,
	// because there are two copies of the IPSets Manager (one for v4 and one for v6) that will
	// call us to make identical queries.
	resultsCache map[string][]string

	// Channel for domain mapping expiry signals.
	mappingExpiryChannel chan *domainMappingExpired
	expiryTimePassed     func(time.Time) bool

	// Shim for starting and returning a timer that will call `onExpiry` after `ttl`.
	makeExpiryTimer func(ttl time.Duration, onExpiry func()) *time.Timer

	// Persistence.
	saveFile     string
	saveInterval time.Duration

	// Reclaiming memory for mappings that are now useless.
	gcTrigger  bool
	gcInterval time.Duration

	// Activity logging.
	collector collector.Collector

	// Handling of DNS request/response timestamps, so that we can measure and report DNS
	// latency.
	measureLatency   bool
	requestTimestamp map[dnsExchangeKey]uint64

	// Handling additional DNS mapping lifetime.
	epoch    int
	extraTTL time.Duration
	resetC   chan struct{}

	dnsResponseDelay time.Duration

	// --- Data for the current set of updates ---
	// These are updates from new DNS packets that have not been handled by the dataplane.

	// Monotonically increasing revision number used to determine what changes have and have not been applied
	// to the dataplane. This number indicates the next revision to apply.
	// Revisions between appliedRevision and currentRevision have been "handled" but not applied.
	currentRevision uint64

	// Set of callbacks for the current (not handled) updates.
	callbacks []func()

	// The collated set of domain name changes for the current (not handled) set of updates. These are always stored
	// lowercase.
	changedNames set.Set // string

	// --- Data for the handled set of updates ---\
	// These are updates that have been handled by the dataplane loop and programmed into the IP set dataplanes,
	// but have not yet been applied to the dataplane.

	// Set of callbacks for handled updates. These are callbacks for updates that have been handled but have not yet
	// been applied to the IP set dataplanes.
	handledCallbacks []func()

	// Whether the dataplane needs a sync from the domain name updates that have been handled. This is only accessed
	// from HandleUpdates() and UpdatesApplied() (which should not occur at the same time) - no lock is required for
	// accessing this field.
	needsDataplaneSync bool

	// --- Data for the applied set of updates ---
	// These are updates that are now programmed in the dataplane.

	// The revision number that has been applied to the dataplane.
	appliedRevision uint64
}

// Signal sent by timers' AfterFunc to the domain info store when a particular name -> IP or name ->
// cname mapping expires.
type domainMappingExpired struct {
	name, value string
}

type DnsConfig struct {
	Collector             collector.Collector
	DNSCacheEpoch         int
	DNSCacheFile          string
	DNSCacheSaveInterval  time.Duration
	DNSExtraTTL           time.Duration
	DNSLogsLatency        bool
	DebugDNSResponseDelay time.Duration
}

func NewDomainInfoStore(config *DnsConfig) *DomainInfoStore {
	return newDomainInfoStoreWithShims(
		config,
		time.AfterFunc,
		func(expiryTime time.Time) bool {
			return expiryTime.Before(time.Now())
		})
}

func newDomainInfoStoreWithShims(
	config *DnsConfig,
	makeExpiryTimer func(time.Duration, func()) *time.Timer,
	expiryTimePassed func(time.Time) bool,
) *DomainInfoStore {
	log.WithField("config", config).Info("Creating domain info store")
	s := &DomainInfoStore{
		// Updates ready channel has capacity 1 since only one notification is required at a time.
		updatesReady:         make(chan struct{}, 1),
		mappings:             make(map[string]*nameData),
		reverse:              make(map[[16]byte]*ipData),
		wildcards:            make(map[string]*regexp.Regexp),
		resultsCache:         make(map[string][]string),
		mappingExpiryChannel: make(chan *domainMappingExpired),
		expiryTimePassed:     expiryTimePassed,
		makeExpiryTimer:      makeExpiryTimer,
		saveFile:             config.DNSCacheFile,
		saveInterval:         config.DNSCacheSaveInterval,
		gcInterval:           13 * time.Second,
		collector:            config.Collector,
		measureLatency:       config.DNSLogsLatency,
		requestTimestamp:     make(map[dnsExchangeKey]uint64),
		epoch:                config.DNSCacheEpoch,
		extraTTL:             config.DNSExtraTTL,
		dnsResponseDelay:     config.DebugDNSResponseDelay,
		// Capacity 1 here is to allow UT to test the use of this channel without
		// needing goroutines.
		resetC: make(chan struct{}, 1),
		// Use a buffered channel here with reasonable capacity, so that the nfnetlink capture
		// thread can handle a burst of DNS response packets without becoming blocked by the reading
		// thread here.  Specifically we say 1000 because that what's we use for flow logs, so we
		// know that works; even though we probably won't need so much capacity for the DNS case.
		msgChannel: make(chan DataWithTimestamp, 1000),

		// Create an empty set of changed names.
		changedNames: set.New(),

		// Current update revision starts at 1.  0 is used to indicate no required updates.
		currentRevision: 1,
	}
	return s
}

func (s *DomainInfoStore) MsgChannel() chan<- DataWithTimestamp {
	return s.msgChannel
}

func (s *DomainInfoStore) UpdatesReadyChannel() <-chan struct{} {
	return s.updatesReady
}

func (s *DomainInfoStore) Start() {
	log.Info("Starting domain info store")

	// If there is a flow collector, register ourselves as a domain lookup cache.
	if s.collector != nil {
		s.collector.SetDomainLookup(s)
	}

	// Ensure that the directory for the persistent file exists.
	if err := os.MkdirAll(path.Dir(s.saveFile), 0755); err != nil {
		log.WithError(err).Fatal("Failed to create persistent file dir")
	}

	// Read mappings from the persistent file (if it exists).
	if err := s.readMappings(); err != nil {
		log.WithError(err).Warning("Failed to read mappings from file")
	}

	// Start repeating timers for periodically saving DNS info to a persistent file, and for
	// garbage collection.
	saveTimerC := time.NewTicker(s.saveInterval).C
	gcTimerC := time.NewTicker(s.gcInterval).C

	go s.loop(saveTimerC, gcTimerC)
}

// Dynamically handle changes to DNSCacheEpoch and DNSExtraTTL.
func (s *DomainInfoStore) OnUpdate(msg interface{}) {
	switch msg := msg.(type) {
	case *proto.ConfigUpdate:
		felixConfig := fc.FromConfigUpdate(msg)
		s.mutex.Lock()
		defer s.mutex.Unlock()
		newEpoch := felixConfig.DNSCacheEpoch
		if newEpoch != s.epoch {
			log.Infof("Update epoch (%v->%v) and send trigger to clear cache", s.epoch, newEpoch)
			s.epoch = newEpoch
			s.resetC <- struct{}{}
		}
		newExtraTTL := felixConfig.GetDNSExtraTTL()
		if newExtraTTL != s.extraTTL {
			log.Infof("Extra TTL is now %v", newExtraTTL)
			s.extraTTL = newExtraTTL
		}
	}
}

// HandleUpdates is called after the dataplane is notified via the UpdatesReadyChannel that updates are ready to apply.
// This calls through to the handlers to notify them of configuration changes. It is up to the dataplane, however,
// to subsequently apply these changes and then call through to UpdatesApplied() to notify the DomainInfoStore that
// changes associated with the pending domain updates are now applied to the dataplane.
//
// Note that during initial sync, HandleUpdates may be called multiple times before UpdatesApplied.
func (s *DomainInfoStore) HandleUpdates() (needsDataplaneSync bool) {
	// Move current data into the pending data.
	s.mutex.Lock()
	s.handledCallbacks = append(s.handledCallbacks, s.callbacks...)
	s.callbacks = nil

	// Increment the current revision, new entries will be added with this revision.
	s.currentRevision++

	changedNames := s.changedNames
	s.changedNames = set.New()
	s.mutex.Unlock()

	// Call into the handlers while we are not holding the lock.  This is important because the handlers will call back
	// into the DomainInfoStore to obtain domain->IP mapping info.
	changedNames.Iter(func(item interface{}) error {
		name := item.(string)
		for ii := range s.handlers {
			if s.handlers[ii].OnDomainChange(name) {
				// Track in member data that the dataplane needs a sync. It is not sufficient to just use a local
				// variable here since HandleUpdates may be called multiple times in a row before UpdatesApplied and
				// changes pending from a previous HandleUpdates invocation may make a later invocation a no-op even
				// though the dataplane changes still needs to be applied.
				s.needsDataplaneSync = true
			}
		}
		return nil
	})

	if !s.needsDataplaneSync {
		// Dataplane does not need any updates, so just call through immediately to UpdatesApplied so that we don't wait
		// unneccessarily for other updates to be applied before we invoke any callbacks.
		s.UpdatesApplied()
	}

	return s.needsDataplaneSync
}

// UpdatesApplied is called by the dataplane when the updates associated after the last invocation of HandleUpdates have
// been applied to the dataplane.
func (s *DomainInfoStore) UpdatesApplied() {
	// Dataplane updates have been applied. Invoke the pending callbacks and update the last applied revision number.
	s.mutex.Lock()
	defer s.mutex.Unlock()

	callbacks := s.handledCallbacks
	s.handledCallbacks = nil
	s.needsDataplaneSync = false

	// We have applied everything that was handled, so the applied revision is up to but not including the current
	// revision.
	s.appliedRevision = s.currentRevision - 1

	// Invoke the callbacks on another goroutine to unblock the main dataplane.
	if len(callbacks) > 0 {
		go func() {
			for i := range callbacks {
				callbacks[i]()
			}
		}()
	}
}

func (s *DomainInfoStore) RegisterHandler(handler DomainInfoChangeHandler) {
	s.handlers = append(s.handlers, handler)
}

func (s *DomainInfoStore) CompleteDeferredWork() error {
	// Nothing to do, we don't defer any work.
	return nil
}

func (s *DomainInfoStore) loop(saveTimerC, gcTimerC <-chan time.Time) {
	for {
		s.loopIteration(saveTimerC, gcTimerC)
	}
}

func (s *DomainInfoStore) loopIteration(saveTimerC, gcTimerC <-chan time.Time) {
	select {
	case msg := <-s.msgChannel:
		// TODO: Test and fix handling of DNS over IPv6.  The `layers.LayerTypeIPv4`
		// in the next line is clearly a v4 assumption, and some of the code inside
		// `nfnetlink.SubscribeDNS` also looks v4-specific.
		packet := gopacket.NewPacket(msg.Data, layers.LayerTypeIPv4, gopacket.Lazy)
		ipv4, _ := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
		if ipv4 != nil {
			log.Debugf("src %v dst %v", ipv4.SrcIP, ipv4.DstIP)
		} else {
			log.Debug("No IPv4 layer")
		}

		// Decode the packet as DNS.  Don't just use LayerTypeDNS here, because that
		// requires port 53.  Here we want to parse as DNS regardless of the port
		// number.
		dns := &layers.DNS{}
		transportLayer := packet.TransportLayer()
		if transportLayer == nil {
			log.Debug("Ignoring packet with no transport layer")
			return
		}
		dnsBytes := transportLayer.LayerPayload()

		// We've seen customers using tools that generate "ping" packets over UDP to test connectivity to
		// their DNS servers. One such tool uses "UDP PING ..." as the UDP payload.  Ignore such packets
		// rather than logging errors downstream.
		const udpPingPrefix = "UDP PING"
		if len(dnsBytes) >= len(udpPingPrefix) && string(dnsBytes[:len(udpPingPrefix)]) == udpPingPrefix {
			log.Debug("Ignoring UDP ping packet")
			prometheusInvalidPacketsInCount.Inc()
			return
		}

		err := dns.DecodeFromBytes(dnsBytes, gopacket.NilDecodeFeedback)
		if err != nil {
			log.WithError(err).Debug("Failed to decode DNS packet")
			prometheusInvalidPacketsInCount.Inc()
			return
		}
		if dns.OpCode != layers.DNSOpCodeQuery {
			log.Debug("Ignoring non-Query DNS packet.")
			prometheusNonQueryPacketsInCount.Inc()
			return
		}
		latencyIfKnown := s.processForLatency(ipv4, dns, msg.Timestamp)
		if dns.QR == true {
			// It's a DNS response.
			if dns.QDCount == 0 || len(dns.Questions) == 0 {
				// No questions; malformed packet?
				log.Debug("Ignoring DNS packet with no questions; malformed packet?")
				prometheusInvalidPacketsInCount.Inc()
				return
			}

			if s.collector != nil {
				if ipv4 != nil {
					s.collector.LogDNS(ipv4.SrcIP, ipv4.DstIP, dns, latencyIfKnown)
				} else {
					log.Warning("Not logging non-IPv4 DNS packet")
				}
			}
			s.processDNSPacket(dns, msg.Callback)
			prometheusRespPacketsInCount.Inc()
		} else {
			prometheusReqPacketsInCount.Inc()
		}
	case expiry := <-s.mappingExpiryChannel:
		s.processMappingExpiry(expiry.name, expiry.value)
	case <-saveTimerC:
		if err := s.SaveMappingsV1(); err != nil {
			log.WithError(err).Warning("Failed to save mappings to file")
		}
	case <-gcTimerC:
		_ = s.collectGarbage()
	case <-s.resetC:
		s.expireAllMappings()
	}
}

// maybeSignalUpdatesReady sends an update ready notification if required.
func (s *DomainInfoStore) maybeSignalUpdatesReady(reason string) {
	// Nothing to do if there are no changed names.
	if s.changedNames.Len() == 0 {
		log.Debug("No changed names")
		return
	}

	// If we need to delay the response, do that now. This is just for testing purposes. Release the lock so we don't
	// lock up the dataplane processing, but make sure we grab it again before sending the signal so that calling code
	// can ensure the signal has been sent but not yet handled (because handling requires access to the lock).
	if s.dnsResponseDelay != 0 {
		log.Debugf("Delaying DNS response for domains %v name for %d millis", s.changedNames, s.dnsResponseDelay)
		s.mutex.Unlock()
		time.Sleep(s.dnsResponseDelay)
		s.mutex.Lock()
	}

	// Signal updates are ready to process.
	select {
	case s.updatesReady <- struct{}{}:
		log.WithField("reason", reason).Debug("Sent update ready notification")
	default:
		log.WithField("reason", reason).Debug("Update ready notification already pending, updates will be handled together")
	}
}

type jsonMappingV1 struct {
	LHS    string
	RHS    string
	Expiry string
	Type   string
}

func (s *DomainInfoStore) readMappings() error {
	// Lock while we populate the cache.
	s.mutex.Lock()
	defer s.mutex.Unlock()

	f, err := os.Open(s.saveFile)
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)

	// Read the first line, which is the format version.
	if scanner.Scan() {
		version := strings.TrimSpace(scanner.Text())
		readerFunc := map[string]func(*bufio.Scanner) error{
			"1": s.readMappingsV1,
		}[version]
		if readerFunc != nil {
			log.Infof("Read mappings in v%v format", version)
			if err = readerFunc(scanner); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("Unrecognised format version: %v", version)
		}
	}
	// If we reach here, there was a problem scanning the version line.
	return scanner.Err()
}

const (
	v1TypeIP   = "ip"
	v1TypeName = "name"
)

func (s *DomainInfoStore) readMappingsV1(scanner *bufio.Scanner) error {
	for scanner.Scan() {
		var jsonMapping jsonMappingV1
		if err := json.Unmarshal(scanner.Bytes(), &jsonMapping); err != nil {
			return err
		}
		expiryTime, err := time.Parse(time.RFC3339, jsonMapping.Expiry)
		if err != nil {
			return err
		}
		ttlNow := time.Until(expiryTime)
		if ttlNow.Seconds() > 1 {
			log.Debugf("Recreate mapping %v", jsonMapping)
			// The mapping may have been saved by a previous version including uppercase letters,
			// so lowercase it now.
			s.storeInfo(strings.ToLower(jsonMapping.LHS), strings.ToLower(jsonMapping.RHS), ttlNow, jsonMapping.Type == v1TypeName)
		} else {
			log.Debugf("Ignore expired mapping %v", jsonMapping)
		}
	}
	s.maybeSignalUpdatesReady("mapping loaded")
	return scanner.Err()
}

func (s *DomainInfoStore) SaveMappingsV1() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	log.WithField("file", s.saveFile).Debug("Saving DNS mappings...")

	// Write first to a temporary save file, so that we can atomically rename it to the intended
	// file once it contains new data.  Thus we avoid overwriting a previous version of the file
	// (which may still be useful) until we're sure we have a complete new file prepared.
	tmpSaveFile := s.saveFile + ".tmp"
	f, err := os.Create(tmpSaveFile)
	if err != nil {
		return err
	}
	fileAlreadyClosed := false
	defer func() {
		if !fileAlreadyClosed {
			if err := f.Close(); err != nil {
				log.WithError(err).Warning("Error closing mappings file")
			}
		}
	}()

	// File format 1.
	if _, err = f.WriteString("1\n"); err != nil {
		return err
	}
	jsonEncoder := json.NewEncoder(f)
	for lhsName, nameData := range s.mappings {
		for rhsName, valueData := range nameData.values {
			jsonMapping := jsonMappingV1{LHS: lhsName, RHS: rhsName, Type: v1TypeIP}
			if valueData.isName {
				jsonMapping.Type = v1TypeName
			}
			jsonMapping.Expiry = valueData.expiryTime.Format(time.RFC3339)
			if err = jsonEncoder.Encode(jsonMapping); err != nil {
				return err
			}
			log.Debugf("Saved mapping: %v", jsonMapping)
		}
	}

	// Close the temporary save file.
	if err = f.Close(); err != nil {
		return err
	}
	fileAlreadyClosed = true

	// Move that file to the non-temporary name.
	if err = os.Rename(tmpSaveFile, s.saveFile); err != nil {
		return err
	}

	log.WithField("file", s.saveFile).Debug("Finished saving DNS mappings")

	return nil
}

func (s *DomainInfoStore) processDNSPacket(dns *layers.DNS, callback func()) {
	log.Debugf("DNS packet with %v answers %v additionals", len(dns.Answers), len(dns.Additionals))
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Update our cache and collect any name updates that we need to signal. We also determine the max revision
	// associated with these changes so that we can determine when to invoke the callbacks (if supplied).
	var maxRevision uint64
	for _, rec := range dns.Answers {
		if revision := s.storeDNSRecordInfo(&rec, "answer"); revision > maxRevision {
			maxRevision = revision
		}
	}
	for _, rec := range dns.Additionals {
		if msgNum := s.storeDNSRecordInfo(&rec, "additional"); msgNum > maxRevision {
			maxRevision = msgNum
		}
	}

	// Maybe signal an update is ready. Since we are holding the lock this is safe to do before the callback handling.
	s.maybeSignalUpdatesReady("mapping added")

	// If there is no callback supplied, just exit now.
	if callback == nil {
		return
	}

	// The DNS packet may have provided new information that is not yet programmed.  If so, add the callback to
	// the set of callbacks associated with the message number. These callbacks will be invoked once the dataplane
	// indicates the messages have been programmed. Otherwise, invoke the callback immediately.
	//
	// Since the message numbers are monotonic and the dataplane handles the messages in order, we use thresholds to
	// determine which messages have been processed. Invoke the callback on a goroutine so that we are not
	// holding the lock.
	switch {
	case maxRevision >= s.currentRevision:
		// The packet contains changes that have not yet been handled, so add the callback to the active set.
		s.callbacks = append(s.callbacks, callback)
	case maxRevision <= s.appliedRevision:
		// The packet only contains changes that are already programmed in the dataplane. Invoke the callback
		// immediately.
		go callback()
	default:
		// The packet has been handled, but not yet programmed, so add to the handled callbacks. This will be invoked
		// on the next call to UpdatesApplied().
		s.handledCallbacks = append(s.handledCallbacks, callback)
	}
}

func (s *DomainInfoStore) processMappingExpiry(name, value string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if nameData := s.mappings[name]; nameData != nil {
		if valueData := nameData.values[value]; (valueData != nil) && s.expiryTimePassed(valueData.expiryTime) {
			log.Debugf("Mapping expiry for %v -> %v", name, value)
			delete(nameData.values, value)
			if !valueData.isName {
				s.removeIPMapping(nameData, value)
			}
			s.gcTrigger = true
			s.compileChangedNames(name)
		} else if valueData != nil {
			log.Debugf("Too early mapping expiry for %v -> %v", name, value)
		} else {
			log.Debugf("Mapping already gone for %v -> %v", name, value)
		}
	}

	s.maybeSignalUpdatesReady("mapping expired")
}

func (s *DomainInfoStore) expireAllMappings() {
	log.Info("Expire all mappings")
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// For each mapping...
	for name, nameData := range s.mappings {
		// ...discard all of its values, being careful to release any reverse IP mappings.
		for value, valueData := range nameData.values {
			if !valueData.isName {
				s.removeIPMapping(nameData, value)
			}
		}
		nameData.values = make(map[string]*valueData)
		s.compileChangedNames(name)
	}

	// Trigger a GC to reclaim the memory that we can.
	s.gcTrigger = true

	s.maybeSignalUpdatesReady("epoch changed")
}

// Add a mapping between an IP and the nameData that directly contains the IP.
func (s *DomainInfoStore) addIPMapping(nameData *nameData, ipStr string) {
	ipBytes, ok := ip.ParseIPAs16Byte(ipStr)
	if !ok {
		return
	}

	ipd := s.reverse[ipBytes]
	if ipd == nil {
		ipd = &ipData{
			nameDatas: set.New(),
		}
		s.reverse[ipBytes] = ipd
	}
	ipd.nameDatas.Add(nameData)
}

// Remove a mapping between an IP and the nameData that directly contained the IP.
func (s *DomainInfoStore) removeIPMapping(nameData *nameData, ipStr string) {
	ipBytes, ok := ip.ParseIPAs16Byte(ipStr)
	if !ok {
		return
	}

	if ipd := s.reverse[ipBytes]; ipd != nil {
		ipd.nameDatas.Discard(nameData)
		if ipd.nameDatas.Len() == 0 {
			delete(s.reverse, ipBytes)
		}
	} else {
		log.Warningf("IP mapping is not cached %v", ipBytes)
	}
}

func (s *DomainInfoStore) storeDNSRecordInfo(rec *layers.DNSResourceRecord, section string) (revision uint64) {
	if rec.Class != layers.DNSClassIN {
		log.Debugf("Ignore DNS response with class %v", rec.Class)
		return
	}

	// Only CNAME type records can have the IP field set to nil
	if rec.IP == nil && rec.Type != layers.DNSTypeCNAME {
		log.Debugf("Ignore %s DNS response with empty or invalid IP", rec.Type.String())
		return
	}

	// All names are stored and looked up as lowercase.
	name := strings.ToLower(string(rec.Name))

	switch rec.Type {
	case layers.DNSTypeA:
		log.Debugf("A: %v -> %v with TTL %v (%v)",
			name,
			rec.IP,
			rec.TTL,
			section,
		)
		revision = s.storeInfo(name, rec.IP.String(), time.Duration(rec.TTL)*time.Second, false)
	case layers.DNSTypeAAAA:
		log.Debugf("AAAA: %v -> %v with TTL %v (%v)",
			name,
			rec.IP,
			rec.TTL,
			section,
		)
		revision = s.storeInfo(name, rec.IP.String(), time.Duration(rec.TTL)*time.Second, false)
	case layers.DNSTypeCNAME:
		cname := strings.ToLower(string(rec.CNAME))
		log.Debugf("CNAME: %v -> %v with TTL %v (%v)",
			name,
			cname,
			rec.TTL,
			section,
		)
		revision = s.storeInfo(name, cname, time.Duration(rec.TTL)*time.Second, true)
	default:
		log.Debugf("Ignore DNS response with type %v", rec.Type)
	}

	return
}

func (s *DomainInfoStore) storeInfo(name, value string, ttl time.Duration, isName bool) (revision uint64) {
	if value == "0.0.0.0" {
		// DNS records sometimes contain 0.0.0.0, but it's not a real routable IP and we
		// must avoid passing it on to ipsets, because ipsets complains with "ipset v6.38:
		// Error in line 1: Null-valued element, cannot be stored in a hash type of set".
		// We don't need to record 0.0.0.0 mappings for any other purpose, so just log and
		// bail out early here.
		log.Debugf("Ignoring zero IP (%v -> %v TTL %v)", name, value, ttl)
		return
	}

	// Add on extra TTL, if configured.
	ttl = time.Duration(int64(ttl) + int64(s.extraTTL))

	// Impose a minimum TTL of 2 seconds - i.e. ensure that the mapping that we store here will
	// not expire for at least 2 seconds.  Otherwise TCP connections that should succeed will
	// fail if they involve a DNS response with TTL 1.  In detail:
	//
	// a. A client does a DNS lookup for an allowed domain.
	// b. DNS response comes back, and is copied here for processing.
	// c. Client sees DNS response and immediately connects to the IP.
	// d. Felix's ipset programming isn't in place yet, so the first connection packet is
	//    dropped.
	// e. TCP sends a retry connection packet after 1 second.
	// f. 1 second should be plenty long enough for Felix's ipset programming, so the retry
	//    connection packet should go through.
	//
	// However, if the mapping learnt from (c) expires after 1 second, the retry connection
	// packet may be dropped as well.  Imposing a minimum expiry of 2 seconds avoids that.
	if int64(ttl) < int64(2*time.Second) {
		ttl = 2 * time.Second
	}

	makeTimer := func() *time.Timer {
		return s.makeExpiryTimer(ttl, func() {
			s.mappingExpiryChannel <- &domainMappingExpired{name: name, value: value}
		})
	}
	if s.mappings[name] == nil {
		s.mappings[name] = &nameData{
			values:        make(map[string]*valueData),
			namesToNotify: set.New(),
		}
	}
	existing := s.mappings[name].values[value]
	if existing == nil {
		// If this is the first value for this name, check whether the name matches any
		// existing wildcards.
		if len(s.mappings[name].values) == 0 {
			for wildcard, regex := range s.wildcards {
				if regex.MatchString(name) {
					s.mappings[name].namesToNotify.Add(wildcard)
				}
			}
		}
		s.mappings[name].values[value] = &valueData{
			expiryTime: time.Now().Add(ttl),
			timer:      makeTimer(),
			isName:     isName,
		}

		if isName {
			// Value is another name. If we don't yet have any information, create a
			// mapping entry for it so we can record that it is a descendant of the name in
			// hand.  Then, when we get information for the descendant name, we can correctly
			// signal changes for the name in hand and any of its ancestors.
			if s.mappings[value] == nil {
				s.mappings[value] = &nameData{
					values:        make(map[string]*valueData),
					namesToNotify: set.New(),
				}
			}
		} else {
			// Value is an IP. Add to our IP mapping.
			s.addIPMapping(s.mappings[name], value)
		}

		// Compile the set of changed names. The calling code will signal that the info has changed for
		// the compiled set of names.
		s.compileChangedNames(name)

		// Set the revision for this entry.
		s.mappings[name].revision = s.currentRevision
		revision = s.currentRevision
	} else {
		newExpiryTime := time.Now().Add(ttl)
		if newExpiryTime.After(existing.expiryTime) {
			// Update the expiry time of the existing mapping.
			existing.timer = makeTimer()
			existing.expiryTime = newExpiryTime
		}

		// Return the revision for this existing mapping.
		revision = s.mappings[name].revision
	}

	return
}

func (s *DomainInfoStore) GetDomainIPs(domain string) []string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	// All names are stored and looked up as lowercase.
	domain = strings.ToLower(domain)
	ips := s.resultsCache[domain]
	if ips == nil {
		var collectIPsForName func(string, set.Set)
		collectIPsForName = func(name string, collectedNames set.Set) {
			if collectedNames.Contains(name) {
				log.Warningf("%v has a CNAME loop back to itself", name)
				return
			}
			collectedNames.Add(name)
			nameData := s.mappings[name]
			log.WithFields(log.Fields{
				"name":     name,
				"nameData": nameData,
			}).Debug("Collect IPs for name")
			if nameData != nil {
				nameData.namesToNotify.Add(domain)
				for value, valueData := range nameData.values {
					if valueData.isName {
						// The RHS of the mapping is another name, so we recurse to pick up
						// its IPs.
						collectIPsForName(value, collectedNames)
					} else {
						// The RHS of the mapping is an IP, so add it to the list that we
						// will return.
						ips = append(ips, value)
					}
				}
			}
		}
		if isWildcard(domain) {
			regex := s.wildcards[domain]
			if regex == nil {
				// Need to build corresponding regexp.
				regexpString := wildcardToRegexpString(domain)
				var err error
				regex, err = regexp.Compile(regexpString)
				if err != nil {
					log.WithError(err).Panicf("Couldn't compile regexp %v for wildcard %v", regexpString, domain)
				}
				s.wildcards[domain] = regex
			}
			for name := range s.mappings {
				if regex.MatchString(name) {
					collectIPsForName(name, set.New())
				}
			}
		} else {
			collectIPsForName(domain, set.New())
		}
		s.resultsCache[domain] = ips
	}
	log.Debugf("GetDomainIPs(%v) -> %v", domain, ips)
	return ips
}

// GetWatchedDomainForIP returns an (arbitrary) watched domain associated with an IP. The "watch" refers to an explicit
// request to GetDomainIPs.
//
// The signature of this method is somewhat specific to how the collector stores connection data and is used to
// minimize allocations during connection processing.
func (s *DomainInfoStore) GetWatchedDomainForIP(ip [16]byte) string {
	// We only need the read lock to access this data.
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var name string
	if ipData := s.reverse[ip]; ipData != nil {
		ipData.nameDatas.Iter(func(item interface{}) error {
			// Just return the first domain name we find. This should cover the most general case where the user adds
			// a single entry for a particular domain. Return the first "name to notify" that we find.
			nd := item.(*nameData)
			nd.namesToNotify.Iter(func(item2 interface{}) error {
				// Just use the first "name to notify".
				name = item2.(string)
				return set.StopIteration
			})
			if name != "" {
				return set.StopIteration
			}
			return nil
		})
	}
	return name
}

func isWildcard(domain string) bool {
	return strings.Contains(domain, "*")
}

func wildcardToRegexpString(wildcard string) string {
	nonWildParts := strings.Split(wildcard, "*")
	for i := range nonWildParts {
		nonWildParts[i] = regexp.QuoteMeta(nonWildParts[i])
	}
	return "^" + strings.Join(nonWildParts, ".*") + "$"
}

func (s *DomainInfoStore) compileChangedNames(name string) {
	s.changedNames.Add(name)
	delete(s.resultsCache, name)
	if nameData := s.mappings[name]; nameData != nil {
		nameData.namesToNotify.Iter(func(item interface{}) error {
			ancestor := item.(string)
			s.changedNames.Add(ancestor)
			delete(s.resultsCache, ancestor)
			return nil
		})
	}
}

func (s *DomainInfoStore) collectGarbage() (numDeleted int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.gcTrigger {
		// Accumulate the mappings that are still useful.
		namesToKeep := set.New()
		for name, nameData := range s.mappings {
			// A mapping is still useful if it has any unexpired values, because policy
			// might be configured at any moment for that mapping's name, and then we'd
			// want to be able to return the corresponding IPs.
			if len(nameData.values) > 0 {
				namesToKeep.Add(name)
			}
			// A mapping X is also still useful if its name is the RHS of another
			// mapping Y, even if we don't currently have any values for X, because
			// there could be a GetDomainIPs(Y) call, and later a new value for X, and
			// in that case we need to be able to signal that the information for Y has
			// changed.
			for rhs, valueData := range nameData.values {
				if valueData.isName {
					namesToKeep.Add(rhs)
					// There must be a mapping for the RHS name.
					if s.mappings[rhs] == nil {
						log.Panicf("Missing mapping for %v, which is a RHS value for %v", rhs, name)
					}
				}
			}
		}
		// Delete the mappings that are now useless.  Since this mapping contains no values, there can be no
		// corresponding reverse mappings to tidy up.
		for name := range s.mappings {
			if !namesToKeep.Contains(name) {
				log.WithField("name", name).Debug("Delete useless mapping")
				delete(s.mappings, name)
				numDeleted += 1
			}
		}
		// Reset the flag that will trigger the next GC.
		s.gcTrigger = false
	}

	return
}

func (s *DomainInfoStore) processForLatency(ipv4 *layers.IPv4, dns *layers.DNS, timestamp uint64) (latencyIfKnown *time.Duration) {
	if !s.measureLatency {
		return
	}

	if ipv4 == nil {
		// DNS request IDs are not globally unique; we need the IP of the client to scope
		// them.  So, when the packet in hand does not have an IPv4 header, we can't process
		// it for latency.
		return
	}

	var key dnsExchangeKey

	if timestamp == 0 {
		// No timestamp on this packet.
		msgType := "request"
		if dns.QR {
			msgType = "response"
		}
		log.Debugf("DNS-LATENCY: Missing timestamp on DNS %v with ID %v", msgType, dns.ID)
		return
	}

	// From here on we know we have a timestamp for the packet in hand.  It's a number of
	// nanoseconds, measured from some arbitrary point in the past.  (Possibly not from the same
	// base point as time.Time, so don't assume that.)
	if dns.QR == false {
		// It's a request.
		key.clientIP = ipv4.SrcIP.String()
		key.dnsID = dns.ID
		if _, exists := s.requestTimestamp[key]; exists {
			log.Warnf("DNS-LATENCY: Already have outstanding DNS request with ID %v", key)
		} else {
			log.Debugf("DNS-LATENCY: DNS request in hand with ID %v", key)
			s.requestTimestamp[key] = timestamp
		}
	} else {
		// It's a response.
		key.clientIP = ipv4.DstIP.String()
		key.dnsID = dns.ID
		if requestTime, exists := s.requestTimestamp[key]; !exists {
			log.Debugf("DNS-LATENCY: Missed DNS request for response with ID %v", key)
		} else {
			delete(s.requestTimestamp, key)
			latency := timestamp - requestTime
			log.Debugf("DNS-LATENCY: %v ns for ID %v", latency, key)
			latencyAsDuration := time.Duration(latency)
			latencyIfKnown = &latencyAsDuration
		}
	}

	// Check for any request timestamps that are now more than 10 seconds old, and discard those
	// so that our map occupancy does not increase over time.
	for key, requestTime := range s.requestTimestamp {
		if time.Duration(timestamp-requestTime) > 10*time.Second {
			log.Warnf("DNS-LATENCY: Missed DNS response for request with ID %v", key)
			delete(s.requestTimestamp, key)
		}
	}
	return
}
