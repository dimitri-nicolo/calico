// Copyright (c) 2017-2019 Tigera, Inc. All rights reserved.
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
	"net"
	"time"

	"github.com/google/gopacket/layers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Domain Info Store", func() {
	var (
		domainStore *domainInfoStore
		mockDNSRec  = layers.DNSResourceRecord{
			Name:       []byte("abc.com"),
			Type:       layers.DNSTypeA,
			Class:      layers.DNSClassIN,
			TTL:        5,
			DataLength: 4,
			Data:       []byte("10.0.0.10"),
			IP:         net.ParseIP("10.0.0.10"),
		}
		invalidDNSRec = layers.DNSResourceRecord{
			Name:       []byte("abc.com"),
			Type:       layers.DNSTypeA,
			Class:      layers.DNSClassIN,
			TTL:        2147483648,
			DataLength: 4,
			Data:       []byte("999.000.999.000"),
			IP:         net.ParseIP("999.000.999.000"),
		}
	)

	// Program any DNS records if this is a domain type IPSet
	programDNSRecs := func(domainStore *domainInfoStore, dnsPacket layers.DNSResourceRecord) {
		var layerDNS layers.DNS
		layerDNS.Answers = append(layerDNS.Answers, dnsPacket)
		domainStore.processDNSPacket(&layerDNS)
	}

	BeforeEach(func() {
		domainInfoChanges := make(chan *domainInfoChanged, 100)
		domainStore = newDomainInfoStore(domainInfoChanges, "/dnsinfo", time.Duration(time.Minute))
	})

	Describe("receiving a DNS record", func() {
		Context("when the DNS data is valid", func() {
			BeforeEach(func() {
				programDNSRecs(domainStore, mockDNSRec)
			})
			It("should result in a domain entry", func() {
				Expect(domainStore.GetDomainIPs(string(mockDNSRec.Name))).To(Equal(mockDNSRec.IP.String()))
			})
		})

		Context("when the DNS data is invalid", func() {
			BeforeEach(func() {
				programDNSRecs(domainStore, invalidDNSRec)
			})
			It("should return nil", func() {
				Expect(domainStore.GetDomainIPs(string(mockDNSRec.Name))).To(BeNil())
			})
		})
	})
})
