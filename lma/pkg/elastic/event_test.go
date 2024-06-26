package elastic_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"time"

	"github.com/olivere/elastic/v7"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

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
				_, err = putSecurityEventWithID(ctx, elasticClientManagement, data, "sample_id_test_542")
				Expect(err).ShouldNot(HaveOccurred())
				// wait for put to reflect
				time.Sleep(5 * time.Second)

				for op := range searchSecurityEvents(ctx, elasticClientManagement, nil, nil, nil, true) {
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
				_, err := putSecurityEventWithID(ctx, elasticClientManagement, data, "sample_id_01_01")
				Expect(err).ShouldNot(HaveOccurred())
			})

			By("querying data from Elasticsearch for a management cluster", func() {
				for op := range searchSecurityEvents(ctx, elasticClientManagement, nil, nil, nil, true) {
					compareEventData(op, data)
				}
			})

			By("querying data from Elasticsearch for managed cluster", func() {
				for range searchSecurityEvents(ctx, elasticClientManaged, nil, nil, nil, false) {
					Fail("Elastic query returned data when not expected.")
				}
			})
		})

		It("should add security events by ID", func() {
			data.Origin = "02_test_lma"

			By("inserting data into Elasticsearch", func() {
				_, err := putSecurityEventWithID(ctx, elasticClientManaged, data, "sample_id_01_02")
				Expect(err).ShouldNot(HaveOccurred())
			})

			By("querying data from the Elasticsearch", func() {
				for op := range searchSecurityEvents(ctx, elasticClientManaged, nil, nil, nil, false) {
					compareEventData(op, data)
				}
			})

			By("updating data in existing security event", func() {
				data.Origin = "02_test_lma_updated"
				_, err := putSecurityEventWithID(ctx, elasticClientManaged, data, "sample_id_01_02")
				Expect(err).ShouldNot(HaveOccurred())

				for op := range searchSecurityEvents(ctx, elasticClientManaged, nil, nil, nil, false) {
					compareEventData(op, data)
				}
			})
		})

		It("should not return error for missing ID while adding events", func() {
			data.Origin = "03_test_lma"
			By("inserting data into Elasticsearch", func() {
				_, err := putSecurityEventWithID(ctx, elasticClientManaged, data, "")
				Expect(err).ShouldNot(HaveOccurred())
			})

			By("Verifying the inserted data in Elasticsearch", func() {
				for op := range searchSecurityEvents(ctx, elasticClientManaged, nil, nil, nil, false) {
					compareEventData(op, data)
				}
			})
		})

		It("should store security events without ID", func() {
			data.Origin = "04_test_lma"

			By("inserting data into Elasticsearch", func() {
				_, err := putSecurityEvent(ctx, elasticClientManagement, data)
				Expect(err).ShouldNot(HaveOccurred())
			})

			By("Verifying the inserted data in Elasticsearch", func() {
				for op := range searchSecurityEvents(ctx, elasticClientManagement, nil, nil, nil, false) {
					compareEventData(op, data)
				}
			})

			By("Verifying the data inserted in management cluster index is not in managed index", func() {
				for range searchSecurityEvents(ctx, elasticClientManaged, nil, nil, nil, false) {
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
				for res := range searchSecurityEvents(ctx, elasticClientManaged, nil, nil, nil, false) {
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
				for res := range searchSecurityEvents(ctx, elasticClientManaged, nil, nil, nil, false) {
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

const (
	resultBucketSize = 1000
)

// searchSecurityEvents is now only used by the above test code.  Ideally the test code should be
// updated to use Linseed, but that hasn't happened yet and so we still have this implementation to
// search using LMA and Elastic directly.
func searchSecurityEvents(ctx context.Context, c Client, start, end *time.Time, filterData []api.EventsSearchFields, allClusters bool) <-chan *api.EventResult {
	resultChan := make(chan *api.EventResult, resultBucketSize)
	var index string
	if allClusters {
		// When allClusters is true use wildcard to query all events index instead of alias to
		// cover older managed clusters that do not use alias for events index.
		index = api.EventIndexWildCardPattern
	} else {
		index = c.ClusterAlias(EventsIndex)
	}
	queries := constructEventLogsQuery(start, end, filterData)
	go func() {
		defer close(resultChan)
		scroll := c.Backend().Scroll(index).
			Size(DefaultPageSize).
			Query(queries).
			Sort(api.EventTime, true)

		// Issue the query to Elasticsearch and send results out through the resultsChan.
		// We only terminate the search if when there are no more results to scroll through.
		for {
			log.Debug("Issuing alerts search query")
			res, err := scroll.Do(ctx)
			if err == io.EOF {
				break
			}
			if err != nil {
				log.WithError(err).Error("Failed to search alert logs")

				resultChan <- &api.EventResult{Err: err}
				return
			}
			if res == nil {
				err = fmt.Errorf("search expected results != nil; got nil")
			} else if res.Hits == nil {
				err = fmt.Errorf("search expected results.Hits != nil; got nil")
			} else if len(res.Hits.Hits) == 0 {
				err = fmt.Errorf("search expected results.Hits.Hits > 0; got 0")
			}
			if err != nil {
				log.WithError(err).Warn("Unexpected results from alert logs search")
				resultChan <- &api.EventResult{Err: err}
				return
			}
			log.WithField("latency (ms)", res.TookInMillis).Debug("query success")

			// Pushes the search results into the channel.
			for _, hit := range res.Hits.Hits {
				var a api.EventsData
				if err := json.Unmarshal(hit.Source, &a); err != nil {
					log.WithFields(log.Fields{"index": hit.Index, "id": hit.Id}).WithError(err).Warn("failed to unmarshal event json")
					continue
				}
				resultChan <- &api.EventResult{EventsData: &a, ID: hit.Id}
			}
		}
	}()

	return resultChan
}

func constructEventLogsQuery(start *time.Time, end *time.Time, filterData []api.EventsSearchFields) elastic.Query {
	queries := []elastic.Query{}
	for _, data := range filterData {
		innerQ := []elastic.Query{}
		v := reflect.ValueOf(data)
		values := make([]interface{}, v.NumField())
		for i := 0; i < v.NumField(); i++ {
			innerQ = append(innerQ, elastic.NewMatchQuery(v.Field(i).String(), values[i]))
		}
		queries = append(queries, elastic.NewBoolQuery().Must(innerQ...))
	}

	if start != nil || end != nil {
		rangeQuery := elastic.NewRangeQuery(api.EventTime)
		if start != nil {
			rangeQuery = rangeQuery.From(*start)
		}
		if end != nil {
			rangeQuery = rangeQuery.To(*end)
		}
		queries = append(queries, rangeQuery)
	}

	return elastic.NewBoolQuery().Must(queries...)
}

func putSecurityEvent(ctx context.Context, c Client, data api.EventsData) (*elastic.IndexResponse, error) {
	alias := c.ClusterAlias(EventsIndex)

	// Marshall the api.EventsData to ignore empty values
	b, err := json.Marshal(data)
	if err != nil {
		log.WithError(err).Error("failed to marshall")
		return nil, err
	}
	return c.Backend().Index().Index(alias).BodyString(string(b)).Do(ctx)
}

func putSecurityEventWithID(ctx context.Context, c Client, data api.EventsData, id string) (*elastic.IndexResponse, error) {
	alias := c.ClusterAlias(EventsIndex)

	// Marshall the api.EventsData to ignore empty values
	b, err := json.Marshal(data)
	if err != nil {
		log.WithError(err).Error("failed to marshall")
		return nil, err
	}
	return c.Backend().Index().Index(alias).Id(id).BodyString(string(b)).Do(ctx)
}
