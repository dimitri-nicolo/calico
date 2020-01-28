// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package api_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/tigera/lma/pkg/api"
)

var _ = Describe("PolicyHit parsing", func() {
	It("handles various policy strings", func() {

		By("creating a PolicyHit from log string: 4|tier1|namespace1/policy1|allow")
		ph1, ok := api.PolicyHitFromFlowLogPolicyString("4|tier1|namespace1/tier1.policy1|allow", 5)
		Expect(ok).To(BeTrue())
		Expect(ph1).To(Equal(api.PolicyHit{
			MatchIndex: 4,
			Tier:       "tier1",
			Namespace:  "namespace1",
			Name:       "tier1.policy1",
			Action:     api.ActionFlagAllow,
			Staged:     false,
			Count:      5,
		}))
		Expect(ph1.IsKubernetes()).To(BeFalse())
		Expect(ph1.FlowLogName()).To(Equal("namespace1/tier1.policy1"))

		By("creating a PolicyHit from log string: 2|tier2|policy3|deny")
		ph2, ok := api.PolicyHitFromFlowLogPolicyString("2|tier2|tier2.policy3|deny", 0)
		Expect(ok).To(BeTrue())
		Expect(ph2).To(Equal(api.PolicyHit{
			MatchIndex: 2,
			Tier:       "tier2",
			Name:       "tier2.policy3",
			Action:     api.ActionFlagDeny,
			Staged:     false,
		}))
		Expect(ph2.IsKubernetes()).To(BeFalse())
		Expect(ph2.FlowLogName()).To(Equal("tier2.policy3"))

		By("creating a PolicyHit from log string: 5|__PROFILE__|__PROFILE.kns.namespace1|allow")
		ph3, ok := api.PolicyHitFromFlowLogPolicyString("5|__PROFILE__|__PROFILE.kns.namespace1|pass", 0)
		Expect(ok).To(BeTrue())
		Expect(ph3).To(Equal(api.PolicyHit{
			MatchIndex: 5,
			Tier:       "__PROFILE__",
			Name:       "__PROFILE.kns.namespace1",
			Action:     api.ActionFlagNextTier,
			Staged:     false,
		}))
		Expect(ph3.IsKubernetes()).To(BeFalse())
		Expect(ph3.FlowLogName()).To(Equal("__PROFILE.kns.namespace1"))

		// Test each section of the comparison
		By("creating a PolicyHit from log string: 1|a|a/a.a|allow")
		ph4, ok := api.PolicyHitFromFlowLogPolicyString("1|a|a/a.a|allow", 0)
		Expect(ok).To(BeTrue())
		Expect(ph4).To(Equal(api.PolicyHit{
			MatchIndex: 1,
			Tier:       "a",
			Namespace:  "a",
			Name:       "a.a",
			Action:     api.ActionFlagAllow,
			Staged:     false,
		}))
		Expect(ph4.IsKubernetes()).To(BeFalse())
		Expect(ph4.FlowLogName()).To(Equal("a/a.a"))

		By("creating a PolicyHit from log string: 1|a|a/staged:a.a|allow")
		ph5, ok := api.PolicyHitFromFlowLogPolicyString("1|a|a/staged:a.a|allow", 0)
		Expect(ok).To(BeTrue())
		Expect(ph5).To(Equal(api.PolicyHit{
			MatchIndex: 1,
			Tier:       "a",
			Namespace:  "a",
			Name:       "a.a",
			Action:     api.ActionFlagAllow,
			Staged:     true,
		}))
		Expect(ph5.IsKubernetes()).To(BeFalse())
		Expect(ph5.FlowLogName()).To(Equal("a/staged:a.a"))

		By("creating a PolicyHit from log string: 1|b|a/b.a|deny")
		ph6, ok := api.PolicyHitFromFlowLogPolicyString("1|b|a/b.a|deny", 0)
		Expect(ok).To(BeTrue())
		Expect(ph6).To(Equal(api.PolicyHit{
			MatchIndex: 1,
			Tier:       "b",
			Namespace:  "a",
			Name:       "b.a",
			Action:     api.ActionFlagDeny,
			Staged:     false,
		}))
		Expect(ph6.IsKubernetes()).To(BeFalse())
		Expect(ph6.FlowLogName()).To(Equal("a/b.a"))

		By("creating a PolicyHit from log string: 1|b|a/b.a|allow")
		ph7, ok := api.PolicyHitFromFlowLogPolicyString("1|b|a/b.a|allow", 0)
		Expect(ok).To(BeTrue())
		Expect(ph7).To(Equal(api.PolicyHit{
			MatchIndex: 1,
			Tier:       "b",
			Namespace:  "a",
			Name:       "b.a",
			Action:     api.ActionFlagAllow,
			Staged:     false,
		}))
		Expect(ph7.IsKubernetes()).To(BeFalse())
		Expect(ph7.FlowLogName()).To(Equal("a/b.a"))

		By("creating a PolicyHit from log string: 1|b|b/b.b|allow")
		ph8, ok := api.PolicyHitFromFlowLogPolicyString("1|b|b/b.b|allow", 0)
		Expect(ok).To(BeTrue())
		Expect(ph8).To(Equal(api.PolicyHit{
			MatchIndex: 1,
			Tier:       "b",
			Namespace:  "b",
			Name:       "b.b",
			Action:     api.ActionFlagAllow,
			Staged:     false,
		}))
		Expect(ph8.IsKubernetes()).To(BeFalse())
		Expect(ph8.FlowLogName()).To(Equal("b/b.b"))

		By("creating a PolicyHit from log string: 1|b|b/b.a|allow")
		ph9, ok := api.PolicyHitFromFlowLogPolicyString("1|b|b/b.a|allow", 0)
		Expect(ok).To(BeTrue())
		Expect(ph9).To(Equal(api.PolicyHit{
			MatchIndex: 1,
			Tier:       "b",
			Namespace:  "b",
			Name:       "b.a",
			Action:     api.ActionFlagAllow,
			Staged:     false,
		}))
		Expect(ph9.IsKubernetes()).To(BeFalse())
		Expect(ph9.FlowLogName()).To(Equal("b/b.a"))

		By("creating a PolicyHit from log string: 1|default|ns1/knp.default.thing|allow")
		ph10, ok := api.PolicyHitFromFlowLogPolicyString("1|default|ns1/knp.default.thing|allow", 0)
		Expect(ok).To(BeTrue())
		Expect(ph10).To(Equal(api.PolicyHit{
			MatchIndex: 1,
			Tier:       "default",
			Namespace:  "ns1",
			Name:       "knp.default.thing",
			Action:     api.ActionFlagAllow,
			Staged:     false,
		}))
		Expect(ph10.IsKubernetes()).To(BeTrue())
		Expect(ph10.FlowLogName()).To(Equal("ns1/knp.default.thing"))

		By("creating a PolicyHit from log string: 1|default|ns1/staged:knp.default.thing|allow")
		ph11, ok := api.PolicyHitFromFlowLogPolicyString("1|default|ns1/staged:knp.default.thing|allow", 0)
		Expect(ok).To(BeTrue())
		Expect(ph11).To(Equal(api.PolicyHit{
			MatchIndex: 1,
			Tier:       "default",
			Namespace:  "ns1",
			Name:       "knp.default.thing",
			Action:     api.ActionFlagAllow,
			Staged:     true,
		}))
		Expect(ph11.IsKubernetes()).To(BeTrue())
		Expect(ph11.FlowLogName()).To(Equal("ns1/staged:knp.default.thing"))
		Expect(ph11.ToFlowLogPolicyString()).To(Equal("1|default|ns1/staged:knp.default.thing|allow"))

		By("failing to create a PolicyHit from log string: 4|tier1|namespace1/policy1|allow|extra")
		_, ok = api.PolicyHitFromFlowLogPolicyString("4|tier1|namespace1/policy1|allow|extra|extra", 0)
		Expect(ok).To(BeFalse())

		By("failing to create a PolicyHit from log string: x|tier1|namespace1/policy1|allow")
		_, ok = api.PolicyHitFromFlowLogPolicyString("x|tier1|namespace1/policy1|allow|extra", 0)
		Expect(ok).To(BeFalse())
	})
})
