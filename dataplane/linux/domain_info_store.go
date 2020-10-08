// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.
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

package intdataplane

import (
	"bufio"
	"bytes"
	"encoding/binary"
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
	"github.com/tigera/nfnetlink"
	"golang.org/x/sys/unix"

	"github.com/projectcalico/felix/collector"
	fc "github.com/projectcalico/felix/config"
	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/felix/rules"
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

func TimestampToTime(ts []uint8) time.Time {
	log.Debugf("DNS-LATENCY: ts=%v", ts)
	var tv unix.Timeval
	err := binary.Read(bytes.NewReader(ts), binary.BigEndian, &tv)
	if err != nil {
		log.WithError(err).Panic("binary.Read failed")
	}
	log.Debugf("DNS-LATENCY: tv=%v", tv)
	return time.Unix(tv.Sec, tv.Usec*1000)
}

type domainInfoStore struct {
	// Channel that we write to when we want DNS response capture to stop.
	stopChannel chan struct{}

	// Channel on which we receive captured DNS responses (beginning with the IP header).
	msgChannel chan nfnetlink.DataWithTimestamp

	// Channel that we write to when new information is available for a domain name.
	domainInfoChanges chan *domainInfoChanged

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
	timestampExpected bool
	requestTimestamp  map[uint16]time.Time

	// Handling additional DNS mapping lifetime.
	epoch    int
	extraTTL time.Duration
	resetC   chan struct{}
}

// Signal sent by the domain info store to the ipsets manager when the information for a given
// domain name changes.  (i.e. when GetDomainIPs(domain) would return a different set of IP
// addresses.)
type domainInfoChanged struct {
	domain string
	reason string
}

// Signal sent by timers' AfterFunc to the domain info store when a particular name -> IP or name ->
// cname mapping expires.
type domainMappingExpired struct {
	name, value string
}

func newDomainInfoStore(domainInfoChanges chan *domainInfoChanged, config *Config) *domainInfoStore {
	return newDomainInfoStoreWithShims(domainInfoChanges, config, func(expiryTime time.Time) bool {
		return expiryTime.Before(time.Now())
	})
}

func newDomainInfoStoreWithShims(domainInfoChanges chan *domainInfoChanged, config *Config, expiryTimePassed func(time.Time) bool) *domainInfoStore {
	log.Info("Creating domain info store")
	s := &domainInfoStore{
		domainInfoChanges:    domainInfoChanges,
		mappings:             make(map[string]*nameData),
		wildcards:            make(map[string]*regexp.Regexp),
		resultsCache:         make(map[string][]string),
		mappingExpiryChannel: make(chan *domainMappingExpired),
		expiryTimePassed:     expiryTimePassed,
		saveFile:             config.DNSCacheFile,
		saveInterval:         config.DNSCacheSaveInterval,
		gcInterval:           13 * time.Second,
		collector:            config.Collector,
		timestampExpected:    config.DNSLogsLatency,
		requestTimestamp:     make(map[uint16]time.Time),
		epoch:                config.DNSCacheEpoch,
		extraTTL:             config.DNSExtraTTL,
		resetC:               make(chan struct{}),
	}
	return s
}

func (s *domainInfoStore) Start() {
	log.Info("Starting domain info store")

	// Use nfnetlink to capture DNS response packets.
	s.stopChannel = make(chan struct{})
	// Use a buffered channel here with reasonable capacity, so that the nfnetlink capture
	// thread can handle a burst of DNS response packets without becoming blocked by the reading
	// thread here.  Specifically we say 1000 because that what's we use for flow logs, so we
	// know that works; even though we probably won't need so much capacity for the DNS case.
	s.msgChannel = make(chan nfnetlink.DataWithTimestamp, 1000)
	nfnetlink.SubscribeDNS(int(rules.NFLOGDomainGroup), 65535, s.msgChannel, s.stopChannel)

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
func (s *domainInfoStore) OnUpdate(msg interface{}) {
	switch msg := msg.(type) {
	case *proto.ConfigUpdate:
		felixConfig := fc.New()
		felixConfig.UpdateFrom(msg.Config, fc.DatastorePerHost)
		s.mutex.Lock()
		defer s.mutex.Unlock()
		if _, specified := msg.Config["DNSCacheEpoch"]; specified && felixConfig.DNSCacheEpoch != s.epoch {
			s.epoch = felixConfig.DNSCacheEpoch
			s.resetC <- struct{}{}
		}
		if _, specified := msg.Config["DNSExtraTTL"]; specified && felixConfig.DNSExtraTTL != s.extraTTL {
			s.extraTTL = felixConfig.DNSExtraTTL
		}
	}
}

func (s *domainInfoStore) CompleteDeferredWork() error {
	// Nothing to do, we don't defer any work.
	return nil
}

func (s *domainInfoStore) loop(saveTimerC, gcTimerC <-chan time.Time) {
	for {
		select {
		case msg := <-s.msgChannel:
			// TODO: Test and fix handling of DNS over IPv6.  The `layers.LayerTypeIPv4`
			// in the next line is clearly a v4 assumption, and some of the code inside
			// `nfnetlink.SubscribeDNS` also looks v4-specific.
			packet := gopacket.NewPacket(msg.Data, layers.LayerTypeIPv4, gopacket.Lazy)
			if ipv4, ok := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4); ok {
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
				latencyIfKnown := s.processForLatency(dns, msg.Timestamp)
				if dns.QR == true {
					// It's a DNS response.
					if s.collector != nil {
						if ipv4, ok := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4); ok {
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
			if err := s.saveMappingsV1(); err != nil {
				log.WithError(err).Warning("Failed to save mappings to file")
			}
		case <-gcTimerC:
			_ = s.collectGarbage()
		case <-s.resetC:
			s.expireAllMappings()
		}
	}
}

type jsonMappingV1 struct {
	LHS    string
	RHS    string
	Expiry string
	Type   string
}

func (s *domainInfoStore) readMappings() error {
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

func (s *domainInfoStore) readMappingsV1(scanner *bufio.Scanner) error {
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

func (s *domainInfoStore) saveMappingsV1() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	log.WithField("file", s.saveFile).Info("Saving DNS mappings...")

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

	log.WithField("file", s.saveFile).Info("Finished saving DNS mappings")

	return nil
}

func (s *domainInfoStore) processDNSPacket(dns *layers.DNS) {
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

func (s *domainInfoStore) processMappingExpiry(name, value string) {
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

func (s *domainInfoStore) expireAllMappings() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// For each mapping...
	for name := range s.mappings {
		// ...discard all of its values.
		s.mappings[name].values = make(map[string]*valueData)
		s.signalDomainInfoChange(name, "epoch change")
	}

	// Trigger a GC to reclaim the memory that we can.
	s.gcTrigger = true
}

func (s *domainInfoStore) storeDNSRecordInfo(rec *layers.DNSResourceRecord, section string) {
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

func (s *domainInfoStore) storeInfo(name, value string, ttl time.Duration, isName bool) {
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
		return time.AfterFunc(ttl, func() {
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

func (s *domainInfoStore) GetDomainIPs(domain string) []string {
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

func (s *domainInfoStore) signalDomainInfoChange(name, reason string) {
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
		s.domainInfoChanges <- &domainInfoChanged{domain: item.(string), reason: reason}
		return nil
	})
}

func (s *domainInfoStore) collectGarbage() (numDeleted int) {
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

func (s *domainInfoStore) processForLatency(dns *layers.DNS, timestamp []uint8) (latencyIfKnown *time.Duration) {
	if len(timestamp) == 0 {
		// No timestamp on this packet.
		if s.timestampExpected {
			msgType := "request"
			if dns.QR {
				msgType = "response"
			}
			log.Warnf("DNS-LATENCY: Missing timestamp on DNS %v with ID %v", msgType, dns.ID)
		}
	} else if dns.QR == false {
		// It's a request.
		if _, exists := s.requestTimestamp[dns.ID]; exists {
			log.Warnf("DNS-LATENCY: Already have outstanding DNS request with ID %v", dns.ID)
		} else {
			log.Debugf("DNS-LATENCY: DNS request in hand with ID %v", dns.ID)
			s.requestTimestamp[dns.ID] = TimestampToTime(timestamp)
		}
	} else {
		// It's a response.
		if requestTime, exists := s.requestTimestamp[dns.ID]; exists {
			latency := TimestampToTime(timestamp).Sub(requestTime)
			log.Debugf("DNS-LATENCY: %v for ID %v", latency, dns.ID)
			delete(s.requestTimestamp, dns.ID)
			latencyIfKnown = &latency
		} else {
			log.Warnf("DNS-LATENCY: Missed DNS request/timestamp for response with ID %v", dns.ID)
		}
	}

	// Check for any request timestamps that are now unfeasibly old, and discard those so that
	// our map occupancy does not increase over time.
	for id, requestTime := range s.requestTimestamp {
		if time.Since(requestTime) > 10*time.Second {
			log.Warnf("DNS-LATENCY: Missed DNS response for request with ID %v", id)
			delete(s.requestTimestamp, dns.ID)
		}
	}
	return
}
