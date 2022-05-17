package middleware

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	"github.com/projectcalico/calico/lma/pkg/rbac"
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
                "source_namespace": "tigera-manager",
                "source_name_aggr": "tigera-manager-778447894c-*",
                "source_type": "wep",
                "dest_namespace": "tigera-elasticsearch",
                "dest_name_aggr": "tigera-secure-es-xg2jxdtnqn",
                "dest_type": "wep"
            },
            "buckets": [
                {
                    "key": {
                        "source_namespace": "tigera-manager",
                        "source_name_aggr": "tigera-manager-778447894c-*",
                        "source_type": "wep",
                        "dest_namespace": "tigera-elasticsearch",
                        "dest_name_aggr": "tigera-secure-es-xg2jxdtnqn",
                        "dest_type": "wep"
                    },
                    "doc_count": 4370
                },
                {
                    "key": {
                        "source_namespace": "default",
                        "source_name_aggr": "test-app-83958379dc",
                        "source_type": "wep",
                        "dest_namespace": "tigera-elasticsearch",
                        "dest_name_aggr": "tigera-secure-es-xg2jxdtnqn",
                        "dest_type": "wep"
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
                "source_namespace": "tigera-compliance",
                "source_name_aggr": "compliance-benchmarker-*",
                "source_type": "wep",
                "dest_namespace": "default",
                "dest_name_aggr": "default/kse.kubernetes",
                "dest_type": "wep"
            },
            "buckets": [
                {
                    "key": {
                        "source_namespace": "tigera-compliance",
                        "source_name_aggr": "compliance-benchmarker-*",
                        "source_type": "wep",
                        "dest_namespace": "tigera-compliance",
                        "dest_name_aggr": "compliance-controller-c7f4b94dd-*",
                        "dest_type": "wep"
                    },
                    "doc_count": 10930
                },
                {
                    "key": {
                        "source_namespace": "tigera-compliance",
                        "source_name_aggr": "compliance-benchmarker-*",
                        "source_type":  "wep",
                        "dest_namespace": "tigera-compliance",
                        "dest_name_aggr": "compliance-server-69c97dffcf-*",
                        "dest_type": "wep"
                    },
                    "doc_count": 4393
                },
                {
                    "key": {
                        "source_namespace": "tigera-compliance",
                        "source_name_aggr": "compliance-controller-c7f4b94dd-*",
                        "source_type": "wep",
                        "dest_namespace": "tigera-compliance",
                        "dest_name_aggr": "compliance-server-69c97dffcf-*",
                        "dest_type": "wep"
                    },
                    "doc_count": 4374
                },
                {
                    "key": {
                        "source_namespace": "tigera-compliance",
                        "source_name_aggr": "compliance-server-69c97dffcf-*",
                        "source_type": "wep",
                        "dest_namespace": "tigera-manager",
                        "dest_name_aggr": "tigera-manager-778447894c-*",
                        "dest_type": "wep"
                    },
                    "doc_count": 4372
                },
                {
                    "key": {
                        "source_namespace": "tigera-compliance",
                        "source_name_aggr": "compliance-server-69c97dffcf-*",
                        "source_type": "wep",
                        "dest_namespace": "tigera-compliance",
                        "dest_name_aggr": "compliance-controller-c7f4b94dd-*",
                        "dest_type": "wep"
                    },
                    "doc_count": 4374
                },
                {
                    "key": {
                        "source_namespace": "tigera-elasticsearch",
                        "source_name_aggr": "tigera-secure-es-xg2jxdtnqn",
                        "source_type": "wep",
                        "dest_namespace": "tigera-manager",
                        "dest_name_aggr": "tigera-manager-778447894c-*",
                        "dest_type": "wep"
                    },
                    "doc_count": 21865
                },
                {
                    "key": {
                        "source_namespace": "tigera-compliance",
                        "source_name_aggr": "compliance-benchmarker-*",
                        "source_type": "wep",
                        "dest_namespace": "default",
                        "dest_name_aggr": "default/kse.kubernetes",
                        "dest_type": "wep"
                    },
                    "doc_count": 2185
                }
            ]
        }
    }
}`
	globalResponse = `{
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
		"source_namespace": "tigera-manager",
                "source_name_aggr": "tigera-manager-778447894c-*",
                "source_type": "wep",
                "dest_namespace": "tigera-elasticsearch",
                "dest_name_aggr": "tigera-secure-es-xg2jxdtnqn",
                "dest_type": "wep"
            },
            "buckets": [
                {
                    "key": {
                        "source_namespace": "-",
                        "source_name_aggr": "tigera-cluster-*",
                        "source_type": "hep",
                        "dest_namespace": "-",
                        "dest_name_aggr": "tigera-global-networkset",
                        "dest_type": "ns"
                    },
                    "doc_count": 4370
                },
                {
                    "key": {
                        "source_namespace": "default",
                        "source_name_aggr": "test-app-83958379dc",
                        "source_type": "ns",
                        "dest_namespace": "tigera-elasticsearch",
                        "dest_name_aggr": "tigera-secure-es-xg2jxdtnqn",
                        "dest_type": "wep"
                    },
                    "doc_count": 3698
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
	httpStatusErrorNamesResponse = `{
    badlyFormedNamesJson
}`
)

var _ = Describe("Test /flowLogNames endpoint functions", func() {
	var esClient lmaelastic.Client

	Context("Test that the validateFlowLogNamesRequest function behaves as expected", func() {
		It("should return an ErrInvalidMethod when passed a request with an http method other than GET", func() {
			By("Creating a request with a POST method")
			req, err := newTestRequest(http.MethodPost)
			Expect(err).NotTo(HaveOccurred())

			params, err := validateFlowLogNamesRequest(req)
			Expect(err).To(BeEquivalentTo(ErrInvalidMethod))
			Expect(params).To(BeNil())

			By("Creating a request with a DELETE method")
			req, err = newTestRequest(http.MethodDelete)
			Expect(err).NotTo(HaveOccurred())

			params, err = validateFlowLogNamesRequest(req)
			Expect(err).To(BeEquivalentTo(ErrInvalidMethod))
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

		It("should return an ErrParseRequest when passed a request with a negative limit parameter", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("limit", "-100")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamesRequest(req)
			Expect(err).To(BeEquivalentTo(ErrParseRequest))
			Expect(params).To(BeNil())
		})

		It("should return an ErrParseRequest when passed a request with word as the limit parameter", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("limit", "ten")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamesRequest(req)
			Expect(err).To(BeEquivalentTo(ErrParseRequest))
			Expect(params).To(BeNil())
		})

		It("should return an ErrParseRequest when passed a request with a floating number as the limit parameter", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("limit", "3.14")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamesRequest(req)
			Expect(err).To(BeEquivalentTo(ErrParseRequest))
			Expect(params).To(BeNil())
		})

		It("should return an ErrParseRequest when passed a request with a max int32 + 1 number as the limit parameter", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("limit", "2147483648")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamesRequest(req)
			Expect(err).To(BeEquivalentTo(ErrParseRequest))
			Expect(params).To(BeNil())
		})

		It("should return an ErrParseRequest when passed a request with a min int32 - 1 number as the limit parameter", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("limit", "-2147483648")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamesRequest(req)
			Expect(err).To(BeEquivalentTo(ErrParseRequest))
			Expect(params).To(BeNil())
		})

		It("should return an ErrParseRequest when passed a request with an invalid unprotected param", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("unprotected", "xvz")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamesRequest(req)
			Expect(err).To(BeEquivalentTo(ErrParseRequest))
			Expect(params).To(BeNil())
		})

		It("should return an ErrParseRequest when passed a request with an invalid combination of actions and unprotected param", func() {
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
			q.Add("namespace", "TIGERA-ELASTICSEARCH")
			req.URL.RawQuery = q.Encode()

			params, err := validateFlowLogNamesRequest(req)
			Expect(err).To(Not(HaveOccurred()))
			Expect(params.Actions[0]).To(BeEquivalentTo("allow"))
			Expect(params.ClusterName).To(BeEquivalentTo("cluster"))
			Expect(params.Namespace).To(BeEquivalentTo("tigera-elasticsearch"))
		})

		It("should return a valid FlowLogNamespaceParams when passed a request with valid start/end time", func() {
			req, err := newTestRequest(http.MethodGet)
			Expect(err).NotTo(HaveOccurred())
			q := req.URL.Query()
			q.Add("actions", "ALLOW")
			q.Add("cluster", "CLUSTER")
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
			Expect(params.Namespace).To(BeEquivalentTo("tigera-elasticsearch"))
			Expect(params.StartDateTime).To(BeEquivalentTo(startTimeObject))
			Expect(params.EndDateTime).To(BeEquivalentTo(endTimeObject))
		})
	})

	Context("Test that the getNamesFromElastic function behaves as expected", func() {
		It("should retrieve all names with prefix tigera", func() {
			By("Creating a mock ES client with a mocked out search results")
			esClient = NewMockSearchClient([]interface{}{defaultResponse})

			By("Creating params with the prefix tigera")
			params := &FlowLogNamesParams{
				Limit:       1000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "tigera",
				ClusterName: "cluster",
				Namespace:   "",
			}

			names, err := getNamesFromElastic(params, esClient, rbac.NewAlwaysAllowFlowHelper())
			Expect(err).To(Not(HaveOccurred()))
			Expect(names).To(HaveLen(2))
			Expect(names[0]).To(Equal("tigera-manager-778447894c-*"))
			Expect(names[1]).To(Equal("tigera-secure-es-xg2jxdtnqn"))
		})

		It("should handle an empty array of names returned from elasticsearch", func() {
			By("Creating a mock ES client with a mocked out search results")
			esClient = NewMockSearchClient([]interface{}{emptyNamesResponse})

			By("Creating params with the prefix tigera")
			params := &FlowLogNamesParams{
				Limit:       1000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "tigera",
				ClusterName: "cluster",
				Namespace:   "",
			}

			names, err := getNamesFromElastic(params, esClient, rbac.NewAlwaysAllowFlowHelper())
			Expect(err).To(Not(HaveOccurred()))
			Expect(names).To(HaveLen(0))
		})

		It("should retrieve an empty array of names when the prefix filters them out", func() {
			By("Creating a mock ES client with a mocked out search results")
			esClient = NewMockSearchClient([]interface{}{emptyNamesResponse})

			By("Creating params with the prefix tigera-elasticccccccc")
			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "tigera-elasticccccccc.*",
				ClusterName: "cluster",
				Namespace:   "tigera-compliance",
			}

			names, err := getNamesFromElastic(params, esClient, rbac.NewAlwaysAllowFlowHelper())
			Expect(err).To(Not(HaveOccurred()))
			Expect(names).To(HaveLen(0))
		})

		It("should retrieve an array of names with no duplicates", func() {
			By("Creating a mock ES client with a mocked out search results containing duplicates")
			esClient = NewMockSearchClient([]interface{}{duplicateNamesResponse})

			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "cluster",
				Namespace:   "",
			}

			names, err := getNamesFromElastic(params, esClient, rbac.NewAlwaysAllowFlowHelper())
			Expect(err).To(Not(HaveOccurred()))
			Expect(names).To(HaveLen(6))
			Expect(names[0]).To(Equal("compliance-benchmarker-*"))
			Expect(names[1]).To(Equal("compliance-controller-c7f4b94dd-*"))
			Expect(names[2]).To(Equal("compliance-server-69c97dffcf-*"))
			Expect(names[3]).To(Equal("default/kse.kubernetes"))
			Expect(names[4]).To(Equal("tigera-manager-778447894c-*"))
			Expect(names[5]).To(Equal("tigera-secure-es-xg2jxdtnqn"))
		})

		It("should retrieve an array of names with no duplicates and only up to the limit", func() {
			By("Creating a mock ES client with a mocked out search results containing duplicates")
			esClient = NewMockSearchClient([]interface{}{duplicateNamesResponse})

			params := &FlowLogNamesParams{
				Limit:       3,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "cluster",
				Namespace:   "tigera-compliance",
			}

			names, err := getNamesFromElastic(params, esClient, rbac.NewAlwaysAllowFlowHelper())
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(names)).To(BeNumerically("==", 3))
			Expect(names[0]).To(Equal("compliance-benchmarker-*"))
			Expect(names[1]).To(Equal("compliance-controller-c7f4b94dd-*"))
			Expect(names[2]).To(Equal("compliance-server-69c97dffcf-*"))
		})

		It("should return an empty response when the search result contains no aggregations", func() {
			By("Creating a mock ES client with a mocked out search results")
			esClient = NewMockSearchClient([]interface{}{missingNamesAggregations})

			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "cluster",
				Namespace:   "tigera-compliance",
			}

			names, err := getNamesFromElastic(params, esClient, rbac.NewAlwaysAllowFlowHelper())
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(names)).To(BeNumerically("==", 0))
		})

		It("should return an empty response when the EndDateTime is in the past", func() {
			By("Creating a mock ES client with a mocked out search results")
			esClient = NewMockSearchClient([]interface{}{missingNamesAggregations})

			_, endTimeObject := getTestStartAndEndTime()

			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "cluster",
				Namespace:   "tigera-compliance",
				EndDateTime: endTimeObject,
			}

			names, err := getNamesFromElastic(params, esClient, rbac.NewAlwaysAllowFlowHelper())
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(names)).To(BeNumerically("==", 0))
		})

		It("should return an empty response when the StartDateTime is in the future", func() {
			By("Creating a mock ES client with a mocked out search results")
			esClient = NewMockSearchClient([]interface{}{missingNamesAggregations})

			startTimeObject := "now + 20d"

			params := &FlowLogNamesParams{
				Limit:         2000,
				Actions:       []string{"allow", "deny", "unknown"},
				Prefix:        "",
				ClusterName:   "cluster",
				Namespace:     "tigera-compliance",
				StartDateTime: startTimeObject,
			}

			names, err := getNamesFromElastic(params, esClient, rbac.NewAlwaysAllowFlowHelper())
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(names)).To(BeNumerically("==", 0))
		})

		It("should return an error when the query fails", func() {
			By("Creating a mock ES client with badly formed search results")
			esClient = NewMockSearchClient([]interface{}{httpStatusErrorNamesResponse})

			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "",
				Namespace:   "tigera-compliance",
			}

			names, err := getNamesFromElastic(params, esClient, rbac.NewAlwaysAllowFlowHelper())
			Expect(err).To(HaveOccurred())
			Expect(names).To(HaveLen(0))
		})

		It("should only return the endpoints for the specified namespace when the namespace is specified", func() {
			By("Creating a mock ES client with a mocked out search results")
			esClient = NewMockSearchClient([]interface{}{duplicateNamesResponse})

			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "",
				Namespace:   "tigera-compliance",
			}

			names, err := getNamesFromElastic(params, esClient, rbac.NewAlwaysAllowFlowHelper())
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(names)).To(BeNumerically("==", 3))
			Expect(names[0]).To(Equal("compliance-benchmarker-*"))
			Expect(names[1]).To(Equal("compliance-controller-c7f4b94dd-*"))
			Expect(names[2]).To(Equal("compliance-server-69c97dffcf-*"))
		})

		It("should return all endpoints and global endpoints with permissive RBAC", func() {
			By("Creating a mock ES client with a mocked out search results")
			esClient = NewMockSearchClient([]interface{}{globalResponse})

			params := &FlowLogNamesParams{
				Limit:       2000,
				Actions:     []string{"allow", "deny", "unknown"},
				Prefix:      "",
				ClusterName: "",
				Namespace:   "",
			}

			names, err := getNamesFromElastic(params, esClient, rbac.NewAlwaysAllowFlowHelper())
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(names)).To(BeNumerically("==", 4))
			Expect(names[0]).To(Equal("test-app-83958379dc"))
			Expect(names[1]).To(Equal("tigera-cluster-*"))
			Expect(names[2]).To(Equal("tigera-global-networkset"))
			Expect(names[3]).To(Equal("tigera-secure-es-xg2jxdtnqn"))
		})

		Context("getNamesFromElastic RBAC filtering", func() {
			var mockFlowHelper *rbac.MockFlowHelper
			BeforeEach(func() {
				mockFlowHelper = new(rbac.MockFlowHelper)
			})

			AfterEach(func() {
				mockFlowHelper.AssertExpectations(GinkgoT())
			})

			It("should only return endpoints when global endpoints are not allowed due to RBAC", func() {
				By("Creating a mock ES client with a mocked out search results")
				esClient = NewMockSearchClient([]interface{}{globalResponse})

				mockFlowHelper.On("CanListHostEndpoints").Return(false, nil)
				mockFlowHelper.On("CanListGlobalNetworkSets").Return(false, nil)
				mockFlowHelper.On("CanListNetworkSets", "").Return(true, nil)
				mockFlowHelper.On("CanListPods", "").Return(true, nil)

				params := &FlowLogNamesParams{
					Limit:       2000,
					Actions:     []string{"allow", "deny", "unknown"},
					Prefix:      "",
					ClusterName: "",
					Namespace:   "",
					Strict:      true,
				}

				names, err := getNamesFromElastic(params, esClient, mockFlowHelper)
				Expect(err).To(Not(HaveOccurred()))
				Expect(names).To(HaveLen(2))
				Expect(names[0]).To(Equal("test-app-83958379dc"))
				Expect(names[1]).To(Equal("tigera-secure-es-xg2jxdtnqn"))
			})

			It("should properly filter endpoints based on allowed namespaces", func() {
				By("Creating a mock ES client with a mocked out search results")
				esClient = NewMockSearchClient([]interface{}{globalResponse})

				mockFlowHelper.On("CanListHostEndpoints").Return(false, nil)
				mockFlowHelper.On("CanListGlobalNetworkSets").Return(false, nil)
				mockFlowHelper.On("CanListNetworkSets", "").Return(false, nil)
				mockFlowHelper.On("CanListNetworkSets", "default").Return(false, nil)
				mockFlowHelper.On("CanListPods", "").Return(false, nil)
				mockFlowHelper.On("CanListPods", "tigera-elasticsearch").Return(true, nil)

				params := &FlowLogNamesParams{
					Limit:       2000,
					Actions:     []string{"allow", "deny", "unknown"},
					Prefix:      "",
					ClusterName: "",
					Namespace:   "",
					Strict:      true,
				}

				names, err := getNamesFromElastic(params, esClient, mockFlowHelper)
				Expect(err).To(Not(HaveOccurred()))
				Expect(names).To(HaveLen(1))
				Expect(names[0]).To(Equal("tigera-secure-es-xg2jxdtnqn"))
			})

			It("should properly filter out endpoints based on the endpoint type per namespace", func() {
				By("Creating a mock ES client with a mocked out search results")
				esClient = NewMockSearchClient([]interface{}{globalResponse})

				mockFlowHelper.On("CanListHostEndpoints").Return(false, nil)
				mockFlowHelper.On("CanListGlobalNetworkSets").Return(false, nil)
				mockFlowHelper.On("CanListNetworkSets", "").Return(false, nil)
				mockFlowHelper.On("CanListNetworkSets", "default").Return(true, nil)
				mockFlowHelper.On("CanListPods", "").Return(false, nil)
				mockFlowHelper.On("CanListPods", "tigera-elasticsearch").Return(false, nil)

				params := &FlowLogNamesParams{
					Limit:       2000,
					Actions:     []string{"allow", "deny", "unknown"},
					Prefix:      "",
					ClusterName: "",
					Namespace:   "",
					Strict:      true,
				}

				names, err := getNamesFromElastic(params, esClient, mockFlowHelper)
				Expect(err).To(Not(HaveOccurred()))
				Expect(names).To(HaveLen(1))
				Expect(names[0]).To(Equal("test-app-83958379dc"))
			})

			It("should properly filter out endpoints based on the endpoint type globally", func() {
				By("Creating a mock ES client with a mocked out search results")
				esClient = NewMockSearchClient([]interface{}{globalResponse})

				mockFlowHelper.On("CanListHostEndpoints").Return(true, nil)
				mockFlowHelper.On("CanListGlobalNetworkSets").Return(false, nil)
				mockFlowHelper.On("CanListNetworkSets", "").Return(false, nil)
				mockFlowHelper.On("CanListNetworkSets", "default").Return(false, nil)
				mockFlowHelper.On("CanListPods", "").Return(false, nil)
				mockFlowHelper.On("CanListPods", "tigera-elasticsearch").Return(false, nil)

				params := &FlowLogNamesParams{
					Limit:       2000,
					Actions:     []string{"allow", "deny", "unknown"},
					Prefix:      "",
					ClusterName: "",
					Namespace:   "",
					Strict:      true,
				}

				names, err := getNamesFromElastic(params, esClient, mockFlowHelper)
				Expect(err).To(Not(HaveOccurred()))
				Expect(names).To(HaveLen(1))
				Expect(names[0]).To(Equal("tigera-cluster-*"))
			})

			It("should return all endpoints as long as RBAC permissions exist for one endpoint in the flow", func() {
				By("Creating a mock ES client with a mocked out search results")
				esClient = NewMockSearchClient([]interface{}{duplicateNamesResponse})
				mockFlowHelper := new(rbac.MockFlowHelper)

				mockFlowHelper.On("CanListHostEndpoints").Return(false, nil)
				mockFlowHelper.On("CanListGlobalNetworkSets").Return(false, nil)
				mockFlowHelper.On("CanListNetworkSets", "").Return(false, nil)
				mockFlowHelper.On("CanListNetworkSets", "default").Return(false, nil)
				mockFlowHelper.On("CanListPods", "").Return(false, nil)
				mockFlowHelper.On("CanListPods", "tigera-manager").Return(false, nil)
				mockFlowHelper.On("CanListPods", "tigera-elasticsearch").Return(false, nil)
				mockFlowHelper.On("CanListPods", "tigera-compliance").Return(true, nil)

				By("Creating params without strict RBAC enforcement")
				params := &FlowLogNamesParams{
					Limit:       2000,
					Actions:     []string{"allow", "deny", "unknown"},
					Prefix:      "",
					ClusterName: "",
					Namespace:   "",
				}

				names, err := getNamesFromElastic(params, esClient, mockFlowHelper)
				Expect(err).To(Not(HaveOccurred()))
				Expect(names).To(HaveLen(5))
				Expect(names[0]).To(Equal("compliance-benchmarker-*"))
				Expect(names[1]).To(Equal("compliance-controller-c7f4b94dd-*"))
				Expect(names[2]).To(Equal("compliance-server-69c97dffcf-*"))
				Expect(names[3]).To(Equal("default/kse.kubernetes"))
				Expect(names[4]).To(Equal("tigera-manager-778447894c-*"))
			})
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
			boolQueryShouldQueries := boolQueryMap["should"].([]interface{})
			Expect(boolQueryShouldQueries).To(HaveLen(2))
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
			boolQueryShouldQueries := boolQueryMap["should"].([]interface{})
			nestedBoolQueryMap := boolQueryFilterMap["bool"].(map[string]interface{})
			Expect(nestedBoolQueryMap).To(HaveLen(1))
			Expect(boolQueryShouldQueries).To(HaveLen(2))
		})

		It("should return a query with endpoint filters", func() {
			By("Creating params with type filters")
			params := &FlowLogNamesParams{
				Limit:       2000,
				Prefix:      "",
				ClusterName: "",
				Namespace:   "",
				SourceType:  []string{"net"},
				DestType:    []string{"wep"},
			}

			query := buildNamesQuery(params)
			queryInf, err := query.Source()
			Expect(err).To(Not(HaveOccurred()))
			queryMap := queryInf.(map[string]interface{})
			boolQueryMap := queryMap["bool"].(map[string]interface{})
			boolQueryShouldQueries := boolQueryMap["should"].([]interface{})
			Expect(boolQueryShouldQueries).To(HaveLen(2))
		})

		It("should return a query with label filters", func() {
			By("Creating params with type filters")
			params := &FlowLogNamesParams{
				Limit:       2000,
				Prefix:      "",
				ClusterName: "",
				Namespace:   "",
				SourceLabels: []LabelSelector{
					LabelSelector{
						Key:      "app",
						Operator: "=",
						Values:   []string{"test-app"},
					},
				},
				DestLabels: []LabelSelector{
					LabelSelector{
						Key:      "otherapp",
						Operator: "=",
						Values:   []string{"not-test-app"},
					},
				},
			}

			query := buildNamesQuery(params)
			queryInf, err := query.Source()
			Expect(err).To(Not(HaveOccurred()))
			queryMap := queryInf.(map[string]interface{})
			boolQueryMap := queryMap["bool"].(map[string]interface{})
			boolQueryShouldQueries := boolQueryMap["should"].([]interface{})
			Expect(boolQueryShouldQueries).To(HaveLen(2))
		})
	})

})
