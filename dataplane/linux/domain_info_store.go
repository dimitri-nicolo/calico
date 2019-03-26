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
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/projectcalico/felix/rules"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/nfnetlink"
)

type timeout struct {
	expiryTime time.Time
	timer      *time.Timer
}

type domainInfoStore struct {
	// Channel that we write to when we want DNS response capture
	// to stop.
	stopChannel chan struct{}

	// Channel on which we receive captured DNS responses
	// (beginning with the IP header).
	msgChannel chan []byte

	// Channel that we write to when new information is available
	// for a domain name.
	domainInfoChanges chan *domainInfoChanged

	// Stores for the information that we glean from DNS responses.
	cnameInfo map[string]map[string]*timeout
	aInfo     map[string]map[string]*timeout
	aaaaInfo  map[string]map[string]*timeout

	// Cache for "what are the IPs for <domain>?".  We have this to halve our processing,
	// because there are two copies of the IPSets Manager (one for v4 and one for v6) that will
	// call us to make identical queries.
	resultsCache map[string][]string
}

func newDomainInfoStore(domainInfoChanges chan *domainInfoChanged, ipv6Enabled bool) *domainInfoStore {
	log.Info("Creating domain info store")
	s := &domainInfoStore{
		domainInfoChanges: domainInfoChanges,
		cnameInfo:         make(map[string]map[string]*timeout),
		aInfo:             make(map[string]map[string]*timeout),
		aaaaInfo:          make(map[string]map[string]*timeout),
		resultsCache:      make(map[string][]string),
	}
	return s
}

func (s *domainInfoStore) Start() {
	log.Info("Starting domain info store")
	s.stopChannel = make(chan struct{})
	s.msgChannel = make(chan []byte)
	nfnetlink.SubscribeDNS(int(rules.NFLOGDomainGroup), 65535, s.msgChannel, s.stopChannel)
	go s.processDNSPackets()
}

func (s *domainInfoStore) processDNSPackets() {
	for msg := range s.msgChannel {
		packet := gopacket.NewPacket(msg, layers.LayerTypeIPv4, gopacket.Default)
		if dnsLayer := packet.Layer(layers.LayerTypeDNS); dnsLayer != nil {
			dns, _ := dnsLayer.(*layers.DNS)
			for _, rec := range dns.Answers {
				s.storeDNSRecordInfo(&rec, "answer")
			}
			for _, rec := range dns.Additionals {
				s.storeDNSRecordInfo(&rec, "additional")
			}
		}
	}
}

func (s *domainInfoStore) storeDNSRecordInfo(rec *layers.DNSResourceRecord, section string) {
	if rec.Class != layers.DNSClassIN {
		log.Debugf("Ignore DNS response with class %v", rec.Class)
		return
	}

	switch rec.Type {
	case layers.DNSTypeA:
		log.Infof("A: %v -> %v with TTL %v (%v)",
			string(rec.Name),
			rec.IP,
			rec.TTL,
			section,
		)
		s.storeInfo(s.aInfo, string(rec.Name), rec.IP.String(), time.Duration(rec.TTL)*time.Second)
	case layers.DNSTypeAAAA:
		log.Infof("AAAA: %v -> %v with TTL %v (%v)",
			string(rec.Name),
			rec.IP,
			rec.TTL,
			section,
		)
		s.storeInfo(s.aaaaInfo, string(rec.Name), rec.IP.String(), time.Duration(rec.TTL)*time.Second)
	case layers.DNSTypeCNAME:
		log.Infof("CNAME: %v -> %v with TTL %v (%v)",
			string(rec.Name),
			string(rec.CNAME),
			rec.TTL,
			section,
		)
		s.storeInfo(s.cnameInfo, string(rec.Name), string(rec.CNAME), time.Duration(rec.TTL)*time.Second)
	default:
		log.Debugf("Ignore DNS response with type %v", rec.Type)
	}

	return
}

func (s *domainInfoStore) storeInfo(infoMap map[string]map[string]*timeout, name, value string, ttl time.Duration) {
	if infoMap[name] == nil {
		infoMap[name] = make(map[string]*timeout)
	}
	existing := infoMap[name][value]
	if existing == nil {
		infoMap[name][value] = &timeout{
			expiryTime: time.Now().Add(ttl),
			timer: time.AfterFunc(ttl, func() {
				delete(infoMap[name], value)
				s.signalDomainInfoChange(name)
			}),
		}
	} else {
		newExpiryTime := time.Now().Add(ttl)
		if newExpiryTime.After(existing.expiryTime) {
			// Update the expiry time of the existing mapping.
			existing.timer.Reset(ttl)
			existing.expiryTime = newExpiryTime
		}
	}
}

func (s *domainInfoStore) GetDomainIPs(domain string) []string {
	cachedIPs := s.resultsCache[domain]
	if cachedIPs != nil {
		return cachedIPs
	}
	ips := []string{}
	addResult := func(ip string) {
		ips = append(ips, ip)
	}
	var handlePossibleAlias func(domain string)
	handlePossibleAlias = func(domain string) {
		if len(s.cnameInfo[domain]) > 0 {
			// domain is an alias (i.e. the LHS of a CNAME record).  We say 'cname' here
			// for the RHS of the stored record, but in fact that might be an alias too,
			// so we recurse to check it.
			for cname := range s.cnameInfo[domain] {
				handlePossibleAlias(cname)
			}
		} else {
			for ipv4 := range s.aInfo[domain] {
				addResult(ipv4)
			}
			for ipv6 := range s.aaaaInfo[domain] {
				addResult(ipv6)
			}
		}
	}
	handlePossibleAlias(domain)
	return ips
}

func (s *domainInfoStore) signalDomainInfoChange(name string) {
	// Discard the results cache.
	//
	// TODO: Something more clever; should just invalid the entries for <name> and all those
	// that depend on <name>.
	s.resultsCache = make(map[string][]string)
	s.domainInfoChanges <- &domainInfoChanged{domain: name}
}
