package middleware

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lmaelastic "github.com/tigera/lma/pkg/elastic"
	"github.com/tigera/lma/pkg/rbac"
)

const (
	prefixResponse = `{
    "took": 98,
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
        "source_dest_namespaces": {
            "after_key": {
                "date": 1494201600000,
		"source_name_aggr": "tigera-eck-operator",
		"dest_name_aggr": "tigera-elasticsearch"
            },
            "buckets": [
                {
                    "key": {
                        "source_namespace": "tigera-eck-operator",
                        "dest_namespace": "tigera-elasticsearch"
                    },
                    "doc_count": 50753
                }
            ]
        }
    }
}
`
	emptyResponse = `{
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
    },
    "aggregations": {
        "source_dest_namespaces": {
            "after_key": {},
            "buckets": []
        }
    }
}`
	duplicatesResponse = `{
    "took": 487,
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
        "source_dest_namespaces": {
            "after_key": {
                "date": 1494201600000,
		"source_name_aggr": "tigera-eck-operator",
		"dest_name_aggr": "tigera-elasticsearch"
            },
            "buckets": [
                {
                    "key": {
                        "source_namespace": "tigera-prometheus",
                        "dest_namespace": "tigera-elasticsearch"
                    },
                    "doc_count": 49209
                },
		{
                    "key": {
                        "source_namespace": "tigera-compliance",
			"dest_namespace": "kube-system"
                    },
                    "doc_count": 26702
                },
                {
                    "key": {
                        "source_namespace": "tigera-fluentd",
                        "dest_namespace": "tigera-prometheus"
                    },
                    "doc_count": 13565
                },
                {
                    "key": {
                        "source_namespace":  "tigera-fluentd",
			"dest_namespace": "tigera-compliance"
                    },
                    "doc_count": 8639
                },
                {
                    "key": {
                        "source_namespace":  "tigera-manager",
			"dest_namespace": "tigera-system"
                    },
                    "doc_count": 4246
                },
                {
                    "key": {
                        "source_namespace":  "tigera-eck-operator",
			"dest_namespace": "tigera-kibana"
                    },
                    "doc_count": 2123
                },
                {
                    "key": {
                        "source_namespace":  "tigera-manager",
			"dest_namespace": "tigera-intrusion-detection"
                    },
                    "doc_count": 1811
                }
            ]
        }
    }
}`
	missingAggregations = `{
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
	malformedResponse = `{
    badlyFormedJson
}`
)

var _ = Describe("Test /flowLogNamespaces endpoint functions", func() {
	var esClient lmaelastic.Client

	Context("Test that the validateFlowLogNamespacesRequest function behaves as expected", func() {
		It("should return an errInvalidMethod when passed a request with an http method other than GET", func() {
			By("Creating a request with a POST method")
			req, err := newTestRequest(http.MethodPost)
			Expect(err).NotTo(HaveOccurred())

			params, err := validateFlowLogNamespacesRequest(req)
			Expect(err).To(BeEquivalentTo(errInvalidMethod))
			Expect(params).To(BeNil())

			By("Creating a request with a DELETE method")
			req, err = newTestRequest(http.MethodDelete)
			Expect(err).NotTo(HaveOccurred())

			params, err = validateFlowLogNamespacesRequest(req)
			Expect(err).To(BeEquivalentTo(errInvalidMethod))
			Expect(params).To(BeNil())
		})

		It("should return a valid params object with the limit set to 1000 when passed an empty limit", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("limit", "")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamespacesRequest(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(params.Limit).To(BeNumerically("==", 1000))
		})

		It("should return a valid params object with the limit set to 1000 when passed a 0 limit", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("limit", "0")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamespacesRequest(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(params.Limit).To(BeNumerically("==", 1000))
		})

		It("should return an errParseRequest when passed a request with a negative limit parameter", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("limit", "-100")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamespacesRequest(req)
			Expect(err).To(BeEquivalentTo(errParseRequest))
			Expect(params).To(BeNil())
		})

		It("should return an errParseRequest when passed a request with a word as the limit parameter", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("limit", "ten")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamespacesRequest(req)
			Expect(err).To(BeEquivalentTo(errParseRequest))
			Expect(params).To(BeNil())
		})

		It("should return an errParseRequest when passed a request with a floating number as the limit parameter", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("limit", "3.14")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamespacesRequest(req)
			Expect(err).To(BeEquivalentTo(errParseRequest))
			Expect(params).To(BeNil())
		})

		It("should return an errParseRequest when passed a request with a max int32 + 1 number as the limit parameter", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("limit", "2147483648")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamespacesRequest(req)
			Expect(err).To(BeEquivalentTo(errParseRequest))
			Expect(params).To(BeNil())
		})

		It("should return an errParseRequest when passed a request with a min int32 - 1 number as the limit parameter", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("limit", "-2147483648")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamespacesRequest(req)
			Expect(err).To(BeEquivalentTo(errParseRequest))
			Expect(params).To(BeNil())
		})

		It("should return an errParseRequest when passed a request with an invalid unprotected param", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("unprotected", "xvz")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamespacesRequest(req)
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

			params, err := validateFlowLogNamespacesRequest(req)
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

			params, err = validateFlowLogNamespacesRequest(req)
			Expect(err).To(BeEquivalentTo(errInvalidAction))
			Expect(params).To(BeNil())
		})

		It("should return a valid FlowLogNamespaceParams object with the Actions from the request", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("actions", "allow")
			q.Add("actions", "deny")
			q.Add("actions", "unknown")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamespacesRequest(req)
			Expect(err).To(Not(HaveOccurred()))
			Expect(params.Actions[0]).To(BeEquivalentTo("allow"))
			Expect(params.Actions[1]).To(BeEquivalentTo("deny"))
			Expect(params.Actions[2]).To(BeEquivalentTo("unknown"))
		})

		It("should return a valid FlowLogNamespaceParams object when passed upper case parameters", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("actions", "ALLOW")
			q.Add("cluster", "CLUSTER")
			q.Add("prefix", "TIGERA-")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamespacesRequest(req)
			Expect(err).To(Not(HaveOccurred()))
			Expect(params.Actions[0]).To(BeEquivalentTo("allow"))
			Expect(params.ClusterName).To(BeEquivalentTo("cluster"))
			Expect(params.Prefix).To(BeEquivalentTo("tigera-.*"))
		})

		It("should return a valid FlowLogNamespaceParams when passed a request with valid start/end time", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("actions", "allow")
			q.Add("actions", "deny")
			startTimeObject, endTimeObject := getTestStartAndEndTime()
			q.Add("startDateTime", startTimeTest)
			q.Add("endDateTime", endTimeTest)

			Expect(err).To(Not(HaveOccurred()))
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamesRequest(req)
			Expect(err).To(Not(HaveOccurred()))
			Expect(params.StartDateTime).To(BeEquivalentTo(startTimeObject))
			Expect(params.EndDateTime).To(BeEquivalentTo(endTimeObject))
		})
	})

	Context("Test that the getNamespacesFromElastic function behaves as expected", func() {
		It("should retrieve all namespaces with prefix tigera-e", func() {
			By("Creating a mock ES client with a mocked out search results")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{prefixResponse})

			By("Creating params with the prefix tigera-e")
			params := &FlowLogNamespaceParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "tigera-e",
				ClusterName: "cluster",
			}

			namespaces, err := getNamespacesFromElastic(params, esClient, rbac.NewAlwaysAllowFlowHelper())
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(namespaces)).To(BeNumerically("==", 2))
			Expect(namespaces[0].Name).To(BeEquivalentTo("tigera-eck-operator"))
			Expect(namespaces[1].Name).To(BeEquivalentTo("tigera-elasticsearch"))
		})

		It("should retrieve an empty array of namespace objects", func() {
			By("Creating a mock ES client with a mocked out search results")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{emptyResponse})

			By("Creating params with the prefix tigera-elasticccccccc")
			params := &FlowLogNamespaceParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "cluster",
			}

			namespaces, err := getNamespacesFromElastic(params, esClient, rbac.NewAlwaysAllowFlowHelper())
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(namespaces)).To(BeNumerically("==", 0))
		})

		It("should retrieve an array of namespace objects with no duplicates", func() {
			By("Creating a mock ES client with a mocked out search results containing duplicates")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{duplicatesResponse})

			params := &FlowLogNamespaceParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "cluster",
			}

			namespaces, err := getNamespacesFromElastic(params, esClient, rbac.NewAlwaysAllowFlowHelper())
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(namespaces)).To(BeNumerically("==", 10))
			Expect(namespaces[0].Name).To(BeEquivalentTo("tigera-prometheus"))
			Expect(namespaces[1].Name).To(BeEquivalentTo("tigera-elasticsearch"))
			Expect(namespaces[2].Name).To(BeEquivalentTo("tigera-compliance"))
			Expect(namespaces[3].Name).To(BeEquivalentTo("kube-system"))
			Expect(namespaces[4].Name).To(BeEquivalentTo("tigera-fluentd"))
			Expect(namespaces[5].Name).To(BeEquivalentTo("tigera-manager"))
			Expect(namespaces[6].Name).To(BeEquivalentTo("tigera-system"))
			Expect(namespaces[7].Name).To(BeEquivalentTo("tigera-eck-operator"))
			Expect(namespaces[8].Name).To(BeEquivalentTo("tigera-kibana"))
			Expect(namespaces[9].Name).To(BeEquivalentTo("tigera-intrusion-detection"))
		})

		It("should retrieve an array of namespace objects with no duplicates and only up to the limit", func() {
			By("Creating a mock ES client with a mocked out search results containing duplicates")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{duplicatesResponse})

			params := &FlowLogNamespaceParams{
				Limit:       3,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "cluster",
			}

			namespaces, err := getNamespacesFromElastic(params, esClient, rbac.NewAlwaysAllowFlowHelper())
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(namespaces)).To(BeNumerically("==", 3))
			Expect(namespaces[0].Name).To(BeEquivalentTo("tigera-prometheus"))
			Expect(namespaces[1].Name).To(BeEquivalentTo("tigera-elasticsearch"))
			Expect(namespaces[2].Name).To(BeEquivalentTo("tigera-compliance"))
		})

		It("should return an empty response when the search result contains no aggregations", func() {
			By("Creating a mock ES client with a mocked out search results")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{missingAggregations})

			params := &FlowLogNamespaceParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "cluster",
			}

			namespaces, err := getNamespacesFromElastic(params, esClient, rbac.NewAlwaysAllowFlowHelper())
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(namespaces)).To(BeNumerically("==", 0))
		})

		It("should return an empty response when the endDateTime is in the past", func() {
			By("Creating a mock ES client with a mocked out search results")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{missingAggregations})

			_, endTimeObject := getTestStartAndEndTime()

			params := &FlowLogNamespaceParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "cluster",
				EndDateTime: endTimeObject,
			}

			namespaces, err := getNamespacesFromElastic(params, esClient, rbac.NewAlwaysAllowFlowHelper())
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(namespaces)).To(BeNumerically("==", 0))
		})

		It("should return an empty response when the startDateTime is in the future", func() {
			By("Creating a mock ES client with a mocked out search results")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{missingAggregations})

			startTimeObject := "now + 20d"

			params := &FlowLogNamespaceParams{
				Limit:         2000,
				Actions:       []string{"allow", "deny", "unknown"},
				Prefix:        "",
				ClusterName:   "cluster",
				StartDateTime: startTimeObject,
			}

			namespaces, err := getNamespacesFromElastic(params, esClient, rbac.NewAlwaysAllowFlowHelper())
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(namespaces)).To(BeNumerically("==", 0))
		})

		It("should return an error when the query fails", func() {
			By("Creating a mock ES client with badly formed search results")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{malformedResponse})

			params := &FlowLogNamespaceParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "",
			}

			namespaces, err := getNamespacesFromElastic(params, esClient, rbac.NewAlwaysAllowFlowHelper())
			Expect(err).To(HaveOccurred())
			Expect(namespaces).To(BeNil())
		})
	})

	Context("Test that the buildESQuery function applies filters only when necessary", func() {
		It("should return a query without filters", func() {
			By("Creating params with no actions")
			params := &FlowLogNamespaceParams{
				Limit:       2000,
				Prefix:      "",
				ClusterName: "",
			}

			query := buildESQuery(params)
			queryInf, err := query.Source()
			Expect(err).To(Not(HaveOccurred()))
			queryMap := queryInf.(map[string]interface{})
			boolQueryMap := queryMap["bool"].(map[string]interface{})
			Expect(len(boolQueryMap)).To(BeNumerically("==", 0))
		})

		It("should return a query with filters", func() {
			By("Creating params with actions")
			params := &FlowLogNamespaceParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "",
			}

			query := buildESQuery(params)
			queryInf, err := query.Source()
			Expect(err).To(Not(HaveOccurred()))
			queryMap := queryInf.(map[string]interface{})
			boolQueryMap := queryMap["bool"].(map[string]interface{})
			Expect(len(boolQueryMap)).To(BeNumerically("==", 1))
		})
	})

	Context("Test that the buildAggregations function creates the appropriate composite aggregation", func() {
		It("should return aggregations without Include filters", func() {
			By("Creating params without a prefix")
			params := &FlowLogNamespaceParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "",
			}
			sourceAggTermsMap, destAggTermsMap, err := getNamespaceAggregationTermsMaps(params)
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(sourceAggTermsMap)).To(BeNumerically("==", 1))
			Expect(len(destAggTermsMap)).To(BeNumerically("==", 1))
		})
	})

	/*
		Context("Test that validateNamespaceRBAC properly validates namespace permissions", func() {
			It("should allow users with access to all namespaces but not specific ones", func() {
				resources := map[string]bool{
					"networksets": true,
					"pods":        true,
				}
				namespaces := map[string]bool{
					"": true,
				}
				flowHelper := rbac.NewMockFlowHelper(resources, false)
				Expect(validateNamespaceRBAC("", flowHelper)).To(BeTrue())
				Expect(validateNamespaceRBAC("tigera-elasticsearch", flowHelper)).To(BeTrue())
				Expect(validateNamespaceRBAC("tigera-system", flowHelper)).To(BeTrue())
				Expect(validateNamespaceRBAC("random-namespace", flowHelper)).To(BeTrue())
			})
		})
	*/
})

func getNamespaceAggregationTermsMaps(params *FlowLogNamespaceParams) (map[string]interface{}, map[string]interface{},
	error) {
	var empty map[string]interface{}
	nameAgg := buildAggregation(params, empty)
	nameAggInf, err := nameAgg.Source()
	if err != nil {
		return nil, nil, err
	}
	// Pull out the terms for the composite aggregation. It should be of the form:
	//    "composite": {
	//        "sources": [
	//            {
	//                "source_namespace": {
	//                    "terms": {
	//                        "field": "source_namespace",
	//                    },
	//                },
	//            },
	//            {
	//                "dest_namespace": {
	//                    "terms": {
	//                        "field": "dest_namespace",
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
		if srcMapInf, exist := mapping[sourceAggregationName]; exist {
			srcMap := srcMapInf.(map[string]interface{})
			sourceAggTermsMap = srcMap["terms"].(map[string]interface{})
		}
		if destMapInf, exist := mapping[destAggregationName]; exist {
			destMap := destMapInf.(map[string]interface{})
			destAggTermsMap = destMap["terms"].(map[string]interface{})
		}
	}
	return sourceAggTermsMap, destAggTermsMap, nil
}
