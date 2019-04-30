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

	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/libcalico-go/lib/set"
)

// For now, we only need four IPSets to stage different scenarios around adding, removing and updating members.
const numMembers int = 4

// Basic structure for a test case. The idea is to have at least one for each IPSetType.
type IPSetsMgrTestCase struct {
	ipsetID      string
	ipsetType    proto.IPSetUpdate_IPSetType
	ipsetMembers [numMembers]string
	dnsRecs      [numMembers]layers.DNSResourceRecord
}

// Main array of test cases. We pass each of these to the test routines during execution.
var ipsetsMgrTestCases = []IPSetsMgrTestCase{
	{
		ipsetID:      "id1",
		ipsetType:    proto.IPSetUpdate_IP,
		ipsetMembers: [numMembers]string{"10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4"},
		dnsRecs:      [numMembers]layers.DNSResourceRecord{},
	},
	{
		ipsetID:      "id2",
		ipsetType:    proto.IPSetUpdate_DOMAIN,
		ipsetMembers: [numMembers]string{"abc.com", "def.com", "ghi.com", "jkl.com"},
		dnsRecs: [numMembers]layers.DNSResourceRecord{
			{
				Name:       []byte("abc.com"),
				Type:       layers.DNSTypeA,
				Class:      layers.DNSClassIN,
				TTL:        5,
				DataLength: 4,
				Data:       []byte("10.0.0.10"),
				IP:         net.ParseIP("10.0.0.10"),
			},
			{
				Name:       []byte("def.com"),
				Type:       layers.DNSTypeA,
				Class:      layers.DNSClassIN,
				TTL:        5,
				DataLength: 4,
				Data:       []byte("10.0.0.20"),
				IP:         net.ParseIP("10.0.0.20"),
			},
			{
				Name:       []byte("ghi.com"),
				Type:       layers.DNSTypeA,
				Class:      layers.DNSClassIN,
				TTL:        5,
				DataLength: 4,
				Data:       []byte("10.0.0.30"),
				IP:         net.ParseIP("10.0.0.30"),
			},
			{
				Name:       []byte("jkl.com"),
				Type:       layers.DNSTypeA,
				Class:      layers.DNSClassIN,
				TTL:        5,
				DataLength: 4,
				Data:       []byte("10.0.0.40"),
				IP:         net.ParseIP("10.0.0.40"),
			},
		},
	},
}

// Program any DNS records if this is a domain type IPSet
func programDNSRecs(ipsetType proto.IPSetUpdate_IPSetType, domainStore *domainInfoStore, dnsPackets [numMembers]layers.DNSResourceRecord) {
	if ipsetType == proto.IPSetUpdate_DOMAIN {
		var layerDNS layers.DNS
		for _, d := range dnsPackets {
			layerDNS.Answers = append(layerDNS.Answers, d)
		}
		domainStore.processDNSPacket(&layerDNS)
	}
}

var _ = Describe("IP Sets manager", func() {
	var (
		ipsetsMgr   *ipSetsManager
		ipSets      *mockIPSets
		domainStore *domainInfoStore
	)

	BeforeEach(func() {
		domainInfoChanges := make(chan *domainInfoChanged, 100)
		domainStore = newDomainInfoStore(domainInfoChanges, "/dnsinfo", time.Duration(time.Minute))
		ipSets = newMockIPSets()
		ipsetsMgr = newIPSetsManager(ipSets, 1024, domainStore)
	})

	// Generic assumptions used during tests. Having them here reduces code duplication and improves readability.
	AssertIPSetMembers := func(id string, members []string) {
		It("IPSet should have the right members", func() {
			Expect(ipSets.Members[id]).To(Equal(set.FromArray(members)))
		})
	}

	AssertIPSetNoMembers := func(id string) {
		It("IPSet should have no members", func() {
			Expect(ipSets.Members[id]).To(BeNil())
		})
	}

	AssertIPSetModified := func() {
		It("IPSet should be modified", func() {
			Expect(ipSets.AddOrReplaceCalled).To(BeTrue())
		})
	}

	AssertIPSetNotModified := func() {
		It("IPSet should not be modified", func() {
			Expect(ipSets.AddOrReplaceCalled).To(BeFalse())
		})
	}

	// Basic add/remove/update test case for different types of IPSets.
	IPsetsMgrTest1 := func(ipsetID string, ipsetType proto.IPSetUpdate_IPSetType, members [numMembers]string, dnsPackets [numMembers]layers.DNSResourceRecord) {
		Describe("after creating an IPSet", func() {
			BeforeEach(func() {
				ipSets.AddOrReplaceCalled = false
				programDNSRecs(ipsetType, domainStore, dnsPackets)
				ipsetsMgr.OnUpdate(&proto.IPSetUpdate{
					Id:      ipsetID,
					Members: []string{members[0], members[1]},
					Type:    ipsetType,
				})
				ipsetsMgr.CompleteDeferredWork()
			})

			AssertIPSetModified()

			// We match domain ipsets with their respective IP addresses.
			if ipsetType == proto.IPSetUpdate_DOMAIN {
				AssertIPSetMembers(ipsetID, []string{dnsPackets[0].IP.String(), dnsPackets[1].IP.String()})
			} else {
				AssertIPSetMembers(ipsetID, []string{members[0], members[1]})
			}

			Describe("after sending a delta update", func() {
				BeforeEach(func() {
					ipSets.AddOrReplaceCalled = false
					programDNSRecs(ipsetType, domainStore, dnsPackets)
					ipsetsMgr.OnUpdate(&proto.IPSetDeltaUpdate{
						Id:             ipsetID,
						AddedMembers:   []string{members[2], members[3]},
						RemovedMembers: []string{members[0]},
					})
					ipsetsMgr.CompleteDeferredWork()
				})

				AssertIPSetNotModified()

				if ipsetType == proto.IPSetUpdate_DOMAIN {
					AssertIPSetMembers(ipsetID, []string{dnsPackets[1].IP.String(), dnsPackets[2].IP.String(),
						dnsPackets[3].IP.String()})
				} else {
					AssertIPSetMembers(ipsetID, []string{members[1], members[2], members[3]})
				}

				Describe("after sending a delete", func() {
					BeforeEach(func() {
						programDNSRecs(ipsetType, domainStore, dnsPackets)
						ipsetsMgr.OnUpdate(&proto.IPSetRemove{
							Id: ipsetID,
						})
						ipsetsMgr.CompleteDeferredWork()
					})
					AssertIPSetNoMembers(ipsetID)
				})
			})

			Describe("after sending another replace", func() {
				BeforeEach(func() {
					ipSets.AddOrReplaceCalled = false
					programDNSRecs(ipsetType, domainStore, dnsPackets)
					ipsetsMgr.OnUpdate(&proto.IPSetUpdate{
						Id:      ipsetID,
						Members: []string{members[1], members[2]},
						Type:    ipsetType,
					})
					ipsetsMgr.CompleteDeferredWork()
				})

				if ipsetType == proto.IPSetUpdate_DOMAIN {
					AssertIPSetMembers(ipsetID, []string{dnsPackets[1].IP.String(), dnsPackets[2].IP.String()})
				} else {
					AssertIPSetModified()
					AssertIPSetMembers(ipsetID, []string{members[1], members[2]})
				}
			})
		})
	}

	for _, testCase := range ipsetsMgrTestCases {
		IPsetsMgrTest1(testCase.ipsetID, testCase.ipsetType, testCase.ipsetMembers, testCase.dnsRecs)
	}
})
