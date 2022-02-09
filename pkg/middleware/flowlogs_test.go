package middleware

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/mock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/libcalico-go/lib/errors"
	"github.com/projectcalico/calico/libcalico-go/lib/resources"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/es-proxy/pkg/pip"
	pipcfg "github.com/tigera/es-proxy/pkg/pip/config"
	"github.com/tigera/lma/pkg/api"
	lmaelastic "github.com/tigera/lma/pkg/elastic"
	"github.com/tigera/lma/pkg/list"
	"github.com/tigera/lma/pkg/rbac"
)

const (
	startTimeTest                = "now-3h"
	endTimeTest                  = "now"
	invalidFlowTypes             = `[network", "networkSSet", "wepp", "heppp"]`
	invalidActions               = `["allowW", "deeny", "unknownn"]`
	httpStatusErrorFlowsResponse = `{badlyFormedNamesJson}`
)

var _ = Describe("Test /flowLogs endpoint functions", func() {
	var esClient lmaelastic.Client
	rbacHelper := &testHelper{
		action: "delete",
		name:   "default.calico-node-alertmanager-mesh",
	}
	Context("Test that the validateFlowLogNamesRequest function behaves as expected", func() {
		It("should return an ErrInvalidMethod when passed a request with an http method other than GET", func() {
			By("Creating a request with a POST method")
			req, err := newTestRequest(http.MethodPost)
			Expect(err).NotTo(HaveOccurred())

			params, err := validateFlowLogsRequest(req)
			Expect(err).To(BeEquivalentTo(ErrInvalidMethod))
			Expect(params).To(BeNil())

			By("Creating a request with a DELETE method")
			req, err = newTestRequest(http.MethodDelete)
			Expect(err).NotTo(HaveOccurred())

			params, err = validateFlowLogsRequest(req)
			Expect(err).To(BeEquivalentTo(ErrInvalidMethod))
			Expect(params).To(BeNil())
		})

		It("should return an ErrParseRequest when passed a request an invalid limit param", func() {
			req, err := newTestRequestWithParam(http.MethodGet, "limit", "-2147483648")
			Expect(err).NotTo(HaveOccurred())
			params, err := validateFlowLogsRequest(req)
			Expect(err).To(BeEquivalentTo(ErrParseRequest))
			Expect(params).To(BeNil())
		})

		It("should return an ErrParseRequest when passed a request an invalid unprotected param", func() {
			req, err := newTestRequestWithParam(http.MethodGet, "unprotected", "xvz")
			Expect(err).NotTo(HaveOccurred())
			params, err := validateFlowLogsRequest(req)
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

		It("should return an ErrParseRequest when passed a request an invalid srcLabels param", func() {
			req, err := newTestRequestWithParams(http.MethodGet, "srcLabels", invalidSelectors)
			Expect(err).NotTo(HaveOccurred())
			params, err := validateFlowLogsRequest(req)
			Expect(err).To(BeEquivalentTo(ErrParseRequest))
			Expect(params).To(BeNil())
		})

		It("should return an ErrParseRequest when passed a request an invalid dstLabels param", func() {
			req, err := newTestRequestWithParams(http.MethodGet, "dstLabels", invalidSelectors)
			Expect(err).NotTo(HaveOccurred())
			params, err := validateFlowLogsRequest(req)
			Expect(err).To(BeEquivalentTo(ErrParseRequest))
			Expect(params).To(BeNil())
		})

		It("should return an ErrParseRequest when passed a request an badly formatted policyPreview param", func() {
			req, err := newTestRequestWithParam(http.MethodGet, "policyPreview", invalidPreview)
			Expect(err).NotTo(HaveOccurred())
			params, err := validateFlowLogsRequest(req)
			Expect(err).To(BeEquivalentTo(ErrParseRequest))
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
			validPreviewBadVerb, err := ioutil.ReadFile("testdata/flow_logs_invalid_preview_bad_verb.json")
			Expect(err).To(Not(HaveOccurred()))
			req, err := newTestRequestWithParam(http.MethodGet, "policyPreview", string(validPreviewBadVerb))
			Expect(err).NotTo(HaveOccurred())
			params, err := validateFlowLogsRequest(req)
			Expect(err).To(BeEquivalentTo(errInvalidPolicyPreview))
			Expect(params).To(BeNil())
		})

		It("should return errInvalidPolicyPreview when passed a request with a policyPreview that has an extra unknown field", func() {
			validPreviewBadVerb, err := ioutil.ReadFile("testdata/flow_logs_invalid_preview_extra_field.json")
			Expect(err).To(Not(HaveOccurred()))
			req, err := newTestRequestWithParam(http.MethodGet, "policyPreview", string(validPreviewBadVerb))
			Expect(err).NotTo(HaveOccurred())
			params, err := validateFlowLogsRequest(req)
			Expect(err.Error()).To(Equal("Error parsing request parameters"))
			Expect(params).To(BeNil())
		})

		It("should return a valid FlowLogsParams object", func() {
			req, err := http.NewRequest(http.MethodGet, "", nil)
			Expect(err).NotTo(HaveOccurred())
			startTimeObject, endTimeObject := getTestStartAndEndTime()
			validPreview, err := ioutil.ReadFile("testdata/flow_logs_valid_preview.json")
			Expect(err).To(Not(HaveOccurred()))
			q := req.URL.Query()
			q.Add("cluster", "cluster2")
			q.Add("limit", "2000")
			q.Add("srcType", "net")
			q.Add("srcType", "ns")
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
			Expect(params.PolicyPreviews).To(HaveLen(1))
			Expect(params.PolicyPreviews[0].NetworkPolicy).To(BeAssignableToTypeOf(&v3.NetworkPolicy{}))
			Expect(params.PolicyPreviews[0].NetworkPolicy.(*v3.NetworkPolicy).Name).To(BeEquivalentTo("default.calico-node-alertmanager-mesh"))
			Expect(params.PolicyPreviews[0].NetworkPolicy.(*v3.NetworkPolicy).Namespace).To(BeEquivalentTo("tigera-prometheus"))
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
				StartDateTime: "",
				EndDateTime:   "",
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
				Expect(err).To(Not(HaveOccurred()))
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
				queryInf, _ := query.Source()
				queryData, err := json.Marshal(queryInf)
				Expect(err).To(Not(HaveOccurred()))
				Expect(queryData).To(MatchJSON(querySelectors))
			})

		It("should return a query with all filters applied", func() {
			By("Creating params object with all possible entries for filters")
			startTime, endTime := getTestStartAndEndTime()
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
				startDateTimeESParm: startTime,
				endDateTimeESParm:   endTime,
			}

			queryAllFilters, err := ioutil.ReadFile("testdata/flow_logs_query_all_filters.json")
			Expect(err).To(Not(HaveOccurred()))
			query := buildFlowLogsQuery(params)
			queryInf, _ := query.Source()
			queryData, err := json.Marshal(queryInf)
			Expect(err).To(Not(HaveOccurred()))
			Expect(queryData).To(MatchJSON(queryAllFilters))
		})
	})

	Context("Test that the getFlowLogsFromElastic function behaves as expected", func() {
		It("should retrieve a search results object", func() {
			By("Creating a mock ES client with a mocked out search results")
			flowLogsResponseJSON, err := ioutil.ReadFile("testdata/flow_logs_aggr_response.json")
			Expect(err).To(Not(HaveOccurred()))
			esClient = lmaelastic.NewMockSearchClient([]interface{}{string(flowLogsResponseJSON)})
			params := &FlowLogsParams{
				Limit: 2,
			}

			searchResults, stat, err := getFlowLogsFromElastic(lmaelastic.NewFlowFilterIncludeAll(), params, esClient)
			Expect(stat).To(Equal(http.StatusOK))
			Expect(err).To(Not(HaveOccurred()))
			Expect(searchResults).To(BeAssignableToTypeOf(&lmaelastic.CompositeAggregationResults{}))
			convertedResults := searchResults.(*lmaelastic.CompositeAggregationResults)
			Expect(convertedResults.TimedOut).To(BeFalse())
			Expect(convertedResults.Aggregations).To(HaveKey("flog_buckets"))
			m := convertedResults.Aggregations["flog_buckets"]
			Expect(m).To(BeAssignableToTypeOf(map[string]interface{}{}))
			b := m.(map[string]interface{})
			Expect(b).To(HaveKey("buckets"))
			Expect(b["buckets"]).To(HaveLen(2))
		})

		It("should fail to retrieve a search results object and return an error", func() {
			By("Creating a mock ES client with a mock http status error response")
			esClient = lmaelastic.NewMockSearchClient([]interface{}{httpStatusErrorFlowsResponse})
			params := &FlowLogsParams{}

			searchResults, stat, err := getFlowLogsFromElastic(lmaelastic.NewFlowFilterIncludeAll(), params, esClient)
			Expect(stat).To(Equal(http.StatusInternalServerError))
			Expect(err).To(HaveOccurred())
			Expect(searchResults).To(BeNil())
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
			previews, err := getPolicyPreviews([]string{string(validPreview)})
			Expect(err).To(Not(HaveOccurred()))
			Expect(previews).To(HaveLen(1))

			listSrc := newMockLister()
			esClient = lmaelastic.NewMockSearchClient([]interface{}{string(esResponse)})
			pipClient := pip.New(pipcfg.MustLoadConfig(), listSrc, esClient)
			params := &FlowLogsParams{
				PolicyPreviews: previews,
				Limit:          1,
			}

			searchResults, stat, err := getPIPFlowLogsFromElastic(lmaelastic.NewFlowFilterIncludeAll(), params, pipClient, rbacHelper)
			Expect(stat).To(Equal(http.StatusOK))
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
			previews, err := getPolicyPreviews([]string{string(validPreview)})
			Expect(err).To(Not(HaveOccurred()))

			listSrc := newMockLister()
			esClient = lmaelastic.NewMockSearchClient([]interface{}{string(esResponse)})
			pipClient := pip.New(pipcfg.MustLoadConfig(), listSrc, esClient)
			params := &FlowLogsParams{
				PolicyPreviews: previews,
				Limit:          2,
			}

			searchResults, stat, err := getPIPFlowLogsFromElastic(lmaelastic.NewFlowFilterIncludeAll(), params, pipClient, rbacHelper)
			Expect(stat).To(Equal(http.StatusOK))
			Expect(err).To(Not(HaveOccurred()))
			Expect(searchResults).To(BeAssignableToTypeOf(&pip.FlowLogResults{}))
			convertedResults := searchResults.(*pip.FlowLogResults)
			// Since bucket ordering can be different just check for the length
			flogBuckets := convertedResults.Aggregations["flog_buckets"].(map[string]interface{})
			buckets := flogBuckets["buckets"].([]map[string]interface{})
			Expect(len(buckets)).To(BeNumerically("==", 2))
		})

		It("should retrieve a FlowLogResults object with no flows because none were impacted", func() {
			err := os.Setenv("TIGERA_PIP_MAX_CALCULATION_TIME", "100s")
			Expect(err).To(Not(HaveOccurred()))
			esResponse, err := ioutil.ReadFile("testdata/flow_logs_aggr_response_2.json")
			Expect(err).To(Not(HaveOccurred()))
			validPreview, err := ioutil.ReadFile("testdata/flow_logs_valid_preview.json")
			Expect(err).To(Not(HaveOccurred()))
			previews, err := getPolicyPreviews([]string{string(validPreview)})
			Expect(err).To(Not(HaveOccurred()))

			listSrc := newMockLister()
			esClient = lmaelastic.NewMockSearchClient([]interface{}{string(esResponse)})
			pipClient := pip.New(pipcfg.MustLoadConfig(), listSrc, esClient)
			params := &FlowLogsParams{
				PolicyPreviews: previews,
				ImpactedOnly:   true,
			}

			searchResults, stat, err := getPIPFlowLogsFromElastic(lmaelastic.NewFlowFilterIncludeAll(), params, pipClient, rbacHelper)
			Expect(stat).To(Equal(http.StatusOK))
			Expect(err).To(Not(HaveOccurred()))
			Expect(searchResults).To(BeAssignableToTypeOf(&pip.FlowLogResults{}))
			convertedResults := searchResults.(*pip.FlowLogResults)
			convertedResults.Took = 3
			flogBuckets := convertedResults.Aggregations["flog_buckets"].(map[string]interface{})
			buckets := flogBuckets["buckets"].([]map[string]interface{})
			Expect(len(buckets)).To(BeNumerically("==", 0))
		})

		It("should fail to retrieve a FlowLogResults object and return an error", func() {
			listSrc := newMockLister()
			esClient = lmaelastic.NewMockSearchClient([]interface{}{""})
			pipClient := pip.New(pipcfg.MustLoadConfig(), listSrc, esClient)
			params := &FlowLogsParams{
				PolicyPreviews: []PolicyPreview{},
			}

			searchResults, stat, err := getPIPFlowLogsFromElastic(lmaelastic.NewFlowFilterIncludeAll(), params, pipClient, rbacHelper)
			Expect(stat).To(Equal(http.StatusBadRequest))
			Expect(err).To(HaveOccurred())
			Expect(searchResults).To(BeNil())
		})

		It("should retrieve a FlowLogResults object with only 1 bucket in each section due to a limit, with results RBAC filtered (non-PIP)", func() {
			err := os.Setenv("TIGERA_PIP_MAX_CALCULATION_TIME", "100s")
			Expect(err).To(Not(HaveOccurred()))
			esResponse, err := ioutil.ReadFile("testdata/flow_logs_aggr_response_2.json")
			Expect(err).To(Not(HaveOccurred()))
			aggResponse, err := ioutil.ReadFile("testdata/flow_logs_1_aggregation_rbac.json")
			Expect(err).To(Not(HaveOccurred()))

			esClient = lmaelastic.NewMockSearchClient([]interface{}{string(esResponse)})
			params := &FlowLogsParams{
				Limit: 1,
			}

			mockFlowHelper := new(rbac.MockFlowHelper)

			// Allow all except HEP and GNPs.  The first result will be excluded.  The second result will have the GNP obfuscated.
			mockFlowHelper.On("CanListEndpoint", api.EndpointTypeHep, api.GlobalEndpointType).Return(false, nil)
			mockFlowHelper.On("CanListEndpoint", api.EndpointTypeNet, api.GlobalEndpointType).Return(false, nil)
			mockFlowHelper.On("CanListEndpoint", api.EndpointTypeWep, mock.Anything).Return(true, nil)
			flowFilter := lmaelastic.NewFlowFilterUserRBAC(mockFlowHelper)

			searchResults, stat, err := getFlowLogsFromElastic(flowFilter, params, esClient)

			mockFlowHelper.AssertExpectations(GinkgoT())

			Expect(stat).To(Equal(http.StatusOK))
			Expect(err).To(Not(HaveOccurred()))
			Expect(searchResults).To(BeAssignableToTypeOf(&lmaelastic.CompositeAggregationResults{}))
			convertedResults := searchResults.(*lmaelastic.CompositeAggregationResults)
			// the took field won't always match the expected response since it is timer based so overwrite it here
			convertedResults.Took = 3
			searchData, err := json.Marshal(convertedResults)
			Expect(err).To(Not(HaveOccurred()))
			Expect(searchData).To(MatchJSON(aggResponse))
		})

		It("should retrieve a FlowLogResults object with only 1 bucket in each section due to a limit, with results RBAC filtered (PIP)", func() {
			err := os.Setenv("TIGERA_PIP_MAX_CALCULATION_TIME", "100s")
			Expect(err).To(Not(HaveOccurred()))
			esResponse, err := ioutil.ReadFile("testdata/flow_logs_aggr_response_2.json")
			Expect(err).To(Not(HaveOccurred()))
			validPreview, err := ioutil.ReadFile("testdata/flow_logs_valid_preview.json")
			Expect(err).To(Not(HaveOccurred()))
			aggResponse, err := ioutil.ReadFile("testdata/flow_logs_pip_1_aggregation_rbac.json")
			Expect(err).To(Not(HaveOccurred()))
			previews, err := getPolicyPreviews([]string{string(validPreview)})
			Expect(err).To(Not(HaveOccurred()))

			listSrc := newMockLister()
			esClient = lmaelastic.NewMockSearchClient([]interface{}{string(esResponse)})
			pipClient := pip.New(pipcfg.MustLoadConfig(), listSrc, esClient)
			params := &FlowLogsParams{
				PolicyPreviews: previews,
				Limit:          1,
			}

			mockFlowHelper := new(rbac.MockFlowHelper)

			// Allow all except HEP and GNPs.  The first result will be excluded.  The second result will have the GNP obfuscated.
			mockFlowHelper.On("CanListPolicy", mock.Anything).Return(false, nil)
			mockFlowHelper.On("CanListEndpoint", api.EndpointTypeHep, api.GlobalEndpointType).Return(false, nil)
			mockFlowHelper.On("CanListEndpoint", api.EndpointTypeNet, api.GlobalEndpointType).Return(false, nil)
			mockFlowHelper.On("CanListEndpoint", api.EndpointTypeWep, mock.Anything).Return(true, nil)

			flowFilter := lmaelastic.NewFlowFilterUserRBAC(mockFlowHelper)
			searchResults, stat, err := getPIPFlowLogsFromElastic(flowFilter, params, pipClient, rbacHelper)

			mockFlowHelper.AssertExpectations(GinkgoT())

			Expect(stat).To(Equal(http.StatusOK))
			Expect(err).To(Not(HaveOccurred()))

			// Check the results.
			Expect(searchResults).To(BeAssignableToTypeOf(&pip.FlowLogResults{}))
			convertedResults := searchResults.(*pip.FlowLogResults)
			// the took field won't always match the expected response since it is timer based so overwrite it here
			convertedResults.Took = 3
			searchData, err := json.Marshal(convertedResults)
			Expect(err).To(Not(HaveOccurred()))
			Expect(searchData).To(MatchJSON(aggResponse))
		})
	})
})

func newTestRequest(method string) (*http.Request, error) {
	req, err := http.NewRequest(method, "", nil)
	return req, err
}

func getTestStartAndEndTime() (string, string) {
	return startTimeTest, endTimeTest
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

// Fake rbacHelper that satisfies the interface
type testHelper struct {
	action string
	name   string
}

func (t *testHelper) CheckCanPreviewPolicyAction(action string, policy resources.Resource) (bool, error) {
	Expect(t.action).To(Equal(action))
	Expect(t.name).To(Equal(policy.GetObjectMeta().GetName()))
	return true, nil
}

// mockList is used by both mockSource and mockDestination.
type mockLister struct {
	data          []*list.TimestampedResourceList
	RetrieveCalls int
}

// Initialize is used by the test to fill the lister with a list for each resource type
//   Useful for replayer.
func newMockLister() *mockLister {
	m := &mockLister{}
	for _, rh := range resources.GetAllResourceHelpers() {
		resList := rh.NewResourceList()
		tm := rh.TypeMeta()
		resList.GetObjectKind().SetGroupVersionKind((&tm).GroupVersionKind())
		m.data = append(m.data, &list.TimestampedResourceList{
			ResourceList:              resList,
			RequestStartedTimestamp:   metav1.Time{Time: time.Now()},
			RequestCompletedTimestamp: metav1.Time{Time: time.Now()}})
	}
	return m
}

// mockLister implements the ClusterAwareLister interface.
func (m *mockLister) RetrieveList(cluster string, tm metav1.TypeMeta) (*list.TimestampedResourceList, error) {
	listToReturn := (*list.TimestampedResourceList)(nil)
	for i := 0; i < len(m.data); i++ {
		resList := m.data[i]
		typeMetaMatches := resList.GetObjectKind().GroupVersionKind() == tm.GroupVersionKind()
		if typeMetaMatches {
			listToReturn = resList
		}
	}
	if listToReturn == nil {
		return nil, errors.ErrorResourceDoesNotExist{}
	}
	return listToReturn, nil
}
