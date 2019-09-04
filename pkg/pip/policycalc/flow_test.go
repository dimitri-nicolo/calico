package policycalc_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tigera/es-proxy/pkg/pip/policycalc"
)

var _ = Describe("PolicyHit parsing", func() {
	It("handles various policy strings", func() {

		By("creating a PolicyHit from log string: 4|tier1|namespace1/policy1|allow")
		ph1, ok := policycalc.PolicyHitFromFlowLogPolicyString("4|tier1|namespace1/policy1|allow")
		Expect(ok).To(BeTrue())
		Expect(ph1).To(Equal(policycalc.PolicyHit{
			MatchIndex: 4,
			Tier:       "tier1",
			Name:       "namespace1/policy1",
			Action:     policycalc.ActionFlagAllow,
			Staged:     false,
		}))

		By("creating a PolicyHit from log string: 2|tier2|policy3|deny")
		ph2, ok := policycalc.PolicyHitFromFlowLogPolicyString("2|tier2|policy3|deny")
		Expect(ok).To(BeTrue())
		Expect(ph2).To(Equal(policycalc.PolicyHit{
			MatchIndex: 2,
			Tier:       "tier2",
			Name:       "policy3",
			Action:     policycalc.ActionFlagDeny,
			Staged:     false,
		}))

		By("creating a PolicyHit from log string: 5|__PROFILE__|__PROFILE.kns.namespace1|allow")
		ph3, ok := policycalc.PolicyHitFromFlowLogPolicyString("5|__PROFILE__|__PROFILE.kns.namespace1|pass")
		Expect(ok).To(BeTrue())
		Expect(ph3).To(Equal(policycalc.PolicyHit{
			MatchIndex: 5,
			Tier:       "__PROFILE__",
			Name:       "__PROFILE.kns.namespace1",
			Action:     policycalc.ActionFlagNextTier,
			Staged:     false,
		}))

		// Test each section of the comparison
		By("creating a PolicyHit from log string: 1|a|a/a.a|allow")
		ph4, ok := policycalc.PolicyHitFromFlowLogPolicyString("1|a|a/a.a|allow")
		Expect(ok).To(BeTrue())
		Expect(ph4).To(Equal(policycalc.PolicyHit{
			MatchIndex: 1,
			Tier:       "a",
			Name:       "a/a.a",
			Action:     policycalc.ActionFlagAllow,
			Staged:     false,
		}))

		By("creating a PolicyHit from log string: 1|a|a/staged:a.a|allow")
		ph5, ok := policycalc.PolicyHitFromFlowLogPolicyString("1|a|a/staged:a.a|allow")
		Expect(ok).To(BeTrue())
		Expect(ph5).To(Equal(policycalc.PolicyHit{
			MatchIndex: 1,
			Tier:       "a",
			Name:       "a/staged:a.a",
			Action:     policycalc.ActionFlagAllow,
			Staged:     true,
		}))

		By("creating a PolicyHit from log string: 1|b|a/b.a|deny")
		ph6, ok := policycalc.PolicyHitFromFlowLogPolicyString("1|b|a/b.a|deny")
		Expect(ok).To(BeTrue())
		Expect(ph6).To(Equal(policycalc.PolicyHit{
			MatchIndex: 1,
			Tier:       "b",
			Name:       "a/b.a",
			Action:     policycalc.ActionFlagDeny,
			Staged:     false,
		}))

		By("creating a PolicyHit from log string: 1|b|a/b.a|allow")
		ph7, ok := policycalc.PolicyHitFromFlowLogPolicyString("1|b|a/b.a|allow")
		Expect(ok).To(BeTrue())
		Expect(ph7).To(Equal(policycalc.PolicyHit{
			MatchIndex: 1,
			Tier:       "b",
			Name:       "a/b.a",
			Action:     policycalc.ActionFlagAllow,
			Staged:     false,
		}))

		By("creating a PolicyHit from log string: 1|b|b/b.b|allow")
		ph8, ok := policycalc.PolicyHitFromFlowLogPolicyString("1|b|b/b.b|allow")
		Expect(ok).To(BeTrue())
		Expect(ph8).To(Equal(policycalc.PolicyHit{
			MatchIndex: 1,
			Tier:       "b",
			Name:       "b/b.b",
			Action:     policycalc.ActionFlagAllow,
			Staged:     false,
		}))

		By("creating a PolicyHit from log string: 1|b|b/b.a|allow")
		ph9, ok := policycalc.PolicyHitFromFlowLogPolicyString("1|b|b/b.a|allow")
		Expect(ok).To(BeTrue())
		Expect(ph9).To(Equal(policycalc.PolicyHit{
			MatchIndex: 1,
			Tier:       "b",
			Name:       "b/b.a",
			Action:     policycalc.ActionFlagAllow,
			Staged:     false,
		}))

		By("failing to create a PolicyHit from log string: 4|tier1|namespace1/policy1|allow|extra")
		_, ok = policycalc.PolicyHitFromFlowLogPolicyString("4|tier1|namespace1/policy1|allow|extra|extra")
		Expect(ok).To(BeFalse())

		By("failing to create a PolicyHit from log string: x|tier1|namespace1/policy1|allow")
		_, ok = policycalc.PolicyHitFromFlowLogPolicyString("x|tier1|namespace1/policy1|allow|extra")
		Expect(ok).To(BeFalse())
	})
})
