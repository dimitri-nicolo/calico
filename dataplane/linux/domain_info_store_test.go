// Copyright (c) 2019 Tigera, Inc. All rights reserved.

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
			TTL:        0,
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

	AssertDomainChanged := func(domainStore *domainInfoStore, d string, r string) {
		receivedInfo := <-domainStore.domainInfoChanges
		Expect(receivedInfo).To(Equal(&domainInfoChanged{domain: d, reason: r}))
	}

	BeforeEach(func() {
		domainChannel := make(chan *domainInfoChanged, 100)
		domainStore = newDomainInfoStore(domainChannel, "/dnsinfo", time.Minute)
	})

	Describe("receiving a DNS packet", func() {
		Context("when the DNS record is valid", func() {
			BeforeEach(func() {
				programDNSRecs(domainStore, mockDNSRec)
				AssertDomainChanged(domainStore, string(mockDNSRec.Name), "mapping added")
			})
			It("should result in a domain entry", func() {
				Expect(domainStore.GetDomainIPs(string(mockDNSRec.Name))).To(Equal([]string{mockDNSRec.IP.String()}))
			})
			It("should expire and signal a domain change", func() {
				domainStore.processMappingExpiry(string(mockDNSRec.Name), mockDNSRec.IP.String())
				AssertDomainChanged(domainStore, string(mockDNSRec.Name), "mapping expired")
			})
		})

		Context("when the DNS record is invalid", func() {
			BeforeEach(func() {
				programDNSRecs(domainStore, invalidDNSRec)
			})
			It("should return nil", func() {
				Expect(domainStore.GetDomainIPs(string(mockDNSRec.Name))).To(BeNil())
			})
		})
	})
})
