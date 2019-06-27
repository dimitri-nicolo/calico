package benchmark_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/tigera/compliance/pkg/benchmark"
	"github.com/tigera/compliance/pkg/benchmark/mock"
	"github.com/tigera/compliance/pkg/config"
)

var _ = Describe("Benchmark", func() {
	var (
		cfg       *config.Config
		mockStore *mock.DB
		mockExec  *mock.Executor
		healthy   func(bool)
		isHealthy bool
	)

	BeforeEach(func() {
		cfg = &config.Config{}
		mockStore = mock.NewMockDB()
		mockExec = new(mock.Executor)
		healthy = func(h bool) { isHealthy = h }
	})

	It("should execute a benchmark", func() {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			<-time.After(time.Second)
			cancel()
		}()
		Run(ctx, cfg, mockExec, mockStore, mockStore, healthy)
		Expect(mockStore.NStoreCalls).To(Equal(1))
		Expect(mockStore.NRetrieveCalls).To(Equal(1))
		Expect(isHealthy).To(BeTrue())
	})

	It("should properly compute Benchmarks equality", func() {
		By("empty benchmarks")
		Expect((Benchmarks{}).Equal(Benchmarks{})).To(BeTrue())

		By("error field")
		Expect((Benchmarks{Error: "an error"}).Equal(Benchmarks{Error: "an error"})).To(BeTrue())
		Expect((Benchmarks{Error: "an error"}).Equal(Benchmarks{Error: "diff error"})).To(BeFalse())

		By("metadata fields")
		Expect((Benchmarks{Version: "1.1"}).Equal(Benchmarks{Version: "1.1"})).To(BeTrue())
		Expect((Benchmarks{Version: "1.1"}).Equal(Benchmarks{Version: "1.1.1"})).To(BeFalse())
		Expect((Benchmarks{Type: TypeKubernetes}).Equal(Benchmarks{Type: TypeKubernetes})).To(BeTrue())
		Expect((Benchmarks{Type: TypeKubernetes}).Equal(Benchmarks{Type: "docker"})).To(BeFalse())
		Expect((Benchmarks{NodeName: "kadm-ms"}).Equal(Benchmarks{NodeName: "kadm-ms"})).To(BeTrue())
		Expect((Benchmarks{NodeName: "kadm-ms"}).Equal(Benchmarks{NodeName: "kadm-node-0"})).To(BeFalse())

		By("tests")
		Expect((Benchmarks{Tests: []Test{{"section", "sectionDesc", "testNum", "testDesc", "testInfo", "status", true}}}).Equal(
			Benchmarks{Tests: []Test{{"section", "sectionDesc", "testNum", "testDesc", "testInfo", "status", true}}},
		)).To(BeTrue())

		Expect((Benchmarks{Tests: []Test{{"section", "sectionDesc", "testNum", "testDesc", "testInfo", "status", true}}}).Equal(
			Benchmarks{Tests: []Test{{"section", "sectionDesc", "testNum", "testDesc", "testInfo", "status", false}}},
		)).To(BeFalse())
	})
})
