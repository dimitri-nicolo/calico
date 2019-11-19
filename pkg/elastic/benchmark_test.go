package elastic_test

import (
	"context"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/lma/pkg/api"
	. "github.com/tigera/lma/pkg/elastic"
)

var _ = Describe("Benchmark elastic tests", func() {
	var (
		elasticClient Client
		// Truncate the time at 1s so that we can compare against results from ES which trucate.
		time8 = metav1.NewTime(time.Now().Truncate(1 * time.Second))
		time7 = metav1.NewTime(time8.Add(-time.Hour))
		time6 = metav1.NewTime(time7.Add(-time.Hour))
		time5 = metav1.NewTime(time6.Add(-time.Hour))
		time4 = metav1.NewTime(time5.Add(-time.Hour))
		time3 = metav1.NewTime(time4.Add(-time.Hour))
		time2 = metav1.NewTime(time3.Add(-time.Hour))
		time1 = metav1.NewTime(time2.Add(-time.Hour))

		br api.BenchmarksResult

		b1 = &api.Benchmarks{
			Version:   "1.0.0",
			Type:      api.TypeKubernetes,
			Timestamp: time2,
			NodeName:  "node1",
		}
		b2 = &api.Benchmarks{
			Version:   "1.0.1",
			Type:      api.TypeKubernetes,
			Timestamp: time4,
			NodeName:  "node2",
		}
		b3 = &api.Benchmarks{
			Version:   "1.0.1",
			Type:      api.TypeKubernetes,
			Timestamp: time6,
			NodeName:  "node3",
		}
		b4 = &api.Benchmarks{
			Version:   "1.0.2",
			Type:      api.TypeKubernetes,
			Timestamp: time7,
			NodeName:  "node2",
		}
	)

	BeforeEach(func() {
		os.Setenv("LOG_LEVEL", "debug")
		os.Setenv("ELASTIC_HOST", "localhost")
		os.Setenv("ELASTIC_INDEX_SUFFIX", "test_cluster")
		os.Setenv("ELASTIC_SCHEME", "http")
		elasticClient = MustGetElasticClient()
		elasticClient.(Resetable).Reset()
	})

	It("should store and retrieve benchmarks properly", func() {
		By("storing a benchmark before the interval")
		elasticClient.StoreBenchmarks(context.Background(), b1)
		elasticClient.StoreBenchmarks(context.Background(), b2)
		elasticClient.StoreBenchmarks(context.Background(), b3)
		elasticClient.StoreBenchmarks(context.Background(), b4)

		By("verifying that we can query each report - this gives ES time to index the benchmarks")
		Eventually(func() (*api.Benchmarks, error) {
			return elasticClient.GetBenchmarks(context.Background(), b4.UID())
		}).ShouldNot(BeNil())
		Eventually(func() (*api.Benchmarks, error) {
			return elasticClient.GetBenchmarks(context.Background(), b3.UID())
		}).ShouldNot(BeNil())
		Eventually(func() (*api.Benchmarks, error) {
			return elasticClient.GetBenchmarks(context.Background(), b2.UID())
		}).ShouldNot(BeNil())
		Eventually(func() (*api.Benchmarks, error) {
			return elasticClient.GetBenchmarks(context.Background(), b1.UID())
		}).ShouldNot(BeNil())

		By("reading benchmarks where earlier benchmarks is omitted")
		res := elasticClient.RetrieveLatestBenchmarks(
			context.Background(), api.TypeKubernetes, nil, time3.Time, time8.Time,
		)

		Eventually(res).Should(Receive(&br))
		Expect(br.Err).To(BeNil())
		Expect(br.Benchmarks.Timestamp).To(Equal(b4.Timestamp))

		Eventually(res).Should(Receive(&br))
		Expect(br.Err).To(BeNil())
		Expect(br.Benchmarks).To(Equal(b3))

		Eventually(res).Should(BeClosed())

		By("changing time range to read previously omitted benchmarks")
		res = elasticClient.RetrieveLatestBenchmarks(
			context.Background(), api.TypeKubernetes, nil, time3.Time, time5.Time,
		)

		Eventually(res).Should(Receive(&br))
		Expect(br.Benchmarks).To(Equal(b2))
		Eventually(res).Should(BeClosed())

		By("filtering in node1 and node2")
		res = elasticClient.RetrieveLatestBenchmarks(
			context.Background(), api.TypeKubernetes,
			[]api.BenchmarkFilter{{NodeNames: []string{"node1", "node2"}}}, time1.Time, time8.Time,
		)

		Eventually(res).Should(Receive(&br))
		Expect(br.Err).To(BeNil())
		Expect(br.Benchmarks).To(Equal(b4))

		Eventually(res).Should(Receive(&br))
		Expect(br.Err).To(BeNil())
		Expect(br.Benchmarks).To(Equal(b1))

		Eventually(res).Should(BeClosed())

		By("filtering in version 1.0.1 *and* node3")
		res = elasticClient.RetrieveLatestBenchmarks(
			context.Background(), api.TypeKubernetes,
			[]api.BenchmarkFilter{{Version: "1.0.1", NodeNames: []string{"node3"}}}, time1.Time, time8.Time,
		)

		Eventually(res).Should(Receive(&br))
		Expect(br.Err).To(BeNil())
		Expect(br.Benchmarks).To(Equal(b3))

		Eventually(res).Should(BeClosed())

		By("filtering in version 1.0.1")
		res = elasticClient.RetrieveLatestBenchmarks(
			context.Background(), api.TypeKubernetes,
			[]api.BenchmarkFilter{{Version: "1.0.1"}}, time1.Time, time8.Time,
		)

		Eventually(res).Should(Receive(&br))
		Expect(br.Err).To(BeNil())
		Expect(br.Benchmarks).To(Equal(b3))

		Eventually(res).Should(Receive(&br))
		Expect(br.Err).To(BeNil())
		Expect(br.Benchmarks).To(Equal(b2))

		Eventually(res).Should(BeClosed())

		By("filtering in version 1.0.1 or node1")
		res = elasticClient.RetrieveLatestBenchmarks(
			context.Background(), api.TypeKubernetes,
			[]api.BenchmarkFilter{{Version: "1.0.1"}, {NodeNames: []string{"node1"}}}, time1.Time, time8.Time,
		)

		Eventually(res).Should(Receive(&br))
		Expect(br.Err).To(BeNil())
		Expect(br.Benchmarks).To(Equal(b3))

		Eventually(res).Should(Receive(&br))
		Expect(br.Err).To(BeNil())
		Expect(br.Benchmarks).To(Equal(b2))

		Eventually(res).Should(Receive(&br))
		Expect(br.Err).To(BeNil())
		Expect(br.Benchmarks).To(Equal(b1))

		Eventually(res).Should(BeClosed())
	})

	It("should create an index with the correct index settings", func() {
		cfg := MustLoadConfig()
		cfg.ElasticReplicas = 2
		cfg.ElasticShards = 7

		elasticClient, err := NewFromConfig(cfg)
		Expect(err).ToNot(HaveOccurred())
		elasticClient.StoreBenchmarks(context.Background(), b1)

		index := elasticClient.ClusterIndex(BenchmarksIndex, b1.Timestamp.Format(IndexTimeFormat))

		testIndexSettings(cfg, index, map[string]string{
			"number_of_replicas": "2",
			"number_of_shards":   "7",
		})
	})
})
