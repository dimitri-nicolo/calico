package benchmark_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"strings"

	"github.com/tigera/compliance/pkg/cis"
)

// Override directory check.
func configExists(cfgPath string) bool {
	paths := strings.Split(cfgPath, "/")
	if len(paths) > 3 {
		version := paths[3]
		switch version {
		// kube-bench supported configuration
		case "1.6", "1.7", "1.8", "1.11", "1.13":
			return true
		// kube-bench possible future support
		case "1.15":
			return true
		}
	}
	return false
}

var _ = Describe("CIS", func() {
	b := &cis.Benchmarker{ConfigChecker: configExists}

	It("should work for differnt k8s versions", func() {
		var ev, dv string

		By("checking basic (non-default) case")
		dv = "1.7.0"
		ev = "1.7"
		Expect(b.GetClosestConfig(dv)).To(Equal(ev))

		By("checking when k8s version is different than CIS benchmark version")
		dv = "1.10.0"
		ev = "1.8"
		Expect(b.GetClosestConfig(dv)).To(Equal(ev))

		By("checking possible future CIS benchmark change")
		dv = "1.15.0"
		ev = "1.15"
		Expect(b.GetClosestConfig(dv)).To(Equal(ev))
	})
})
