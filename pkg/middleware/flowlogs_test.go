package middleware

import (
	"encoding/json"
	"github.com/olivere/elastic/v7"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	listMock "github.com/tigera/compliance/pkg/list/mock"
	"github.com/tigera/es-proxy/pkg/pip"
	pipcfg "github.com/tigera/es-proxy/pkg/pip/config"
	lmaelastic "github.com/tigera/lma/pkg/elastic"
	"io/ioutil"
	"os"

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

		It("should return an errParseRequest when passed a request an badly formatted policyPreview param", func() {
			req, err := newTestRequestWithParam(http.MethodGet, "policyPreview", invalidPreview)
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

		It("should return errInvalidPolicyPreview when passed a request with a policyPreview that has an invalid verb", func() {
			validPreviewBadVerb, err := ioutil.ReadFile("testdata/flow_logs_valid_preview_bad_verb.json")
			Expect(err).To(Not(HaveOccurred()))
			req, err := newTestRequestWithParam(http.MethodGet, "policyPreview", string(validPreviewBadVerb))
			Expect(err).NotTo(HaveOccurred())
			params, err := validateFlowLogsRequest(req)
			Expect(err).To(BeEquivalentTo(errInvalidPolicyPreview))
			Expect(params).To(BeNil())
		})

		It("should return a valid FlowLogsParams object", func() {
			req, err := http.NewRequest(http.MethodGet, "", nil)
			Expect(err).NotTo(HaveOccurred())
			startTimeObject, endTimeObject, err := getTestStartAndEndTime()
			Expect(err).To(Not(HaveOccurred()))
			validPreview, err := ioutil.ReadFile("testdata/flow_logs_valid_preview.json")
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
			q.Add("policyPreview", string(validPreview))
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
			Expect(params.PolicyPreview.NetworkPolicy.Name).To(BeEquivalentTo("calico-node-alertmanager-mesh"))
			Expect(params.PolicyPreview.NetworkPolicy.Namespace).To(BeEquivalentTo("tigera-prometheus"))
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

			searchResults, err := getFLowLogsFromElastic(params, esClient, nil)
			Expect(err).To(Not(HaveOccurred()))
			Expect(searchResults).To(BeAssignableToTypeOf(&elastic.SearchResult{}))
			convertedResults := searchResults.(*elastic.SearchResult)
			Expect(convertedResults.Hits.Hits[0].Index).To(BeEquivalentTo("tigera_secure_ee_flows.cluster.20191204"))
			Expect(convertedResults.Hits.Hits[0].Id).To(BeEquivalentTo("Y552z24BCJYmuA0kSFuo"))
			Expect(convertedResults.Hits.Hits[0].Type).To(BeEquivalentTo("_doc"))
		})

		It("should fail to retrieve a search results object and return an error", func() {
			By("Creating a mock ES client with a mock malformed response")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{malformedFlowsResponse})
			params := &FlowLogsParams{}

			searchResults, err := getFLowLogsFromElastic(params, esClient, nil)
			Expect(err).To(HaveOccurred())
			Expect(searchResults).To(BeNil())
		})

		It("should retrieve a FlowLogResults object", func() {
			err := os.Setenv("TIGERA_PIP_MAX_CALCULATION_TIME", "100s")
			Expect(err).To(Not(HaveOccurred()))
			esResponse, err := ioutil.ReadFile("testdata/flow_logs_aggr_response.json")
			Expect(err).To(Not(HaveOccurred()))
			validPreview, err := ioutil.ReadFile("testdata/flow_logs_valid_preview.json")
			Expect(err).To(Not(HaveOccurred()))
			aggResponse, err := ioutil.ReadFile("testdata/flow_logs_pip_aggregations.json")
			Expect(err).To(Not(HaveOccurred()))
			preview, err := getPolicyPreview(string(validPreview))
			Expect(err).To(Not(HaveOccurred()))

			listSrc := listMock.NewSource()
			listSrc.Initialize(time.Now())
			esClient = lmaelastic.NewMockSearchClient([]interface{}{string(esResponse)})
			pipClient := pip.New(pipcfg.MustLoadConfig(), listSrc, esClient)
			params := &FlowLogsParams{
				PolicyPreview: preview,
			}

			searchResults, err := getFLowLogsFromElastic(params, esClient, pipClient)
			Expect(err).To(Not(HaveOccurred()))
			Expect(searchResults).To(BeAssignableToTypeOf(&pip.FlowLogResults{}))
			convertedResults := searchResults.(*pip.FlowLogResults)
			// the took field won't always match the expected response since it is timer based so overwrite it here
			convertedResults.Took = 3
			searchData, err := json.Marshal(convertedResults)
			Expect(err).To(Not(HaveOccurred()))
			Expect(searchData).To(MatchJSON(aggResponse))
		})

		It("should retrieve a FlowLogResults object with only 1 bucket in each section due to a limit", func() {
			err := os.Setenv("TIGERA_PIP_MAX_CALCULATION_TIME", "100s")
			Expect(err).To(Not(HaveOccurred()))
			esResponse, err := ioutil.ReadFile("testdata/flow_logs_aggr_response_2.json")
			Expect(err).To(Not(HaveOccurred()))
			validPreview, err := ioutil.ReadFile("testdata/flow_logs_valid_preview.json")
			Expect(err).To(Not(HaveOccurred()))
			aggResponse, err := ioutil.ReadFile("testdata/flow_logs_pip_1_aggregation.json")
			Expect(err).To(Not(HaveOccurred()))
			preview, err := getPolicyPreview(string(validPreview))
			Expect(err).To(Not(HaveOccurred()))

			listSrc := listMock.NewSource()
			listSrc.Initialize(time.Now())
			esClient = lmaelastic.NewMockSearchClient([]interface{}{string(esResponse)})
			pipClient := pip.New(pipcfg.MustLoadConfig(), listSrc, esClient)
			params := &FlowLogsParams{
				PolicyPreview: preview,
				Limit:         1,
			}

			searchResults, err := getFLowLogsFromElastic(params, esClient, pipClient)
			Expect(err).To(Not(HaveOccurred()))
			Expect(searchResults).To(BeAssignableToTypeOf(&pip.FlowLogResults{}))
			convertedResults := searchResults.(*pip.FlowLogResults)
			// the took field won't always match the expected response since it is timer based so overwrite it here
			convertedResults.Took = 3
			searchData, err := json.Marshal(convertedResults)
			Expect(err).To(Not(HaveOccurred()))
			Expect(searchData).To(MatchJSON(aggResponse))
		})

		It("should retrieve a FlowLogResults object with 2 buckets in each section due to a limit", func() {
			err := os.Setenv("TIGERA_PIP_MAX_CALCULATION_TIME", "100s")
			Expect(err).To(Not(HaveOccurred()))
			esResponse, err := ioutil.ReadFile("testdata/flow_logs_aggr_response_2.json")
			Expect(err).To(Not(HaveOccurred()))
			validPreview, err := ioutil.ReadFile("testdata/flow_logs_valid_preview.json")
			Expect(err).To(Not(HaveOccurred()))
			preview, err := getPolicyPreview(string(validPreview))
			Expect(err).To(Not(HaveOccurred()))

			listSrc := listMock.NewSource()
			listSrc.Initialize(time.Now())
			esClient = lmaelastic.NewMockSearchClient([]interface{}{string(esResponse)})
			pipClient := pip.New(pipcfg.MustLoadConfig(), listSrc, esClient)
			params := &FlowLogsParams{
				PolicyPreview: preview,
				Limit:         2,
			}

			searchResults, err := getFLowLogsFromElastic(params, esClient, pipClient)
			Expect(err).To(Not(HaveOccurred()))
			Expect(searchResults).To(BeAssignableToTypeOf(&pip.FlowLogResults{}))
			convertedResults := searchResults.(*pip.FlowLogResults)
			// Since bucket ordering can be different just check for the length
			flogBuckets := convertedResults.Aggregations["flog_buckets"].(map[string]interface{})
			buckets := flogBuckets["buckets"].([]map[string]interface{})
			Expect(len(buckets)).To(BeNumerically("==", 2))
		})

		It("should fail to retrieve a FlowLogResults object and return an error", func() {
			listSrc := listMock.NewSource()
			listSrc.Initialize(time.Now())
			esClient = lmaelastic.NewMockSearchClient([]interface{}{""})
			pipClient := pip.New(pipcfg.MustLoadConfig(), listSrc, esClient)
			params := &FlowLogsParams{
				PolicyPreview: &PolicyPreview{},
			}

			searchResults, err := getFLowLogsFromElastic(params, esClient, pipClient)
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
