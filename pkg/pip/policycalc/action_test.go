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
		Expect(policycalc.ActionFlagAllow.ToAction()).To(Equal(policycalc.ActionAllow))
		Expect(policycalc.ActionFlagDeny.ToAction()).To(Equal(policycalc.ActionDeny))
		Expect(policycalc.ActionFlagNextTier.ToAction()).To(Equal(policycalc.ActionNextTier))
		Expect((policycalc.ActionFlagAllow | policycalc.ActionFlagDeny).ToAction()).To(Equal(policycalc.ActionUnknown))
		Expect((policycalc.ActionFlagAllow | policycalc.ActionFlagNextTier).ToAction()).To(Equal(policycalc.ActionInvalid))

		By("checking names to flags")
		Expect(policycalc.ActionFlagFromAction(policycalc.ActionAllow)).To(Equal(policycalc.ActionFlagAllow))
		Expect(policycalc.ActionFlagFromAction(policycalc.ActionDeny)).To(Equal(policycalc.ActionFlagDeny))
		Expect(policycalc.ActionFlagFromAction(policycalc.ActionNextTier)).To(Equal(policycalc.ActionFlagNextTier))
		Expect(policycalc.ActionFlagFromAction(policycalc.ActionUnknown)).To(BeZero())
		Expect(policycalc.ActionFlagFromAction(policycalc.ActionInvalid)).To(BeZero())
	})
})
