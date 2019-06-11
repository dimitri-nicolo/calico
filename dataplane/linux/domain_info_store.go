// Copyright (c) 2019 Tigera, Inc. All rights reserved.
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
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/nfnetlink"

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
	// Known values for this name.  Map keys are the actual values (i.e. IPs or alias names),
	// and valueData is as above.
	values map[string]*valueData
	// Other names whose cached IP list should be invalidated when the info for _this_ name
	// changes.
	ancestorNames []string
}

type domainInfoStore struct {
	// Channel that we write to when we want DNS response capture to stop.
	stopChannel chan struct{}

	// Channel on which we receive captured DNS responses (beginning with the IP header).
	msgChannel chan []byte

	// Channel that we write to when new information is available for a domain name.
	domainInfoChanges chan *domainInfoChanged

	// Stores for the information that we glean from DNS responses.  Note: IPs are held here as
	// strings, and also passed to the ipsets manager as strings.
	mutex    sync.Mutex
	mappings map[string]*nameData

	// Cache for "what are the IPs for <domain>?".  We have this to halve our processing,
	// because there are two copies of the IPSets Manager (one for v4 and one for v6) that will
	// call us to make identical queries.
	resultsCache map[string][]string

	// Channel for domain mapping expiry signals.
	mappingExpiryChannel chan *domainMappingExpired

	// Persistence.
	saveFile     string
	saveInterval time.Duration
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

func newDomainInfoStore(domainInfoChanges chan *domainInfoChanged, saveFile string, saveInterval time.Duration) *domainInfoStore {
	log.Info("Creating domain info store")
	s := &domainInfoStore{
		domainInfoChanges:    domainInfoChanges,
		mappings:             make(map[string]*nameData),
		resultsCache:         make(map[string][]string),
		mappingExpiryChannel: make(chan *domainMappingExpired),
		saveFile:             saveFile,
		saveInterval:         saveInterval,
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
	s.msgChannel = make(chan []byte, 1000)
	nfnetlink.SubscribeDNS(int(rules.NFLOGDomainGroup), 65535, s.msgChannel, s.stopChannel)

	// Ensure that the directory for the persistent file exists.
	if err := os.MkdirAll(path.Dir(s.saveFile), 0755); err != nil {
		log.WithError(err).Fatal("Failed to create persistent file dir")
	}

	// Read mappings from the persistent file (if it exists).
	if err := s.readMappings(); err != nil {
		log.WithError(err).Warning("Failed to read mappings from file")
	}

	// Start a repeating timer for periodically saving DNS info to a persistent file.
	saveTimerC := time.NewTicker(s.saveInterval).C

	go s.loop(saveTimerC)
}

func (s *domainInfoStore) loop(saveTimerC <-chan time.Time) {
	for {
		select {
		case msg := <-s.msgChannel:
			packet := gopacket.NewPacket(msg, layers.LayerTypeIPv4, gopacket.Default)
			if dnsLayer := packet.Layer(layers.LayerTypeDNS); dnsLayer != nil {
				dns, _ := dnsLayer.(*layers.DNS)
				s.processDNSPacket(dns)
			}
		case expiry := <-s.mappingExpiryChannel:
			s.processMappingExpiry(expiry.name, expiry.value)
		case <-saveTimerC:
			if err := s.saveMappingsV1(); err != nil {
				log.WithError(err).Warning("Failed to save mappings to file")
			}
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
			s.storeInfo(jsonMapping.LHS, jsonMapping.RHS, ttlNow, jsonMapping.Type == v1TypeName)
		} else {
			log.Debugf("Ignore expired mapping %v", jsonMapping)
		}
	}
	return scanner.Err()
}

func (s *domainInfoStore) saveMappingsV1() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

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

	// Sync and close the temporary save file.
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	fileAlreadyClosed = true

	// Move that file to the non-temporary name.
	if err = os.Rename(tmpSaveFile, s.saveFile); err != nil {
		return err
	}

	return nil
}

func (s *domainInfoStore) processDNSPacket(dns *layers.DNS) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
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
		if valueData := nameData.values[value]; (valueData != nil) && valueData.expiryTime.Before(time.Now()) {
			log.Debugf("Mapping expiry for %v -> %v", name, value)
			delete(nameData.values, value)
			if len(nameData.values)+len(nameData.ancestorNames) == 0 {
				delete(s.mappings, name)
			}
			s.signalDomainInfoChange(name, "mapping expired")
		} else if valueData != nil {
			log.Debugf("Too early mapping expiry for %v -> %v", name, value)
		} else {
			log.Debugf("Mapping already gone for %v -> %v", name, value)
		}
	}
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

	switch rec.Type {
	case layers.DNSTypeA:
		log.Debugf("A: %v -> %v with TTL %v (%v)",
			string(rec.Name),
			rec.IP,
			rec.TTL,
			section,
		)
		s.storeInfo(string(rec.Name), rec.IP.String(), time.Duration(rec.TTL)*time.Second, false)
	case layers.DNSTypeAAAA:
		log.Debugf("AAAA: %v -> %v with TTL %v (%v)",
			string(rec.Name),
			rec.IP,
			rec.TTL,
			section,
		)
		s.storeInfo(string(rec.Name), rec.IP.String(), time.Duration(rec.TTL)*time.Second, false)
	case layers.DNSTypeCNAME:
		log.Debugf("CNAME: %v -> %v with TTL %v (%v)",
			string(rec.Name),
			string(rec.CNAME),
			rec.TTL,
			section,
		)
		s.storeInfo(string(rec.Name), string(rec.CNAME), time.Duration(rec.TTL)*time.Second, true)
	default:
		log.Debugf("Ignore DNS response with type %v", rec.Type)
	}

	return
}

func (s *domainInfoStore) storeInfo(name, value string, ttl time.Duration, isName bool) {
	makeTimer := func() *time.Timer {
		return time.AfterFunc(ttl, func() {
			s.mappingExpiryChannel <- &domainMappingExpired{name: name, value: value}
		})
	}
	if s.mappings[name] == nil {
		s.mappings[name] = &nameData{values: make(map[string]*valueData)}
	}
	existing := s.mappings[name].values[value]
	if existing == nil {
		s.mappings[name].values[value] = &valueData{
			expiryTime: time.Now().Add(ttl),
			timer:      makeTimer(),
			isName:     isName,
		}
		s.signalDomainInfoChange(name, "mapping added")
		// If value is another name, for which we don't yet have any information, create a
		// mapping entry for it so we can record that it is a descendant of the name in
		// hand.  Then, when we get information for the descendant name, we can correctly
		// signal changes for the name in hand and any of its ancestors.
		if isName && s.mappings[value] == nil {
			s.mappings[value] = &nameData{values: make(map[string]*valueData)}
		}
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
	ips := s.resultsCache[domain]
	if ips == nil {
		var collectIPsForName func(string, []string)
		collectIPsForName = func(domain string, ancestorNames []string) {
			nameData := s.mappings[domain]
			log.WithFields(log.Fields{
				"domain":        domain,
				"ancestorNames": ancestorNames,
				"nameData":      nameData,
			}).Debug("Collect IPs for name")
			if nameData != nil {
				nameData.ancestorNames = ancestorNames
				for value, valueData := range nameData.values {
					if valueData.isName {
						// The RHS of the mapping is another name, so we recurse to pick up
						// its IPs.
						collectIPsForName(value, append(ancestorNames, domain))
					} else {
						// The RHS of the mapping is an IP, so add it to the list that we
						// will return.
						ips = append(ips, value)
					}
				}
			}
		}
		collectIPsForName(domain, nil)
		s.resultsCache[domain] = ips
	}
	log.Infof("GetDomainIPs(%v) -> %v", domain, ips)
	return ips
}

func (s *domainInfoStore) signalDomainInfoChange(name, reason string) {
	changedNames := set.From(name)
	delete(s.resultsCache, name)
	if nameData := s.mappings[name]; nameData != nil {
		for _, ancestor := range nameData.ancestorNames {
			changedNames.Add(ancestor)
			delete(s.resultsCache, ancestor)
		}
	}
	// Release the mutex to send change signals, so that we can't get a deadlock between this
	// thread and the int_dataplane thread.
	s.mutex.Unlock()
	defer s.mutex.Lock()
	changedNames.Iter(func(item interface{}) error {
		s.domainInfoChanges <- &domainInfoChanged{domain: item.(string), reason: reason}
		return nil
	})
}
