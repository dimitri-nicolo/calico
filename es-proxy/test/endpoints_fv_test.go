// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package fv_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	authzv1 "k8s.io/api/authorization/v1"

	"k8s.io/apiserver/pkg/authentication/user"

	"github.com/projectcalico/calico/es-proxy/pkg/middleware"
	lapi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	lsclient "github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
	querycacheclient "github.com/projectcalico/calico/ts-queryserver/pkg/querycache/client"
	"github.com/projectcalico/calico/ts-queryserver/queryserver/client"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

// The user authentication review mock struct implementing the authentication review interface.
type userAuthorizationReviewMock struct {
	verbs []v3.AuthorizedResourceVerbs
	err   error
}

// PerformReviewForElasticLogs wraps a mocked version of the authorization review method
// PerformReviewForElasticLogs.
func (a userAuthorizationReviewMock) PerformReview(
	ctx context.Context, cluster string,
) ([]v3.AuthorizedResourceVerbs, error) {
	return a.verbs, a.err
}

type mockAuthorizationReview struct {
	isAuthorized bool
	err          error
}

func (az mockAuthorizationReview) Authorize(user.Info, *authzv1.ResourceAttributes, *authzv1.NonResourceAttributes) (bool, error) {
	return az.isAuthorized, az.err
}

var _ = Describe("Test EndpointsAggregation handler", func() {
	var (
		server       *httptest.Server
		qsconfig     *client.QueryServerConfig
		req          *http.Request
		mocklsclient lsclient.MockClient
		CAFilePath   = "ca"

		authz mockAuthorizationReview
	)

	BeforeEach(func() {
		// initiliaze queryserver config
		qsconfig = &client.QueryServerConfig{
			QueryServerTunnelURL: "",
			QueryServerURL:       "",
			QueryServerCA:        CAFilePath,
		}

		// Create mock client certificate
		CA_file, err := os.Create(CAFilePath)
		Expect(err).ShouldNot(HaveOccurred())
		defer CA_file.Close()
	})

	AfterEach(func() {
		// Delete mock client certificate and auth token files
		Expect(os.Remove(CAFilePath)).Error().ShouldNot(HaveOccurred())
	})

	Context("when there are denied flowlogs", func() {
		var authReview userAuthorizationReviewMock
		BeforeEach(func() {
			// prepare mock authreview
			authReview = userAuthorizationReviewMock{
				verbs: []v3.AuthorizedResourceVerbs{
					{
						APIGroup: "lma.tigera.io",
						Resource: "flows",
						Verbs: []v3.AuthorizedResourceVerb{
							{
								Verb:           "get",
								ResourceGroups: nil,
							},
						},
					},
				},
				err: nil,
			}

			// prepare mock linseed client
			linseedResults := []rest.MockResult{
				{
					Body: lapi.List[lapi.FlowLog]{
						Items: []lapi.FlowLog{
							{
								SourceName:      "-",
								SourceNameAggr:  "ep-src-*",
								SourceNamespace: "ns-src",
								DestName:        "-",
								DestNameAggr:    "ep-dst-*",
								DestNamespace:   "ns-dst",
								Action:          "deny",
							},
						},
						AfterKey:  nil,
						TotalHits: 1,
					},
				},
			}
			mocklsclient = lsclient.NewMockClient("", linseedResults...)
		})

		It("return denied endpoints", func() {
			By("preparing the server")
			serverResponse := middleware.EndpointsAggregationResponse{
				Count: 2,
				Item: []middleware.AggregatedEndpoint{
					{
						Endpoint: querycacheclient.Endpoint{
							Namespace: "ns-src",
							Pod:       "ep-src-1234",
						},
					},
					{
						Endpoint: querycacheclient.Endpoint{
							Namespace: "ns-dst",
							Pod:       "ep-dst-1234",
						},
					},
				},
			}

			server = createServer(&serverResponse)
			defer server.Close()

			// update queryserver url
			qsconfig.QueryServerURL = server.URL

			// prepare request
			endpointReq := &middleware.EndpointsAggregationRequest{
				ShowDeniedEndpoints: true,
			}

			reqBodyBytes, err := json.Marshal(endpointReq)
			Expect(err).ShouldNot(HaveOccurred())

			req, err = http.NewRequest("POST", server.URL, bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Authorization", "Bearer tokentoken")
			Expect(err).ShouldNot(HaveOccurred())

			// prepare response recorder
			rr := httptest.NewRecorder()

			By("calling EndpointsAggregationHandler")

			// set mock authorizer
			authz.isAuthorized = true
			authz.err = nil

			handler := middleware.EndpointsAggregationHandler(authz, authReview, qsconfig, mocklsclient)
			handler.ServeHTTP(rr, req)

			By("validating server response")
			Expect(rr.Code).To(Equal(http.StatusOK))

			response := &middleware.EndpointsAggregationResponse{}
			err = json.Unmarshal(rr.Body.Bytes(), response)
			Expect(err).ShouldNot(HaveOccurred())

			Expect(response.Count).To(Equal(2))
			for _, item := range response.Item {
				Expect(item.HasFlowAccess).To(BeTrue())
				Expect(*item.HasDeniedTraffic).To(BeTrue())
			}
		})

		It("return all endpoints", func() {
			By("preparing the server")
			serverResponse := middleware.EndpointsAggregationResponse{
				Count: 3,
				Item: []middleware.AggregatedEndpoint{
					{
						Endpoint: querycacheclient.Endpoint{
							Namespace: "ns-src",
							Pod:       "ep-src-1234",
						},
					},
					{
						Endpoint: querycacheclient.Endpoint{
							Namespace: "ns-dst",
							Pod:       "ep-dst-1234",
						},
					},
					{
						Endpoint: querycacheclient.Endpoint{
							Namespace: "ns-allow",
							Pod:       "ep-allow-1234",
						},
					},
				},
			}

			server = createServer(&serverResponse)
			defer server.Close()

			// update queryserver url
			qsconfig.QueryServerURL = server.URL

			// prepare request
			endpointReq := &middleware.EndpointsAggregationRequest{
				ShowDeniedEndpoints: true,
			}

			reqBodyBytes, err := json.Marshal(endpointReq)
			Expect(err).ShouldNot(HaveOccurred())

			req, err = http.NewRequest("POST", server.URL, bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Authorization", "Bearer tokentoken")
			Expect(err).ShouldNot(HaveOccurred())

			// prepare response recorder
			rr := httptest.NewRecorder()

			By("calling EndpointsAggregationHandler")
			// set mock authz
			authz.isAuthorized = true
			authz.err = nil
			handler := middleware.EndpointsAggregationHandler(authz, authReview, qsconfig, mocklsclient)
			handler.ServeHTTP(rr, req)

			By("validating server response")
			Expect(rr.Code).To(Equal(http.StatusOK))

			response := &middleware.EndpointsAggregationResponse{}
			err = json.Unmarshal(rr.Body.Bytes(), response)
			Expect(err).ShouldNot(HaveOccurred())

			Expect(response.Count).To(Equal(3))
			for _, item := range response.Item {
				Expect(item.HasFlowAccess).To(BeTrue())
				if item.Namespace == "ns-allow" {
					Expect(*item.HasDeniedTraffic).To(BeFalse())
				} else {
					Expect(*item.HasDeniedTraffic).To(BeTrue())
				}
			}
		})
	})
})

func createServer(response *middleware.EndpointsAggregationResponse) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "" {
			w.WriteHeader(http.StatusForbidden)
		}
		if r.Header.Get("Accept") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
		}
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		w.WriteHeader(http.StatusOK)

		bytes, err := json.Marshal(response)
		Expect(err).ShouldNot(HaveOccurred())

		_, err = w.Write(bytes)
		Expect(err).ShouldNot(HaveOccurred())
	}))
}
