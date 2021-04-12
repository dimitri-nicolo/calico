// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.
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

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/collector"
	fc "github.com/projectcalico/felix/config"
	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/libcalico-go/lib/set"
)

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
	// Other names that we should notify a "change of information" for, and whose cached IP list
	// should be invalidated, when the info for _this_ name changes.
	namesToNotify set.Set
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
}

type DomainInfoStore struct {
	// Channel that we write to when we want DNS response capture to stop.
	stopChannel chan struct{}

	// Channel on which we receive captured DNS responses (beginning with the IP header).
	MsgChannel chan DataWithTimestamp

	// Channel that we write to when new information is available for a domain name.
	domainInfoChanges chan *DomainInfoChanged

	// Stores for the information that we glean from DNS responses.  Note: IPs are held here as
	// strings, and also passed to the ipsets manager as strings.
	mutex    sync.Mutex
	mappings map[string]*nameData

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
}

// Signal sent by the domain info store to the ipsets manager when the information for a given
// domain name changes.  (i.e. when GetDomainIPs(domain) would return a different set of IP
// addresses.)
type DomainInfoChanged struct {
	Domain string
	Reason string
}

// Signal sent by timers' AfterFunc to the domain info store when a particular name -> IP or name ->
// cname mapping expires.
type domainMappingExpired struct {
	name, value string
}

type DnsConfig struct {
	Collector            collector.Collector
	DNSCacheEpoch        int
	DNSCacheFile         string
	DNSCacheSaveInterval time.Duration
	DNSExtraTTL          time.Duration
	DNSLogsLatency       bool
}

func NewDomainInfoStore(domainInfoChanges chan *DomainInfoChanged, config *DnsConfig) *DomainInfoStore {
	return newDomainInfoStoreWithShims(
		domainInfoChanges,
		config,
		time.AfterFunc,
		func(expiryTime time.Time) bool {
			return expiryTime.Before(time.Now())
		})
}

func newDomainInfoStoreWithShims(
	domainInfoChanges chan *DomainInfoChanged,
	config *DnsConfig,
	makeExpiryTimer func(time.Duration, func()) *time.Timer,
	expiryTimePassed func(time.Time) bool,
) *DomainInfoStore {
	log.Info("Creating domain info store")
	s := &DomainInfoStore{
		domainInfoChanges:    domainInfoChanges,
		mappings:             make(map[string]*nameData),
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
		// Capacity 1 here is to allow UT to test the use of this channel without
		// needing goroutines.
		resetC: make(chan struct{}, 1),
	}
	return s
}

func (s *DomainInfoStore) Start() {
	log.Info("Starting domain info store")

	// Use a buffered channel here with reasonable capacity, so that the nfnetlink capture
	// thread can handle a burst of DNS response packets without becoming blocked by the reading
	// thread here.  Specifically we say 1000 because that what's we use for flow logs, so we
	// know that works; even though we probably won't need so much capacity for the DNS case.
	s.MsgChannel = make(chan DataWithTimestamp, 1000)

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
		newExtraTTL := felixConfig.DNSExtraTTL
		if newExtraTTL != s.extraTTL {
			log.Infof("Extra TTL is now %v", newExtraTTL)
			s.extraTTL = newExtraTTL
		}
	}
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
	case msg := <-s.MsgChannel:
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
		err := dns.DecodeFromBytes(packet.TransportLayer().LayerPayload(), gopacket.NilDecodeFeedback)
		if err == nil {
			latencyIfKnown := s.processForLatency(ipv4, dns, msg.Timestamp)
			if dns.QR == true {
				// It's a DNS response.
				if s.collector != nil {
					if ipv4 != nil {
						s.collector.LogDNS(ipv4.SrcIP, ipv4.DstIP, dns, latencyIfKnown)
					} else {
						log.Warning("Not logging non-IPv4 DNS packet")
					}
				}
				s.processDNSPacket(dns)
			}
		} else {
			log.WithError(err).Debug("No DNS layer")
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

type jsonMappingV1 struct {
	LHS    string
	RHS    string
	Expiry string
	Type   string
}

func (s *DomainInfoStore) readMappings() error {
	// This happens before the domain info store thread is started, so we don't need locking for
	// concurrency reasons.  But we do need to lock the mutex because we'll be calling through
	// to subroutines that assume it's locked and briefly unlock it.
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

func (s *DomainInfoStore) processDNSPacket(dns *layers.DNS) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	log.Debugf("DNS packet with %v answers %v additionals", len(dns.Answers), len(dns.Additionals))
	for _, rec := range dns.Answers {
		s.storeDNSRecordInfo(&rec, "answer")
	}
	for _, rec := range dns.Additionals {
		s.storeDNSRecordInfo(&rec, "additional")
	}
}

func (s *DomainInfoStore) processMappingExpiry(name, value string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if nameData := s.mappings[name]; nameData != nil {
		if valueData := nameData.values[value]; (valueData != nil) && s.expiryTimePassed(valueData.expiryTime) {
			log.Debugf("Mapping expiry for %v -> %v", name, value)
			delete(nameData.values, value)
			s.gcTrigger = true
			s.signalDomainInfoChange(name, "mapping expired")
		} else if valueData != nil {
			log.Debugf("Too early mapping expiry for %v -> %v", name, value)
		} else {
			log.Debugf("Mapping already gone for %v -> %v", name, value)
		}
	}
}

func (s *DomainInfoStore) expireAllMappings() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	log.Info("Expire all mappings")

	// For each mapping...
	for name := range s.mappings {
		// ...discard all of its values.
		s.mappings[name].values = make(map[string]*valueData)
		s.signalDomainInfoChange(name, "epoch change")
	}

	// Trigger a GC to reclaim the memory that we can.
	s.gcTrigger = true
}

func (s *DomainInfoStore) storeDNSRecordInfo(rec *layers.DNSResourceRecord, section string) {
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
		s.storeInfo(name, rec.IP.String(), time.Duration(rec.TTL)*time.Second, false)
	case layers.DNSTypeAAAA:
		log.Debugf("AAAA: %v -> %v with TTL %v (%v)",
			name,
			rec.IP,
			rec.TTL,
			section,
		)
		s.storeInfo(name, rec.IP.String(), time.Duration(rec.TTL)*time.Second, false)
	case layers.DNSTypeCNAME:
		cname := strings.ToLower(string(rec.CNAME))
		log.Debugf("CNAME: %v -> %v with TTL %v (%v)",
			name,
			cname,
			rec.TTL,
			section,
		)
		s.storeInfo(name, cname, time.Duration(rec.TTL)*time.Second, true)
	default:
		log.Debugf("Ignore DNS response with type %v", rec.Type)
	}

	return
}

func (s *DomainInfoStore) storeInfo(name, value string, ttl time.Duration, isName bool) {
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

		// If value is another name, for which we don't yet have any information, create a
		// mapping entry for it so we can record that it is a descendant of the name in
		// hand.  Then, when we get information for the descendant name, we can correctly
		// signal changes for the name in hand and any of its ancestors.
		if isName && s.mappings[value] == nil {
			s.mappings[value] = &nameData{
				values:        make(map[string]*valueData),
				namesToNotify: set.New(),
			}
		}

		// Now signal that the available domain info for this name has changed.  It's
		// important, in order to preserve the correctness of mapping entries knowing the
		// names that they should notify for, to do this _after_ the previous block that
		// creates a mapping entry for a CNAME value, because signalDomainInfoChange
		// releases the mutex and so allows other goroutines to call GetDomainIPs for one of
		// the domain names that is signaled as changed.
		//
		// With the ordering here:
		// - the CNAME value mapping <entry> must exist before GetDomainIPs(<ancestor>) is
		//   called
		// - therefore, GetDomainIPs(<ancestor>) will update <entry>.namesToNotify so that
		//   it includes <ancestor>
		// - then, when further information is learned for <entry>, <ancestor> will be
		//   signaled again.
		//
		// On the other hand, if we signaled before creating the CNAME value mapping
		// <entry>, this would be possible:
		// - GetDomainIPs(<ancestor>) is called (in response to the current signaling)
		// - <entry> does not exist yet, so its namesToNotify is not updated
		// - now this thread creates <entry>, with an empty set for namesToNotify
		// - when further information is learned for <entry>, <ancestor> will not be
		//   signaled, unless there has been a further intervening call to
		//   GetDomainIPs(<ancestor>).  (And such a call would only happen if there was an
		//   independent update to one of the mappings in the tree below <ancestor>, which
		//   is unlikely.)
		s.signalDomainInfoChange(name, "mapping added")
	} else {
		newExpiryTime := time.Now().Add(ttl)
		if newExpiryTime.After(existing.expiryTime) {
			// Update the expiry time of the existing mapping.
			existing.timer = makeTimer()
			existing.expiryTime = newExpiryTime
		}
	}
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
	log.Infof("GetDomainIPs(%v) -> %v", domain, ips)
	return ips
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

func (s *DomainInfoStore) signalDomainInfoChange(name, reason string) {
	changedNames := set.From(name)
	delete(s.resultsCache, name)
	if nameData := s.mappings[name]; nameData != nil {
		nameData.namesToNotify.Iter(func(item interface{}) error {
			ancestor := item.(string)
			changedNames.Add(ancestor)
			delete(s.resultsCache, ancestor)
			return nil
		})
	}
	// Release the mutex to send change signals, so that we can't get a deadlock between this
	// thread and the int_dataplane thread.
	s.mutex.Unlock()
	defer s.mutex.Lock()
	changedNames.Iter(func(item interface{}) error {
		log.Debugf("Signal domain change for %v -> %v", name, item.(string))
		s.domainInfoChanges <- &DomainInfoChanged{Domain: item.(string), Reason: reason}
		return nil
	})
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
			// A mapping X is also still useful it its name is the RHS of another
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
		// Delete the mappings that are now useless.
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
		log.Warnf("DNS-LATENCY: Missing timestamp on DNS %v with ID %v", msgType, dns.ID)
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
			log.Warnf("DNS-LATENCY: Missed DNS request for response with ID %v", key)
		} else {
			delete(s.requestTimestamp, key)
			if requestTime == 0 {
				log.Debug("DNS-LATENCY: Can't calculate latency because timestamp was missing on request")
			} else if timestamp == 0 {
				log.Debug("DNS-LATENCY: Can't calculate latency because timestamp is missing on response")
			} else {
				latency := timestamp - requestTime
				log.Debugf("DNS-LATENCY: %v ns for ID %v", latency, key)
				latencyAsDuration := time.Duration(latency)
				latencyIfKnown = &latencyAsDuration
			}
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
