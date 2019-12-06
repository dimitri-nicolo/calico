package middleware

import (
	"encoding/json"
	"github.com/olivere/elastic/v7"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"

	lmaelastic "github.com/tigera/lma/pkg/elastic"
	"net/http"
	"time"
)

const (
	startTimeTest          = "2019-12-03T21:45:57-08:00"
	endTimeTest            = "2019-12-03T21:51:01-08:00"
	invalidFlowTypes       = `[network", "networkSSet", "wepp", "heppp"]`
	invalidActions         = `["allowW", "deeny", "unknownn"]`
	malformedFlowsResponse = `{badlyFormedNamesJson}`
)

var _ = Describe("Test /flowLogs endpoint functions", func() {
	var esClient lmaelastic.Client

	Context("Test that the validateFlowLogNamesRequest function behaves as expected", func() {
		It("should return an errInvalidMethod when passed a request with an http method other than GET", func() {
			By("Creating a request with a POST method")
			req, err := newTestRequest(http.MethodPost)
			Expect(err).NotTo(HaveOccurred())

			params, err := validateFlowLogsRequest(req)
			Expect(err).To(BeEquivalentTo(errInvalidMethod))
			Expect(params).To(BeNil())

			By("Creating a request with a DELETE method")
			req, err = newTestRequest(http.MethodDelete)
			Expect(err).NotTo(HaveOccurred())

			params, err = validateFlowLogsRequest(req)
			Expect(err).To(BeEquivalentTo(errInvalidMethod))
			Expect(params).To(BeNil())
		})

		It("should return an errParseRequest when passed a request an invalid limit param", func() {
			req, err := newTestRequestWithParam(http.MethodGet, "limit", "-2147483648")
			Expect(err).NotTo(HaveOccurred())
			params, err := validateFlowLogsRequest(req)
			Expect(err).To(BeEquivalentTo(errParseRequest))
			Expect(params).To(BeNil())
		})

		It("should return an errParseRequest when passed a request an invalid srcLabels param", func() {
			req, err := newTestRequestWithParams(http.MethodGet, "srcLabels", invalidSelectors)
			Expect(err).NotTo(HaveOccurred())
			params, err := validateFlowLogsRequest(req)
			Expect(err).To(BeEquivalentTo(errParseRequest))
			Expect(params).To(BeNil())
		})

		It("should return an errParseRequest when passed a request an invalid dstLabels param", func() {
			req, err := newTestRequestWithParams(http.MethodGet, "dstLabels", invalidSelectors)
			Expect(err).NotTo(HaveOccurred())
			params, err := validateFlowLogsRequest(req)
			Expect(err).To(BeEquivalentTo(errParseRequest))
			Expect(params).To(BeNil())
		})

		It("should return an errParseRequest when passed a request an badly formatted startDateTime param", func() {
			req, err := newTestRequestWithParam(http.MethodGet, "startDateTime", "20199-13-0321:51:01-08:00")
			Expect(err).NotTo(HaveOccurred())
			params, err := validateFlowLogsRequest(req)
			Expect(err).To(BeEquivalentTo(errParseRequest))
			Expect(params).To(BeNil())
		})

		It("should return an errParseRequest when passed a request an badly formatted endDateTime param", func() {
			req, err := newTestRequestWithParam(http.MethodGet, "endDateTime", "20199-13-0321:51:01-08:00")
			Expect(err).NotTo(HaveOccurred())
			params, err := validateFlowLogsRequest(req)
			Expect(err).To(BeEquivalentTo(errParseRequest))
			Expect(params).To(BeNil())
		})

		It("should return an errInvalidFlowType when passed a request with an invalid srcType param", func() {
			req, err := newTestRequestWithParam(http.MethodGet, "srcType", invalidFlowTypes)
			Expect(err).NotTo(HaveOccurred())
			params, err := validateFlowLogsRequest(req)
			Expect(err).To(BeEquivalentTo(errInvalidFlowType))
			Expect(params).To(BeNil())
		})

		It("should return an errInvalidFlowType when passed a request with an invalid dstType param", func() {
			req, err := newTestRequestWithParam(http.MethodGet, "dstType", invalidFlowTypes)
			Expect(err).NotTo(HaveOccurred())
			params, err := validateFlowLogsRequest(req)
			Expect(err).To(BeEquivalentTo(errInvalidFlowType))
			Expect(params).To(BeNil())
		})

		It("should return an errInvalidLabelSelector when passed a request with a valid srcLabels param but invalid operator", func() {
			req, err := newTestRequestWithParams(http.MethodGet, "srcLabels", validSelectorsBadOperators)
			Expect(err).NotTo(HaveOccurred())
			params, err := validateFlowLogsRequest(req)
			Expect(err).To(BeEquivalentTo(errInvalidLabelSelector))
			Expect(params).To(BeNil())
		})

		It("should return an errInvalidLabelSelector when passed a request with a valid dstLabels param but invalid operator", func() {
			req, err := newTestRequestWithParams(http.MethodGet, "dstLabels", validSelectorsBadOperators)
			Expect(err).NotTo(HaveOccurred())
			params, err := validateFlowLogsRequest(req)
			Expect(err).To(BeEquivalentTo(errInvalidLabelSelector))
			Expect(params).To(BeNil())
		})

		It("should return an errInvalidAction when passed a request with an actions param containing invalid actions", func() {
			req, err := newTestRequestWithParam(http.MethodGet, "actions", invalidActions)
			Expect(err).NotTo(HaveOccurred())
			params, err := validateFlowLogsRequest(req)
			Expect(err).To(BeEquivalentTo(errInvalidAction))
			Expect(params).To(BeNil())
		})

		It("should return a valid FlowLogsParams object", func() {
			req, err := http.NewRequest(http.MethodGet, "", nil)
			Expect(err).NotTo(HaveOccurred())
			startTimeObject, endTimeObject, err := getTestStartAndEndTime()
			Expect(err).To(Not(HaveOccurred()))
			q := req.URL.Query()
			q.Add("cluster", "cluster2")
			q.Add("limit", "2000")
			q.Add("srcType", "network")
			q.Add("srcType", "networkSet")
			q.Add("dstType", "wep")
			q.Add("dstType", "hep")
			q.Add("srcLabels", validSelectors[0])
			q.Add("srcLabels", validSelectors[1])
			q.Add("dstLabels", validSelectors[0])
			q.Add("dstLabels", validSelectors[1])
			q.Add("startDateTime", startTimeTest)
			q.Add("endDateTime", endTimeTest)
			q.Add("actions", "allow")
			q.Add("actions", "unknown")
			q.Add("namespace", "tigera-elasticsearch")
			q.Add("srcDstNamePrefix", "coredns")
			req.URL.RawQuery = q.Encode()
			params, err := validateFlowLogsRequest(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(params.ClusterName).To(BeEquivalentTo("cluster2"))
			Expect(params.Limit).To(BeNumerically("==", 2000))
			Expect(params.SourceType[0]).To(BeEquivalentTo("net"))
			Expect(params.SourceType[1]).To(BeEquivalentTo("ns"))
			Expect(params.DestType[0]).To(BeEquivalentTo("wep"))
			Expect(params.DestType[1]).To(BeEquivalentTo("hep"))
			Expect(params.SourceLabels[0].Key).To(BeEquivalentTo("key1"))
			Expect(params.SourceLabels[1].Key).To(BeEquivalentTo("key2"))
			Expect(params.SourceLabels[0].Operator).To(BeEquivalentTo("="))
			Expect(params.SourceLabels[1].Operator).To(BeEquivalentTo("!="))
			Expect(params.SourceLabels[0].Values[0]).To(BeEquivalentTo("hi"))
			Expect(params.SourceLabels[0].Values[1]).To(BeEquivalentTo("hello"))
			Expect(params.DestLabels[0].Key).To(BeEquivalentTo("key1"))
			Expect(params.DestLabels[1].Key).To(BeEquivalentTo("key2"))
			Expect(params.DestLabels[0].Operator).To(BeEquivalentTo("="))
			Expect(params.DestLabels[1].Operator).To(BeEquivalentTo("!="))
			Expect(params.DestLabels[0].Values[0]).To(BeEquivalentTo("hi"))
			Expect(params.DestLabels[0].Values[1]).To(BeEquivalentTo("hello"))
			Expect(params.StartDateTime).To(BeEquivalentTo(startTimeObject))
			Expect(params.EndDateTime).To(BeEquivalentTo(endTimeObject))
			Expect(params.Actions[0]).To(BeEquivalentTo("allow"))
			Expect(params.Actions[1]).To(BeEquivalentTo("unknown"))
			Expect(params.Namespace).To(BeEquivalentTo("tigera-elasticsearch"))
			Expect(params.SourceDestNamePrefix).To(BeEquivalentTo("coredns"))
		})
	})

	Context("Test that the buildFlowLogsQuery function applies filters only when necessary", func() {
		It("should return a query without filters when passed an empty params object", func() {
			By("Creating empty params")
			params := &FlowLogsParams{}

			query := buildFlowLogsQuery(params)
			queryInf, err := query.Source()
			Expect(err).To(Not(HaveOccurred()))
			queryMap := queryInf.(map[string]interface{})
			boolQueryMap := queryMap["bool"].(map[string]interface{})
			Expect(len(boolQueryMap)).To(BeNumerically("==", 0))
		})

		It("should return a query without filters when passed a params object with zero start and end time", func() {
			params := &FlowLogsParams{
				StartDateTime: time.Time{},
				EndDateTime:   time.Time{},
			}

			query := buildFlowLogsQuery(params)
			queryInf, err := query.Source()
			Expect(err).To(Not(HaveOccurred()))
			queryMap := queryInf.(map[string]interface{})
			boolQueryMap := queryMap["bool"].(map[string]interface{})
			Expect(len(boolQueryMap)).To(BeNumerically("==", 0))
		})

		It("should return a query with a nested filter for dest labels containing one term and two terms queries",
			func() {
				params := &FlowLogsParams{
					DestLabels: []LabelSelector{
						{Key: "key1", Operator: "=", Values: []string{"test"}},
						{Key: "key2", Operator: "!=", Values: []string{"test", "test2"}},
						{Key: "key3", Operator: "=", Values: []string{"test", "test2", "test3"}},
					},
				}

				querySelectors, err := ioutil.ReadFile("testdata/flow_logs_query_dest_selectors.json")
				Expect(err).To(Not(HaveOccurred()))
				query := buildFlowLogsQuery(params)
				queryInf, err := query.Source()
				queryData, err := json.Marshal(queryInf)
				Expect(err).To(Not(HaveOccurred()))
				Expect(queryData).To(MatchJSON(querySelectors))
			})

		It("should return a query with a nested filter for source labels containing one term and two terms queries",
			func() {
				params := &FlowLogsParams{
					SourceLabels: []LabelSelector{
						{Key: "key1", Operator: "=", Values: []string{"test"}},
						{Key: "key2", Operator: "!=", Values: []string{"test", "test2"}},
						{Key: "key3", Operator: "=", Values: []string{"test", "test2", "test3"}},
					},
				}

				querySelectors, err := ioutil.ReadFile("testdata/flow_logs_query_source_selectors.json")
				Expect(err).To(Not(HaveOccurred()))
				query := buildFlowLogsQuery(params)
				queryInf, err := query.Source()
				queryData, err := json.Marshal(queryInf)
				Expect(err).To(Not(HaveOccurred()))
				Expect(queryData).To(MatchJSON(querySelectors))
			})

		It("should return a query with all filters applied", func() {
			By("Creating params object with all possible entries for filters")
			startTime, endTime, err := getTestStartAndEndTime()
			Expect(err).To(Not(HaveOccurred()))
			params := &FlowLogsParams{
				Actions:              []string{"allow", "deny", "unknown"},
				SourceType:           []string{"net", "ns", "wep", "hep"},
				DestType:             []string{"net", "ns", "wep", "hep"},
				StartDateTime:        startTime,
				EndDateTime:          endTime,
				Namespace:            "tigera-elasticsearch",
				SourceDestNamePrefix: "coredns",
				SourceLabels: []LabelSelector{
					{Key: "key1", Operator: "=", Values: []string{"test", "test2"}},
					{Key: "key2", Operator: "!=", Values: []string{"test", "test2"}},
				},
				DestLabels: []LabelSelector{
					{Key: "key1", Operator: "=", Values: []string{"test", "test2"}},
					{Key: "key2", Operator: "!=", Values: []string{"test", "test2"}},
				},
			}

			queryAllFilters, err := ioutil.ReadFile("testdata/flow_logs_query_all_filters.json")
			Expect(err).To(Not(HaveOccurred()))
			query := buildFlowLogsQuery(params)
			queryInf, err := query.Source()
			queryData, err := json.Marshal(queryInf)
			Expect(err).To(Not(HaveOccurred()))
			Expect(queryData).To(MatchJSON(queryAllFilters))
		})
	})

	Context("Test that the getFLowLogsFromElastic function behaves as expected", func() {
		It("should retrieve a search results object", func() {
			By("Creating a mock ES client with a mocked out search results")
			flowLogsResponseJSON, err := ioutil.ReadFile("testdata/flow_logs_response.json")
			Expect(err).To(Not(HaveOccurred()))
			esClient = lmaelastic.NewMockSearchClient([]interface{}{string(flowLogsResponseJSON)})
			params := &FlowLogsParams{}

			searchResults, err := getFLowLogsFromElastic(params, esClient)
			Expect(err).To(Not(HaveOccurred()))
			Expect(searchResults).To(BeAssignableToTypeOf(&elastic.SearchResult{}))
		})

		It("should fail to retrieve a search results object and return an error", func() {
			By("Creating a mock ES client with a mock malformed response")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{malformedFlowsResponse})
			params := &FlowLogsParams{}

			searchResults, err := getFLowLogsFromElastic(params, esClient)
			Expect(err).To(HaveOccurred())
			Expect(searchResults).To(BeNil())
		})
	})
})

func newTestRequest(method string) (*http.Request, error) {
	req, err := http.NewRequest(method, "", nil)
	return req, err
}

func getTestStartAndEndTime() (time.Time, time.Time, error) {
	startTimeObject, err := time.Parse(time.RFC3339, startTimeTest)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	endTimeObject, err := time.Parse(time.RFC3339, endTimeTest)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return startTimeObject, endTimeObject, nil
}

func newTestRequestWithParams(method string, key string, values []string) (*http.Request, error) {
	req, err := http.NewRequest(method, "", nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	for _, value := range values {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()
	return req, nil
}
