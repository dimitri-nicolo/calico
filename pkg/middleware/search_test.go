// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/olivere/elastic/v7"

	eselastic "github.com/tigera/es-proxy/pkg/elastic"
	"github.com/tigera/es-proxy/pkg/httputils"
	esSearch "github.com/tigera/es-proxy/pkg/search"
	calicojson "github.com/tigera/lma/pkg/test/json"
	"github.com/tigera/lma/pkg/test/thirdpartymock"
)

const (
	validRequestBody = `
{
  "cluster": "c_val",
  "page_size": 152,
  "search_after": "sa_val"
}`
	validRequestBodyPageSizeGreaterThanLTE = `
{
  "cluster": "c_val",
  "page_size": 1001,
  "search_after": "sa_val"
}`
	validRequestBodyPageSizeLessThanGTE = `
{
  "cluster": "c_val",
  "page_size": -1,
  "search_after": "sa_val"
}`
	badlyFormedAtPosisitonRequestBody = `
{
  "cluster": c_val,
  "page_size": 152,
  "search_after": "sa_val"
}`
)

var _ = Describe("SearchElasticHits", func() {
	var (
		mockDoer *thirdpartymock.MockDoer
	)

	type SomeLog struct {
		Timestamp time.Time `json:"@timestamp"`
		StartTime time.Time `json:"start_time"`
		EndTime   time.Time `json:"end_time"`
		Action    string    `json:"action"`
		BytesIn   *uint64   `json:"bytes_in"`
		BytesOut  *uint64   `json:"bytes_out"`
	}

	BeforeEach(func() {
		mockDoer = new(thirdpartymock.MockDoer)
	})

	AfterEach(func() {
		mockDoer.AssertExpectations(GinkgoT())
	})

	Context("Elasticsearch /search request and response validation", func() {
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
				Timestamp: t1,
				StartTime: st1,
				EndTime:   et1,
				Action:    "action1",
				BytesIn:   &bytesIn1,
				BytesOut:  &bytesOut1,
			},
			{
				Timestamp: t2,
				StartTime: st2,
				EndTime:   et2,
				Action:    "action2",
				BytesIn:   &bytesIn2,
				BytesOut:  &bytesOut2,
			},
		}

		It("Should return a valid Elastic search response", func() {
			mockDoer = new(thirdpartymock.MockDoer)

			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).ShouldNot(HaveOccurred())

			exp := calicojson.Map{
				"query": calicojson.Map{"bool": calicojson.Map{}},
				"size":  100,
			}

			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
				defer GinkgoRecover()
				req := args.Get(0).(*http.Request)

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(req.Body.Close()).ShouldNot(HaveOccurred())

				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))

				requestJson := map[string]interface{}{}
				Expect(json.Unmarshal(body, &requestJson)).ShouldNot(HaveOccurred())
				Expect(calicojson.MustUnmarshalToStandardObject(body)).
					Should(Equal(calicojson.MustUnmarshalToStandardObject(exp)))
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

			params := &SearchParams{
				ClusterName: "cl_name_val",
				PageSize:    100,
				SearchAfter: nil,
			}
			results, err := search(eselastic.GetFlowsIndex, params, client)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(results.TotalHits).Should(Equal(int64(2)))
			Expect(results.TimedOut).Should(Equal(false))
			Expect(results.TookInMillis).Should(Equal(int64(631)))
			var someLog *SomeLog
			for i, hit := range results.RawHits {
				s, _ := hit.MarshalJSON()
				umerr := json.Unmarshal(s, &someLog)
				Expect(umerr).ShouldNot(HaveOccurred())
				Expect(someLog.Timestamp).Should(Equal(expectedJSONResponse[i].Timestamp))
				Expect(someLog.StartTime).Should(Equal(expectedJSONResponse[i].StartTime))
				Expect(someLog.EndTime).Should(Equal(expectedJSONResponse[i].EndTime))
				Expect(someLog.Action).Should(Equal(expectedJSONResponse[i].Action))
				Expect(someLog.BytesIn).Should(Equal(expectedJSONResponse[i].BytesIn))
				Expect(someLog.BytesOut).Should(Equal(expectedJSONResponse[i].BytesOut))
			}
		})

		It("Should return no hits when TotalHits are equal to zero", func() {
			mockDoer = new(thirdpartymock.MockDoer)

			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).ShouldNot(HaveOccurred())

			exp := calicojson.Map{
				"query": calicojson.Map{"bool": calicojson.Map{}},
				"size":  100,
			}

			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
				defer GinkgoRecover()
				req := args.Get(0).(*http.Request)

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(req.Body.Close()).ShouldNot(HaveOccurred())

				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))

				requestJson := map[string]interface{}{}
				Expect(json.Unmarshal(body, &requestJson)).ShouldNot(HaveOccurred())
				Expect(calicojson.MustUnmarshalToStandardObject(body)).
					Should(Equal(calicojson.MustUnmarshalToStandardObject(exp)))
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

			params := &SearchParams{
				ClusterName: "cl_name_val",
				PageSize:    100,
				SearchAfter: nil,
			}
			results, err := search(eselastic.GetFlowsIndex, params, client)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(results.TotalHits).Should(Equal(int64(0)))
			Expect(results.TimedOut).Should(Equal(false))
			Expect(results.TookInMillis).Should(Equal(int64(631)))
			var emptyHitsResponse []json.RawMessage
			Expect(results.RawHits).Should(Equal(emptyHitsResponse))
		})

		It("Should return no hits when ElasticSearch Hits are empty (nil)", func() {
			mockDoer = new(thirdpartymock.MockDoer)

			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).ShouldNot(HaveOccurred())

			exp := calicojson.Map{
				"query": calicojson.Map{"bool": calicojson.Map{}},
				"size":  100,
			}

			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
				defer GinkgoRecover()
				req := args.Get(0).(*http.Request)

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(req.Body.Close()).ShouldNot(HaveOccurred())

				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))

				requestJson := map[string]interface{}{}
				Expect(json.Unmarshal(body, &requestJson)).ShouldNot(HaveOccurred())
				Expect(calicojson.MustUnmarshalToStandardObject(body)).
					Should(Equal(calicojson.MustUnmarshalToStandardObject(exp)))
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

			params := &SearchParams{
				ClusterName: "cl_name_val",
				PageSize:    100,
				SearchAfter: nil,
			}
			results, err := search(eselastic.GetFlowsIndex, params, client)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(results.TotalHits).Should(Equal(int64(0)))
			Expect(results.TimedOut).Should(Equal(false))
			Expect(results.TookInMillis).Should(Equal(int64(631)))
			var emptyHitsResponse []json.RawMessage
			Expect(results.RawHits).Should(Equal(emptyHitsResponse))
		})

		It("Should return an error with data when ElasticSearch returns TimeOut==true", func() {
			mockDoer = new(thirdpartymock.MockDoer)

			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).ShouldNot(HaveOccurred())

			exp := calicojson.Map{
				"query": calicojson.Map{"bool": calicojson.Map{}},
				"size":  100,
			}

			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
				defer GinkgoRecover()
				req := args.Get(0).(*http.Request)

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(req.Body.Close()).ShouldNot(HaveOccurred())

				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))

				requestJson := map[string]interface{}{}
				Expect(json.Unmarshal(body, &requestJson)).ShouldNot(HaveOccurred())
				Expect(calicojson.MustUnmarshalToStandardObject(body)).
					Should(Equal(calicojson.MustUnmarshalToStandardObject(exp)))
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

			params := &SearchParams{
				ClusterName: "cl_name_val",
				PageSize:    100,
				SearchAfter: nil,
			}
			results, err := search(eselastic.GetFlowsIndex, params, client)
			Expect(err).Should(HaveOccurred())
			var se *httputils.HttpStatusError
			Expect(true).Should(BeEquivalentTo(errors.As(err, &se)))
			Expect(500).Should(BeEquivalentTo(se.Status))
			Expect("timed out querying tigera_secure_ee_flows.cl_name_val.*").
				Should(BeEquivalentTo(se.Msg))
			var response *esSearch.ESResults
			Expect(response).Should(BeEquivalentTo(results))
		})

		It("Should return an error when ElasticSearch returns an error", func() {
			mockDoer = new(thirdpartymock.MockDoer)

			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).ShouldNot(HaveOccurred())

			exp := calicojson.Map{
				"query": calicojson.Map{"bool": calicojson.Map{}},
				"size":  100,
			}

			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
				defer GinkgoRecover()
				req := args.Get(0).(*http.Request)

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(req.Body.Close()).ShouldNot(HaveOccurred())

				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))

				requestJson := map[string]interface{}{}
				Expect(json.Unmarshal(body, &requestJson)).ShouldNot(HaveOccurred())
				Expect(calicojson.MustUnmarshalToStandardObject(body)).
					Should(Equal(calicojson.MustUnmarshalToStandardObject(exp)))
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

			params := &SearchParams{
				ClusterName: "cl_name_val",
				PageSize:    100,
				SearchAfter: nil,
			}
			results, err := search(eselastic.GetFlowsIndex, params, client)
			Expect(err).Should(HaveOccurred())
			var se *httputils.HttpStatusError
			Expect(true).Should(BeEquivalentTo(errors.As(err, &se)))
			Expect(500).Should(BeEquivalentTo(se.Status))
			Expect("ESError: Elastic search generic error").Should(BeEquivalentTo(se.Msg))
			var response *esSearch.ESResults
			Expect(response).Should(BeEquivalentTo(results))
		})
	})

	Context("parseRequestBodyForParams response validation", func() {
		It("Should return a SearchError when http request not POST or GET", func() {
			var w http.ResponseWriter
			r, err := http.NewRequest(http.MethodPut, "", bytes.NewReader([]byte(validRequestBody)))
			Expect(err).NotTo(HaveOccurred())

			_, err = parseRequestBodyForParams(w, r)
			Expect(err).To(HaveOccurred())
			var se *httputils.HttpStatusError
			Expect(true).To(BeEquivalentTo(errors.As(err, &se)))
		})

		It("Should return a HttpStatusError when parsing a http status error body", func() {
			r, err := http.NewRequest(
				http.MethodGet, "", bytes.NewReader([]byte(badlyFormedAtPosisitonRequestBody)))
			Expect(err).NotTo(HaveOccurred())

			var w http.ResponseWriter
			_, err = parseRequestBodyForParams(w, r)
			Expect(err).To(HaveOccurred())

			var mr *httputils.HttpStatusError
			Expect(true).To(BeEquivalentTo(errors.As(err, &mr)))
		})

		It("Should return an error when parsing a page size that is greater than lte", func() {
			r, err := http.NewRequest(
				http.MethodGet, "", bytes.NewReader([]byte(validRequestBodyPageSizeGreaterThanLTE)))
			Expect(err).NotTo(HaveOccurred())

			var w http.ResponseWriter
			_, err = parseRequestBodyForParams(w, r)
			Expect(err).To(HaveOccurred())

			var se *httputils.HttpStatusError
			Expect(true).To(BeEquivalentTo(errors.As(err, &se)))
			Expect(400).To(BeEquivalentTo(se.Status))
			Expect("error with field PageSize = '1001' (Reason: failed to validate Field: PageSize " +
				"because of Tag: lte )").To(BeEquivalentTo(se.Msg))
		})

		It("Should return an error when parsing a page size that is less than gte", func() {
			r, err := http.NewRequest(
				http.MethodGet, "", bytes.NewReader([]byte(validRequestBodyPageSizeLessThanGTE)))
			Expect(err).NotTo(HaveOccurred())

			var w http.ResponseWriter
			_, err = parseRequestBodyForParams(w, r)
			Expect(err).To(HaveOccurred())

			var se *httputils.HttpStatusError
			Expect(true).To(BeEquivalentTo(errors.As(err, &se)))
			Expect(400).To(BeEquivalentTo(se.Status))
			Expect("error with field PageSize = '-1' (Reason: failed to validate Field: PageSize " +
				"because of Tag: gte )").To(BeEquivalentTo(se.Msg))
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
