// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package process

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
	//go:embed testdata/process_request_from_manager.json
	processRequest string

	// requests from es-proxy to elastic
	//go:embed testdata/flow_search_request.json
	flowSearchRequest string

	// responses from elastic to es-proxy
	//go:embed testdata/flow_search_response.json
	flowSearchResponse string
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
				Expect(req.URL.Path).To(Equal("/tigera_secure_ee_flows.test-cluster-name.*/_search"))

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(body).To(Equal([]byte(flowSearchRequest)))

				Expect(req.Body.Close()).NotTo(HaveOccurred())
				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			}).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBuffer([]byte(flowSearchResponse))),
			}, nil)

			// mock es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())

			// validate responses
			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(processRequest)))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			idxHelper := lmaindex.FlowLogs()
			handler := ProcessHandler(idxHelper, userAuthReview, client)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))

			var resp v1.ProcessResponse
			err = json.Unmarshal(rr.Body.Bytes(), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Processes).To(HaveLen(9))

			// sort process slice as the order isn't guaranteed when translated from map.
			sort.Slice(resp.Processes, func(i, j int) bool {
				return resp.Processes[i].Name < resp.Processes[j].Name
			})

			Expect(resp.Processes[0].Name).To(Equal("/app/cartservice"))
			Expect(resp.Processes[0].Endpoint).To(Equal("cartservice-74f56fd4b-*"))
			Expect(resp.Processes[0].InstanceCount).To(Equal(3))
			Expect(resp.Processes[1].Name).To(Equal("/src/checkoutservice"))
			Expect(resp.Processes[1].Endpoint).To(Equal("checkoutservice-69c8ff664b-*"))
			Expect(resp.Processes[1].InstanceCount).To(Equal(4))
			Expect(resp.Processes[2].Name).To(Equal("/src/server"))
			Expect(resp.Processes[2].Endpoint).To(Equal("frontend-99684f7f8-*"))
			Expect(resp.Processes[2].InstanceCount).To(Equal(3))
			Expect(resp.Processes[3].Name).To(Equal("/usr/local/bin/locust"))
			Expect(resp.Processes[3].Endpoint).To(Equal("loadgenerator-555fbdc87d-*"))
			Expect(resp.Processes[3].InstanceCount).To(Equal(1))
			Expect(resp.Processes[4].Name).To(Equal("/usr/local/bin/python"))
			Expect(resp.Processes[4].Endpoint).To(Equal("loadgenerator-555fbdc87d-*"))
			Expect(resp.Processes[4].InstanceCount).To(Equal(2))
			Expect(resp.Processes[5].Name).To(Equal("/usr/local/bin/python"))
			Expect(resp.Processes[5].Endpoint).To(Equal("recommendationservice-5f8c456796-*"))
			Expect(resp.Processes[5].InstanceCount).To(Equal(2))
			Expect(resp.Processes[6].Name).To(Equal("/usr/local/openjdk-8/bin/java"))
			Expect(resp.Processes[6].Endpoint).To(Equal("adservice-77d5cd745d-*"))
			Expect(resp.Processes[6].InstanceCount).To(Equal(3))
			Expect(resp.Processes[7].Name).To(Equal("python"))
			Expect(resp.Processes[7].Endpoint).To(Equal("recommendationservice-5f8c456796-*"))
			Expect(resp.Processes[7].InstanceCount).To(Equal(2))
			Expect(resp.Processes[8].Name).To(Equal("wget"))
			Expect(resp.Processes[8].Endpoint).To(Equal("loadgenerator-555fbdc87d-*"))
			Expect(resp.Processes[8].InstanceCount).To(Equal(1))
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
			idxHelper := lmaindex.FlowLogs()
			handler := ProcessHandler(idxHelper, userAuthReview, client)
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
			idxHelper := lmaindex.FlowLogs()
			handler := ProcessHandler(idxHelper, userAuthReview, client)
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
				Expect(req.URL.Path).To(Equal("/tigera_secure_ee_flows.test-cluster-name.*/_search"))

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(body).To(Equal([]byte(flowSearchRequest)))

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
			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(processRequest)))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			idxHelper := lmaindex.FlowLogs()
			handler := ProcessHandler(idxHelper, userAuthReview, client)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})

		It("should return error when index helper failed to create a new selector query", func() {
			// mock index helper
			mockIdxHelper := lmaindex.MockHelper{}
			mockIdxHelper.On("GetIndex", mock.Anything).Return("any-index")
			mockIdxHelper.On("NewSelectorQuery", mock.Anything).Return(nil, fmt.Errorf("NewSelectorQuery failed"))

			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(processRequest)))
			Expect(err).NotTo(HaveOccurred())

			// mock es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := ProcessHandler(&mockIdxHelper, userAuthReview, client)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})

		It("should return error when index helper failed to create a new RBAC query", func() {
			// mock index helper
			mockIdxHelper := lmaindex.MockHelper{}
			now := time.Now()
			mockIdxHelper.On("GetIndex", mock.Anything).Return("any-index")
			mockIdxHelper.On("NewSelectorQuery", mock.Anything).Return(elastic.NewBoolQuery(), nil)
			mockIdxHelper.On("NewTimeRangeQuery", mock.Anything, mock.Anything).
				Return(elastic.NewRangeQuery("any-time-field").Gt(now.Unix()).Lte(now.Unix()), nil)
			mockIdxHelper.On("NewRBACQuery", mock.Anything).Return(nil, fmt.Errorf("NewRBACQuery failed"))

			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(processRequest)))
			Expect(err).NotTo(HaveOccurred())

			// mock es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := ProcessHandler(&mockIdxHelper, userAuthReview, client)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})

		It("should return error when failed to perform AuthorizationReview", func() {
			// mock index helper
			mockIdxHelper := lmaindex.MockHelper{}
			now := time.Now()
			mockIdxHelper.On("GetIndex", mock.Anything).Return("any-index")
			mockIdxHelper.On("NewSelectorQuery", mock.Anything).Return(elastic.NewBoolQuery(), nil)
			mockIdxHelper.On("NewTimeRangeQuery", mock.Anything, mock.Anything).
				Return(elastic.NewRangeQuery("any-time-field").Gt(now.Unix()).Lte(now.Unix()), nil)

			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(processRequest)))
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
			handler := ProcessHandler(&mockIdxHelper, mockUserAuthReviewFailed, client)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})
	})
})
