package policycalc_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/tigera/es-proxy/pkg/pip/policycalc"
)

var _ = Describe("Test action flags", func() {
	It("handles flag checks correctly", func() {
		By("checking for indeterminate")
		Expect((policycalc.ActionFlagAllow | policycalc.ActionFlagDeny).Indeterminate()).To(BeTrue())
		Expect((policycalc.ActionFlagAllow | policycalc.ActionFlagDeny | policycalc.ActionFlagNextTier).Indeterminate()).To(BeTrue())
		Expect((policycalc.ActionFlagDeny).Indeterminate()).To(BeFalse())
		Expect((policycalc.ActionFlagAllow | policycalc.ActionFlagNextTier).Indeterminate()).To(BeFalse())
	})

	It("handles conversion correctly", func() {
		By("checking flags to names")
		Expect(policycalc.ActionFlagAllow.ToFlowActionString()).To(Equal(policycalc.ActionAllow))
		Expect(policycalc.ActionFlagDeny.ToFlowActionString()).To(Equal(policycalc.ActionDeny))
		Expect(policycalc.ActionFlagEndOfTierDeny.ToFlowActionString()).To(Equal(policycalc.ActionDeny))
		Expect((policycalc.ActionFlagDeny | policycalc.ActionFlagEndOfTierDeny).ToFlowActionString()).To(Equal(policycalc.ActionDeny))
		Expect(policycalc.ActionFlagNextTier.ToFlowActionString()).To(Equal(policycalc.ActionInvalid))
		Expect((policycalc.ActionFlagAllow | policycalc.ActionFlagDeny).ToFlowActionString()).To(Equal(policycalc.ActionUnknown))
		Expect((policycalc.ActionFlagAllow | policycalc.ActionFlagEndOfTierDeny).ToFlowActionString()).To(Equal(policycalc.ActionUnknown))
		Expect((policycalc.ActionFlagAllow | policycalc.ActionFlagNextTier).ToFlowActionString()).To(Equal(policycalc.ActionAllow))
		Expect((policycalc.ActionFlagDeny | policycalc.ActionFlagNextTier).ToFlowActionString()).To(Equal(policycalc.ActionDeny))
	})
})
