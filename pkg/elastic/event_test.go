package elastic_test

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/olivere/elastic/v7"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tigera/lma/pkg/api"
	. "github.com/tigera/lma/pkg/elastic"
)

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
		elasticClientManagement.(Resetable).Reset()

		err = os.Setenv("ELASTIC_INDEX_SUFFIX", managedClusterName)
		Expect(err).NotTo(HaveOccurred())
		elasticClientManaged = MustGetElasticClient()
		elasticClientManaged.(Resetable).Reset()

		cfg = MustLoadConfig()
	})

	Context("create index", func() {
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

		It("attaches alias to both old and new events index", func() {
			oldIndexName := fmt.Sprintf("%s.%s", EventsIndex, managementClusterName)
			newIndexName := fmt.Sprintf("%s.%s.lma", EventsIndex, managementClusterName)
			alias := fmt.Sprintf("%s.%s.", EventsIndex, managementClusterName)
			_, err := esClient.Index().Index(oldIndexName).BodyJson(map[string]interface{}{"description": "test old index"}).Do(ctx)
			Expect(err).ShouldNot(HaveOccurred())

			err = elasticClientManagement.CreateEventsIndex(ctx)
			Expect(err).ShouldNot(HaveOccurred())

			aliases, err := esClient.Aliases().Index(oldIndexName, newIndexName).Do(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(len(aliases.Indices)).Should(Equal(2))
			Expect(aliases.Indices[oldIndexName].Aliases[0].AliasName).Should(Equal(alias))
			Expect(aliases.Indices[oldIndexName].Aliases[0].IsWriteIndex).Should(BeFalse())

			Expect(aliases.Indices[newIndexName].Aliases[0].AliasName).Should(Equal(alias))
			Expect(aliases.Indices[newIndexName].Aliases[0].IsWriteIndex).Should(BeTrue())
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
			data.Type = "alert"
			data.Description = "test es in management cluster"
			data.SourceIP = sptr("1.2.3.4")
		})

		AfterEach(func() {
			deleteIndex(cfg, fmt.Sprintf("%s*", EventsIndex))
		})

		It("reads from both old and new events index", func() {
			esClient, err := getESClient(cfg)
			Expect(err).ShouldNot(HaveOccurred())

			By("putting documents into old index", func() {
				data.Origin = "old_events_index"
				oldIndexName := fmt.Sprintf("%s.%s", EventsIndex, managementClusterName)
				_, err = esClient.Index().Index(oldIndexName).BodyJson(data).Do(ctx)
				Expect(err).ShouldNot(HaveOccurred())
			})

			By("putting document into new index", func() {
				data.Origin = "new_events_index"
				_, err = elasticClientManagement.PutSecurityEvent(ctx, data)
				Expect(err).ShouldNot(HaveOccurred())
			})

			// wait for put to reflect
			time.Sleep(5 * time.Second)

			count := 0
			for op := range elasticClientManagement.SearchSecurityEvents(ctx, nil, nil, nil, true) {
				Expect(op.Origin).Should(BeElementOf([]string{"new_events_index", "old_events_index"}))
				count++
			}
			Expect(count).Should(Equal(2))
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

		It("should not return error for missing docID while adding events", func() {
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
			Eventually(func() int { return bulkCommit }).Should(Equal(1))
			time.Sleep(5 * time.Second)

			eventCount := 0
			for res := range elasticClientManaged.SearchSecurityEvents(ctx, nil, nil, nil, false) {
				eventCount++
				Expect(res.Err).ShouldNot(HaveOccurred())
			}
			Expect(eventCount).Should(BeNumerically("==", 22))
		})

		It("should send bulk events to Elasticsearch when bulkaction is set", func() {
			bulkCommit := 0
			afterFn := func(executionId int64, requests []elastic.BulkableRequest, response *elastic.BulkResponse, err error) {
				Expect(response.Errors).Should(BeFalse())
				Expect(err).ShouldNot(HaveOccurred())
				bulkCommit++
			}
			err := elasticClientManaged.BulkProcessorInitializeWithFlush(ctx, afterFn, 5)
			Expect(err).ShouldNot(HaveOccurred())

			for i := 0; i < 22; i++ {
				data.Origin = fmt.Sprintf("06_test_lma_%d", i)
				err = elasticClientManaged.PutBulkSecurityEvent(data)
				Expect(err).ShouldNot(HaveOccurred())
			}

			By("verifying requests are flushed after reaching the bulkaction count", func() {
				// wait for bulk to commit
				Eventually(func() int { return bulkCommit }).Should(Equal(4))
				time.Sleep(5 * time.Second)

				eventCount := 0
				for res := range elasticClientManaged.SearchSecurityEvents(ctx, nil, nil, nil, false) {
					eventCount++
					Expect(res.Err).ShouldNot(HaveOccurred())
				}
				Expect(eventCount).Should(BeNumerically("==", 20))
			})

			By("verifying that pending requests are flushed on closing bulk processor service", func() {
				err = elasticClientManaged.BulkProcessorClose()
				Expect(err).ShouldNot(HaveOccurred())

				// wait for bulk to commit
				Eventually(func() int { return bulkCommit }).Should(Equal(5))
				time.Sleep(5 * time.Second)

				eventCount := 0
				for res := range elasticClientManaged.SearchSecurityEvents(ctx, nil, nil, nil, false) {
					eventCount++
					Expect(res.Err).ShouldNot(HaveOccurred())
				}
				Expect(eventCount).Should(BeNumerically("==", 22))
			})
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
			Eventually(func() int { return bulkCommit }).Should(Equal(1))
			time.Sleep(5 * time.Second)

			eventCount := 0
			for res := range elasticClientManaged.SearchSecurityEvents(ctx, nil, nil, nil, false) {
				eventCount++
				Expect(res.Err).ShouldNot(HaveOccurred())
			}
			Expect(eventCount).Should(BeNumerically("==", 220))
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
