// Copyright (c) 2021-2022 Tigera, Inc. All rights reserved.
package search

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/mock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	libcalicov3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"
	"github.com/tigera/compliance/pkg/datastore"
	v1 "github.com/tigera/es-proxy/pkg/apis/v1"

	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	lmaindex "github.com/projectcalico/calico/lma/pkg/elastic/index"
	"github.com/projectcalico/calico/lma/pkg/httputils"
	calicojson "github.com/projectcalico/calico/lma/pkg/test/json"
	"github.com/projectcalico/calico/lma/pkg/test/thirdpartymock"
)

var (
	//go:embed testdata/valid_request_body.json
	validRequestBody string
	//go:embed testdata/valid_request_body_no_cluster.json
	validRequestBodyNoCluster string
	//go:embed testdata/valid_request_body_page_size_greater_than_lte.json
	validRequestBodyPageSizeGreaterThanLTE string
	//go:embed testdata/valid_request_body_page_size_less_than_gte.json
	validRequestBodyPageSizeLessThanGTE string
	//go:embed testdata/invalid_request_body_badly_formed_string_value.json
	invalidRequestBodyBadlyFormedStringValue string
	//go:embed testdata/invalid_request_body_time_range_contains_invalid_time_value.json
	invalidRequestBodyTimeRangeContainsInvalidTimeValue string

	//go:embed testdata/event_search_request_from_manager.json
	eventSearchRequestFromManager string
	//go:embed testdata/event_search_request.json
	eventSearchRequest string
	//go:embed testdata/event_search_request_selector.json
	eventSearchRequestSelector string
	//go:embed testdata/event_search_request_selector_invalid.json
	eventSearchRequestSelectorInvalid string
	//go:embed testdata/event_search_response.json
	eventSearchResponse string
)

// The user authentication review mock struct implementing the authentication review interface.
type userAuthorizationReviewMock struct {
	verbs []libcalicov3.AuthorizedResourceVerbs
	err   error
}

// PerformReviewForElasticLogs wraps a mocked version of the authorization review method
// PerformReviewForElasticLogs.
func (a userAuthorizationReviewMock) PerformReviewForElasticLogs(
	ctx context.Context, cluster string,
) ([]libcalicov3.AuthorizedResourceVerbs, error) {
	return a.verbs, a.err
}

var _ = Describe("SearchElasticHits", func() {
	var (
		fakeClientSet  datastore.ClientSet
		mockDoer       *thirdpartymock.MockDoer
		userAuthReview userAuthorizationReviewMock
	)

	type Source struct {
		Timestamp time.Time `json:"@timestamp"`
		StartTime time.Time `json:"start_time"`
		EndTime   time.Time `json:"end_time"`
		Action    string    `json:"action"`
		BytesIn   *uint64   `json:"bytes_in"`
		BytesOut  *uint64   `json:"bytes_out"`
	}

	type SomeLog struct {
		ID     string `json:"id"`
		Index  string `json:"index"`
		Source Source `json:"source"`
	}

	BeforeEach(func() {
		fakeClientSet = datastore.NewClientSet(nil, fake.NewSimpleClientset().ProjectcalicoV3())
		mockDoer = new(thirdpartymock.MockDoer)
		userAuthReview = userAuthorizationReviewMock{verbs: []libcalicov3.AuthorizedResourceVerbs{
			{
				APIGroup: "APIGroupVal1",
				Resource: "hostendpoints",
				Verbs: []libcalicov3.AuthorizedResourceVerb{
					{
						Verb: "list",
						ResourceGroups: []libcalicov3.AuthorizedResourceGroup{
							{
								Tier:      "tierVal1",
								Namespace: "namespaceVal1",
							},
							{
								Tier:      "tierVal2",
								Namespace: "namespaceVal2",
							},
						},
					},
					{
						Verb: "list",
						ResourceGroups: []libcalicov3.AuthorizedResourceGroup{
							{
								Tier:      "tierVal1",
								Namespace: "namespaceVal1",
							},
							{
								Tier:      "tierVal2",
								Namespace: "namespaceVal2",
							},
						},
					},
				},
			},
		},
			err: nil,
		}

	})

	AfterEach(func() {
		mockDoer.AssertExpectations(GinkgoT())
	})

	Context("Elasticsearch /search request and response validation", func() {
		fromTime := time.Date(2021, 04, 19, 14, 25, 30, 169827009, time.Local)
		toTime := time.Date(2021, 04, 19, 15, 25, 30, 169827009, time.Local)

		esResponse := []*elastic.SearchHit{
			{
				Index: "tigera_secure_ee_flows",
				Type:  "_doc",
				Id:    "2021-04-19 14:25:30.169827011 -0700 PDT m=+0.121726716",
				Source: calicojson.MustMarshal(calicojson.Map{
					"@timestamp": "2021-04-19T14:25:30.169827011-07:00",
					"start_time": "2021-04-19T14:25:30.169821857-07:00",
					"end_time":   "2021-04-19T14:25:30.169827009-07:00",
					"action":     "action1",
					"bytes_in":   uint64(5456),
					"bytes_out":  uint64(48245),
				}),
			},
			{
				Index: "tigera_secure_ee_flows",
				Type:  "_doc",
				Id:    "2021-04-19 14:25:30.169827010 -0700 PDT m=+0.121726716",
				Source: calicojson.MustMarshal(calicojson.Map{
					"@timestamp": "2021-04-19T15:25:30.169827010-07:00",
					"start_time": "2021-04-19T15:25:30.169821857-07:00",
					"end_time":   "2021-04-19T15:25:30.169827009-07:00",
					"action":     "action2",
					"bytes_in":   uint64(3436),
					"bytes_out":  uint64(68547),
				}),
			},
		}

		t1, _ := time.Parse(time.RFC3339, "2021-04-19T14:25:30.169827011-07:00")
		st1, _ := time.Parse(time.RFC3339, "2021-04-19T14:25:30.169821857-07:00")
		et1, _ := time.Parse(time.RFC3339, "2021-04-19T14:25:30.169827009-07:00")
		bytesIn1 := uint64(5456)
		bytesOut1 := uint64(48245)
		t2, _ := time.Parse(time.RFC3339, "2021-04-19T15:25:30.169827010-07:00")
		st2, _ := time.Parse(time.RFC3339, "2021-04-19T15:25:30.169821857-07:00")
		et2, _ := time.Parse(time.RFC3339, "2021-04-19T15:25:30.169827009-07:00")
		bytesIn2 := uint64(3436)
		bytesOut2 := uint64(68547)
		expectedJSONResponse := []*SomeLog{
			{
				ID:    "id1",
				Index: "index1",
				Source: Source{
					Timestamp: t1,
					StartTime: st1,
					EndTime:   et1,
					Action:    "action1",
					BytesIn:   &bytesIn1,
					BytesOut:  &bytesOut1,
				},
			},
			{
				ID:    "id2",
				Index: "index2",
				Source: Source{
					Timestamp: t2,
					StartTime: st2,
					EndTime:   et2,
					Action:    "action2",
					BytesIn:   &bytesIn2,
					BytesOut:  &bytesOut2,
				},
			},
		}

		It("Should return a valid Elastic search response", func() {
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())

			exp := calicojson.Map{
				"from": 0,
				"query": calicojson.Map{
					"bool": calicojson.Map{
						"filter": []calicojson.Map{
							{
								"range": calicojson.Map{
									"end_time": calicojson.Map{
										"from":          fromTime.Unix(),
										"include_lower": false,
										"include_upper": true,
										"to":            toTime.Unix(),
									},
								},
							},
							{
								"bool": calicojson.Map{
									"should": []calicojson.Map{
										{
											"term": calicojson.Map{"source_type": "hep"},
										},
										{
											"term": calicojson.Map{"dest_type": "hep"},
										},
										{
											"term": calicojson.Map{"source_type": "hep"},
										},
										{
											"term": calicojson.Map{"dest_type": "hep"},
										},
									},
								},
							},
						},
					},
				},
				"size": 100,
				"sort": []calicojson.Map{
					{
						"test": calicojson.Map{
							"order": "desc",
						},
					},
					{
						"test2": calicojson.Map{
							"order": "asc",
						},
					},
				},
			}

			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
				defer GinkgoRecover()
				req := args.Get(0).(*http.Request)

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(req.Body.Close()).NotTo(HaveOccurred())

				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))

				requestJson := map[string]interface{}{}
				Expect(json.Unmarshal(body, &requestJson)).NotTo(HaveOccurred())
				Expect(calicojson.MustUnmarshalToStandardObject(body)).
					To(Equal(calicojson.MustUnmarshalToStandardObject(exp)))
			}).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body: esSearchHitsResultToResponseBody(elastic.SearchResult{
					TookInMillis: 631,
					TimedOut:     false,
					Hits: &elastic.SearchHits{
						Hits:      esResponse,
						TotalHits: &elastic.TotalHits{Value: 2},
					},
				}),
			}, nil)

			params := &v1.SearchRequest{
				ClusterName: "cl_name_val",
				PageSize:    100,
				PageNum:     0,
				TimeRange: &lmav1.TimeRange{
					From: fromTime,
					To:   toTime,
				},
				SortBy: []v1.SearchRequestSortBy{{
					Field:      "test",
					Descending: true,
				}, {
					Field:      "test2",
					Descending: false,
				}},
				Timeout: &metav1.Duration{Duration: 60 * time.Second},
			}

			r, err := http.NewRequest(
				http.MethodGet, "", bytes.NewReader([]byte(validRequestBody)))
			Expect(err).NotTo(HaveOccurred())

			results, err := search(lmaindex.FlowLogs(), params, userAuthReview, fakeClientSet, client, r)
			Expect(err).NotTo(HaveOccurred())
			Expect(results.NumPages).To(Equal(1))
			Expect(results.TotalHits).To(Equal(2))
			Expect(results.TimedOut).To(BeFalse())
			Expect(results.Took.Milliseconds()).To(Equal(int64(631)))
			var someLog *SomeLog
			for i, hit := range results.Hits {
				s, _ := hit.MarshalJSON()
				umerr := json.Unmarshal(s, &someLog)
				Expect(umerr).NotTo(HaveOccurred())
				Expect(someLog.Source.Timestamp).To(Equal(expectedJSONResponse[i].Source.Timestamp))
				Expect(someLog.Source.StartTime).To(Equal(expectedJSONResponse[i].Source.StartTime))
				Expect(someLog.Source.EndTime).To(Equal(expectedJSONResponse[i].Source.EndTime))
				Expect(someLog.Source.Action).To(Equal(expectedJSONResponse[i].Source.Action))
				Expect(someLog.Source.BytesIn).To(Equal(expectedJSONResponse[i].Source.BytesIn))
				Expect(someLog.Source.BytesOut).To(Equal(expectedJSONResponse[i].Source.BytesOut))
			}
		})

		It("Should return a valid Elastic search response (search request with filter)", func() {
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())

			exp := calicojson.Map{
				"from": 0,
				"query": calicojson.Map{
					"bool": calicojson.Map{
						"filter": []calicojson.Map{
							{
								"range": calicojson.Map{
									"time": calicojson.Map{
										"gte": "2022-01-24T00:00:00Z",
										"lte": "2022-01-31T23:59:59Z",
									},
								},
							},
							{
								"term": calicojson.Map{
									"type": "global_alert",
								},
							},
						},
					},
				},
				"size": 100,
				"sort": []calicojson.Map{
					{
						"time": calicojson.Map{
							"order": "desc",
						},
					},
				},
			}

			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
				defer GinkgoRecover()
				req := args.Get(0).(*http.Request)

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(req.Body.Close()).NotTo(HaveOccurred())

				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))

				requestJson := map[string]interface{}{}
				Expect(json.Unmarshal(body, &requestJson)).NotTo(HaveOccurred())
				Expect(calicojson.MustUnmarshalToStandardObject(body)).
					To(Equal(calicojson.MustUnmarshalToStandardObject(exp)))
			}).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body: esSearchHitsResultToResponseBody(elastic.SearchResult{
					TookInMillis: 123,
					TimedOut:     false,
					Hits: &elastic.SearchHits{
						Hits:      esResponse,
						TotalHits: &elastic.TotalHits{Value: 2},
					},
				}),
			}, nil)

			params := &v1.SearchRequest{
				ClusterName: "cl_name_val",
				PageSize:    100,
				PageNum:     0,
				Filter: []json.RawMessage{
					json.RawMessage(`{"range":{"time":{"gte":"2022-01-24T00:00:00Z","lte":"2022-01-31T23:59:59Z"}}}`),
					json.RawMessage(`{"term":{"type":"global_alert"}}`),
				},
				SortBy: []v1.SearchRequestSortBy{{
					Field:      "time",
					Descending: true,
				}},
				Timeout: &metav1.Duration{Duration: 60 * time.Second},
			}

			r, err := http.NewRequest(
				http.MethodGet, "", bytes.NewReader([]byte(validRequestBody)))
			Expect(err).NotTo(HaveOccurred())

			results, err := search(lmaindex.Alerts(), params, userAuthReview, fakeClientSet, client, r)
			Expect(err).NotTo(HaveOccurred())
			Expect(results.NumPages).To(Equal(1))
			Expect(results.TotalHits).To(Equal(2))
			Expect(results.TimedOut).To(BeFalse())
			Expect(results.Took.Milliseconds()).To(Equal(int64(123)))
			var someLog *SomeLog
			for i, hit := range results.Hits {
				s, _ := hit.MarshalJSON()
				umerr := json.Unmarshal(s, &someLog)
				Expect(umerr).NotTo(HaveOccurred())
				Expect(someLog.Source.Timestamp).To(Equal(expectedJSONResponse[i].Source.Timestamp))
				Expect(someLog.Source.StartTime).To(Equal(expectedJSONResponse[i].Source.StartTime))
				Expect(someLog.Source.EndTime).To(Equal(expectedJSONResponse[i].Source.EndTime))
				Expect(someLog.Source.Action).To(Equal(expectedJSONResponse[i].Source.Action))
				Expect(someLog.Source.BytesIn).To(Equal(expectedJSONResponse[i].Source.BytesIn))
				Expect(someLog.Source.BytesOut).To(Equal(expectedJSONResponse[i].Source.BytesOut))
			}
		})

		It("Should return no hits when TotalHits are equal to zero", func() {
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())

			exp := calicojson.Map{
				"from": 0,
				"query": calicojson.Map{
					"bool": calicojson.Map{
						"filter": []calicojson.Map{
							{
								"range": calicojson.Map{
									"end_time": calicojson.Map{
										"from":          fromTime.Unix(),
										"include_lower": false,
										"include_upper": true,
										"to":            toTime.Unix(),
									},
								},
							},
							{
								"bool": calicojson.Map{
									"should": []calicojson.Map{
										{
											"term": calicojson.Map{"source_type": "hep"},
										},
										{
											"term": calicojson.Map{"dest_type": "hep"},
										},
										{
											"term": calicojson.Map{"source_type": "hep"},
										},
										{
											"term": calicojson.Map{"dest_type": "hep"},
										},
									},
								},
							},
						},
					},
				},
				"size": 100,
			}

			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
				defer GinkgoRecover()
				req := args.Get(0).(*http.Request)

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(req.Body.Close()).NotTo(HaveOccurred())

				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))

				requestJson := map[string]interface{}{}
				Expect(json.Unmarshal(body, &requestJson)).NotTo(HaveOccurred())
				Expect(calicojson.MustUnmarshalToStandardObject(body)).
					To(Equal(calicojson.MustUnmarshalToStandardObject(exp)))
			}).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body: esSearchHitsResultToResponseBody(elastic.SearchResult{
					TookInMillis: 631,
					TimedOut:     false,
					Hits: &elastic.SearchHits{
						Hits:      esResponse,
						TotalHits: &elastic.TotalHits{Value: 0},
					},
				}),
			}, nil)

			params := &v1.SearchRequest{
				ClusterName: "cl_name_val",
				PageSize:    100,
				PageNum:     0,
				TimeRange: &lmav1.TimeRange{
					From: fromTime,
					To:   toTime,
				},
				Timeout: &metav1.Duration{Duration: 60 * time.Second},
			}

			r, err := http.NewRequest(
				http.MethodGet, "", bytes.NewReader([]byte(validRequestBody)))
			Expect(err).NotTo(HaveOccurred())

			results, err := search(lmaindex.FlowLogs(), params, userAuthReview, fakeClientSet, client, r)
			Expect(err).NotTo(HaveOccurred())
			Expect(results.NumPages).To(Equal(1))
			Expect(results.TotalHits).To(Equal(0))
			Expect(results.TimedOut).To(BeFalse())
			Expect(results.Took.Milliseconds()).To(Equal(int64(631)))
			var emptyHitsResponse []json.RawMessage
			Expect(results.Hits).To(Equal(emptyHitsResponse))
		})

		It("Should return no hits when ElasticSearch Hits are empty (nil)", func() {
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())

			exp := calicojson.Map{
				"from": 0,
				"query": calicojson.Map{
					"bool": calicojson.Map{
						"filter": []calicojson.Map{
							{
								"range": calicojson.Map{
									"end_time": calicojson.Map{
										"from":          fromTime.Unix(),
										"include_lower": false,
										"include_upper": true,
										"to":            toTime.Unix(),
									},
								},
							},
							{
								"bool": calicojson.Map{
									"should": []calicojson.Map{
										{
											"term": calicojson.Map{"source_type": "hep"},
										},
										{
											"term": calicojson.Map{"dest_type": "hep"},
										},
										{
											"term": calicojson.Map{"source_type": "hep"},
										},
										{
											"term": calicojson.Map{"dest_type": "hep"},
										},
									},
								},
							},
						},
					},
				},
				"size": 100,
			}

			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
				defer GinkgoRecover()
				req := args.Get(0).(*http.Request)

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(req.Body.Close()).NotTo(HaveOccurred())

				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))

				requestJson := map[string]interface{}{}
				Expect(json.Unmarshal(body, &requestJson)).NotTo(HaveOccurred())
				Expect(calicojson.MustUnmarshalToStandardObject(body)).
					To(Equal(calicojson.MustUnmarshalToStandardObject(exp)))
			}).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body: esSearchHitsResultToResponseBody(elastic.SearchResult{
					TookInMillis: 631,
					TimedOut:     false,
					Hits: &elastic.SearchHits{
						Hits:      nil,
						TotalHits: &elastic.TotalHits{Value: 0},
					},
				}),
			}, nil)

			params := &v1.SearchRequest{
				ClusterName: "cl_name_val",
				PageSize:    100,
				PageNum:     0,
				TimeRange: &lmav1.TimeRange{
					From: fromTime,
					To:   toTime,
				},
				Timeout: &metav1.Duration{Duration: 60 * time.Second},
			}

			r, err := http.NewRequest(
				http.MethodGet, "", bytes.NewReader([]byte(validRequestBody)))
			Expect(err).NotTo(HaveOccurred())

			results, err := search(lmaindex.FlowLogs(), params, userAuthReview, fakeClientSet, client, r)
			Expect(err).NotTo(HaveOccurred())
			Expect(results.NumPages).To(Equal(1))
			Expect(results.TotalHits).To(Equal(0))
			Expect(results.TimedOut).To(BeFalse())
			Expect(results.Took.Milliseconds()).To(Equal(int64(631)))
			var emptyHitsResponse []json.RawMessage
			Expect(results.Hits).To(Equal(emptyHitsResponse))
		})

		It("Should return an error with data when ElasticSearch returns TimeOut==true", func() {
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())

			exp := calicojson.Map{
				"from": 0,
				"query": calicojson.Map{
					"bool": calicojson.Map{
						"filter": []calicojson.Map{
							{
								"range": calicojson.Map{
									"end_time": calicojson.Map{
										"from":          fromTime.Unix(),
										"include_lower": false,
										"include_upper": true,
										"to":            toTime.Unix(),
									},
								},
							},
							{
								"bool": calicojson.Map{
									"should": []calicojson.Map{
										{
											"term": calicojson.Map{"source_type": "hep"},
										},
										{
											"term": calicojson.Map{"dest_type": "hep"},
										},
										{
											"term": calicojson.Map{"source_type": "hep"},
										},
										{
											"term": calicojson.Map{"dest_type": "hep"},
										},
									},
								},
							},
						},
					},
				},
				"size": 100,
			}

			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
				defer GinkgoRecover()
				req := args.Get(0).(*http.Request)

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(req.Body.Close()).NotTo(HaveOccurred())

				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))

				requestJson := map[string]interface{}{}
				Expect(json.Unmarshal(body, &requestJson)).NotTo(HaveOccurred())
				Expect(calicojson.MustUnmarshalToStandardObject(body)).
					To(Equal(calicojson.MustUnmarshalToStandardObject(exp)))
			}).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body: esSearchHitsResultToResponseBody(elastic.SearchResult{
					TookInMillis: 10000,
					TimedOut:     true,
					Hits: &elastic.SearchHits{
						Hits:      esResponse,
						TotalHits: &elastic.TotalHits{Value: 2},
					},
				}),
			}, nil)

			params := &v1.SearchRequest{
				ClusterName: "cl_name_val",
				PageSize:    100,
				PageNum:     0,
				TimeRange: &lmav1.TimeRange{
					From: fromTime,
					To:   toTime,
				},
				Timeout: &metav1.Duration{Duration: 60 * time.Second},
			}

			r, err := http.NewRequest(
				http.MethodGet, "", bytes.NewReader([]byte(validRequestBody)))
			Expect(err).NotTo(HaveOccurred())
			results, err := search(lmaindex.FlowLogs(), params, userAuthReview, fakeClientSet, client, r)
			Expect(err).To(HaveOccurred())
			var se *httputils.HttpStatusError
			Expect(errors.As(err, &se)).To(BeTrue())
			Expect(se.Status).To(Equal(500))
			Expect(se.Msg).
				To(Equal("timed out querying tigera_secure_ee_flows.cl_name_val.*"))
			Expect(results).To(BeNil())
		})

		It("Should return an error when ElasticSearch returns an error", func() {
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())

			exp := calicojson.Map{
				"from": 0,
				"query": calicojson.Map{
					"bool": calicojson.Map{
						"filter": []calicojson.Map{
							{
								"range": calicojson.Map{
									"end_time": calicojson.Map{
										"from":          fromTime.Unix(),
										"include_lower": false,
										"include_upper": true,
										"to":            toTime.Unix(),
									},
								},
							},
							{
								"bool": calicojson.Map{
									"should": []calicojson.Map{
										{
											"term": calicojson.Map{"source_type": "hep"},
										},
										{
											"term": calicojson.Map{"dest_type": "hep"},
										},
										{
											"term": calicojson.Map{"source_type": "hep"},
										},
										{
											"term": calicojson.Map{"dest_type": "hep"},
										},
									},
								},
							},
						},
					},
				},
				"size": 100,
			}

			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
				defer GinkgoRecover()
				req := args.Get(0).(*http.Request)

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(req.Body.Close()).NotTo(HaveOccurred())

				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))

				requestJson := map[string]interface{}{}
				Expect(json.Unmarshal(body, &requestJson)).NotTo(HaveOccurred())
				Expect(calicojson.MustUnmarshalToStandardObject(body)).
					To(Equal(calicojson.MustUnmarshalToStandardObject(exp)))
			}).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body: esSearchHitsResultToResponseBody(elastic.SearchResult{
					TookInMillis: 10000,
					TimedOut:     true,
					Hits: &elastic.SearchHits{
						Hits:      esResponse,
						TotalHits: &elastic.TotalHits{Value: 2},
					},
				}),
			}, errors.New("ESError: Elastic search generic error"))

			params := &v1.SearchRequest{
				ClusterName: "cl_name_val",
				PageSize:    100,
				PageNum:     0,
				TimeRange: &lmav1.TimeRange{
					From: fromTime,
					To:   toTime,
				},
				Timeout: &metav1.Duration{Duration: 60 * time.Second},
			}

			r, err := http.NewRequest(
				http.MethodGet, "", bytes.NewReader([]byte(validRequestBody)))
			Expect(err).NotTo(HaveOccurred())

			results, err := search(lmaindex.FlowLogs(), params, userAuthReview, fakeClientSet, client, r)
			Expect(err).To(HaveOccurred())

			var httpErr *httputils.HttpStatusError
			Expect(errors.As(err, &httpErr)).To(BeTrue())
			Expect(httpErr.Status).To(Equal(500))
			Expect(httpErr.Msg).To(Equal("ESError: Elastic search generic error"))
			Expect(results).To(BeNil())
		})
	})

	Context("parseRequestBodyForParams response validation", func() {
		It("Should parse x-cluster-id in the request header when cluster is missing in body", func() {
			r, err := http.NewRequest(
				http.MethodGet, "", bytes.NewReader([]byte(validRequestBodyNoCluster)))
			Expect(err).NotTo(HaveOccurred())
			r.Header.Add("x-cluster-id", "cluster-id-in-header")

			var w http.ResponseWriter
			searchRequest, err := parseRequestBodyForParams(w, r)
			Expect(err).NotTo(HaveOccurred())
			Expect(searchRequest.ClusterName).To(Equal("cluster-id-in-header"))
		})

		It("Should return a SearchError when http request not POST or GET", func() {
			var w http.ResponseWriter
			r, err := http.NewRequest(http.MethodPut, "", bytes.NewReader([]byte(validRequestBody)))
			Expect(err).NotTo(HaveOccurred())

			_, err = parseRequestBodyForParams(w, r)
			Expect(err).To(HaveOccurred())
			var se *httputils.HttpStatusError
			Expect(errors.As(err, &se)).To(BeTrue())
		})

		It("Should return a HttpStatusError when parsing a http status error body", func() {
			r, err := http.NewRequest(
				http.MethodGet, "", bytes.NewReader([]byte(invalidRequestBodyBadlyFormedStringValue)))
			Expect(err).NotTo(HaveOccurred())

			var w http.ResponseWriter
			_, err = parseRequestBodyForParams(w, r)
			Expect(err).To(HaveOccurred())

			var mr *httputils.HttpStatusError
			Expect(errors.As(err, &mr)).To(BeTrue())
		})

		It("Should return an error when parsing a page size that is greater than lte", func() {
			r, err := http.NewRequest(
				http.MethodGet, "", bytes.NewReader([]byte(validRequestBodyPageSizeGreaterThanLTE)))
			Expect(err).NotTo(HaveOccurred())

			var w http.ResponseWriter
			_, err = parseRequestBodyForParams(w, r)
			Expect(err).To(HaveOccurred())

			var se *httputils.HttpStatusError
			Expect(errors.As(err, &se)).To(BeTrue())
			Expect(se.Status).To(Equal(400))
			Expect(se.Msg).To(Equal("error with field PageSize = '1001' (Reason: failed to validate Field: PageSize " +
				"because of Tag: lte )"))
		})

		It("Should return an error when parsing a page size that is less than gte", func() {
			r, err := http.NewRequest(
				http.MethodGet, "", bytes.NewReader([]byte(validRequestBodyPageSizeLessThanGTE)))
			Expect(err).NotTo(HaveOccurred())

			var w http.ResponseWriter
			_, err = parseRequestBodyForParams(w, r)
			Expect(err).To(HaveOccurred())

			var se *httputils.HttpStatusError
			Expect(errors.As(err, &se)).To(BeTrue())
			Expect(se.Status).To(Equal(400))
			Expect(se.Msg).To(Equal("error with field PageSize = '-1' (Reason: failed to validate Field: PageSize "+
				"because of Tag: gte )"), se.Msg)
		})

		It("Should return an error when parsing an invalid value for time_range value", func() {
			r, err := http.NewRequest(
				http.MethodGet, "", bytes.NewReader([]byte(invalidRequestBodyTimeRangeContainsInvalidTimeValue)))
			Expect(err).NotTo(HaveOccurred())

			var w http.ResponseWriter
			_, err = parseRequestBodyForParams(w, r)
			Expect(err).To(HaveOccurred())

			var se *httputils.HttpStatusError
			Expect(errors.As(err, &se)).To(BeTrue())
			Expect(se.Status).To(Equal(400))
			Expect(se.Msg).To(Equal("Request body contains an invalid value for the \"time_range\" "+
				"field (at position 20)"), se.Msg)
		})
	})

	Context("Elasticsearch /events/search request and response validation", func() {
		It("should inject alert exceptions in search request", func() {
			// mock http client
			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
				defer GinkgoRecover()
				req := args.Get(0).(*http.Request)

				// Elastic _search request
				Expect(req.Method).To(Equal(http.MethodPost))
				Expect(req.URL.Path).To(Equal("/tigera_secure_ee_events.cluster*/_search"))

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).NotTo(HaveOccurred())
				// Elastic search request json
				Expect(body).To(Equal([]byte(eventSearchRequest)))

				Expect(req.Body.Close()).NotTo(HaveOccurred())
				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			}).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBuffer([]byte(eventSearchResponse))),
			}, nil)

			// create some alert exceptions
			alertExceptions := []*v3.AlertException{
				// no expiry
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "alert-exception-no-expiry",
						CreationTimestamp: metav1.Now(),
					},
					Spec: v3.AlertExceptionSpec{
						Description: "AlertException no expiry",
						Selector:    "origin = origin1",
					},
				},
				// not expired
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "alert-exception-not-expired",
						CreationTimestamp: metav1.Now(),
					},
					Spec: v3.AlertExceptionSpec{
						Description: "AlertException not expired",
						Selector:    "origin = origin2",
						Period:      &metav1.Duration{Duration: time.Hour},
					},
				},
				// expired
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "alert-exception-expired",
						CreationTimestamp: metav1.Time{Time: metav1.Now().Add(-2 * time.Hour)}, // make this one expire
					},
					Spec: v3.AlertExceptionSpec{
						Description: "AlertException expired",
						Selector:    "origin = origin3",
						Period:      &metav1.Duration{Duration: time.Hour},
					},
				},
			}
			for _, alertException := range alertExceptions {
				_, err := fakeClientSet.AlertExceptions().Create(context.Background(), alertException, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
			}

			// mock lma es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())
			lmaClient := lmaelastic.NewWithClient(client)

			// validate responses
			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(eventSearchRequestFromManager)))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := SearchHandler(lmaindex.Alerts(), userAuthReview, fakeClientSet, lmaClient.Backend())
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))

			var resp v1.SearchResponse
			err = json.Unmarshal(rr.Body.Bytes(), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Hits).To(HaveLen(2))
			Expect(resp.NumPages).To(Equal(1))
			Expect(resp.TimedOut).To(BeFalse())
			Expect(resp.TotalHits).To(Equal(2))
		})

		It("should handle alert exceptions selector AND/OR conditions", func() {
			// mock http client
			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
				defer GinkgoRecover()
				req := args.Get(0).(*http.Request)

				// Elastic _search request
				Expect(req.Method).To(Equal(http.MethodPost))
				Expect(req.URL.Path).To(Equal("/tigera_secure_ee_events.cluster*/_search"))

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).NotTo(HaveOccurred())
				// Elastic search request json
				Expect(body).To(Equal([]byte(eventSearchRequestSelector)))

				Expect(req.Body.Close()).NotTo(HaveOccurred())
				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			}).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBuffer([]byte(eventSearchResponse))),
			}, nil)

			// create some alert exceptions
			alertExceptions := []*v3.AlertException{
				// AND
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "alert-exception-and",
						CreationTimestamp: metav1.Now(),
					},
					Spec: v3.AlertExceptionSpec{
						Description: "AlertException all AND",
						Selector:    "origin = origin1 AND type = global_alert",
					},
				},
				// OR
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "alert-exception-or",
						CreationTimestamp: metav1.Now(),
					},
					Spec: v3.AlertExceptionSpec{
						Description: "AlertException OR",
						Selector:    "origin = origin2 OR type = honeypod",
					},
				},
				// mixed AND / OR
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "alert-exception-and-or",
						CreationTimestamp: metav1.Now(),
					},
					Spec: v3.AlertExceptionSpec{
						Description: "AlertException AND OR",
						Selector:    "origin = origin3 AND type = alert OR source_namespace = ns3",
					},
				},
			}
			for _, alertException := range alertExceptions {
				_, err := fakeClientSet.AlertExceptions().Create(context.Background(), alertException, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
			}

			// mock lma es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())
			lmaClient := lmaelastic.NewWithClient(client)

			// validate responses
			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(eventSearchRequestFromManager)))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := SearchHandler(lmaindex.Alerts(), userAuthReview, fakeClientSet, lmaClient.Backend())
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))

			var resp v1.SearchResponse
			err = json.Unmarshal(rr.Body.Bytes(), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Hits).To(HaveLen(2))
			Expect(resp.NumPages).To(Equal(1))
			Expect(resp.TimedOut).To(BeFalse())
			Expect(resp.TotalHits).To(Equal(2))
		})

		It("should skip invalid alert exceptions selector", func() {
			// mock http client
			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
				defer GinkgoRecover()
				req := args.Get(0).(*http.Request)

				// Elastic _search request
				Expect(req.Method).To(Equal(http.MethodPost))
				Expect(req.URL.Path).To(Equal("/tigera_secure_ee_events.cluster*/_search"))

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).NotTo(HaveOccurred())
				// Elastic search request json
				Expect(body).To(Equal([]byte(eventSearchRequestSelectorInvalid)))

				Expect(req.Body.Close()).NotTo(HaveOccurred())
				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			}).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBuffer([]byte(eventSearchResponse))),
			}, nil)

			// create some alert exceptions
			alertExceptions := []*v3.AlertException{
				// valid selector
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "alert-exception-valid-selector",
						CreationTimestamp: metav1.Now(),
					},
					Spec: v3.AlertExceptionSpec{
						Description: "AlertException valid selector",
						Selector:    "origin = origin1",
					},
				},
				// invalid selector
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "alert-exception-invalid-selector",
						CreationTimestamp: metav1.Now(),
					},
					Spec: v3.AlertExceptionSpec{
						Description: "AlertException invalid selector",
						Selector:    "invalid selector",
					},
				},
			}
			for _, alertException := range alertExceptions {
				_, err := fakeClientSet.AlertExceptions().Create(context.Background(), alertException, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
			}

			// mock lma es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())
			lmaClient := lmaelastic.NewWithClient(client)

			// validate responses
			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(eventSearchRequestFromManager)))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := SearchHandler(lmaindex.Alerts(), userAuthReview, fakeClientSet, lmaClient.Backend())
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))

			var resp v1.SearchResponse
			err = json.Unmarshal(rr.Body.Bytes(), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Hits).To(HaveLen(2))
			Expect(resp.NumPages).To(Equal(1))
			Expect(resp.TimedOut).To(BeFalse())
			Expect(resp.TotalHits).To(Equal(2))
		})

		It("should return error when request is not GET or POST", func() {
			// mock lma es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())
			lmaClient := lmaelastic.NewWithClient(client)

			req, err := http.NewRequest(http.MethodPatch, "", bytes.NewReader([]byte("any")))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := SearchHandler(lmaindex.Alerts(), userAuthReview, fakeClientSet, lmaClient.Backend())
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusMethodNotAllowed))
		})

		It("should return error when request body is not valid", func() {
			// mock lma es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())
			lmaClient := lmaelastic.NewWithClient(client)

			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte("invalid-json-body")))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := SearchHandler(lmaindex.Alerts(), userAuthReview, fakeClientSet, lmaClient.Backend())
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})
	})
})

func esSearchHitsResultToResponseBody(searchResult elastic.SearchResult) io.ReadCloser {
	byts, err := json.Marshal(searchResult)
	if err != nil {
		panic(err)
	}

	return ioutil.NopCloser(bytes.NewBuffer(byts))
}
