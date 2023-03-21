package elastic_test

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/olivere/elastic/v7"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/lma/pkg/api"
	. "github.com/projectcalico/calico/lma/pkg/elastic"
)

func reset(c Client) {
	_, _ = c.Backend().DeleteIndex(
		c.ClusterIndex("tigera_secure_ee_compliance_reports", "*"),
		c.ClusterIndex("tigera_secure_ee_snapshots", "*"),
		c.ClusterIndex("tigera_secure_ee_audit_", "*"),
		c.ClusterIndex("tigera_secure_ee_benchmark_results", "*"),
	).Do(context.Background())
}

var _ = Describe("Elasticsearch events index", func() {
	var (
		elasticClientManagement                   Client
		elasticClientManaged                      Client
		ctx                                       context.Context
		cfg                                       *Config
		data                                      api.EventsData
		managementClusterName, managedClusterName string
	)

	BeforeSuite(func() {
		managementClusterName = "test_management_cluster"
		managedClusterName = "test_managed_cluster"
		err := os.Setenv("LOG_LEVEL", "debug")
		Expect(err).NotTo(HaveOccurred())
		err = os.Setenv("ELASTIC_HOST", "localhost")
		Expect(err).NotTo(HaveOccurred())
		err = os.Setenv("ELASTIC_SCHEME", "http")
		Expect(err).NotTo(HaveOccurred())

		err = os.Setenv("ELASTIC_INDEX_SUFFIX", managementClusterName)
		Expect(err).NotTo(HaveOccurred())
		elasticClientManagement = MustGetElasticClient()
		reset(elasticClientManagement)

		err = os.Setenv("ELASTIC_INDEX_SUFFIX", managedClusterName)
		Expect(err).NotTo(HaveOccurred())
		elasticClientManaged = MustGetElasticClient()
		reset(elasticClientManaged)

		cfg = MustLoadConfig()
	})

	// TODO CASEY: Port these tests to Linseed
	Context("create and update events index", func() {
		var (
			esClient *elastic.Client
			err      error
		)
		BeforeEach(func() {
			ctx = context.Background()
			esClient, err = getESClient(cfg)
			Expect(err).ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			fmt.Printf("\nDelete after each")
			deleteIndex(cfg, EventsIndex)
		})

		It("creates events index with dynamic mapping turned off", func() {
			err := elasticClientManagement.CreateEventsIndex(ctx)
			Expect(err).ShouldNot(HaveOccurred())

			indexName := fmt.Sprintf("%s.%s.lma", EventsIndex, managementClusterName)
			mapping, err := esClient.GetMapping().Index(indexName).Do(ctx)
			Expect(err).ShouldNot(HaveOccurred())

			// unwrap the mappings object and verify properties
			m1, ok := mapping[indexName]
			Expect(ok).Should(BeTrue())
			m2, ok := m1.(map[string]interface{})
			Expect(ok).Should(BeTrue())
			m3, ok := m2["mappings"]
			Expect(ok).Should(BeTrue())
			m4, ok := m3.(map[string]interface{})
			Expect(ok).Should(BeTrue())
			dynamicSetting, ok := m4["dynamic"]
			Expect(ok).Should(BeTrue())
			Expect(dynamicSetting).Should(Equal("false"))
		})

		It("should update index mapping for both old and new events indices", func() {
			newIndexName := fmt.Sprintf("%s.%s.lma", EventsIndex, managementClusterName)

			err = elasticClientManagement.CreateEventsIndex(ctx)
			Expect(err).NotTo(HaveOccurred())

			// update to some random mappings
			randomMapping := `{"properties":{"description":{"type":"keyword"}}}`
			_, err = esClient.PutMapping().Index(newIndexName).BodyString(randomMapping).Do(ctx)
			Expect(err).NotTo(HaveOccurred())

			// CreateEventsIndex will check for existence and update to the latest mapping
			err = elasticClientManagement.CreateEventsIndex(ctx)
			Expect(err).NotTo(HaveOccurred())

			resp, err := esClient.GetMapping().Index(newIndexName).Do(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).To(HaveKey(newIndexName))
			v, ok := resp[newIndexName].(map[string]interface{})
			Expect(ok).To(BeTrue())
			mappings, ok := v["mappings"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			properties, ok := mappings["properties"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(properties).To(HaveKey("dismissed"))
		})
	})

	Context("read and write data", func() {
		BeforeEach(func() {
			ctx = context.Background()
			err := elasticClientManagement.CreateEventsIndex(ctx)
			Expect(err).ShouldNot(HaveOccurred())

			err = elasticClientManaged.CreateEventsIndex(ctx)
			Expect(err).ShouldNot(HaveOccurred())

			data.Time = time.Now().Unix()
			data.Type = "global_alert"
			data.Description = "test es in management cluster"
			data.SourceIP = sptr("1.2.3.4")
		})

		AfterEach(func() {
			deleteIndex(cfg, fmt.Sprintf("%s*", EventsIndex))
		})

		It("writes only into new events index", func() {
			esClient, err := getESClient(cfg)
			Expect(err).ShouldNot(HaveOccurred())

			By("putting document into index and verifying it exists", func() {
				data.Origin = "new_events_index_00"
				_, err = elasticClientManagement.PutSecurityEventWithID(ctx, data, "sample_id_test_542")
				Expect(err).ShouldNot(HaveOccurred())
				// wait for put to reflect
				time.Sleep(5 * time.Second)

				for op := range elasticClientManagement.SearchSecurityEvents(ctx, nil, nil, nil, true) {
					compareEventData(op, data)
				}
			})

			By("verifying old index doesn't have the indexed document", func() {
				oldIndexName := fmt.Sprintf("%s.%s", EventsIndex, managementClusterName)
				_, err = esClient.Get().Index(oldIndexName).Id("sample_id_test_542").Do(ctx)
				Expect(err).Should(HaveOccurred())
			})
		})

		It("should store events data in Elasticsearch", func() {
			data.Origin = "01_test_lma"

			By("inserting data into Elasticsearch for management cluster", func() {
				_, err := elasticClientManagement.PutSecurityEventWithID(ctx, data, "sample_id_01_01")
				Expect(err).ShouldNot(HaveOccurred())
			})

			By("querying data from Elasticsearch for a management cluster", func() {
				for op := range elasticClientManagement.SearchSecurityEvents(ctx, nil, nil, nil, true) {
					compareEventData(op, data)
				}
			})

			By("querying data from Elasticsearch for managed cluster", func() {
				for range elasticClientManaged.SearchSecurityEvents(ctx, nil, nil, nil, false) {
					Fail("Elastic query returned data when not expected.")
				}
			})
		})

		It("should add security events by ID", func() {
			data.Origin = "02_test_lma"

			By("inserting data into Elasticsearch", func() {
				_, err := elasticClientManaged.PutSecurityEventWithID(ctx, data, "sample_id_01_02")
				Expect(err).ShouldNot(HaveOccurred())
			})

			By("querying data from the Elasticsearch", func() {
				for op := range elasticClientManaged.SearchSecurityEvents(ctx, nil, nil, nil, false) {
					compareEventData(op, data)
				}
			})

			By("updating data in existing security event", func() {
				data.Origin = "02_test_lma_updated"
				_, err := elasticClientManaged.PutSecurityEventWithID(ctx, data, "sample_id_01_02")
				Expect(err).ShouldNot(HaveOccurred())

				for op := range elasticClientManaged.SearchSecurityEvents(ctx, nil, nil, nil, false) {
					compareEventData(op, data)
				}
			})
		})

		It("should not return error for missing ID while adding events", func() {
			data.Origin = "03_test_lma"
			By("inserting data into Elasticsearch", func() {
				_, err := elasticClientManaged.PutSecurityEventWithID(ctx, data, "")
				Expect(err).ShouldNot(HaveOccurred())
			})

			By("Verifying the inserted data in Elasticsearch", func() {
				for op := range elasticClientManaged.SearchSecurityEvents(ctx, nil, nil, nil, false) {
					compareEventData(op, data)
				}
			})
		})

		It("should store security events without ID", func() {
			data.Origin = "04_test_lma"

			By("inserting data into Elasticsearch", func() {
				_, err := elasticClientManagement.PutSecurityEvent(ctx, data)
				Expect(err).ShouldNot(HaveOccurred())
			})

			By("Verifying the inserted data in Elasticsearch", func() {
				for op := range elasticClientManagement.SearchSecurityEvents(ctx, nil, nil, nil, false) {
					compareEventData(op, data)
				}
			})

			By("Verifying the data inserted in management cluster index is not in managed index", func() {
				for range elasticClientManaged.SearchSecurityEvents(ctx, nil, nil, nil, false) {
					Fail("Elastic query returned data when not expected.")
				}
			})
		})

		It("should send bulk events to Elasticsearch", func() {
			bulkCommit := 0
			afterFn := func(executionId int64, requests []elastic.BulkableRequest, response *elastic.BulkResponse, err error) {
				Expect(response.Errors).Should(BeFalse())
				Expect(err).ShouldNot(HaveOccurred())
				bulkCommit++
			}
			err := elasticClientManaged.BulkProcessorInitialize(ctx, afterFn)
			Expect(err).ShouldNot(HaveOccurred())

			for i := 0; i < 22; i++ {
				data.Origin = fmt.Sprintf("05_test_lma_%d", i)
				err = elasticClientManaged.PutBulkSecurityEvent(data)
				Expect(err).ShouldNot(HaveOccurred())
			}
			err = elasticClientManaged.BulkProcessorClose()
			Expect(err).ShouldNot(HaveOccurred())

			// wait for bulk to commit
			Eventually(bulkCommit).Should(Equal(1))

			Eventually(func() int {
				eventCount := 0
				for res := range elasticClientManaged.SearchSecurityEvents(ctx, nil, nil, nil, false) {
					eventCount++
					Expect(res.Err).ShouldNot(HaveOccurred())
				}
				return eventCount
			}, 10*time.Second, 3*time.Second).Should(Equal(22))
		})

		It("should return paginated security events", func() {
			bulkCommit := 0
			afterFn := func(executionId int64, requests []elastic.BulkableRequest, response *elastic.BulkResponse, err error) {
				Expect(response.Errors).Should(BeFalse())
				Expect(err).ShouldNot(HaveOccurred())
				bulkCommit++
			}
			err := elasticClientManaged.BulkProcessorInitialize(ctx, afterFn)
			Expect(err).ShouldNot(HaveOccurred())

			for i := 0; i < 220; i++ {
				data.Origin = fmt.Sprintf("07_test_lma_%d", i)
				err = elasticClientManaged.PutBulkSecurityEvent(data)
				Expect(err).ShouldNot(HaveOccurred())
			}
			err = elasticClientManaged.BulkProcessorClose()
			Expect(err).ShouldNot(HaveOccurred())

			// wait for bulk to commit
			Eventually(bulkCommit).Should(Equal(1))

			Eventually(func() int {
				eventCount := 0
				for res := range elasticClientManaged.SearchSecurityEvents(ctx, nil, nil, nil, false) {
					eventCount++
					Expect(res.Err).ShouldNot(HaveOccurred())
				}
				return eventCount
			}, 10*time.Second, 3*time.Second).Should(Equal(220))
		})
	})
})

func sptr(s string) *string {
	sCopy := s
	return &sCopy
}

func compareEventData(actual *api.EventResult, expected api.EventsData) {
	Expect(actual.Err).ShouldNot(HaveOccurred())
	Expect(actual.Type).Should(Equal(expected.Type))
	Expect(actual.Description).Should(Equal(expected.Description))
	Expect(actual.Origin).Should(Equal(expected.Origin))
	Expect(actual.SourceIP).Should(Equal(expected.SourceIP))
}
