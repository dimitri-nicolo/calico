// Copyright (c) 2018-2019 Tigera, Inc. All rights reserved.

package calc_test

import (
	"net"

	"github.com/projectcalico/felix/rules"

	. "github.com/projectcalico/felix/calc"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var (
	float1_0 = float64(1.0)
	float2_0 = float64(2.0)
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
			_, ok = ec.GetEndpoint(addrB)
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
			_, ok = ec.GetEndpoint(addrB)
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
		origRemoteWepLabels := map[string]string{
			"id": "rem-ep-1",
			"x":  "x",
			"y":  "y",
		}
		ec.OnUpdate(update)

		verifyIpToEndpoint := func(key model.Key, ipAddr net.IP, exists bool, labels map[string]string) {
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
				if labels != nil {
					switch ep := ed.Endpoint.(type) {
					case *model.WorkloadEndpoint:
						Expect(ep.Labels).To(Equal(labels), ec.DumpEndpoints())
					case *model.HostEndpoint:
						Expect(ep.Labels).To(Equal(labels), ec.DumpEndpoints())
					}
				}
			} else {
				Expect(ok).To(BeFalse(), name+".\n"+ec.DumpEndpoints())
			}
		}

		By("verifying all IPv4 and IPv6 addresses of the workload endpoint are present in the mapping")
		for _, ipv4 := range remoteWlEp1.IPv4Nets {
			verifyIpToEndpoint(remoteWlEpKey1, ipv4.IP, true, origRemoteWepLabels)
		}
		for _, ipv6 := range remoteWlEp1.IPv6Nets {
			verifyIpToEndpoint(remoteWlEpKey1, ipv6.IP, true, origRemoteWepLabels)
		}

		By("adding a host endpoint with multiple ipv4 and ipv6 ip addresses")
		update = api.Update{
			KVPair: model.KVPair{
				Key:   hostEpWithNameKey,
				Value: &hostEpWithName,
			},
			UpdateType: api.UpdateTypeKVNew,
		}
		hepLabels := map[string]string{
			"id": "loc-ep-1",
			"a":  "a",
			"b":  "b",
		}
		ec.OnUpdate(update)

		By("verifying all IPv4 and IPv6 addresses of the host endpoint are present in the mapping")
		for _, ipv4 := range hostEpWithName.ExpectedIPv4Addrs {
			verifyIpToEndpoint(hostEpWithNameKey, ipv4.IP, true, hepLabels)
		}
		for _, ipv6 := range hostEpWithName.ExpectedIPv6Addrs {
			verifyIpToEndpoint(hostEpWithNameKey, ipv6.IP, true, hepLabels)
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
			verifyIpToEndpoint(hostEpWithNameKey, ipv4.IP, false, nil)
		}
		for _, ipv6 := range hostEpWithName.ExpectedIPv6Addrs {
			verifyIpToEndpoint(hostEpWithNameKey, ipv6.IP, false, nil)
		}

		By("updating the workload endpoint and adding new labels")
		update = api.Update{
			KVPair: model.KVPair{
				Key:   remoteWlEpKey1,
				Value: &remoteWlEp1UpdatedLabels,
			},
			UpdateType: api.UpdateTypeKVUpdated,
		}
		ec.OnUpdate(update)

		updatedRemoteWepLabels := map[string]string{
			"id": "rem-ep-1",
			"x":  "x",
			"y":  "y",
			"z":  "z",
		}
		By("verifying all IPv4 and IPv6 addresses are present with updated labels")
		// For verification we iterate using the original WEP with IPv6 so that it is easy to
		// get a list of Ipv6 addresses to check against.
		for _, ipv4 := range remoteWlEp1.IPv4Nets {
			verifyIpToEndpoint(remoteWlEpKey1, ipv4.IP, true, updatedRemoteWepLabels)
		}
		for _, ipv6 := range remoteWlEp1.IPv6Nets {
			verifyIpToEndpoint(remoteWlEpKey1, ipv6.IP, true, updatedRemoteWepLabels)
		}

		By("updating the workload endpoint and removing all IPv6 addresses, and reverting labels back to original")
		update = api.Update{
			KVPair: model.KVPair{
				Key:   remoteWlEpKey1,
				Value: &remoteWlEp1NoIpv6,
			},
			UpdateType: api.UpdateTypeKVUpdated,
		}
		ec.OnUpdate(update)

		By("verifying all IPv4 are present and no Ipv6 addresses are present")
		// For verification we iterate using the original WEP with IPv6 so that it is easy to
		// get a list of Ipv6 addresses to check against.
		for _, ipv4 := range remoteWlEp1.IPv4Nets {
			verifyIpToEndpoint(remoteWlEpKey1, ipv4.IP, true, origRemoteWepLabels)
		}
		for _, ipv6 := range remoteWlEp1.IPv6Nets {
			verifyIpToEndpoint(remoteWlEpKey1, ipv6.IP, false, nil)
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
			verifyIpToEndpoint(remoteWlEpKey1, ipv4.IP, true, origRemoteWepLabels)
		}
		for _, ipv6 := range remoteWlEp1.IPv6Nets {
			verifyIpToEndpoint(remoteWlEpKey1, ipv6.IP, false, nil)
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
			verifyIpToEndpoint(remoteWlEpKey1, ipv4.IP, false, nil)
		}
		for _, ipv6 := range remoteWlEp1.IPv6Nets {
			verifyIpToEndpoint(remoteWlEpKey1, ipv6.IP, false, nil)
		}
	})

	It("should process local endpoints correctly with no staged policies and one tier per ingress and egress", func() {
		By("adding a host endpoint with ingress policies in tier1 and egress policies in tier default")
		p1k := model.PolicyKey{Name: "tier1.pol1"}
		p1 := &model.Policy{
			Order:        &float1_0,
			Types:        []string{"ingress"},
			InboundRules: []model.Rule{{Action: "next-tier"}, {Action: "allow"}, {Action: "deny"}},
		}
		p1id := PolicyID{Name: "pol1", Tier: "tier1"}

		p2k := model.PolicyKey{Name: "ns1/default.pol2"}
		p2 := &model.Policy{
			Namespace: "ns1",
			Order:     &float1_0,
			Types:     []string{"egress"},
		}
		p2id := PolicyID{Name: "pol2", Tier: "default", Namespace: "ns1"}

		p3k := model.PolicyKey{Name: "ns1/default.pol3"}
		p3 := &model.Policy{
			Namespace: "ns1",
			Order:     &float2_0,
			Types:     []string{"egress"},
		}
		p3id := PolicyID{Name: "pol3", Tier: "default", Namespace: "ns1"}

		t1 := NewTierInfo("tier1")
		t1.Order = &float1_0
		t1.Valid = true
		t1.OrderedPolicies = []PolKV{{Key: p1k, Value: p1}}

		td := NewTierInfo("default")
		td.Order = &float2_0
		td.Valid = true
		td.OrderedPolicies = []PolKV{{Key: p2k, Value: p2}, {Key: p3k, Value: p3}}

		ts := NewTierInfoSlice()
		ts = append(ts, *t1, *td)

		ed := ec.CreateEndpointData(hostEpWithNameKey, &hostEpWithName, ts)

		By("checking endpoint data")
		Expect(ed.Key).To(Equal(hostEpWithNameKey))
		Expect(ed.IsLocal).To(BeTrue())
		Expect(ed.IsHostEndpoint()).To(BeTrue())
		Expect(ed.Endpoint).To(Equal(&hostEpWithName))

		By("checking compiled ingress data")
		Expect(ed.Ingress).ToNot(BeNil())
		Expect(ed.Ingress.PolicyMatches).To(HaveLen(1))
		Expect(ed.Ingress.PolicyMatches).To(HaveKey(p1id))
		Expect(ed.Ingress.PolicyMatches[p1id]).To(Equal(0))
		Expect(ed.Ingress.ProfileMatchIndex).To(Equal(1))
		Expect(ed.Ingress.TierData).To(HaveLen(1))
		Expect(ed.Ingress.TierData).To(HaveKey("tier1"))
		Expect(ed.Ingress.TierData["tier1"]).ToNot(BeNil())
		Expect(ed.Ingress.TierData["tier1"].ImplicitDropRuleID).To(Equal(
			NewRuleID("tier1", "pol1", "", RuleIDIndexImplicitDrop, rules.RuleDirIngress, rules.RuleActionDeny)))
		Expect(ed.Ingress.TierData["tier1"].EndOfTierMatchIndex).To(Equal(0))

		By("checking compiled egress data")
		Expect(ed.Egress).ToNot(BeNil())
		Expect(ed.Egress.PolicyMatches).To(HaveLen(2))
		Expect(ed.Egress.PolicyMatches).To(HaveKey(p2id))
		Expect(ed.Egress.PolicyMatches[p2id]).To(Equal(0))
		Expect(ed.Egress.PolicyMatches).To(HaveKey(p3id))
		Expect(ed.Egress.PolicyMatches[p3id]).To(Equal(0))
		Expect(ed.Egress.ProfileMatchIndex).To(Equal(1))
		Expect(ed.Egress.TierData).To(HaveLen(1))
		Expect(ed.Egress.TierData).To(HaveKey("default"))
		Expect(ed.Egress.TierData["default"]).ToNot(BeNil())
		Expect(ed.Egress.TierData["default"].ImplicitDropRuleID).To(Equal(
			NewRuleID("default", "pol3", "ns1", RuleIDIndexImplicitDrop, rules.RuleDirEgress, rules.RuleActionDeny)))
		Expect(ed.Egress.TierData["default"].EndOfTierMatchIndex).To(Equal(0))
	})

	DescribeTable(
		"should process local endpoints correctly with staged policies and multiple tiers",
		func(ingress bool) {
			var dir string
			if ingress {
				dir = "ingress"
			} else {
				dir = "egress"
			}

			By("adding a workloadendpoint with mixed staged/non-staged policies in tier1")
			sp1k := model.PolicyKey{Name: "staged:tier1.pol1"}
			sp1 := &model.Policy{
				Order: &float1_0,
				Types: []string{dir},
			}
			sp1id := PolicyID{Name: "staged:pol1", Tier: "tier1"}

			p1k := model.PolicyKey{Name: "tier1.pol1"}
			p1 := &model.Policy{
				Order: &float1_0,
				Types: []string{dir},
			}
			p1id := PolicyID{Name: "pol1", Tier: "tier1"}

			sp2k := model.PolicyKey{Name: "ns1/staged:tier1.pol2"}
			sp2 := &model.Policy{
				Namespace: "ns1",
				Order:     &float2_0,
				Types:     []string{dir},
			}
			sp2id := PolicyID{Name: "staged:pol2", Tier: "tier1", Namespace: "ns1"}

			p2k := model.PolicyKey{Name: "ns1/tier1.pol2"}
			p2 := &model.Policy{
				Namespace: "ns1",
				Order:     &float2_0,
				Types:     []string{dir},
			}
			p2id := PolicyID{Name: "pol2", Tier: "tier1", Namespace: "ns1"}

			t1 := NewTierInfo("tier1")
			t1.Order = &float1_0
			t1.Valid = true
			t1.OrderedPolicies = []PolKV{{Key: sp1k, Value: sp1}, {Key: p1k, Value: p1}, {Key: sp2k, Value: sp2}, {Key: p2k, Value: p2}}

			By("and adding staged policies in tier default")
			sp3k := model.PolicyKey{Name: "ns2/staged:knp.default.pol3"}
			sp3 := &model.Policy{
				Order: &float1_0,
				Types: []string{dir},
			}
			sp3id := PolicyID{Name: "staged:knp.default.pol3", Tier: "default", Namespace: "ns2"}

			sp4k := model.PolicyKey{Name: "staged:default.pol4"}
			sp4 := &model.Policy{
				Order: &float2_0,
				Types: []string{dir},
			}
			sp4id := PolicyID{Name: "staged:pol4", Tier: "default"}

			td := NewTierInfo("default")
			td.Valid = true
			td.OrderedPolicies = []PolKV{{Key: sp3k, Value: sp3}, {Key: sp4k, Value: sp4}}

			By("Creating the endpoint data")
			ts := NewTierInfoSlice()
			ts = append(ts, *t1, *td)

			ed := ec.CreateEndpointData(localWlEpKey1, &localWlEp1, ts)

			By("checking endpoint data")
			Expect(ed.Key).To(Equal(localWlEpKey1))
			Expect(ed.IsLocal).To(BeTrue())
			Expect(ed.IsHostEndpoint()).To(BeFalse())
			Expect(ed.Endpoint).To(Equal(&localWlEp1))

			By("checking compiled data size for both tiers")
			var data, other *MatchData
			var ruleDir rules.RuleDir
			if ingress {
				data = ed.Ingress
				other = ed.Egress
				ruleDir = rules.RuleDirIngress
			} else {
				data = ed.Egress
				other = ed.Ingress
				ruleDir = rules.RuleDirEgress
			}

			Expect(data).ToNot(BeNil())
			Expect(data.PolicyMatches).To(HaveLen(6))
			Expect(other.PolicyMatches).To(HaveLen(0))
			Expect(data.TierData).To(HaveLen(2))
			Expect(other.TierData).To(HaveLen(0))
			Expect(data.TierData["tier1"]).ToNot(BeNil())
			Expect(data.TierData["default"]).ToNot(BeNil())

			By("checking compiled match data for tier1")
			// Staged policy increments the next index.
			Expect(data.PolicyMatches).To(HaveKey(sp1id))
			Expect(data.PolicyMatches[sp1id]).To(Equal(0))

			// Enforced policy leaves next index unchanged.
			Expect(data.PolicyMatches).To(HaveKey(p1id))
			Expect(data.PolicyMatches[p1id]).To(Equal(1))

			// Staged policy increments the next index.
			Expect(data.PolicyMatches).To(HaveKey(sp2id))
			Expect(data.PolicyMatches[sp2id]).To(Equal(1))

			// Enforced policy leaves next index unchanged.
			Expect(data.PolicyMatches).To(HaveKey(p2id))
			Expect(data.PolicyMatches[p2id]).To(Equal(2))

			// Tier contains enforced policy, so has a real implicit drop rule ID.
			Expect(data.TierData["tier1"].EndOfTierMatchIndex).To(Equal(2))
			Expect(data.TierData["tier1"].ImplicitDropRuleID).To(Equal(
				NewRuleID("tier1", "pol2", "ns1", RuleIDIndexImplicitDrop, ruleDir, rules.RuleActionDeny)))

			By("checking compiled match data for default tier")
			// Staged policy increments the next index.
			Expect(data.PolicyMatches).To(HaveKey(sp3id))
			Expect(data.PolicyMatches[sp3id]).To(Equal(3))

			// Staged policy increments the next index.
			Expect(data.PolicyMatches).To(HaveKey(sp4id))
			Expect(data.PolicyMatches[sp4id]).To(Equal(4))

			// Tier contains only staged policy so does not contain an implicit drop rule ID.
			Expect(data.TierData["default"].EndOfTierMatchIndex).To(Equal(5))
			Expect(data.TierData["default"].ImplicitDropRuleID).To(BeNil())

			By("checking profile match index")
			Expect(data.ProfileMatchIndex).To(Equal(6))
			Expect(other.ProfileMatchIndex).To(Equal(0))
		},
		Entry("ingress", true),
		Entry("egress", false),
	)
})
