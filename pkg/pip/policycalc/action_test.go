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

		By("checking for deny")
		Expect((policycalc.ActionFlagAllow | policycalc.ActionFlagDeny).Deny()).To(BeFalse())
		Expect((policycalc.ActionFlagDeny).Deny()).To(BeTrue())
		Expect((policycalc.ActionFlagDeny | policycalc.ActionFlagNextTier).Deny()).To(BeTrue())

		By("checking for allow")
		Expect((policycalc.ActionFlagAllow | policycalc.ActionFlagDeny).Allow()).To(BeFalse())
		Expect((policycalc.ActionFlagAllow).Allow()).To(BeTrue())
		Expect((policycalc.ActionFlagAllow | policycalc.ActionFlagNextTier).Allow()).To(BeTrue())
	})
})
