// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package aggregation_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	. "github.com/projectcalico/calico/es-proxy/pkg/middleware/aggregation"

	"github.com/olivere/elastic/v7"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	v1 "github.com/projectcalico/calico/es-proxy/pkg/apis/v1"

	lmaapi "github.com/projectcalico/calico/lma/pkg/apis/v1"
	lmaindex "github.com/projectcalico/calico/lma/pkg/elastic/index"
	"github.com/projectcalico/calico/lma/pkg/httputils"
)

// MockBackend implements a mock backend for test purposes.
type MockBackend struct {
	// Fill in by test.
	AuthorizationReviewResp    []v3.AuthorizedResourceVerbs
	AuthorizationReviewRespErr error
	RunQueryResp               elastic.Aggregations
	RunQueryRespErr            error

	// Filled in by backend processing.
	RequestData *RequestData
	Query       elastic.Query
}

func (m *MockBackend) PerformUserAuthorizationReview(ctx context.Context, rd *RequestData) ([]v3.AuthorizedResourceVerbs, error) {
	m.RequestData = rd
	return m.AuthorizationReviewResp, m.AuthorizationReviewRespErr
}

func (m *MockBackend) RunQuery(cxt context.Context, rd *RequestData, query elastic.Query) (elastic.Aggregations, error) {
	m.RequestData = rd
	m.Query = query
	return m.RunQueryResp, m.RunQueryRespErr
}

var (
	timeFrom, _     = time.Parse(time.RFC3339, "2021-05-30T21:23:10Z")
	timeTo5Mins, _  = time.Parse(time.RFC3339, "2021-05-30T21:28:10Z")
	timeTo60Mins, _ = time.Parse(time.RFC3339, "2021-05-30T22:23:10Z")
)

const (
	// The following are used across multiple tests, so define once here.
	query5Mins = `{
          "bool": {
            "must": [
              {
                "term": {
                  "dest_namespace": {
                    "value": "abc"
                  }
                }
              },
              {
                "bool": {
                  "should": [
                    {
                      "bool": {
                        "must": [
                          {
                            "term": {
                              "source_type": "ns"
                            }
                          },
                          {
                            "term": {
                              "source_namespace": "ns1"
                            }
                          }
                        ]
                      }
                    },
                    {
                      "bool": {
                        "must": [
                          {
                            "term": {
                              "dest_type": "ns"
                            }
                          },
                          {
                            "term": {
                              "dest_namespace": "ns1"
                            }
                          }
                        ]
                      }
                    }
                  ]
                }
              },
              {
                "range": {
                  "end_time": {
                    "from": 1622409790,
                    "include_lower": false,
                    "include_upper": true,
                    "to": 1622410090
                  }
                }
              }
            ]
          }
        }`

	query60Mins = `{
          "bool": {
            "must": [
              {
                "term": {
                  "dest_namespace": {
                    "value": "abc"
                  }
                }
              },
              {
                "bool": {
                  "should": [
                    {
                      "bool": {
                        "must": [
                          {
                            "term": {
                              "source_type": "ns"
                            }
                          },
                          {
                            "term": {
                              "source_namespace": "ns1"
                            }
                          }
                        ]
                      }
                    },
                    {
                      "bool": {
                        "must": [
                          {
                            "term": {
                              "dest_type": "ns"
                            }
                          },
                          {
                            "term": {
                              "dest_namespace": "ns1"
                            }
                          }
                        ]
                      }
                    }
                  ]
                }
              },
              {
                "range": {
                  "end_time": {
                    "from": 1622409790,
                    "include_lower": false,
                    "include_upper": true,
                    "to": 1622413390
                  }
                }
              }
            ]
          }
        }`

	bucketsNoTimeSeries = `{
            "buckets": [
                {
                    "start_time": "2021-05-30T21:23:10Z",
                    "aggregations": {
                        "agg1": {"abc": "123"}
                    }
                }
            ]
        }`
)

var _ = Describe("Aggregation tests", func() {
	DescribeTable("valid request parameters",
		func(sgr v1.AggregationRequest, backend *MockBackend, code int, query string, resp string) {
			// Create a service graph.
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			sg := NewAggregationHandlerWithBackend(lmaindex.FlowLogs(), backend)

			// Marshal the request and create an HTTP request
			sgrb, err := json.Marshal(sgr)
			Expect(err).NotTo(HaveOccurred())
			body := ioutil.NopCloser(bytes.NewReader(sgrb))
			req, err := http.NewRequest("POST", "/aggregation", body)
			Expect(err).NotTo(HaveOccurred())
			req = req.WithContext(ctx)

			// Pass it through the handler
			writer := httptest.NewRecorder()
			sg.ServeHTTP(writer, req)
			Expect(writer.Code).To(Equal(code))

			// The remaining checks are only applicable if the response was 200 OK.
			if code != http.StatusOK {
				Expect(strings.TrimSpace(writer.Body.String())).To(Equal(resp))
				return
			}

			// Check the query matches.
			if backend.AuthorizationReviewRespErr == nil {
				source, err := backend.Query.Source()
				Expect(err).NotTo(HaveOccurred())
				j, err := json.Marshal(source)
				Expect(err).NotTo(HaveOccurred())
				Expect(j).To(MatchJSON(query))
			}

			// Parse the response. Unmarshal into a generic map for easier comparison (also we haven't implemented
			// all the unmarshal methods required)
			var actual v1.AggregationResponse
			err = json.Unmarshal(writer.Body.Bytes(), &actual)
			Expect(err).NotTo(HaveOccurred())

			var expected v1.AggregationResponse
			err = json.Unmarshal([]byte(resp), &expected)
			Expect(err).NotTo(HaveOccurred())

			Expect(writer.Body.String()).To(MatchJSON(resp), writer.Body.String())
		},

		Entry("Simple request with selector, 5 min interval, no time series",
			v1.AggregationRequest{
				Cluster:           "",
				TimeRange:         &lmaapi.TimeRange{From: timeFrom, To: timeTo5Mins},
				Selector:          "dest_namespace = 'abc'",
				IncludeTimeSeries: false,
				Aggregations:      map[string]json.RawMessage{"agg1": json.RawMessage(`{"abc": "def"}`)},
				Timeout:           1000,
			}, &MockBackend{
				AuthorizationReviewResp: []v3.AuthorizedResourceVerbs{{
					APIGroup: "projectcalico.org",
					Resource: "networksets",
					Verbs: []v3.AuthorizedResourceVerb{{
						Verb: "list",
						ResourceGroups: []v3.AuthorizedResourceGroup{{
							Namespace: "ns1",
						}},
					}},
				}},
				AuthorizationReviewRespErr: nil,
				RunQueryResp: map[string]json.RawMessage{
					"agg1": json.RawMessage(`{"abc": "123"}`),
				},
				RunQueryRespErr: nil,
			},
			http.StatusOK,
			query5Mins,
			bucketsNoTimeSeries,
		),

		Entry("Simple request with selector, 5 min interval, request time series - but range too small, so non-time series selected",
			v1.AggregationRequest{
				Cluster:           "",
				TimeRange:         &lmaapi.TimeRange{From: timeFrom, To: timeTo5Mins},
				Selector:          "dest_namespace = 'abc'",
				IncludeTimeSeries: true,
				Aggregations:      map[string]json.RawMessage{"agg1": json.RawMessage(`{"abc": "def"}`)},
				Timeout:           1000,
			}, &MockBackend{
				AuthorizationReviewResp: []v3.AuthorizedResourceVerbs{{
					APIGroup: "projectcalico.org",
					Resource: "networksets",
					Verbs: []v3.AuthorizedResourceVerb{{
						Verb: "list",
						ResourceGroups: []v3.AuthorizedResourceGroup{{
							Namespace: "ns1",
						}},
					}},
				}},
				AuthorizationReviewRespErr: nil,
				RunQueryResp: map[string]json.RawMessage{
					"agg1": json.RawMessage(`{"abc": "123"}`),
				},
				RunQueryRespErr: nil,
			},
			http.StatusOK,
			query5Mins,
			bucketsNoTimeSeries,
		),

		Entry("Simple request with selector, 45 min interval, request time series",
			v1.AggregationRequest{
				Cluster:           "",
				TimeRange:         &lmaapi.TimeRange{From: timeFrom, To: timeTo60Mins},
				Selector:          "dest_namespace = 'abc'",
				IncludeTimeSeries: true,
				Aggregations:      map[string]json.RawMessage{"agg1": json.RawMessage(`{"abc": "def"}`)},
				Timeout:           1000,
			}, &MockBackend{
				AuthorizationReviewResp: []v3.AuthorizedResourceVerbs{{
					APIGroup: "projectcalico.org",
					Resource: "networksets",
					Verbs: []v3.AuthorizedResourceVerb{{
						Verb: "list",
						ResourceGroups: []v3.AuthorizedResourceGroup{{
							Namespace: "ns1",
						}},
					}},
				}},
				AuthorizationReviewRespErr: nil,
				RunQueryResp: map[string]json.RawMessage{
					"tb": json.RawMessage(`{"buckets":[
                        {"key":1622409790,"agg1":{"abc": "def0"}},
                        {"key":1622410690,"agg1":{"abc": "def1"}},
                        {"key":1622411590,"agg1":{"abc": "def2"}},
                        {"key":1622412490,"agg1":{"abc": "def3"}}
                    ]}`),
				},
				RunQueryRespErr: nil,
			},
			http.StatusOK,
			query60Mins,
			`{
          "buckets": [
            {
              "start_time": "1970-01-19T18:40:09Z",
              "aggregations": {
                "agg1": {
                  "abc": "def0"
                }
              }
            },
            {
              "start_time": "1970-01-19T18:40:10Z",
              "aggregations": {
                "agg1": {
                  "abc": "def1"
                }
              }
            },
            {
              "start_time": "1970-01-19T18:40:11Z",
              "aggregations": {
                "agg1": {
                  "abc": "def2"
                }
              }
            },
            {
              "start_time": "1970-01-19T18:40:12Z",
              "aggregations": {
                "agg1": {
                  "abc": "def3"
                }
              }
            }
          ]
        }`,
		),

		Entry("Elastic responds with bad request",
			v1.AggregationRequest{
				Cluster:           "",
				TimeRange:         &lmaapi.TimeRange{From: timeFrom, To: timeTo60Mins},
				Selector:          "dest_namespace = 'abc'",
				IncludeTimeSeries: true,
				Aggregations:      map[string]json.RawMessage{"agg1": json.RawMessage("[]")},
				Timeout:           1000,
			},
			&MockBackend{
				AuthorizationReviewResp: []v3.AuthorizedResourceVerbs{{
					APIGroup: "projectcalico.org",
					Resource: "networksets",
					Verbs: []v3.AuthorizedResourceVerb{{
						Verb: "list",
						ResourceGroups: []v3.AuthorizedResourceGroup{{
							Namespace: "ns1",
						}},
					}},
				}},
				AuthorizationReviewRespErr: nil,
				RunQueryResp:               nil,
				RunQueryRespErr: &elastic.Error{
					Status: http.StatusBadRequest,
				},
			},
			http.StatusBadRequest,
			"",
			"elastic: Error 400 (Bad Request)",
		),

		Entry("Forbidden response from authorization review",
			v1.AggregationRequest{
				Cluster:           "",
				TimeRange:         &lmaapi.TimeRange{From: timeFrom, To: timeTo60Mins},
				Selector:          "dest_namespace = 'abc'",
				IncludeTimeSeries: true,
				Aggregations:      map[string]json.RawMessage{"agg1": json.RawMessage("[]")},
				Timeout:           1000,
			},
			&MockBackend{
				AuthorizationReviewResp: nil,
				AuthorizationReviewRespErr: &httputils.HttpStatusError{
					Status: http.StatusForbidden,
					Msg:    "Forbidden",
				},
				RunQueryResp:    nil,
				RunQueryRespErr: nil,
			},
			http.StatusForbidden,
			"",
			"Forbidden",
		),

		Entry("Empty response from authorization review",
			v1.AggregationRequest{
				Cluster:           "",
				TimeRange:         &lmaapi.TimeRange{From: timeFrom, To: timeTo60Mins},
				Selector:          "dest_namespace = 'abc'",
				IncludeTimeSeries: true,
				Aggregations:      map[string]json.RawMessage{"agg1": json.RawMessage("[]")},
				Timeout:           1000,
			},
			&MockBackend{
				AuthorizationReviewResp:    []v3.AuthorizedResourceVerbs{},
				AuthorizationReviewRespErr: nil,
				RunQueryResp:               nil,
				RunQueryRespErr:            nil,
			},
			http.StatusForbidden,
			"",
			"Forbidden",
		),

		Entry("Invalid field name in selector",
			v1.AggregationRequest{
				Cluster:           "",
				TimeRange:         &lmaapi.TimeRange{From: timeFrom, To: timeTo60Mins},
				Selector:          "dest_namex = 'abc'",
				IncludeTimeSeries: true,
				Aggregations:      map[string]json.RawMessage{"agg1": json.RawMessage("[]")},
				Timeout:           1000,
			},
			&MockBackend{
				AuthorizationReviewResp:    []v3.AuthorizedResourceVerbs{},
				AuthorizationReviewRespErr: nil,
				RunQueryResp:               nil,
				RunQueryRespErr:            nil,
			},
			http.StatusBadRequest,
			"",
			"Invalid selector (dest_namex = 'abc') in request: invalid key: dest_namex",
		),
	)

	DescribeTable("invalid request parameters",
		func(reqest string, code int, resp string) {
			// Create a service graph.
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			sg := NewAggregationHandlerWithBackend(lmaindex.FlowLogs(), &MockBackend{
				AuthorizationReviewRespErr: errors.New("should not hit this"),
				RunQueryRespErr:            errors.New("should not hit this"),
			})

			// Marshal the request and create an HTTP request
			body := ioutil.NopCloser(strings.NewReader(reqest))
			req, err := http.NewRequest("POST", "/aggregation", body)
			Expect(err).NotTo(HaveOccurred())
			req = req.WithContext(ctx)

			// Pass it through the handler
			writer := httptest.NewRecorder()
			sg.ServeHTTP(writer, req)
			Expect(writer.Code).To(Equal(code))
			Expect(strings.TrimSpace(writer.Body.String())).To(Equal(resp), writer.Body.String())
		},

		Entry("Missing time range",
			`{"aggregations": {"test": {}}}`,
			http.StatusBadRequest,
			"Request body contains invalid data: error with field TimeRange = '<nil>' (Reason: failed to validate Field: TimeRange because of Tag: required )",
		),

		Entry("Missing time range fields",
			`{"time_range": {}, "aggregations": {"test": {}}}`,
			http.StatusBadRequest,
			"Request body contains an invalid value for the time range: missing `from` field",
		),
	)
})
