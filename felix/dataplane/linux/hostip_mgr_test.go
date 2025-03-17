// Copyright (c) 2017-2021 Tigera, Inc. All rights reserved.
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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	dpsets "github.com/projectcalico/calico/felix/dataplane/ipsets"
	"github.com/projectcalico/calico/felix/proto"
	"github.com/projectcalico/calico/libcalico-go/lib/set"
)

var _ = Describe("Host ip manager", func() {
	var (
		hostIPMgr *hostIPManager
		ipSets    *dpsets.MockIPSets
	)

	BeforeEach(func() {
		ipSets = dpsets.NewMockIPSets()
		hostIPMgr = newHostIPManager([]string{"cali"}, "this-host", ipSets, 1024, "all-tunnel")
	})

	Describe("after sending a route update", func() {
		BeforeEach(func() {
			hostIPMgr.OnUpdate(&proto.RouteUpdate{
				Type: proto.RouteType_REMOTE_TUNNEL,
				Dst:  "192.0.0.1/32",
			})
			hostIPMgr.OnUpdate(&proto.RouteUpdate{
				Type: proto.RouteType_REMOTE_TUNNEL,
				Dst:  "192.0.0.2/32",
			})
			err := hostIPMgr.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
		})
		It("should create the IP set", func() {
			Expect(ipSets.AddOrReplaceCalled).To(BeTrue())
		})
		It("should add the right members", func() {
			Expect(ipSets.Members).To(HaveLen(1))
			expIPs := set.From("192.0.0.1", "192.0.0.2")
			Expect(ipSets.Members["all-tunnel"]).To(Equal(expIPs))
		})
		Describe("after sending a route update with same CIDR but different type", func() {
			BeforeEach(func() {
				hostIPMgr.OnUpdate(&proto.RouteUpdate{
					Type: proto.RouteType_REMOTE_WORKLOAD,
					Dst:  "192.0.0.1/32",
				})
				err := hostIPMgr.CompleteDeferredWork()
				Expect(err).ToNot(HaveOccurred())
			})
			It("should delete the right member", func() {
				Expect(ipSets.Members).To(HaveLen(1))
				Expect(ipSets.Members["all-tunnel"].Len()).To(Equal(1))
				expIPs := set.From("192.0.0.2")
				Expect(ipSets.Members["all-tunnel"]).To(Equal(expIPs))
			})
		})
		Describe("after sending a route remove", func() {
			BeforeEach(func() {
				hostIPMgr.OnUpdate(&proto.RouteRemove{
					Dst: "192.0.0.2/32",
				})
				hostIPMgr.OnUpdate(&proto.RouteRemove{
					Dst: "192.0.0.1/32",
				})
				err := hostIPMgr.CompleteDeferredWork()
				Expect(err).ToNot(HaveOccurred())
			})
			It("should delete the right member", func() {
				Expect(ipSets.Members).To(HaveLen(1))
				Expect(ipSets.Members["all-tunnel"].Len()).To(Equal(0))
			})
		})
	})

	Describe("after sending a replace", func() {
		BeforeEach(func() {
			hostIPMgr.OnUpdate(&ifaceAddrsUpdate{
				Name:  "eth0",
				Addrs: set.From("10.0.0.1", "10.0.0.2"),
			})
			err := hostIPMgr.CompleteDeferredWork()
			Expect(err).ToNot(HaveOccurred())
		})
		It("should create the IP set", func() {
			Expect(ipSets.AddOrReplaceCalled).To(BeTrue())
		})
		It("should add the right members", func() {
			Expect(ipSets.Members).To(HaveLen(2))
			expIPs := set.From("10.0.0.1", "10.0.0.2")
			Expect(ipSets.Members["this-host"]).To(Equal(expIPs))
		})

		Describe("after sending a delete", func() {
			BeforeEach(func() {
				hostIPMgr.OnUpdate(&ifaceAddrsUpdate{
					Name: "eth0",
				})
				err := hostIPMgr.CompleteDeferredWork()
				Expect(err).ToNot(HaveOccurred())
			})
			It("should remove the IP set", func() {
				Expect(ipSets.Members["this-host"]).To(Equal(set.New[string]()))
			})
		})

		Describe("after sending a workload interface update", func() {
			BeforeEach(func() {
				ipSets.AddOrReplaceCalled = false
				hostIPMgr.OnUpdate(&ifaceAddrsUpdate{
					Name:  "cali1234",
					Addrs: set.From("10.0.0.8", "10.0.0.9"),
				})
				err := hostIPMgr.CompleteDeferredWork()
				Expect(err).ToNot(HaveOccurred())
			})
			It("should not create the IP set", func() {
				Expect(ipSets.AddOrReplaceCalled).To(BeFalse())
			})
			It("should have old members", func() {
				Expect(ipSets.Members).To(HaveLen(2))
				expIPs := set.From("10.0.0.1", "10.0.0.2")
				Expect(ipSets.Members["this-host"]).To(Equal(expIPs))
			})
		})

		Describe("after sending update for new interface", func() {
			BeforeEach(func() {
				ipSets.AddOrReplaceCalled = false
				hostIPMgr.OnUpdate(&ifaceAddrsUpdate{
					Name:  "eth1",
					Addrs: set.From("10.0.0.8", "10.0.0.9"),
				})
				err := hostIPMgr.CompleteDeferredWork()
				Expect(err).ToNot(HaveOccurred())
			})
			It("should not create the IP set", func() {
				Expect(ipSets.AddOrReplaceCalled).To(BeTrue())
			})
			It("should have old members", func() {
				Expect(ipSets.Members).To(HaveLen(2))
				expIPs := set.From("10.0.0.1", "10.0.0.2", "10.0.0.8", "10.0.0.9")
				Expect(ipSets.Members["this-host"]).To(Equal(expIPs))
			})
		})

		Describe("after sending another replace", func() {
			BeforeEach(func() {
				ipSets.AddOrReplaceCalled = false
				hostIPMgr.OnUpdate(&ifaceAddrsUpdate{
					Name:  "eth0",
					Addrs: set.From("10.0.0.2", "10.0.0.3"),
				})
				err := hostIPMgr.CompleteDeferredWork()
				Expect(err).ToNot(HaveOccurred())
			})
			It("should replace the IP set", func() {
				Expect(ipSets.AddOrReplaceCalled).To(BeTrue())
			})
			It("should add the right members", func() {
				Expect(ipSets.Members).To(HaveLen(2))
				expIPs := set.From("10.0.0.2", "10.0.0.3")
				Expect(ipSets.Members["this-host"]).To(Equal(expIPs))
			})
		})
	})
})
