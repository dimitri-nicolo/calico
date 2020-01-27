package middleware

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lmaelastic "github.com/tigera/lma/pkg/elastic"
	"github.com/tigera/lma/pkg/rbac"
)

const (
	defaultResponse = `{
    "took": 65,
    "timed_out": false,
    "_shards": {
        "total": 40,
        "successful": 40,
        "skipped": 0,
        "failed": 0
    },
    "hits": {
        "total": {
            "value": 10000,
            "relation": "gte"
        },
        "max_score": null,
        "hits": []
    },
    "aggregations": {
        "source_dest_name_aggrs": {
            "after_key": {
                "date": 1494201600000,
		"source_name_aggr": "tigera-manager-778447894c-*",
		"dest_name_aggr": "tigera-secure-es-xg2jxdtnqn"
            },
            "buckets": [
                {
                    "key": {
                        "source_name_aggr": "tigera-manager-778447894c-*",
			"source_namespace": "tigera-manager",
                        "dest_name_aggr": "tigera-secure-es-xg2jxdtnqn",
			"dest_namespace": "tigera-elasticsearch"
                    },
                    "doc_count": 4370
                },
                {
                    "key": {
                        "source_name_aggr": "test-app-83958379dc",
			"source_namespace": "default",
                        "dest_name_aggr": "tigera-secure-es-xg2jxdtnqn",
			"dest_namespace": "tigera-elasticsearch"
                    },
                    "doc_count": 3698
                }
            ]
        }
    }
}
`
	emptyNamesResponse = `{
    "took": 1066,
    "timed_out": false,
    "_shards": {
        "total": 40,
        "successful": 40,
        "skipped": 0,
        "failed": 0
    },
    "hits": {
        "total": {
            "value": 10000,
            "relation": "gte"
        },
        "max_score": null,
        "hits": []
    },
    "aggregations": {
        "source_dest_name_aggrs": {
            "after_key": {},
            "buckets": []
        }
    }
}
`
	filterPrefixResponse = `{
    "took": 65,
    "timed_out": false,
    "_shards": {
        "total": 40,
        "successful": 40,
        "skipped": 0,
        "failed": 0
    },
    "hits": {
        "total": {
            "value": 10000,
            "relation": "gte"
        },
        "max_score": null,
        "hits": []
    },
    "aggregations": {
        "source_dest_name_aggrs": {
            "after_key": {
                "date": 1494201600000,
		"source_name_aggr": "tigera-manager-778447894c-*",
		"dest_name_aggr": "tigera-secure-es-xg2jxdtnqn"
            },
            "buckets": [
                {
                    "key": {
                        "source_name_aggr": "tigera-manager-778447894c-*",
			"source_namespace": "tigera-manager",
                        "dest_name_aggr": "tigera-secure-es-xg2jxdtnqn",
			"dest_namespace": "tigera-elasticsearch"
                    },
                    "doc_count": 4370
                },
                {
                    "key": {
                        "source_name_aggr": "tigera-app-83958379dc",
			"source_namespace": "default",
                        "dest_name_aggr": "tigera-secure-es-xg2jxdtnqn",
			"dest_namespace": "tigera-elasticsearch"
                    },
                    "doc_count": 3698
                }
            ]
        }
    }
}
`
	duplicateNamesResponse = `{
    "took": 13,
    "timed_out": false,
    "_shards": {
        "total": 40,
        "successful": 40,
        "skipped": 0,
        "failed": 0
    },
    "hits": {
        "total": {
            "value": 10000,
            "relation": "gte"
        },
        "max_score": null,
        "hits": []
    },
    "aggregations": {
        "source_dest_name_aggrs": {
            "after_key": {
                "date": 1494201600000,
		"source_name_aggr": "compliance-benchmarker-*",
		"dest_name_aggr": "default/kse.kubernetes"
            },
            "buckets": [
                {
                    "key": {
                        "source_name_aggr": "compliance-benchmarker-*",
			"source_namespace": "tigera-compliance",
                        "dest_name_aggr": "compliance-controller-c7f4b94dd-*",
			"dest_namespace": "tigera-compliance"
                    },
                    "doc_count": 10930
                },
                {
                    "key": {
                        "source_name_aggr": "compliance-benchmarker-*",
			"source_namespace": "tigera-compliance",
                        "dest_name_aggr": "compliance-server-69c97dffcf-*",
			"dest_namespace": "tigera-compliance"
                    },
                    "doc_count": 4393
                },
                {
                    "key": {
                        "source_name_aggr": "compliance-controller-c7f4b94dd-*",
			"source_namespace": "tigera-compliance",
                        "dest_name_aggr": "compliance-server-69c97dffcf-*",
			"dest_namespace": "tigera-compliance"
                    },
                    "doc_count": 4374
                },
                {
                    "key": {
                        "source_name_aggr": "compliance-server-69c97dffcf-*",
			"source_namespace": "tigera-compliance",
                        "dest_name_aggr": "tigera-manager-778447894c-*",
			"dest_namespace": "tigera-manager"
                    },
                    "doc_count": 4372
                },
                {
                    "key": {
                        "source_name_aggr": "compliance-server-69c97dffcf-*",
			"source_namespace": "tigera-compliance",
                        "dest_name_aggr": "compliance-controller-c7f4b94dd-*",
			"dest_namespace": "tigera-compliance"
                    },
                    "doc_count": 4374
                },
                {
                    "key": {
                        "source_name_aggr": "tigera-secure-es-xg2jxdtnqn",
			"source_namespace": "tigera-elasticsearch",
                        "dest_name_aggr": "tigera-manager-778447894c-*",
			"dest_namespace": "tigera-manager"
                    },
                    "doc_count": 21865
                },
                {
                    "key": {
                        "source_name_aggr": "compliance-benchmarker-*",
			"source_namespace": "tigera-compliance",
                        "dest_name_aggr": "default/kse.kubernetes",
			"dest_namespace": "default"
                    },
                    "doc_count": 2185
                }
            ]
        }
    }
}`
	missingNamesAggregations = `{
    "took": 155,
    "timed_out": false,
    "_shards": {
        "total": 40,
        "successful": 40,
        "skipped": 0,
        "failed": 0
    },
    "hits": {
        "total": {
            "value": 10000,
            "relation": "gte"
        },
        "max_score": null,
        "hits": []
    }
}`
	malformedNamesResponse = `{
    badlyFormedNamesJson
}`
)

var _ = Describe("Test /flowLogNames endpoint functions", func() {
	var esClient lmaelastic.Client

	Context("Test that the validateFlowLogNamesRequest function behaves as expected", func() {
		It("should return an errInvalidMethod when passed a request with an http method other than GET", func() {
			By("Creating a request with a POST method")
			req, err := newTestRequest(http.MethodPost)
			Expect(err).NotTo(HaveOccurred())

			params, err := validateFlowLogNamesRequest(req)
			Expect(err).To(BeEquivalentTo(errInvalidMethod))
			Expect(params).To(BeNil())

			By("Creating a request with a DELETE method")
			req, err = newTestRequest(http.MethodDelete)
			Expect(err).NotTo(HaveOccurred())

			params, err = validateFlowLogNamesRequest(req)
			Expect(err).To(BeEquivalentTo(errInvalidMethod))
			Expect(params).To(BeNil())
		})

		It("should return a valid params object with the limit set to 1000 when passed an empty limit", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("limit", "")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamesRequest(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(params.Limit).To(BeNumerically("==", 1000))
		})

		It("should return a valid params object with the limit set to 1000 when passed a 0 limit", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("limit", "0")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamesRequest(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(params.Limit).To(BeNumerically("==", 1000))
		})

		It("should return an errParseRequest when passed a request with a negative limit parameter", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("limit", "-100")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamesRequest(req)
			Expect(err).To(BeEquivalentTo(errParseRequest))
			Expect(params).To(BeNil())
		})

		It("should return an errParseRequest when passed a request with word as the limit parameter", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("limit", "ten")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamesRequest(req)
			Expect(err).To(BeEquivalentTo(errParseRequest))
			Expect(params).To(BeNil())
		})

		It("should return an errParseRequest when passed a request with a floating number as the limit parameter", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("limit", "3.14")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamesRequest(req)
			Expect(err).To(BeEquivalentTo(errParseRequest))
			Expect(params).To(BeNil())
		})

		It("should return an errParseRequest when passed a request with a max int32 + 1 number as the limit parameter", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("limit", "2147483648")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamesRequest(req)
			Expect(err).To(BeEquivalentTo(errParseRequest))
			Expect(params).To(BeNil())
		})

		It("should return an errParseRequest when passed a request with a min int32 - 1 number as the limit parameter", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("limit", "-2147483648")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamesRequest(req)
			Expect(err).To(BeEquivalentTo(errParseRequest))
			Expect(params).To(BeNil())
		})

		It("should return an errParseRequest when passed a request with an invalid unprotected param", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("unprotected", "xvz")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamesRequest(req)
			Expect(err).To(BeEquivalentTo(errParseRequest))
			Expect(params).To(BeNil())
		})

		It("should return an errParseRequest when passed a request with an invalid combination of actions and unprotected param", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("actions", "allow")
			q.Add("actions", "deny")
			q.Add("unprotected", "true")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamesRequest(req)
			Expect(err).To(BeEquivalentTo(errInvalidActionUnprotected))
			Expect(params).To(BeNil())
		})

		It("should return an errInvalidAction when passed a request with an unacceptable actions parameter", func() {
			By("Forming a request with an invalid actions value")
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("actions", "alloow")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamesRequest(req)
			Expect(err).To(BeEquivalentTo(errInvalidAction))
			Expect(params).To(BeNil())

			By("Forming a request with a few valid actions and one invalid")
			req, err = newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q = req.URL.Query()
			q.Add("actions", "allow")
			q.Add("actions", "deny")
			q.Add("actions", "invalid")
			req.URL.RawQuery = q.Encode()

			params, err = validateFlowLogNamesRequest(req)
			Expect(err).To(BeEquivalentTo(errInvalidAction))
			Expect(params).To(BeNil())
		})

		It("should return a valid FlowLogNamesParams object with the Actions and Namespace from the request", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("actions", "allow")
			q.Add("actions", "deny")
			q.Add("actions", "unknown")
			q.Add("namespace", "tigera-elasticsearch")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamesRequest(req)
			Expect(err).To(Not(HaveOccurred()))
			Expect(params.Actions[0]).To(BeEquivalentTo("allow"))
			Expect(params.Actions[1]).To(BeEquivalentTo("deny"))
			Expect(params.Actions[2]).To(BeEquivalentTo("unknown"))
			Expect(params.Namespace).To(BeEquivalentTo("tigera-elasticsearch"))
		})

		It("should return a valid FlowLogNamesParams object when passed upper case parameters", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("actions", "ALLOW")
			q.Add("cluster", "CLUSTER")
			q.Add("prefix", "TIGERA-")
			q.Add("namespace", "TIGERA-ELASTICSEARCH")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamesRequest(req)
			Expect(err).To(Not(HaveOccurred()))
			Expect(params.Actions[0]).To(BeEquivalentTo("allow"))
			Expect(params.ClusterName).To(BeEquivalentTo("cluster"))
			Expect(params.Prefix).To(BeEquivalentTo("tigera-.*"))
			Expect(params.Namespace).To(BeEquivalentTo("tigera-elasticsearch"))
		})

		It("should return a valid FlowLogNamespaceParams when passed a request with valid start/end time", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("actions", "ALLOW")
			q.Add("cluster", "CLUSTER")
			q.Add("prefix", "TIGERA-")
			q.Add("namespace", "TIGERA-ELASTICSEARCH")
			startTimeObject, endTimeObject := getTestStartAndEndTime()
			Expect(err).To(Not(HaveOccurred()))
			q.Add("startDateTime", startTimeTest)
			q.Add("endDateTime", endTimeTest)

			Expect(err).To(Not(HaveOccurred()))
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamesRequest(req)
			Expect(err).To(Not(HaveOccurred()))
			Expect(params.Actions[0]).To(BeEquivalentTo("allow"))
			Expect(params.ClusterName).To(BeEquivalentTo("cluster"))
			Expect(params.Prefix).To(BeEquivalentTo("tigera-.*"))
			Expect(params.Namespace).To(BeEquivalentTo("tigera-elasticsearch"))
			Expect(params.StartDateTime).To(BeEquivalentTo(startTimeObject))
			Expect(params.EndDateTime).To(BeEquivalentTo(endTimeObject))
		})
	})

	Context("Test that the getNamesFromElastic function behaves as expected", func() {
		It("should retrieve all names with prefix tigera", func() {
			By("Creating a mock ES client with a mocked out search results")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{defaultResponse})

			By("Creating params with the prefix tigera")
			params := &FlowLogNamesParams{
				Limit:       1000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "tigera",
				ClusterName: "cluster",
				Namespace:   "tigera-compliance",
			}

			names, err := getNamesFromElastic(params, esClient)
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(names)).To(BeNumerically("==", 2))
			Expect(names[0]).To(BeEquivalentTo("tigera-manager-778447894c-*"))
			Expect(names[1]).To(BeEquivalentTo("tigera-secure-es-xg2jxdtnqn"))
		})

		It("should handle an empty array of names returned from elasticsearch", func() {
			By("Creating a mock ES client with a mocked out search results")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{emptyNamesResponse})

			By("Creating params with the prefix tigera")
			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "tigera",
				ClusterName: "cluster",
				Namespace:   "tigera-compliance",
			}

			names, err := getNamesFromElastic(params, esClient)
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(names)).To(BeNumerically("==", 0))
		})

		It("should retrieve an empty array of names when the prefix filters them out", func() {
			By("Creating a mock ES client with a mocked out search results")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{filterPrefixResponse})

			By("Creating params with the prefix tigera-elasticccccccc")
			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "tigera-elasticccccccc.*",
				ClusterName: "cluster",
				Namespace:   "tigera-compliance",
			}

			names, err := getNamesFromElastic(params, esClient)
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(names)).To(BeNumerically("==", 0))
		})

		It("should retrieve an array of names with no duplicates", func() {
			By("Creating a mock ES client with a mocked out search results containing duplicates")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{duplicateNamesResponse})

			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "cluster",
				Namespace:   "",
			}

			names, err := getNamesFromElastic(params, esClient)
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(names)).To(BeNumerically("==", 6))
			Expect(names[0]).To(BeEquivalentTo("compliance-benchmarker-*"))
			Expect(names[1]).To(BeEquivalentTo("compliance-controller-c7f4b94dd-*"))
			Expect(names[2]).To(BeEquivalentTo("compliance-server-69c97dffcf-*"))
			Expect(names[3]).To(BeEquivalentTo("tigera-manager-778447894c-*"))
			Expect(names[4]).To(BeEquivalentTo("tigera-secure-es-xg2jxdtnqn"))
			Expect(names[5]).To(BeEquivalentTo("default/kse.kubernetes"))
		})

		It("should retrieve an array of names with no duplicates and only up to the limit", func() {
			By("Creating a mock ES client with a mocked out search results containing duplicates")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{duplicateNamesResponse})

			params := &FlowLogNamesParams{
				Limit:       3,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "cluster",
				Namespace:   "tigera-compliance",
			}

			names, err := getNamesFromElastic(params, esClient)
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(names)).To(BeNumerically("==", 3))
			Expect(names[0]).To(BeEquivalentTo("compliance-benchmarker-*"))
			Expect(names[1]).To(BeEquivalentTo("compliance-controller-c7f4b94dd-*"))
			Expect(names[2]).To(BeEquivalentTo("compliance-server-69c97dffcf-*"))
		})

		It("should return an empty response when the search result contains no aggregations", func() {
			By("Creating a mock ES client with a mocked out search results")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{missingNamesAggregations})

			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "cluster",
				Namespace:   "tigera-compliance",
			}

			names, err := getNamesFromElastic(params, esClient)
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(names)).To(BeNumerically("==", 0))
		})

		It("should return an empty response when the EndDateTime is in the past", func() {
			By("Creating a mock ES client with a mocked out search results")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{missingNamesAggregations})

			_, endTimeObject := getTestStartAndEndTime()

			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "cluster",
				Namespace:   "tigera-compliance",
				EndDateTime: endTimeObject,
			}

			names, err := getNamesFromElastic(params, esClient)
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(names)).To(BeNumerically("==", 0))
		})

		It("should return an empty response when the StartDateTime is in the future", func() {
			By("Creating a mock ES client with a mocked out search results")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{missingNamesAggregations})

			startTimeObject := "now + 20d"

			params := &FlowLogNamesParams{
				Limit:         2000,
				Actions:       []string{"allow", "deny", "unknown"},
				Prefix:        "",
				ClusterName:   "cluster",
				Namespace:     "tigera-compliance",
				StartDateTime: startTimeObject,
			}

			names, err := getNamesFromElastic(params, esClient)
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(names)).To(BeNumerically("==", 0))
		})

		It("should return an error when the query fails", func() {
			By("Creating a mock ES client with badly formed search results")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{malformedNamesResponse})

			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "",
				Namespace:   "tigera-compliance",
			}

			names, err := getNamesFromElastic(params, esClient)
			Expect(err).To(HaveOccurred())
			Expect(names).To(BeNil())
		})
	})

	Context("Test that the buildNamesQuery function applies filters only when necessary", func() {
		It("should return a query without filters", func() {
			By("Creating params with no actions")
			params := &FlowLogNamesParams{
				Limit:       2000,
				Prefix:      "",
				ClusterName: "",
				Namespace:   "tigera-compliance",
				filterNamespaces: []NamespacePermissions{
					{
						Namespace:     "tigera-compliance",
						EndpointTypes: map[string]struct{}{"wep": struct{}{}},
					},
				},
			}

			query := buildNamesQuery(params)
			queryInf, err := query.Source()
			Expect(err).To(Not(HaveOccurred()))
			queryMap := queryInf.(map[string]interface{})
			boolQueryMap := queryMap["bool"].(map[string]interface{})
			boolQueryFilterMap := boolQueryMap["filter"].(map[string]interface{})
			nestedBoolQueryMap := boolQueryFilterMap["bool"].(map[string]interface{})
			nestedBoolQueryShouldSlice := nestedBoolQueryMap["should"].([]interface{})
			Expect(len(nestedBoolQueryMap)).To(BeNumerically("==", 1))
			Expect(nestedBoolQueryShouldSlice).To(HaveLen(2))
		})

		It("should return a query with filters", func() {
			By("Creating params with actions")
			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "",
				Namespace:   "tigera-compliance",
				filterNamespaces: []NamespacePermissions{
					{
						Namespace:     "tigera-compliance",
						EndpointTypes: map[string]struct{}{"wep": struct{}{}},
					},
				},
			}

			query := buildNamesQuery(params)
			queryInf, err := query.Source()
			Expect(err).To(Not(HaveOccurred()))
			queryMap := queryInf.(map[string]interface{})
			boolQueryMap := queryMap["bool"].(map[string]interface{})
			boolQueryFilterMap := boolQueryMap["filter"].(map[string]interface{})
			nestedBoolQueryMap := boolQueryFilterMap["bool"].(map[string]interface{})
			nestedBoolQueryShouldSlice := nestedBoolQueryMap["should"].([]interface{})
			Expect(len(nestedBoolQueryMap)).To(BeNumerically("==", 2))
			Expect(nestedBoolQueryShouldSlice).To(HaveLen(2))
		})
	})

	Context("Test that the buildNameAggregation function creates the appropriate composite aggregation", func() {
		It("should return aggregations without Include filters", func() {
			By("Creating params without a prefix")
			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "",
				Namespace:   "tigera-compliance",
			}

			sourceAggTermsMap, destAggTermsMap, err := getNamesAggregationTermsMaps(params)
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(sourceAggTermsMap)).To(BeNumerically("==", 1))
			Expect(len(destAggTermsMap)).To(BeNumerically("==", 1))
		})
	})

	Context("Test that the buildAllowedEndpointsByNamespace function properly transforms the params", func() {
		It("should add all returned namespaces and the global denomination with permissive RBAC", func() {
			By("Creating params without a namespace")
			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "",
			}

			By("Creating a mock ES Client that will return two namespaces")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{prefixResponse})
			newParams, err := buildAllowedEndpointsByNamespace(params, esClient, rbac.NewAlwaysAllowFlowHelper())
			Expect(err).To(Not(HaveOccurred()))
			Expect(newParams.filterNamespaces).To(HaveLen(1))
			Expect(newParams.filterNamespaces[0].Namespace).To(Equal(""))
			Expect(newParams.filterNamespaces[0].EndpointTypes).To(BeEmpty())
		})

		It("should add all returned namespaces and endpoints for each namespace when allowed and without permissions to view global endpoints", func() {
			By("Creating params without a namespace")
			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "",
			}
			By("Creating a mock ES Client that will return two namespaces")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{prefixResponse})
			mockFlowHelper := rbac.NewMockFlowHelper(map[string][]string{
				"pods":        []string{""},
				"networksets": []string{""},
			})
			newParams, err := buildAllowedEndpointsByNamespace(params, esClient, mockFlowHelper)
			Expect(err).To(Not(HaveOccurred()))
			Expect(newParams.filterNamespaces).To(HaveLen(2))
			Expect(newParams.filterNamespaces[0].Namespace).To(Equal("tigera-eck-operator"))
			Expect(newParams.filterNamespaces[0].EndpointTypes).To(HaveKey("wep"))
			Expect(newParams.filterNamespaces[0].EndpointTypes).To(HaveKey("ns"))
			Expect(newParams.filterNamespaces[1].Namespace).To(Equal("tigera-elasticsearch"))
			Expect(newParams.filterNamespaces[1].EndpointTypes).To(HaveKey("wep"))
			Expect(newParams.filterNamespaces[1].EndpointTypes).To(HaveKey("ns"))
		})

		It("should return permissions to view global endpoints as well as namespaced ones when specific namespaces all allowed", func() {
			By("Creating params without a namespace")
			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "",
			}
			By("Creating a mock ES Client that will return two namespaces")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{prefixResponse})
			mockFlowHelper := rbac.NewMockFlowHelper(map[string][]string{
				"pods":              []string{"tigera-eck-operator", "tigera-elasticsearch"},
				"networksets":       []string{"tigera-eck-operator", "tigera-elasticsearch"},
				"hostendpoints":     []string{""},
				"globalnetworksets": []string{""},
			})
			newParams, err := buildAllowedEndpointsByNamespace(params, esClient, mockFlowHelper)
			Expect(err).To(Not(HaveOccurred()))
			Expect(newParams.filterNamespaces).To(HaveLen(3))
			Expect(newParams.filterNamespaces[0].Namespace).To(Equal("tigera-eck-operator"))
			Expect(newParams.filterNamespaces[0].EndpointTypes).To(HaveKey("wep"))
			Expect(newParams.filterNamespaces[0].EndpointTypes).To(HaveKey("ns"))
			Expect(newParams.filterNamespaces[1].Namespace).To(Equal("tigera-elasticsearch"))
			Expect(newParams.filterNamespaces[1].EndpointTypes).To(HaveKey("wep"))
			Expect(newParams.filterNamespaces[1].EndpointTypes).To(HaveKey("ns"))
			Expect(newParams.filterNamespaces[2].Namespace).To(Equal("-"))
			Expect(newParams.filterNamespaces[2].EndpointTypes).To(HaveKey("hep"))
			Expect(newParams.filterNamespaces[2].EndpointTypes).To(HaveKey("ns"))
		})

		It("should return permissions to view global endpoints when that is all that is allowed", func() {
			By("Creating params without a namespace")
			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "",
			}
			By("Creating a mock ES Client that will return two namespaces")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{prefixResponse})
			mockFlowHelper := rbac.NewMockFlowHelper(map[string][]string{
				"hostendpoints":     []string{""},
				"globalnetworksets": []string{""},
			})
			newParams, err := buildAllowedEndpointsByNamespace(params, esClient, mockFlowHelper)
			Expect(err).To(Not(HaveOccurred()))
			Expect(newParams.filterNamespaces).To(HaveLen(1))
			Expect(newParams.filterNamespaces[0].Namespace).To(Equal("-"))
			Expect(newParams.filterNamespaces[0].EndpointTypes).To(HaveKey("hep"))
			Expect(newParams.filterNamespaces[0].EndpointTypes).To(HaveKey("ns"))
		})

		It("should add all returned namespaces with the proper endpoint type permissions for each depending on RBAC", func() {
			By("Creating params without a namespace")
			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "",
			}
			By("Creating a mock ES Client that will return two namespaces")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{prefixResponse})
			mockFlowHelper := rbac.NewMockFlowHelper(map[string][]string{
				"pods":          []string{"tigera-elasticsearch"},
				"networksets":   []string{"tigera-eck-operator"},
				"hostendpoints": []string{""},
			})
			newParams, err := buildAllowedEndpointsByNamespace(params, esClient, mockFlowHelper)
			Expect(err).To(Not(HaveOccurred()))
			Expect(newParams.filterNamespaces).To(HaveLen(3))
			Expect(newParams.filterNamespaces[0].Namespace).To(Equal("tigera-eck-operator"))
			Expect(newParams.filterNamespaces[0].EndpointTypes).NotTo(HaveKey("wep"))
			Expect(newParams.filterNamespaces[0].EndpointTypes).To(HaveKey("ns"))
			Expect(newParams.filterNamespaces[1].Namespace).To(Equal("tigera-elasticsearch"))
			Expect(newParams.filterNamespaces[1].EndpointTypes).To(HaveKey("wep"))
			Expect(newParams.filterNamespaces[1].EndpointTypes).NotTo(HaveKey("ns"))
			Expect(newParams.filterNamespaces[2].Namespace).To(Equal("-"))
			Expect(newParams.filterNamespaces[2].EndpointTypes).To(HaveKey("hep"))
			Expect(newParams.filterNamespaces[2].EndpointTypes).NotTo(HaveKey("ns"))
		})

		It("should properly add permissions on some endpoint types across all namespaces to all the namespaces that need it", func() {
			By("Creating params without a namespace")
			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "",
			}
			By("Creating a mock ES Client that will return two namespaces")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{prefixResponse})
			mockFlowHelper := rbac.NewMockFlowHelper(map[string][]string{
				"pods":          []string{""},
				"networksets":   []string{"tigera-eck-operator"},
				"hostendpoints": []string{""},
			})
			newParams, err := buildAllowedEndpointsByNamespace(params, esClient, mockFlowHelper)
			Expect(err).To(Not(HaveOccurred()))
			Expect(newParams.filterNamespaces).To(HaveLen(3))
			Expect(newParams.filterNamespaces[0].Namespace).To(Equal("tigera-eck-operator"))
			Expect(newParams.filterNamespaces[0].EndpointTypes).To(HaveKey("wep"))
			Expect(newParams.filterNamespaces[0].EndpointTypes).To(HaveKey("ns"))
			Expect(newParams.filterNamespaces[1].Namespace).To(Equal("tigera-elasticsearch"))
			Expect(newParams.filterNamespaces[1].EndpointTypes).To(HaveKey("wep"))
			Expect(newParams.filterNamespaces[1].EndpointTypes).NotTo(HaveKey("ns"))
			Expect(newParams.filterNamespaces[2].Namespace).To(Equal("-"))
			Expect(newParams.filterNamespaces[2].EndpointTypes).To(HaveKey("hep"))
			Expect(newParams.filterNamespaces[2].EndpointTypes).NotTo(HaveKey("ns"))
		})

		It("should only return the specified namespace if it is permitted", func() {
			By("Creating params with a namespace")
			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "",
				Namespace:   "tigera-compliance",
			}

			By("Creating a mock ES Client that should not get called")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{malformedResponse})
			permissions := map[string][]string{
				"pods": []string{"fake-namespace"},
			}
			newParams, err := buildAllowedEndpointsByNamespace(params, esClient, rbac.NewMockFlowHelper(permissions))
			Expect(err).To(HaveOccurred())
			Expect(newParams.filterNamespaces).To(HaveLen(0))
		})

		It("should only return the specified namespace if it is permitted with the correct endpoint types for the global case", func() {
			By("Creating params with a namespace")
			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "",
				Namespace:   "tigera-eck-operator",
			}
			By("Creating a mock ES Client that will return two namespaces")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{prefixResponse})
			mockFlowHelper := rbac.NewMockFlowHelper(map[string][]string{
				"pods":          []string{""},
				"networksets":   []string{"tigera-eck-operator"},
				"hostendpoints": []string{""},
			})
			newParams, err := buildAllowedEndpointsByNamespace(params, esClient, mockFlowHelper)
			Expect(err).To(Not(HaveOccurred()))
			Expect(newParams.filterNamespaces).To(HaveLen(1))
			Expect(newParams.filterNamespaces[0].Namespace).To(Equal("tigera-eck-operator"))
			Expect(newParams.filterNamespaces[0].EndpointTypes).To(HaveKey("wep"))
			Expect(newParams.filterNamespaces[0].EndpointTypes).To(HaveKey("ns"))
		})
	})
})

func getNamesAggregationTermsMaps(params *FlowLogNamesParams) (map[string]interface{}, map[string]interface{},
	error) {
	var empty map[string]interface{}
	nameAgg := buildNameAggregation(params, empty)
	nameAggInf, err := nameAgg.Source()
	if err != nil {
		return nil, nil, err
	}
	// Pull out the terms for the composite aggregation. It should be of the form:
	//    "composite": {
	//        "sources": [
	//            {
	//                "source_name_aggr": {
	//                    "terms": {
	//                        "field": "source_name_aggr",
	//                    },
	//                },
	//            },
	//            {
	//                "dest_name_aggr": {
	//                    "terms": {
	//                        "field": "dest_name_aggr",
	//                    },
	//                },
	//            },
	//        ],
	//        "size": 4000,
	//    }
	nameAggMap := nameAggInf.(map[string]interface{})
	compAggMap := nameAggMap["composite"].(map[string]interface{})
	srcSliceInf := compAggMap["sources"].([]interface{})
	var sourceAggTermsMap map[string]interface{}
	var destAggTermsMap map[string]interface{}
	for _, mappingInf := range srcSliceInf {
		mapping := mappingInf.(map[string]interface{})
		if srcMapInf, exist := mapping[sourceAggName]; exist {
			srcMap := srcMapInf.(map[string]interface{})
			sourceAggTermsMap = srcMap["terms"].(map[string]interface{})
		}
		if destMapInf, exist := mapping[destAggName]; exist {
			destMap := destMapInf.(map[string]interface{})
			destAggTermsMap = destMap["terms"].(map[string]interface{})
		}
	}
	return sourceAggTermsMap, destAggTermsMap, nil
}
