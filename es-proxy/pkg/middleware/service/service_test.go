// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package service

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sort"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/mock"

	v1 "github.com/projectcalico/calico/es-proxy/pkg/apis/v1"
	"github.com/projectcalico/calico/es-proxy/test/thirdpartymock"

	libcalicov3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

var (
	// requests from manager to es-proxy
	//go:embed testdata/service_request_from_manager.json
	serviceRequest string
	//go:embed testdata/service_request_with_selector_from_manager.json
	serviceRequestWithSelector string

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

var _ = Describe("Service middleware tests", func() {
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
			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(serviceRequest)))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := ServiceHandler(userAuthReview, client)
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
			Expect(resp.Services[0].Latency).To(Equal(0.0))
			Expect(resp.Services[0].InboundThroughput).To(BeNumerically("~", 14990.34, 0.01))
			Expect(resp.Services[0].OutboundThroughput).To(BeNumerically("~", 12463.77, 0.01))
			Expect(resp.Services[0].RequestThroughput).To(BeNumerically("~", 0.095, 0.001))

			Expect(resp.Services[1].Name).To(Equal("frontend-99684f7f8-*"))
			Expect(resp.Services[1].ErrorRate).To(BeNumerically("~", 5.577, 0.001))
			Expect(resp.Services[1].Latency).To(Equal(0.0))
			Expect(resp.Services[1].InboundThroughput).To(BeNumerically("~", 4251.80, 0.01))
			Expect(resp.Services[1].OutboundThroughput).To(BeNumerically("~", 19909.94, 0.01))
			Expect(resp.Services[1].RequestThroughput).To(BeNumerically("~", 0.804, 0.001))
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
			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(serviceRequestWithSelector)))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := ServiceHandler(userAuthReview, client)
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
			Expect(resp.Services[0].Latency).To(Equal(0.0))
			Expect(resp.Services[0].InboundThroughput).To(BeNumerically("~", 16202.02, 0.01))
			Expect(resp.Services[0].OutboundThroughput).To(BeNumerically("~", 13464.65, 0.01))
			Expect(resp.Services[0].RequestThroughput).To(BeNumerically("~", 0.102, 0.001))

			Expect(resp.Services[1].Name).To(Equal("frontend-99684f7f8-*"))
			Expect(resp.Services[1].ErrorRate).To(BeNumerically("~", 100, 0.001))
			Expect(resp.Services[1].Latency).To(Equal(0.0))
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
			handler := ServiceHandler(userAuthReview, client)
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
			handler := ServiceHandler(userAuthReview, client)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

	})
})
