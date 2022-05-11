package benchmark_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/lma/pkg/api"
)

var _ = Describe("Benchmark", func() {

	It("should properly compute Benchmarks equality", func() {
		By("empty benchmarks")
		Expect((api.Benchmarks{}).Equal(api.Benchmarks{})).To(BeTrue())

		By("error field")
		Expect((api.Benchmarks{Error: "an error"}).Equal(api.Benchmarks{Error: "an error"})).To(BeTrue())
		Expect((api.Benchmarks{Error: "an error"}).Equal(api.Benchmarks{Error: "diff error"})).To(BeFalse())

		By("metadata fields")
		Expect((api.Benchmarks{Version: "1.1"}).Equal(api.Benchmarks{Version: "1.1"})).To(BeTrue())
		Expect((api.Benchmarks{Version: "1.1"}).Equal(api.Benchmarks{Version: "1.1.1"})).To(BeFalse())
		Expect((api.Benchmarks{Type: api.TypeKubernetes}).Equal(api.Benchmarks{Type: api.TypeKubernetes})).To(BeTrue())
		Expect((api.Benchmarks{Type: api.TypeKubernetes}).Equal(api.Benchmarks{Type: "docker"})).To(BeFalse())
		Expect((api.Benchmarks{NodeName: "kadm-ms"}).Equal(api.Benchmarks{NodeName: "kadm-ms"})).To(BeTrue())
		Expect((api.Benchmarks{NodeName: "kadm-ms"}).Equal(api.Benchmarks{NodeName: "kadm-node-0"})).To(BeFalse())

		By("tests")
		Expect((api.Benchmarks{Tests: []api.BenchmarkTest{{Section: "section", SectionDesc: "sectionDesc", TestNumber: "testNum", TestDesc: "testDesc", TestInfo: "testInfo", Status: "status", Scored: true}}}).Equal(
			api.Benchmarks{Tests: []api.BenchmarkTest{{Section: "section", SectionDesc: "sectionDesc", TestNumber: "testNum", TestDesc: "testDesc", TestInfo: "testInfo", Status: "status", Scored: true}}},
		)).To(BeTrue())

		Expect((api.Benchmarks{Tests: []api.BenchmarkTest{{Section: "section", SectionDesc: "sectionDesc", TestNumber: "testNum", TestDesc: "testDesc", TestInfo: "testInfo", Status: "status", Scored: true}}}).Equal(
			api.Benchmarks{Tests: []api.BenchmarkTest{{Section: "section", SectionDesc: "sectionDesc", TestNumber: "testNum", TestDesc: "testDesc", TestInfo: "testInfo", Status: "status", Scored: false}}},
		)).To(BeFalse())
	})
})
