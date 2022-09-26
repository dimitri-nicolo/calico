// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package application

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sort"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/mock"

	v1 "github.com/projectcalico/calico/es-proxy/pkg/apis/v1"
	"github.com/projectcalico/calico/es-proxy/test/thirdpartymock"
	lmaindex "github.com/projectcalico/calico/lma/pkg/elastic/index"

	libcalicov3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

var (
	// requests from manager to es-proxy
	//go:embed testdata/application_request_from_manager.json
	ApplicationRequest string
	//go:embed testdata/application_request_with_selector_from_manager.json
	ApplicationRequestWithSelector string

	// requests from es-proxy to elastic
	//go:embed testdata/l7_search_request.json
	l7SearchRequest string
	//go:embed testdata/l7_search_request_with_namespace.json
	l7SearchRequestWithNamespace string

	// responses from elastic to es-proxy
	//go:embed testdata/l7_search_response.json
	l7SearchResponse string
	//go:embed testdata/l7_search_response_with_namespace.json
	l7SearchResponseWithNamespace string
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

var _ = Describe("Application middleware tests", func() {
	var (
		mockDoer       *thirdpartymock.MockDoer
		userAuthReview userAuthorizationReviewMock
	)

	BeforeEach(func() {
		mockDoer = new(thirdpartymock.MockDoer)
		userAuthReview = userAuthorizationReviewMock{
			verbs: []libcalicov3.AuthorizedResourceVerbs{
				{
					APIGroup: "api-group-1",
					Resource: "pods",
					Verbs: []libcalicov3.AuthorizedResourceVerb{
						{
							Verb: "list",
							ResourceGroups: []libcalicov3.AuthorizedResourceGroup{
								{Tier: "tier-1"},
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

	Context("Elasticsearch /services request and response validation", func() {
		It("should return a valid services response", func() {
			// mock http client
			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
				defer GinkgoRecover()
				req := args.Get(0).(*http.Request)

				// Elastic _search request
				Expect(req.Method).To(Equal(http.MethodPost))
				Expect(req.URL.Path).To(Equal("/tigera_secure_ee_l7.test-cluster-name.*/_search"))

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(body).To(Equal([]byte(l7SearchRequest)))

				Expect(req.Body.Close()).NotTo(HaveOccurred())
				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			}).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBuffer([]byte(l7SearchResponse))),
			}, nil)

			// mock es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())

			// validate responses
			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(ApplicationRequest)))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			idxHelper := lmaindex.L7Logs()
			handler := ApplicationHandler(idxHelper, userAuthReview, client, ApplicationTypeService)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))

			var resp v1.ServiceResponse
			err = json.Unmarshal(rr.Body.Bytes(), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Services).To(HaveLen(3))

			// sort services slice as the order isn't guaranteed when translated from map.
			sort.Slice(resp.Services, func(i, j int) bool {
				return resp.Services[i].Name < resp.Services[j].Name
			})

			Expect(resp.Services[0].Name).To(Equal("checkoutservice-69c8ff664b-*"))
			Expect(resp.Services[0].ErrorRate).To(Equal(0.0))
			Expect(resp.Services[0].Latency).To(BeNumerically("~", 3449.98, 0.01))
			Expect(resp.Services[0].InboundThroughput).To(BeNumerically("~", 14990.34, 0.01))
			Expect(resp.Services[0].OutboundThroughput).To(BeNumerically("~", 12463.77, 0.01))
			Expect(resp.Services[0].RequestThroughput).To(BeNumerically("~", 0.095, 0.001))

			Expect(resp.Services[1].Name).To(Equal("frontend-99684f7f8-*"))
			Expect(resp.Services[1].ErrorRate).To(BeNumerically("~", 5.577, 0.001))
			Expect(resp.Services[1].Latency).To(BeNumerically("~", 5338.46, 0.01))
			Expect(resp.Services[1].InboundThroughput).To(BeNumerically("~", 4251.80, 0.01))
			Expect(resp.Services[1].OutboundThroughput).To(BeNumerically("~", 19909.94, 0.01))
			Expect(resp.Services[1].RequestThroughput).To(BeNumerically("~", 0.804, 0.001))

			Expect(resp.Services[2].Name).To(Equal("pvt"))
			Expect(resp.Services[2].ErrorRate).To(Equal(0.0))
			Expect(resp.Services[2].Latency).To(BeNumerically("~", 1461.38, 0.01))
			Expect(resp.Services[2].InboundThroughput).To(BeNumerically("~", 9862.310, 0.01))
			Expect(resp.Services[2].OutboundThroughput).To(BeNumerically("~", 67543.816, 0.01))
			Expect(resp.Services[2].RequestThroughput).To(BeNumerically("~", 1.572, 0.001))
		})

		It("should return a valid services response filtered by namespace", func() {
			// mock http client
			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
				defer GinkgoRecover()
				req := args.Get(0).(*http.Request)

				// Elastic _search request
				Expect(req.Method).To(Equal(http.MethodPost))
				Expect(req.URL.Path).To(Equal("/tigera_secure_ee_l7.test-cluster-name.*/_search"))

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(body).To(Equal([]byte(l7SearchRequestWithNamespace)))

				Expect(req.Body.Close()).NotTo(HaveOccurred())
				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			}).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBuffer([]byte(l7SearchResponseWithNamespace))),
			}, nil)

			// mock es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())

			// validate responses
			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(ApplicationRequestWithSelector)))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			idxHelper := lmaindex.L7Logs()
			handler := ApplicationHandler(idxHelper, userAuthReview, client, ApplicationTypeService)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))

			var resp v1.ServiceResponse
			err = json.Unmarshal(rr.Body.Bytes(), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Services).To(HaveLen(2))

			// sort services slice as the order isn't guaranteed when translated from map.
			sort.Slice(resp.Services, func(i, j int) bool {
				return resp.Services[i].Name < resp.Services[j].Name
			})

			Expect(resp.Services[0].Name).To(Equal("checkoutservice-69c8ff664b-*"))
			Expect(resp.Services[0].ErrorRate).To(Equal(0.0))
			Expect(resp.Services[0].Latency).To(BeNumerically("~", 3193.52, 0.01))
			Expect(resp.Services[0].InboundThroughput).To(BeNumerically("~", 16202.02, 0.01))
			Expect(resp.Services[0].OutboundThroughput).To(BeNumerically("~", 13464.65, 0.01))
			Expect(resp.Services[0].RequestThroughput).To(BeNumerically("~", 0.102, 0.001))

			Expect(resp.Services[1].Name).To(Equal("frontend-99684f7f8-*"))
			Expect(resp.Services[1].ErrorRate).To(BeNumerically("~", 100, 0.001))
			Expect(resp.Services[1].Latency).To(BeNumerically("~", 72103.41, 0.01))
			Expect(resp.Services[1].InboundThroughput).To(BeNumerically("~", 2316.12, 0.01))
			Expect(resp.Services[1].OutboundThroughput).To(BeNumerically("~", 3217.60, 0.01))
			Expect(resp.Services[1].RequestThroughput).To(BeNumerically("~", 0.090, 0.001))
		})

		It("should return error when request is not POST", func() {
			req, err := http.NewRequest(http.MethodGet, "", bytes.NewReader([]byte("any")))
			Expect(err).NotTo(HaveOccurred())

			// mock es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			idxHelper := lmaindex.L7Logs()
			handler := ApplicationHandler(idxHelper, userAuthReview, client, ApplicationTypeService)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusMethodNotAllowed))
		})

		It("should return error when request body is not valid", func() {
			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte("invalid-json-body")))
			Expect(err).NotTo(HaveOccurred())

			// mock es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			idxHelper := lmaindex.L7Logs()
			handler := ApplicationHandler(idxHelper, userAuthReview, client, ApplicationTypeService)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("should return error when response from elastic is not valid", func() {
			// mock http client
			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
				defer GinkgoRecover()
				req := args.Get(0).(*http.Request)

				// Elastic _search request
				Expect(req.Method).To(Equal(http.MethodPost))
				Expect(req.URL.Path).To(Equal("/tigera_secure_ee_l7.test-cluster-name.*/_search"))

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(body).To(Equal([]byte(l7SearchRequestWithNamespace)))

				Expect(req.Body.Close()).NotTo(HaveOccurred())
				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			}).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBuffer([]byte("invalid-elastic-response"))),
			}, nil)

			// mock es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())

			// validate responses
			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(ApplicationRequestWithSelector)))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			idxHelper := lmaindex.L7Logs()
			handler := ApplicationHandler(idxHelper, userAuthReview, client, ApplicationTypeService)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})

		It("should return error when index helper failed to create a new selector query", func() {
			// mock index helper
			mockIdxHelper := lmaindex.MockHelper{}
			mockIdxHelper.On("GetIndex", mock.Anything).Return("any-index")
			mockIdxHelper.On("NewSelectorQuery", mock.Anything).Return(nil, fmt.Errorf("NewSelectorQuery failed"))

			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(ApplicationRequestWithSelector)))
			Expect(err).NotTo(HaveOccurred())

			// mock es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := ApplicationHandler(&mockIdxHelper, userAuthReview, client, ApplicationTypeService)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})

		It("should return error when index helper failed to create a new RBAC query", func() {
			// mock index helper
			mockIdxHelper := lmaindex.MockHelper{}
			now := time.Now()
			mockIdxHelper.On("GetIndex", mock.Anything).Return("any-index")
			mockIdxHelper.On("NewTimeRangeQuery", mock.Anything, mock.Anything).
				Return(elastic.NewRangeQuery("any-time-field").Gt(now.Unix()).Lte(now.Unix()), nil)
			mockIdxHelper.On("NewRBACQuery", mock.Anything).Return(nil, fmt.Errorf("NewRBACQuery failed"))

			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(ApplicationRequest)))
			Expect(err).NotTo(HaveOccurred())

			// mock es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := ApplicationHandler(&mockIdxHelper, userAuthReview, client, ApplicationTypeService)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})

		It("should return error when failed to perform AuthorizationReview", func() {
			// mock index helper
			mockIdxHelper := lmaindex.MockHelper{}
			now := time.Now()
			mockIdxHelper.On("GetIndex", mock.Anything).Return("any-index")
			mockIdxHelper.On("NewTimeRangeQuery", mock.Anything, mock.Anything).
				Return(elastic.NewRangeQuery("any-time-field").Gt(now.Unix()).Lte(now.Unix()), nil)

			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(ApplicationRequest)))
			Expect(err).NotTo(HaveOccurred())

			// mock es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())

			// mock auth review returns error
			mockUserAuthReviewFailed := userAuthorizationReviewMock{
				verbs: []libcalicov3.AuthorizedResourceVerbs{},
				err:   fmt.Errorf("PerformReviewForElasticLogs failed"),
			}

			rr := httptest.NewRecorder()
			handler := ApplicationHandler(&mockIdxHelper, mockUserAuthReviewFailed, client, ApplicationTypeService)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Context("Elasticsearch /urls request and response validation", func() {
		It("should return a valid urls response", func() {
			// mock http client
			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
				defer GinkgoRecover()
				req := args.Get(0).(*http.Request)

				// Elastic _search request
				Expect(req.Method).To(Equal(http.MethodPost))
				Expect(req.URL.Path).To(Equal("/tigera_secure_ee_l7.test-cluster-name.*/_search"))

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(body).To(Equal([]byte(l7SearchRequest)))

				Expect(req.Body.Close()).NotTo(HaveOccurred())
				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			}).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBuffer([]byte(l7SearchResponse))),
			}, nil)

			// mock es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())

			// validate responses
			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(ApplicationRequest)))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			idxHelper := lmaindex.L7Logs()
			handler := ApplicationHandler(idxHelper, userAuthReview, client, ApplicationTypeURL)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))

			var resp v1.URLResponse
			err = json.Unmarshal(rr.Body.Bytes(), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.URLs).To(HaveLen(4))

			// sort urls slice as the order isn't guaranteed when translated from map.
			sort.Slice(resp.URLs, func(i, j int) bool {
				li := resp.URLs[i].URL
				si := resp.URLs[i].Service
				lj := resp.URLs[j].URL
				sj := resp.URLs[j].Service
				return li+si < lj+sj
			})

			Expect(resp.URLs[0].URL).To(Equal("adservice:9555/hipstershop.AdService/GetAds"))
			Expect(resp.URLs[0].Service).To(Equal("frontend-99684f7f8-*"))
			Expect(resp.URLs[0].RequestCount).To(Equal(491))

			Expect(resp.URLs[1].URL).To(Equal("adservice:9555/hipstershop.AdService/GetAds"))
			Expect(resp.URLs[1].Service).To(Equal("pvt"))
			Expect(resp.URLs[1].RequestCount).To(Equal(492))

			Expect(resp.URLs[2].URL).To(Equal("checkoutservice:5050/hipstershop.CheckoutService/PlaceOrder"))
			Expect(resp.URLs[2].Service).To(Equal("frontend-99684f7f8-*"))
			Expect(resp.URLs[2].RequestCount).To(Equal(29))

			Expect(resp.URLs[3].URL).To(Equal("paymentservice:50051/hipstershop.PaymentService/Charge"))
			Expect(resp.URLs[3].Service).To(Equal("checkoutservice-69c8ff664b-*"))
			Expect(resp.URLs[3].RequestCount).To(Equal(60)) // 31+29
		})

		It("should return a valid services response filtered by namespace", func() {
			// mock http client
			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
				defer GinkgoRecover()
				req := args.Get(0).(*http.Request)

				// Elastic _search request
				Expect(req.Method).To(Equal(http.MethodPost))
				Expect(req.URL.Path).To(Equal("/tigera_secure_ee_l7.test-cluster-name.*/_search"))

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(body).To(Equal([]byte(l7SearchRequestWithNamespace)))

				Expect(req.Body.Close()).NotTo(HaveOccurred())
				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			}).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBuffer([]byte(l7SearchResponseWithNamespace))),
			}, nil)

			// mock es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())

			// validate responses
			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(ApplicationRequestWithSelector)))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			idxHelper := lmaindex.L7Logs()
			handler := ApplicationHandler(idxHelper, userAuthReview, client, ApplicationTypeURL)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))

			var resp v1.URLResponse
			err = json.Unmarshal(rr.Body.Bytes(), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.URLs).To(HaveLen(2))

			// sort urls slice as the order isn't guaranteed when translated from map.
			sort.Slice(resp.URLs, func(i, j int) bool {
				li := resp.URLs[i].URL
				si := resp.URLs[i].Service
				lj := resp.URLs[j].URL
				sj := resp.URLs[j].Service
				return li+si < lj+sj
			})

			Expect(resp.URLs[0].URL).To(Equal("checkoutservice:5050/hipstershop.CheckoutService/PlaceOrder"))
			Expect(resp.URLs[0].Service).To(Equal("frontend-99684f7f8-*"))
			Expect(resp.URLs[0].RequestCount).To(Equal(29))

			Expect(resp.URLs[1].URL).To(Equal("paymentservice:50051/hipstershop.PaymentService/Charge"))
			Expect(resp.URLs[1].Service).To(Equal("checkoutservice-69c8ff664b-*"))
			Expect(resp.URLs[1].RequestCount).To(Equal(31))
		})

		It("should return error when request is not POST", func() {
			req, err := http.NewRequest(http.MethodGet, "", bytes.NewReader([]byte("any")))
			Expect(err).NotTo(HaveOccurred())

			// mock es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			idxHelper := lmaindex.L7Logs()
			handler := ApplicationHandler(idxHelper, userAuthReview, client, ApplicationTypeURL)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusMethodNotAllowed))
		})

		It("should return error when request body is not valid", func() {
			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte("invalid-json-body")))
			Expect(err).NotTo(HaveOccurred())

			// mock es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			idxHelper := lmaindex.L7Logs()
			handler := ApplicationHandler(idxHelper, userAuthReview, client, ApplicationTypeURL)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("should return error when response from elastic is not valid", func() {
			// mock http client
			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
				defer GinkgoRecover()
				req := args.Get(0).(*http.Request)

				// Elastic _search request
				Expect(req.Method).To(Equal(http.MethodPost))
				Expect(req.URL.Path).To(Equal("/tigera_secure_ee_l7.test-cluster-name.*/_search"))

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(body).To(Equal([]byte(l7SearchRequestWithNamespace)))

				Expect(req.Body.Close()).NotTo(HaveOccurred())
				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			}).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBuffer([]byte("invalid-elastic-response"))),
			}, nil)

			// mock es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())

			// validate responses
			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(ApplicationRequestWithSelector)))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			idxHelper := lmaindex.L7Logs()
			handler := ApplicationHandler(idxHelper, userAuthReview, client, ApplicationTypeURL)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})

		It("should return error when index helper failed to create a new selector query", func() {
			// mock index helper
			mockIdxHelper := lmaindex.MockHelper{}
			mockIdxHelper.On("GetIndex", mock.Anything).Return("any-index")
			mockIdxHelper.On("NewSelectorQuery", mock.Anything).Return(nil, fmt.Errorf("NewSelectorQuery failed"))

			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(ApplicationRequestWithSelector)))
			Expect(err).NotTo(HaveOccurred())

			// mock es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := ApplicationHandler(&mockIdxHelper, userAuthReview, client, ApplicationTypeURL)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})

		It("should return error when index helper failed to create a new RBAC query", func() {
			// mock index helper
			mockIdxHelper := lmaindex.MockHelper{}
			now := time.Now()
			mockIdxHelper.On("GetIndex", mock.Anything).Return("any-index")
			mockIdxHelper.On("NewTimeRangeQuery", mock.Anything, mock.Anything).
				Return(elastic.NewRangeQuery("any-time-field").Gt(now.Unix()).Lte(now.Unix()), nil)
			mockIdxHelper.On("NewRBACQuery", mock.Anything).Return(nil, fmt.Errorf("NewRBACQuery failed"))

			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(ApplicationRequest)))
			Expect(err).NotTo(HaveOccurred())

			// mock es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := ApplicationHandler(&mockIdxHelper, userAuthReview, client, ApplicationTypeURL)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})

		It("should return error when failed to perform AuthorizationReview", func() {
			// mock index helper
			mockIdxHelper := lmaindex.MockHelper{}
			now := time.Now()
			mockIdxHelper.On("GetIndex", mock.Anything).Return("any-index")
			mockIdxHelper.On("NewTimeRangeQuery", mock.Anything, mock.Anything).
				Return(elastic.NewRangeQuery("any-time-field").Gt(now.Unix()).Lte(now.Unix()), nil)

			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(ApplicationRequest)))
			Expect(err).NotTo(HaveOccurred())

			// mock es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())

			// mock auth review returns error
			mockUserAuthReviewFailed := userAuthorizationReviewMock{
				verbs: []libcalicov3.AuthorizedResourceVerbs{},
				err:   fmt.Errorf("PerformReviewForElasticLogs failed"),
			}

			rr := httptest.NewRecorder()
			handler := ApplicationHandler(&mockIdxHelper, mockUserAuthReviewFailed, client, ApplicationTypeURL)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})
	})
})
