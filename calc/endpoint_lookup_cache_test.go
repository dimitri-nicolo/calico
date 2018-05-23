// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package calc_test

import (
	"net"

	. "github.com/projectcalico/felix/calc"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("EndpointLookupsCache tests", func() {
	ec := NewEndpointLookupsCache()

	DescribeTable(
		"Check adding/deleting workload endpoint modifies the cache",
		func(key model.WorkloadEndpointKey, wep *model.WorkloadEndpoint, ipAddr net.IP) {
			c := "WEP(" + key.Hostname + "/" + key.OrchestratorID + "/" + key.WorkloadID + "/" + key.EndpointID + ")"
			update := api.Update{
				KVPair: model.KVPair{
					Key:   key,
					Value: wep,
				},
				UpdateType: api.UpdateTypeKVNew,
			}
			var addrB [16]byte
			copy(addrB[:], ipAddr.To16()[:16])

			ec.OnUpdate(update)
			ed, ok := ec.GetEndpoint(addrB)
			Expect(ok).To(BeTrue(), c)
			Expect(ed.Key).To(Equal(key))

			update = api.Update{
				KVPair: model.KVPair{
					Key: key,
				},
				UpdateType: api.UpdateTypeKVDeleted,
			}
			ec.OnUpdate(update)
			ed, ok = ec.GetEndpoint(addrB)
			Expect(ok).To(BeFalse(), c)
		},
		Entry("remote WEP1 IPv4", remoteWlEpKey1, &remoteWlEp1, remoteWlEp1.IPv4Nets[0].IP),
		Entry("remote WEP1 IPv6", remoteWlEpKey1, &remoteWlEp1, remoteWlEp1.IPv6Nets[0].IP),
	)

	DescribeTable(
		"Check adding/deleting host endpoint modifies the cache",
		func(key model.HostEndpointKey, hep *model.HostEndpoint, ipAddr net.IP) {
			c := "HEP(" + key.Hostname + "/" + key.EndpointID + ")"
			update := api.Update{
				KVPair: model.KVPair{
					Key:   key,
					Value: hep,
				},
				UpdateType: api.UpdateTypeKVNew,
			}
			var addrB [16]byte
			copy(addrB[:], ipAddr.To16()[:16])

			ec.OnUpdate(update)
			ed, ok := ec.GetEndpoint(addrB)
			Expect(ok).To(BeTrue(), c)
			Expect(ed.Key).To(Equal(key))

			update = api.Update{
				KVPair: model.KVPair{
					Key: key,
				},
				UpdateType: api.UpdateTypeKVDeleted,
			}
			ec.OnUpdate(update)
			ed, ok = ec.GetEndpoint(addrB)
			Expect(ok).To(BeFalse(), c)
		},
		Entry("Host Endpoint IPv4", hostEpWithNameKey, &hostEpWithName, hostEpWithName.ExpectedIPv4Addrs[0].IP),
		Entry("Host Endpoint IPv6", hostEpWithNameKey, &hostEpWithName, hostEpWithName.ExpectedIPv6Addrs[0].IP),
	)

	It("should process both workload and host endpoints each with multiple IP addresses", func() {
		By("adding a workload endpoint with multiple ipv4 and ipv6 ip addresses")
		update := api.Update{
			KVPair: model.KVPair{
				Key:   remoteWlEpKey1,
				Value: &remoteWlEp1,
			},
			UpdateType: api.UpdateTypeKVNew,
		}
		ec.OnUpdate(update)

		verifyIpToEndpoint := func(key model.Key, ipAddr net.IP, exists bool) {
			var name string
			switch k := key.(type) {
			case model.WorkloadEndpointKey:
				name = "WEP(" + k.Hostname + "/" + k.OrchestratorID + "/" + k.WorkloadID + "/" + k.EndpointID + ")"
			case model.HostEndpointKey:
				name = "HEP(" + k.Hostname + "/" + k.EndpointID + ")"
			}
			var addrB [16]byte
			copy(addrB[:], ipAddr.To16()[:16])

			ed, ok := ec.GetEndpoint(addrB)
			if exists {
				Expect(ok).To(BeTrue(), name+"\n"+ec.DumpEndpoints())
				Expect(ed.Key).To(Equal(key), ec.DumpEndpoints())
			} else {
				Expect(ok).To(BeFalse(), name+".\n"+ec.DumpEndpoints())
			}
		}

		By("verifying all IPv4 and IPv6 addresses of the workload endpoint are present in the mapping")
		for _, ipv4 := range remoteWlEp1.IPv4Nets {
			verifyIpToEndpoint(remoteWlEpKey1, ipv4.IP, true)
		}
		for _, ipv6 := range remoteWlEp1.IPv6Nets {
			verifyIpToEndpoint(remoteWlEpKey1, ipv6.IP, true)
		}

		By("adding a host endpoint with multiple ipv4 and ipv6 ip addresses")
		update = api.Update{
			KVPair: model.KVPair{
				Key:   hostEpWithNameKey,
				Value: &hostEpWithName,
			},
			UpdateType: api.UpdateTypeKVNew,
		}
		ec.OnUpdate(update)

		By("verifying all IPv4 and IPv6 addresses of the host endpoint are present in the mapping")
		for _, ipv4 := range hostEpWithName.ExpectedIPv4Addrs {
			verifyIpToEndpoint(hostEpWithNameKey, ipv4.IP, true)
		}
		for _, ipv6 := range hostEpWithName.ExpectedIPv6Addrs {
			verifyIpToEndpoint(hostEpWithNameKey, ipv6.IP, true)
		}

		By("deleting the host endpoint")
		update = api.Update{
			KVPair: model.KVPair{
				Key: hostEpWithNameKey,
			},
			UpdateType: api.UpdateTypeKVDeleted,
		}
		ec.OnUpdate(update)

		By("verifying all IPv4 and IPv6 addresses of the host endpoint are not present in the mapping")
		for _, ipv4 := range hostEpWithName.ExpectedIPv4Addrs {
			verifyIpToEndpoint(hostEpWithNameKey, ipv4.IP, false)
		}
		for _, ipv6 := range hostEpWithName.ExpectedIPv6Addrs {
			verifyIpToEndpoint(hostEpWithNameKey, ipv6.IP, false)
		}

		By("updating the workload endpoint and removing all IPv6 addresses")
		update = api.Update{
			KVPair: model.KVPair{
				Key:   remoteWlEpKey1,
				Value: &remoteWlEp1NoIpv6,
			},
			UpdateType: api.UpdateTypeKVUpdated,
		}
		ec.OnUpdate(update)

		By("verifying all IPv4 are present but no Ipv6 addresses are present")
		// For verification we iterate using the original WEP with IPv6 so that it is easy to
		// get a list of Ipv6 addresses to check against.
		for _, ipv4 := range remoteWlEp1.IPv4Nets {
			verifyIpToEndpoint(remoteWlEpKey1, ipv4.IP, true)
		}
		for _, ipv6 := range remoteWlEp1.IPv6Nets {
			verifyIpToEndpoint(remoteWlEpKey1, ipv6.IP, false)
		}

		By("updating the workload endpoint keeping all the information as before")
		update = api.Update{
			KVPair: model.KVPair{
				Key:   remoteWlEpKey1,
				Value: &remoteWlEp1NoIpv6,
			},
			UpdateType: api.UpdateTypeKVUpdated,
		}
		ec.OnUpdate(update)

		By("verifying all IPv4 are present but no Ipv6 addresses are present")
		// For verification we iterate using the original WEP with IPv6 so that it is easy to
		// get a list of Ipv6 addresses to check against.
		for _, ipv4 := range remoteWlEp1.IPv4Nets {
			verifyIpToEndpoint(remoteWlEpKey1, ipv4.IP, true)
		}
		for _, ipv6 := range remoteWlEp1.IPv6Nets {
			verifyIpToEndpoint(remoteWlEpKey1, ipv6.IP, false)
		}

		By("finally removing the WEP and no mapping is present")
		update = api.Update{
			KVPair: model.KVPair{
				Key: remoteWlEpKey1,
			},
			UpdateType: api.UpdateTypeKVDeleted,
		}
		ec.OnUpdate(update)

		By("verifying all there are no mapping present")
		// For verification we iterate using the original WEP with IPv6 so that it is easy to
		// get a list of Ipv6 addresses to check against.
		for _, ipv4 := range remoteWlEp1.IPv4Nets {
			verifyIpToEndpoint(remoteWlEpKey1, ipv4.IP, false)
		}
		for _, ipv6 := range remoteWlEp1.IPv6Nets {
			verifyIpToEndpoint(remoteWlEpKey1, ipv6.IP, false)
		}
	})

})
