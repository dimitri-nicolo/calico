// Copyright (c) 2017 Tigera, Inc. All rights reserved.

package intdataplane

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/felix/dataplane/ipsets"
	"github.com/projectcalico/calico/felix/proto"
	"github.com/projectcalico/calico/libcalico-go/lib/set"
)

var _ = Describe("allHostsIpsetManager IP set updates", func() {
	var (
		allHostsMgr *allHostsIpsetManager
		ipSets      *ipsets.MockIPSets
	)

	const (
		externalCIDR = "11.0.0.1/32"
	)

	BeforeEach(func() {
		ipSets = ipsets.NewMockIPSets()
		allHostsMgr = newAllHostsIpsetManager(ipSets, 1024, []string{externalCIDR})
	})

	It("should not create the IP set until first call to CompleteDeferredWork()", func() {
		Expect(ipSets.AddOrReplaceCalled).To(BeFalse())
		Expect(allHostsMgr.CompleteDeferredWork()).NotTo(HaveOccurred())
		Expect(ipSets.AddOrReplaceCalled).To(BeTrue())
	})

	allHostsSet := func() set.Set[string] {
		Expect(ipSets.Members).To(HaveLen(1))
		return ipSets.Members["all-hosts-net"]
	}

	Describe("after adding an IP for host1", func() {
		BeforeEach(func() {
			allHostsMgr.OnUpdate(&proto.HostMetadataUpdate{
				Hostname: "host1",
				Ipv4Addr: "10.0.0.1",
			})
			Expect(allHostsMgr.CompleteDeferredWork()).NotTo(HaveOccurred())
		})

		It("should add host1's IP to the IP set", func() {
			Expect(allHostsSet()).To(Equal(set.From[string]("10.0.0.1", externalCIDR)))
		})

		Describe("after adding an IP for host2", func() {
			BeforeEach(func() {
				allHostsMgr.OnUpdate(&proto.HostMetadataUpdate{
					Hostname: "host2",
					Ipv4Addr: "10.0.0.2",
				})
				Expect(allHostsMgr.CompleteDeferredWork()).NotTo(HaveOccurred())
			})
			It("should add the IP to the IP set", func() {
				Expect(allHostsSet()).To(Equal(set.From[string]("10.0.0.1", "10.0.0.2", externalCIDR)))
			})
		})

		Describe("after adding a duplicate IP", func() {
			BeforeEach(func() {
				allHostsMgr.OnUpdate(&proto.HostMetadataUpdate{
					Hostname: "host2",
					Ipv4Addr: "10.0.0.1",
				})
				Expect(allHostsMgr.CompleteDeferredWork()).NotTo(HaveOccurred())
			})
			It("should tolerate the duplicate", func() {
				Expect(allHostsSet()).To(Equal(set.From[string]("10.0.0.1", externalCIDR)))
			})

			Describe("after removing a duplicate IP", func() {
				BeforeEach(func() {
					allHostsMgr.OnUpdate(&proto.HostMetadataRemove{
						Hostname: "host2",
					})
					Expect(allHostsMgr.CompleteDeferredWork()).NotTo(HaveOccurred())
				})
				It("should keep the IP in the IP set", func() {
					Expect(allHostsSet()).To(Equal(set.From[string]("10.0.0.1", externalCIDR)))
				})

				Describe("after removing initial copy of IP", func() {
					BeforeEach(func() {
						allHostsMgr.OnUpdate(&proto.HostMetadataRemove{
							Hostname: "host1",
						})
						Expect(allHostsMgr.CompleteDeferredWork()).NotTo(HaveOccurred())
					})
					It("should remove the IP", func() {
						Expect(allHostsSet()).To(Equal(set.From[string](externalCIDR)))
					})
				})
			})
		})

		Describe("after adding/removing a duplicate IP in one batch", func() {
			BeforeEach(func() {
				allHostsMgr.OnUpdate(&proto.HostMetadataUpdate{
					Hostname: "host2",
					Ipv4Addr: "10.0.0.1",
				})
				allHostsMgr.OnUpdate(&proto.HostMetadataRemove{
					Hostname: "host2",
				})
				Expect(allHostsMgr.CompleteDeferredWork()).NotTo(HaveOccurred())
			})
			It("should keep the IP in the IP set", func() {
				Expect(allHostsSet()).To(Equal(set.From[string]("10.0.0.1", externalCIDR)))
			})
		})

		Describe("after changing IP for host1", func() {
			BeforeEach(func() {
				allHostsMgr.OnUpdate(&proto.HostMetadataUpdate{
					Hostname: "host1",
					Ipv4Addr: "10.0.0.2",
				})
				Expect(allHostsMgr.CompleteDeferredWork()).NotTo(HaveOccurred())
			})
			It("should update the IP set", func() {
				Expect(allHostsSet()).To(Equal(set.From[string]("10.0.0.2", externalCIDR)))
			})
		})

		Describe("after a no-op batch", func() {
			BeforeEach(func() {
				ipSets.AddOrReplaceCalled = false
				Expect(allHostsMgr.CompleteDeferredWork()).NotTo(HaveOccurred())
			})
			It("shouldn't rewrite the IP set", func() {
				Expect(ipSets.AddOrReplaceCalled).To(BeFalse())
			})
		})
	})
})
