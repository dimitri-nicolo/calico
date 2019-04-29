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

	IPSets_Tests_1 := func(ipsetID string, ipsetType proto.IPSetUpdate_IPSetType, members [4]string, dnsPackets [4]layers.DNSResourceRecord) {
		Describe("after creating an IPSet", func() {
			BeforeEach(func() {
				var layerDNS0, layerDNS1 layers.DNS
				layerDNS0.Answers = append(layerDNS0.Answers, dnsPackets[0])
				layerDNS1.Answers = append(layerDNS1.Answers, dnsPackets[1])

				domainStore.processDNSPacket(&layerDNS0)
				domainStore.processDNSPacket(&layerDNS1)

				ipsetsMgr.OnUpdate(&proto.IPSetUpdate{
					Id:      ipsetID,
					Members: []string{members[0], members[1]},
					Type:    ipsetType,
				})
				ipsetsMgr.CompleteDeferredWork()
			})
			AssertIPSetModified()
			AssertIPSetMembers(ipsetID, []string{members[0], members[1]})

			Describe("after sending a delta update", func() {
				BeforeEach(func() {
					ipSets.AddOrReplaceCalled = false
					ipsetsMgr.OnUpdate(&proto.IPSetDeltaUpdate{
						Id:             ipsetID,
						AddedMembers:   []string{members[2], members[3]},
						RemovedMembers: []string{members[0]},
					})
					ipsetsMgr.CompleteDeferredWork()
				})
				AssertIPSetNotModified()
				AssertIPSetMembers(ipsetID, []string{members[1], members[2], members[3]})

				Describe("after sending a delete", func() {
					BeforeEach(func() {
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
					ipsetsMgr.OnUpdate(&proto.IPSetUpdate{
						Id:      ipsetID,
						Members: []string{members[1], members[2]},
						Type:    ipsetType,
					})
					ipsetsMgr.CompleteDeferredWork()
				})
				AssertIPSetModified()
				AssertIPSetMembers(ipsetID, []string{members[1], members[2]})
			})
		})
	}

	IPSets_Tests_1("id1", proto.IPSetUpdate_IP, [4]string{"10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4"}, nil)
	IPSets_Tests_1("id2", proto.IPSetUpdate_DOMAIN, [4]string{"abc.com", "def.com", "ghi.com", "jkl.com"},
		[4]layers.DNSResourceRecord{
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
		})
})
