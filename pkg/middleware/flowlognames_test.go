package middleware

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lmaelastic "github.com/tigera/lma/pkg/elastic"
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
        "source_name_aggrs": {
            "doc_count_error_upper_bound": 0,
            "sum_other_doc_count": 0,
            "buckets": [
                {
                    "key": "tigera-manager-778447894c-*",
                    "doc_count": 4370
                }
            ]
        },
        "dest_name_aggrs": {
            "doc_count_error_upper_bound": 0,
            "sum_other_doc_count": 0,
            "buckets": [
                {
                    "key": "tigera-secure-es-xg2jxdtnqn",
                    "doc_count": 21855
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
        "source_name_aggrs": {
            "doc_count_error_upper_bound": 0,
            "sum_other_doc_count": 0,
            "buckets": []
        },
        "dest_name_aggrs": {
            "doc_count_error_upper_bound": 0,
            "sum_other_doc_count": 0,
            "buckets": []
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
        "source_name_aggrs": {
            "doc_count_error_upper_bound": 0,
            "sum_other_doc_count": 0,
            "buckets": [
                {
                    "key": "compliance-benchmarker-*",
                    "doc_count": 10930
                },
                {
                    "key": "compliance-benchmarker-*",
                    "doc_count": 4393
                },
                {
                    "key": "compliance-controller-c7f4b94dd-*",
                    "doc_count": 4374
                },
                {
                    "key": "compliance-server-69c97dffcf-*",
                    "doc_count": 4372
                },
                {
                    "key": "tigera-manager-778447894c-*",
                    "doc_count": 4372
                }
            ]
        },
        "dest_name_aggrs": {
            "doc_count_error_upper_bound": 0,
            "sum_other_doc_count": 0,
            "buckets": [
                {
                    "key": "tigera-secure-es-xg2jxdtnqn",
                    "doc_count": 21865
                },
                {
                    "key": "compliance-benchmarker-*",
                    "doc_count": 4372
                },
                {
                    "key": "default/kse.kubernetes",
                    "doc_count": 2185
                },
                {
                    "key": "compliance-benchmarker-*",
                    "doc_count": 19
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

	missingDestNamesAggregations = `{
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
        "source_name_aggrs": {
            "doc_count_error_upper_bound": 0,
            "sum_other_doc_count": 0,
            "buckets": [
                {
                    "key": "compliance-benchmarker-*",
                    "doc_count": 10930
                },
                {
                    "key": "compliance-benchmarker-*",
                    "doc_count": 4393
                },
                {
                    "key": "compliance-controller-c7f4b94dd-*",
                    "doc_count": 4374
                }
            ]
        }
    }
}`

	missingSourceNamesAggregations = `{
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
        "dest_name_aggrs": {
            "doc_count_error_upper_bound": 0,
            "sum_other_doc_count": 0,
            "buckets": [
                {
                    "key": "tigera-secure-es-xg2jxdtnqn",
                    "doc_count": 21865
                },
                {
                    "key": "compliance-benchmarker-*",
                    "doc_count": 4372
                }
            ]
        }
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

		It("should retrieve an empty array of names", func() {
			By("Creating a mock ES client with a mocked out search results")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{emptyNamesResponse})

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
				Namespace:   "tigera-compliance",
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

		It("should return an array of names when the search result contains no source aggregations", func() {
			By("Creating a mock ES client with a mocked out search results")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{missingSourceNamesAggregations})

			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "cluster",
				Namespace:   "",
			}

			names, err := getNamesFromElastic(params, esClient)
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(names)).To(BeNumerically("==", 2))
			Expect(names[0]).To(BeEquivalentTo("tigera-secure-es-xg2jxdtnqn"))
			Expect(names[1]).To(BeEquivalentTo("compliance-benchmarker-*"))
		})

		It("should return an array of names when the search result contains no dest aggregations", func() {
			By("Creating a mock ES client with a mocked out search results")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{missingDestNamesAggregations})

			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "cluster",
				Namespace:   "",
			}

			names, err := getNamesFromElastic(params, esClient)
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(names)).To(BeNumerically("==", 2))
			Expect(names[0]).To(BeEquivalentTo("compliance-benchmarker-*"))
			Expect(names[1]).To(BeEquivalentTo("compliance-controller-c7f4b94dd-*"))
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
			}

			query := buildNamesQuery(params)
			queryInf, err := query.Source()
			Expect(err).To(Not(HaveOccurred()))
			queryMap := queryInf.(map[string]interface{})
			boolQueryMap := queryMap["bool"].(map[string]interface{})
			boolQueryFilterMap := boolQueryMap["filter"].(map[string]interface{})
			nestedBoolQueryMap := boolQueryFilterMap["bool"].(map[string]interface{})
			Expect(len(nestedBoolQueryMap)).To(BeNumerically("==", 2))
		})

		It("should return a query with filters", func() {
			By("Creating params with actions")
			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "",
				Namespace:   "tigera-compliance",
			}

			query := buildNamesQuery(params)
			queryInf, err := query.Source()
			Expect(err).To(Not(HaveOccurred()))
			queryMap := queryInf.(map[string]interface{})
			boolQueryMap := queryMap["bool"].(map[string]interface{})
			boolQueryFilterMap := boolQueryMap["filter"].(map[string]interface{})
			nestedBoolQueryMap := boolQueryFilterMap["bool"].(map[string]interface{})
			Expect(len(nestedBoolQueryMap)).To(BeNumerically("==", 3))
		})
	})

	Context("Test that the buildNameAggregations function applies Include filters only when there is a prefix", func() {
		It("should return aggregations with Include filters", func() {
			By("Creating params with a prefix")
			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "tigera*",
				ClusterName: "",
				Namespace:   "tigera-compliance",
			}

			sourceAggTermsMap, destAggTermsMap, err := getNamesAggregationTermsMaps(params)
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(sourceAggTermsMap)).To(BeNumerically("==", 3))
			Expect(len(destAggTermsMap)).To(BeNumerically("==", 3))
		})

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
			Expect(len(sourceAggTermsMap)).To(BeNumerically("==", 2))
			Expect(len(destAggTermsMap)).To(BeNumerically("==", 2))
		})
	})
})

func getNamesAggregationTermsMaps(params *FlowLogNamesParams) (map[string]interface{}, map[string]interface{},
	error) {
	sourceAgg, destAgg := buildNameAggregations(params)
	sourceAggInf, err := sourceAgg.Source()
	if err != nil {
		return nil, nil, err
	}
	destAggInf, err := destAgg.Source()
	if err != nil {
		return nil, nil, err
	}
	sourceAggMap := sourceAggInf.(map[string]interface{})
	destAggMap := destAggInf.(map[string]interface{})
	sourceAggTermsMap := sourceAggMap["terms"].(map[string]interface{})
	destAggTermsMap := destAggMap["terms"].(map[string]interface{})
	return sourceAggTermsMap, destAggTermsMap, nil
}
