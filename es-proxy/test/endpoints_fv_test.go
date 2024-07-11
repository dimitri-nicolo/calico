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

	"github.com/projectcalico/calico/es-proxy/pkg/middleware"
	lapi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	lsclient "github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
	"github.com/projectcalico/calico/lma/pkg/httputils"
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

var _ = Describe("Test EndpointsAggregation handler", func() {
	var (
		server       *httptest.Server
		qsconfig     *client.QueryServerConfig
		req          *http.Request
		mocklsclient lsclient.MockClient

		tokenFilePath = "token"
		CAFilePath    = "ca"
	)

	BeforeEach(func() {
		// initiliaze queryserver config
		qsconfig = &client.QueryServerConfig{
			QueryServerTunnelURL: "",
			QueryServerURL:       "",
			QueryServerCA:        CAFilePath,
			QueryServerToken:     tokenFilePath,
		}

		// Create mock client certificate and auth token
		CA_file, err := os.Create(CAFilePath)
		Expect(err).ShouldNot(HaveOccurred())
		defer CA_file.Close()

		token_file, err := os.Create(tokenFilePath)
		Expect(err).ShouldNot(HaveOccurred())
		defer token_file.Close()
	})

	AfterEach(func() {
		// Delete mock client certificate and auth token files
		Expect(os.Remove(CAFilePath)).Error().ShouldNot(HaveOccurred())

		Expect(os.Remove(tokenFilePath)).Error().ShouldNot(HaveOccurred())
	})

	Context("when there are denied flowlogs", func() {
		var authReview userAuthorizationReviewMock
		BeforeEach(func() {
			// prepare mock authreview
			authReview = userAuthorizationReviewMock{
				verbs: []v3.AuthorizedResourceVerbs{},
				err:   nil,
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
			deniedEndPointsResponse := querycacheclient.QueryEndpointsResp{
				Count: 2,
				Items: []querycacheclient.Endpoint{
					{
						Namespace: "ns-src",
						Pod:       "ep-src-1234",
					},
					{
						Namespace: "ns-dst",
						Pod:       "ep-dst-1234",
					},
				},
			}
			server = createFakeQueryServer(&deniedEndPointsResponse, func(requestBody *querycacheclient.QueryEndpointsReqBody) {
				// If showDeniedEndpointsOnly is true, the endpoints aggregation handler will generate
				// an endpoint list based on the result of the linseedResults.
				Expect(requestBody.EndpointsList).Should(ConsistOf([]string{
					".*ns-src/.*-ep--src--*",
					".*ns-dst/.*-ep--dst--*",
				}))
			})
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
			Expect(err).ShouldNot(HaveOccurred())

			// prepare response recorder
			rr := httptest.NewRecorder()

			By("calling EndpointsAggregationHandler")
			handler := middleware.EndpointsAggregationHandler(authReview, qsconfig, mocklsclient)
			handler.ServeHTTP(rr, req)

			By("validating server response")
			Expect(rr.Code).To(Equal(http.StatusOK))

			response := &middleware.EndpointsAggregationResponse{}
			err = json.Unmarshal(rr.Body.Bytes(), response)
			Expect(err).ShouldNot(HaveOccurred())

			Expect(response.Count).To(Equal(2))
			for _, item := range response.Item {
				Expect(item.HasDeniedTraffic).To(BeTrue())
			}
		})

		It("return all endpoints", func() {
			By("preparing the server")
			allEndPointsResponse := querycacheclient.QueryEndpointsResp{
				Count: 3,
				Items: []querycacheclient.Endpoint{
					{
						Namespace: "ns-src",
						Pod:       "ep-src-1234",
					},
					{
						Namespace: "ns-dst",
						Pod:       "ep-dst-1234",
					},
					{
						Namespace: "ns-allow",
						Pod:       "ep-allow-1234",
					},
				},
			}
			server = createFakeQueryServer(&allEndPointsResponse, func(requestBody *querycacheclient.QueryEndpointsReqBody) {
				// If showDeniedEndpointsOnly is false, the endpoints aggregation handler will NOT generate
				// an endpoint list and will return all endpoints as a result.
				Expect(requestBody.EndpointsList).Should(ConsistOf([]string{}))
			})
			defer server.Close()

			// update queryserver url
			qsconfig.QueryServerURL = server.URL

			// prepare request
			endpointReq := &middleware.EndpointsAggregationRequest{
				ShowDeniedEndpoints: false,
			}

			reqBodyBytes, err := json.Marshal(endpointReq)
			Expect(err).ShouldNot(HaveOccurred())

			req, err = http.NewRequest("POST", server.URL, bytes.NewBuffer(reqBodyBytes))
			Expect(err).ShouldNot(HaveOccurred())

			// prepare response recorder
			rr := httptest.NewRecorder()

			By("calling EndpointsAggregationHandler")
			handler := middleware.EndpointsAggregationHandler(authReview, qsconfig, mocklsclient)
			handler.ServeHTTP(rr, req)

			By("validating server response")
			Expect(rr.Code).To(Equal(http.StatusOK))

			response := &middleware.EndpointsAggregationResponse{}
			err = json.Unmarshal(rr.Body.Bytes(), response)
			Expect(err).ShouldNot(HaveOccurred())

			Expect(response.Count).To(Equal(3))
			Expect(response.Item).To(HaveLen(3))
			for _, item := range response.Item {
				if item.Namespace == "ns-allow" {
					Expect(item.HasDeniedTraffic).To(BeFalse())
				} else {
					Expect(item.HasDeniedTraffic).To(BeTrue())
				}
			}
		})
	})
})

// createFakeQueryServer sets up a fake Query Server instance for tests.
func createFakeQueryServer(response *querycacheclient.QueryEndpointsResp, test func(requestBody *querycacheclient.QueryEndpointsReqBody)) *httptest.Server {

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

		// Make sure we get a valid request
		var requestBody querycacheclient.QueryEndpointsReqBody
		err := httputils.Decode(w, r, &requestBody)
		Expect(err).ShouldNot(HaveOccurred())

		// Run any extra test supplied as a parameter
		if test != nil {
			test(&requestBody)
		}

		bytes, err := json.Marshal(response)
		Expect(err).ShouldNot(HaveOccurred())

		_, err = w.Write(bytes)
		Expect(err).ShouldNot(HaveOccurred())
	}))

}
