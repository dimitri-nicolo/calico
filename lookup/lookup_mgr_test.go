// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package lookup_test

import (
	. "github.com/projectcalico/felix/lookup"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/felix/rules"
)

var (
	// Define a set of policies and profiles - these just contain the bare bones info for the
	// tests.

	// GlobalNetworkPolicy tier-1.policy1, three variations
	gnp1_t1_0i0e = &proto.ActivePolicyUpdate{
		Id: &proto.PolicyID{
			Tier: "tier-1",
			Name: "tier-1.policy-1",
		},
		Policy: &proto.Policy{},
	}
	gnp1_t1_1i1e = &proto.ActivePolicyUpdate{
		Id: &proto.PolicyID{
			Tier: "tier-1",
			Name: "tier-1.policy-1",
		},
		Policy: &proto.Policy{
			InboundRules: []*proto.Rule{
				{Action: "allow"},
			},
			OutboundRules: []*proto.Rule{
				{Action: "deny"},
			},
		},
	}
	prefix_gnp1_t1_i0A = toprefix("API0|tier-1.policy-1")
	ruleID_gnp1_t1_i0A = &RuleID{
		Tier:      "tier-1",
		Name:      "policy-1",
		Namespace: "",
		Direction: rules.RuleDirIngress,
		Index:     0,
		IndexStr:  "0",
		Action:    rules.RuleActionAllow,
	}
	prefix_gnp1_t1_e0D = toprefix("DPE0|tier-1.policy-1")
	ruleID_gnp1_t1_e0D = &RuleID{
		Tier:      "tier-1",
		Name:      "policy-1",
		Namespace: "",
		Direction: rules.RuleDirEgress,
		Index:     0,
		IndexStr:  "0",
		Action:    rules.RuleActionDeny,
	}

	gnp1_t1_4i2e = &proto.ActivePolicyUpdate{
		Id: &proto.PolicyID{
			Tier: "tier-1",
			Name: "tier-1.policy-1",
		},
		Policy: &proto.Policy{
			InboundRules: []*proto.Rule{
				{Action: "allow"}, {Action: "deny"}, {Action: "pass"}, {Action: "next-tier"},
			},
			OutboundRules: []*proto.Rule{
				{Action: "allow"}, {Action: "allow"},
			},
		},
	}
	//prefix_gnp1_t1_i0A defined above
	//ruleID_gnp1_t1_i0A defined above
	prefix_gnp1_t1_i1D = toprefix("DPI1|tier-1.policy-1")
	ruleID_gnp1_t1_i1D = &RuleID{
		Tier:      "tier-1",
		Name:      "policy-1",
		Namespace: "",
		Direction: rules.RuleDirIngress,
		Index:     1,
		IndexStr:  "1",
		Action:    rules.RuleActionDeny,
	}
	prefix_gnp1_t1_i2N = toprefix("NPI2|tier-1.policy-1")
	ruleID_gnp1_t1_i2N = &RuleID{
		Tier:      "tier-1",
		Name:      "policy-1",
		Namespace: "",
		Direction: rules.RuleDirIngress,
		Index:     2,
		IndexStr:  "2",
		Action:    rules.RuleActionNextTier,
	}
	prefix_gnp1_t1_i3P = toprefix("NPI3|tier-1.policy-1")
	ruleID_gnp1_t1_i3P = &RuleID{
		Tier:      "tier-1",
		Name:      "policy-1",
		Namespace: "",
		Direction: rules.RuleDirIngress,
		Index:     3,
		IndexStr:  "3",
		Action:    rules.RuleActionNextTier,
	}
	prefix_gnp1_t1_e0A = toprefix("APE0|tier-1.policy-1")
	ruleID_gnp1_t1_e0A = &RuleID{
		Tier:      "tier-1",
		Name:      "policy-1",
		Namespace: "",
		Direction: rules.RuleDirEgress,
		Index:     0,
		IndexStr:  "0",
		Action:    rules.RuleActionAllow,
	}
	prefix_gnp1_t1_e1A = toprefix("APE1|tier-1.policy-1")
	ruleID_gnp1_t1_e1A = &RuleID{
		Tier:      "tier-1",
		Name:      "policy-1",
		Namespace: "",
		Direction: rules.RuleDirEgress,
		Index:     1,
		IndexStr:  "1",
		Action:    rules.RuleActionAllow,
	}

	// NetworkPolicy namespace-1/tier-1.policy-1
	np1_t1_0i1e = &proto.ActivePolicyUpdate{
		Id: &proto.PolicyID{
			Tier: "tier-1",
			Name: "namespace-1/tier-1.policy-2",
		},
		Policy: &proto.Policy{
			OutboundRules: []*proto.Rule{
				{Action: "allow"},
			},
		},
	}
	prefix_np1_t1_e0A = toprefix("APE0|namespace-1/tier-1.policy-2")
	ruleID_np1_t1_e0A = &RuleID{
		Tier:      "tier-1",
		Name:      "policy-2",
		Namespace: "namespace-1",
		Direction: rules.RuleDirEgress,
		Index:     0,
		IndexStr:  "0",
		Action:    rules.RuleActionAllow,
	}

	// K8s NetworkPolicy namespace-1/knp.default.policy-1
	knp1_t1_1i0e = &proto.ActivePolicyUpdate{
		Id: &proto.PolicyID{
			Tier: "default",
			Name: "namespace-1/knp.default.policy-1",
		},
		Policy: &proto.Policy{
			InboundRules: []*proto.Rule{
				{Action: "deny"},
			},
		},
	}
	prefix_knp1_t1_i0D = toprefix("DPI0|namespace-1/knp.default.policy-1")
	ruleID_knp1_t1_i0D = &RuleID{
		Tier:      "default",
		Name:      "knp.default.policy-1",
		Namespace: "namespace-1",
		Direction: rules.RuleDirIngress,
		Index:     0,
		IndexStr:  "0",
		Action:    rules.RuleActionDeny,
	}

	// Profile profile-1
	pr1_1i1e = &proto.ActiveProfileUpdate{
		Id: &proto.ProfileID{
			Name: "profile-1",
		},
		Profile: &proto.Profile{
			InboundRules: []*proto.Rule{
				{Action: "deny"},
			},
			OutboundRules: []*proto.Rule{
				{Action: "deny"},
			},
		},
	}
	prefix_prof_i0D = toprefix("DRI0|profile-1")
	prefix_prof_e0D = toprefix("DRE0|profile-1")
	ruleID_prof_i0D = &RuleID{
		Tier:      "",
		Name:      "profile-1",
		Namespace: "",
		Direction: rules.RuleDirIngress,
		Index:     0,
		IndexStr:  "0",
		Action:    rules.RuleActionDeny,
	}
	ruleID_prof_e0D = &RuleID{
		Tier:      "",
		Name:      "profile-1",
		Namespace: "",
		Direction: rules.RuleDirEgress,
		Index:     0,
		IndexStr:  "0",
		Action:    rules.RuleActionDeny,
	}

	// Tier no-matches
	prefix_nomatch_t1_i = toprefix("DPI|tier-1")
	ruleID_nomatch_t1_i = &RuleID{
		Tier:      "tier-1",
		Name:      "",
		Namespace: "",
		Direction: rules.RuleDirIngress,
		Index:     0,
		IndexStr:  "0",
		Action:    rules.RuleActionDeny,
	}
	prefix_nomatch_t1_e = toprefix("DPE|tier-1")
	ruleID_nomatch_t1_e = &RuleID{
		Tier:      "tier-1",
		Name:      "",
		Namespace: "",
		Direction: rules.RuleDirEgress,
		Index:     0,
		IndexStr:  "0",
		Action:    rules.RuleActionDeny,
	}
	prefix_nomatch_td_i = toprefix("DPI|default")
	ruleID_nomatch_td_i = &RuleID{
		Tier:      "default",
		Name:      "",
		Namespace: "",
		Direction: rules.RuleDirIngress,
		Index:     0,
		IndexStr:  "0",
		Action:    rules.RuleActionDeny,
	}
	prefix_nomatch_td_e = toprefix("DPE|default")
	ruleID_nomatch_td_e = &RuleID{
		Tier:      "default",
		Name:      "",
		Namespace: "",
		Direction: rules.RuleDirEgress,
		Index:     0,
		IndexStr:  "0",
		Action:    rules.RuleActionDeny,
	}

	// Profile no-matches
	prefix_nomatch_prof_i = toprefix("DRI")
	ruleID_nomatch_prof_i = &RuleID{
		Tier:      "",
		Name:      "",
		Namespace: "",
		Direction: rules.RuleDirIngress,
		Index:     0,
		IndexStr:  "0",
		Action:    rules.RuleActionDeny,
	}
	prefix_nomatch_prof_e = toprefix("DRE")
	ruleID_nomatch_prof_e = &RuleID{
		Tier:      "",
		Name:      "",
		Namespace: "",
		Direction: rules.RuleDirEgress,
		Index:     0,
		IndexStr:  "0",
		Action:    rules.RuleActionDeny,
	}
)

var _ = Describe("LookupManager tests", func() {
	lm := NewLookupManager()

	DescribeTable(
		"Check default rules are installed",
		func(prefix [64]byte, expectedRuleID *RuleID) {
			rid := lm.GetRuleIDFromNFLOGPrefix(prefix)
			Expect(rid).NotTo(BeNil())
			Expect(*rid).To(Equal(*expectedRuleID))
		},
		Entry("Ingress profile no-match", prefix_nomatch_prof_i, ruleID_nomatch_prof_i),
		Entry("Egress profile no-match", prefix_nomatch_prof_e, ruleID_nomatch_prof_e),
	)

	DescribeTable(
		"Check adding/deleting policy installs/uninstalls rules",
		func(pu *proto.ActivePolicyUpdate, prefix [64]byte, expectedRuleID *RuleID) {
			// Send the policy update and check that the entry is now in the cache
			c := "Querying prefix " + string(prefix[:]) + "\n"
			lm.OnUpdate(pu)
			rid := lm.GetRuleIDFromNFLOGPrefix(prefix)
			Expect(rid).NotTo(BeNil(), c+lm.Dump())
			Expect(*rid).To(Equal(*expectedRuleID))

			// Send a policy delete and check that the entry is not in the cache
			lm.OnUpdate(&proto.ActivePolicyRemove{
				Id: pu.Id,
			})
			rid = lm.GetRuleIDFromNFLOGPrefix(prefix)
			Expect(rid).To(BeNil(), c+lm.Dump())
		},
		Entry("GNP1 (0i0e) no match tier-1 ingress", gnp1_t1_0i0e, prefix_nomatch_t1_i, ruleID_nomatch_t1_i),
		Entry("GNP1 (0i0e) no match tier-1 egress", gnp1_t1_0i0e, prefix_nomatch_t1_e, ruleID_nomatch_t1_e),
		Entry("GNP1 (1i1e) i0", gnp1_t1_1i1e, prefix_gnp1_t1_i0A, ruleID_gnp1_t1_i0A),
		Entry("GNP1 (1i1e) e0", gnp1_t1_1i1e, prefix_gnp1_t1_e0D, ruleID_gnp1_t1_e0D),
		Entry("GNP1 (4i2e) i0", gnp1_t1_4i2e, prefix_gnp1_t1_i0A, ruleID_gnp1_t1_i0A),
		Entry("GNP1 (4i2e) i1", gnp1_t1_4i2e, prefix_gnp1_t1_i1D, ruleID_gnp1_t1_i1D),
		Entry("GNP1 (4i2e) i2", gnp1_t1_4i2e, prefix_gnp1_t1_i2N, ruleID_gnp1_t1_i2N),
		Entry("GNP1 (4i2e) i3", gnp1_t1_4i2e, prefix_gnp1_t1_i3P, ruleID_gnp1_t1_i3P),
		Entry("GNP1 (4i2e) e0", gnp1_t1_4i2e, prefix_gnp1_t1_e0A, ruleID_gnp1_t1_e0A),
		Entry("GNP1 (4i2e) e1", gnp1_t1_4i2e, prefix_gnp1_t1_e1A, ruleID_gnp1_t1_e1A),
		Entry("NP1 (0i1e) no match tier-1 ingress", np1_t1_0i1e, prefix_nomatch_t1_i, ruleID_nomatch_t1_i),
		Entry("NP1 (0i1e) no match tier-1 egress", gnp1_t1_0i0e, prefix_nomatch_t1_e, ruleID_nomatch_t1_e),
		Entry("NP1 (0i1e) i1", np1_t1_0i1e, prefix_np1_t1_e0A, ruleID_np1_t1_e0A),
		Entry("KNP1 (1i0e) no match default ingress", knp1_t1_1i0e, prefix_nomatch_td_i, ruleID_nomatch_td_i),
		Entry("KNP1 (1i0e) no match default egress", knp1_t1_1i0e, prefix_nomatch_td_e, ruleID_nomatch_td_e),
		Entry("KNP1 (1i0e) i0", knp1_t1_1i0e, prefix_knp1_t1_i0D, ruleID_knp1_t1_i0D),
	)

	DescribeTable(
		"Check adding/deleting profile installs/uninstalls rules",
		func(pu *proto.ActiveProfileUpdate, prefix [64]byte, expectedRuleID *RuleID) {
			// Send the policy update and check that the entry is now in the cache
			c := "Querying prefix " + string(prefix[:]) + "\n"
			lm.OnUpdate(pu)
			rid := lm.GetRuleIDFromNFLOGPrefix(prefix)
			Expect(rid).NotTo(BeNil(), c+lm.Dump())
			Expect(*rid).To(Equal(*expectedRuleID))

			// Send a policy delete and check that the entry is not in the cache
			lm.OnUpdate(&proto.ActiveProfileRemove{
				Id: pu.Id,
			})
			rid = lm.GetRuleIDFromNFLOGPrefix(prefix)
			Expect(rid).To(BeNil(), c+lm.Dump())
		},
		Entry("Pr1 (1i1e) i0", pr1_1i1e, prefix_prof_i0D, ruleID_prof_i0D),
		Entry("Pr1 (1i1e) e0", pr1_1i1e, prefix_prof_e0D, ruleID_prof_e0D),
	)

	It("should handle tier drops when there are multiple policies in the same tier", func() {
		By("Creating policy GNP1 in tier 1")
		lm.OnUpdate(gnp1_t1_0i0e)

		By("Checking the default tier drops are cached")
		rid := lm.GetRuleIDFromNFLOGPrefix(prefix_nomatch_t1_i)
		Expect(rid).NotTo(BeNil(), lm.Dump())
		Expect(*rid).To(Equal(*ruleID_nomatch_t1_i))
		rid = lm.GetRuleIDFromNFLOGPrefix(prefix_nomatch_t1_e)
		Expect(rid).NotTo(BeNil(), lm.Dump())
		Expect(*rid).To(Equal(*ruleID_nomatch_t1_e))

		By("Creating policy NP1 in tier 1")
		lm.OnUpdate(np1_t1_0i1e)

		By("Checking the default tier drops are cached")
		rid = lm.GetRuleIDFromNFLOGPrefix(prefix_nomatch_t1_i)
		Expect(rid).NotTo(BeNil(), lm.Dump())
		Expect(*rid).To(Equal(*ruleID_nomatch_t1_i))
		rid = lm.GetRuleIDFromNFLOGPrefix(prefix_nomatch_t1_e)
		Expect(rid).NotTo(BeNil(), lm.Dump())
		Expect(*rid).To(Equal(*ruleID_nomatch_t1_e))

		By("Deleting policy GNP1 in tier 1")
		lm.OnUpdate(&proto.ActivePolicyRemove{
			Id: gnp1_t1_0i0e.Id,
		})

		By("Checking the default tier drops are cached")
		rid = lm.GetRuleIDFromNFLOGPrefix(prefix_nomatch_t1_i)
		Expect(rid).NotTo(BeNil(), lm.Dump())
		Expect(*rid).To(Equal(*ruleID_nomatch_t1_i))
		rid = lm.GetRuleIDFromNFLOGPrefix(prefix_nomatch_t1_e)
		Expect(rid).NotTo(BeNil(), lm.Dump())
		Expect(*rid).To(Equal(*ruleID_nomatch_t1_e))

		By("Deleting policy NP1 in tier 1")
		lm.OnUpdate(&proto.ActivePolicyRemove{
			Id: np1_t1_0i1e.Id,
		})

		By("Checking the default tier drops are no longer cached")
		rid = lm.GetRuleIDFromNFLOGPrefix(prefix_nomatch_t1_i)
		Expect(rid).To(BeNil(), lm.Dump())
		rid = lm.GetRuleIDFromNFLOGPrefix(prefix_nomatch_t1_e)
		Expect(rid).To(BeNil(), lm.Dump())
	})

	It("should handle a policy being updated", func() {
		By("Creating policy GNP1 in tier 1")
		lm.OnUpdate(gnp1_t1_1i1e)

		By("Checking the ingress and egress rules are cached")
		rid := lm.GetRuleIDFromNFLOGPrefix(prefix_gnp1_t1_i0A)
		Expect(rid).NotTo(BeNil(), lm.Dump())
		Expect(*rid).To(Equal(*ruleID_gnp1_t1_i0A))
		rid = lm.GetRuleIDFromNFLOGPrefix(prefix_gnp1_t1_e0D)
		Expect(rid).NotTo(BeNil(), lm.Dump())
		Expect(*rid).To(Equal(*ruleID_gnp1_t1_e0D))

		By("Checking that some ingress and egress rules are not yet cached")
		rid = lm.GetRuleIDFromNFLOGPrefix(prefix_gnp1_t1_i1D)
		Expect(rid).To(BeNil(), lm.Dump())
		rid = lm.GetRuleIDFromNFLOGPrefix(prefix_gnp1_t1_e1A)
		Expect(rid).To(BeNil(), lm.Dump())

		By("Creating policy GNP1 in tier 1")
		lm.OnUpdate(gnp1_t1_4i2e)

		By("Checking the old ingress rule is still cached (it is unchanged by the update)")
		rid = lm.GetRuleIDFromNFLOGPrefix(prefix_gnp1_t1_i0A)
		Expect(rid).NotTo(BeNil(), lm.Dump())
		Expect(*rid).To(Equal(*ruleID_gnp1_t1_i0A))

		By("Checking the old egress rule has been replaced (the rule action has changed)")
		rid = lm.GetRuleIDFromNFLOGPrefix(prefix_gnp1_t1_e0D)
		Expect(rid).To(BeNil(), lm.Dump())
		rid = lm.GetRuleIDFromNFLOGPrefix(prefix_gnp1_t1_e0A)
		Expect(rid).NotTo(BeNil(), lm.Dump())
		Expect(*rid).To(Equal(*ruleID_gnp1_t1_e0A))

		By("Checking the some ingress and egress rules are now cached")
		rid = lm.GetRuleIDFromNFLOGPrefix(prefix_gnp1_t1_i1D)
		Expect(rid).NotTo(BeNil(), lm.Dump())
		Expect(*rid).To(Equal(*ruleID_gnp1_t1_i1D))
		rid = lm.GetRuleIDFromNFLOGPrefix(prefix_gnp1_t1_e1A)
		Expect(rid).NotTo(BeNil(), lm.Dump())
		Expect(*rid).To(Equal(*ruleID_gnp1_t1_e1A))

		By("Update policy GNP1 in tier 1 with more rules")
		lm.OnUpdate(gnp1_t1_1i1e)

		By("Checking the ingress and egress rules are cached")
		rid = lm.GetRuleIDFromNFLOGPrefix(prefix_gnp1_t1_i0A)
		Expect(rid).NotTo(BeNil(), lm.Dump())
		Expect(*rid).To(Equal(*ruleID_gnp1_t1_i0A))
		rid = lm.GetRuleIDFromNFLOGPrefix(prefix_gnp1_t1_e0D)
		Expect(rid).NotTo(BeNil(), lm.Dump())
		Expect(*rid).To(Equal(*ruleID_gnp1_t1_e0D))

		By("Checking the some ingress and egress rules are not cached")
		rid = lm.GetRuleIDFromNFLOGPrefix(prefix_gnp1_t1_i1D)
		Expect(rid).To(BeNil(), lm.Dump())
		rid = lm.GetRuleIDFromNFLOGPrefix(prefix_gnp1_t1_e1A)
		Expect(rid).To(BeNil(), lm.Dump())

		By("Update policy GNP1 in tier 1 with no rules")
		lm.OnUpdate(gnp1_t1_0i0e)

		By("Checking the ingress and egress rules are not cached")
		rid = lm.GetRuleIDFromNFLOGPrefix(prefix_gnp1_t1_i0A)
		Expect(rid).To(BeNil(), lm.Dump())
		rid = lm.GetRuleIDFromNFLOGPrefix(prefix_gnp1_t1_e0D)
		Expect(rid).To(BeNil(), lm.Dump())
		rid = lm.GetRuleIDFromNFLOGPrefix(prefix_gnp1_t1_i1D)
		Expect(rid).To(BeNil(), lm.Dump())
		rid = lm.GetRuleIDFromNFLOGPrefix(prefix_gnp1_t1_e1A)
		Expect(rid).To(BeNil(), lm.Dump())
	})
})

func toprefix(s string) [64]byte {
	p := [64]byte{}
	copy(p[:], []byte(s))
	return p
}
