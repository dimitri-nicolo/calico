// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package event

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/es-proxy/test/thirdpartymock"
	lapi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
)

var (
	// requests from manager to es-proxy
	eventStatisticsRequest string = `{
  "field_values": {
    "type": {"count": true}
  }
}`
	emptyEventStatisticsRequest   string = `{"field_values": {}}`
	invalidEventStatisticsRequest string = `{
		"field_values": {
		  "type": {}
		}
	  }`

	// responses from linseed to es-proxy
	eventStatisticsResponse string = `{
  "field_values": {
    "type": [
      {
        "value": "suspicious_dns_query",
        "count": 2
      },
      {
        "value": "TODO",
        "count": 1
      }
    ]
  }
}`
	emptyEventStatisticsResponse string = `{}`
)

var _ = Describe("EventStatistics middleware tests", func() {
	var mockDoer *thirdpartymock.MockDoer
	var lsclient client.MockClient

	BeforeEach(func() {
		mockDoer = new(thirdpartymock.MockDoer)

		lsclient = client.NewMockClient("")
	})

	AfterEach(func() {
		mockDoer.AssertExpectations(GinkgoT())
	})

	Context("Elasticsearch /events/statistics request and response validation", func() {
		It("should return a valid event statistics response", func() {
			// Set up a response from Linseed.
			var linseedResponse lapi.EventStatistics
			err := json.Unmarshal([]byte(eventStatisticsResponse), &linseedResponse)
			Expect(err).NotTo(HaveOccurred())
			res := rest.MockResult{Body: linseedResponse}
			lsclient.SetResults(res)

			// Setup request
			req, err := http.NewRequest(http.MethodPost, "", strings.NewReader(eventStatisticsRequest))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := EventStatisticsHandler(lsclient)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))

			// Check that response matches expectations
			var resp lapi.EventStatistics
			err = json.Unmarshal(rr.Body.Bytes(), &resp)
			Expect(err).NotTo(HaveOccurred())

			formattedJson, err := json.MarshalIndent(resp, "", "  ")
			Expect(err).NotTo(HaveOccurred())

			Expect(string(formattedJson)).To(Equal(eventStatisticsResponse))
		})

		It("should return an empty event statistics response for an empty request", func() {

			// Set up a response from Linseed.
			var linseedResponse lapi.EventStatistics
			err := json.Unmarshal([]byte(emptyEventStatisticsResponse), &linseedResponse)
			Expect(err).NotTo(HaveOccurred())
			res := rest.MockResult{Body: linseedResponse}
			lsclient.SetResults(res)

			// Setup request
			req, err := http.NewRequest(http.MethodPost, "", strings.NewReader(emptyEventStatisticsRequest))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := EventStatisticsHandler(lsclient)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))

			// Check that response matches expectations
			var resp lapi.EventStatistics
			err = json.Unmarshal(rr.Body.Bytes(), &resp)
			Expect(err).NotTo(HaveOccurred())

			formattedJson, err := json.MarshalIndent(resp, "", "  ")
			Expect(err).NotTo(HaveOccurred())

			Expect(string(formattedJson)).To(Equal(emptyEventStatisticsResponse))
		})

		It("should return an error for an invalid request", func() {

			// Set up a response from Linseed.
			res := rest.MockResult{StatusCode: http.StatusInternalServerError}
			lsclient.SetResults(res)

			// Setup request
			req, err := http.NewRequest(http.MethodPost, "", strings.NewReader(invalidEventStatisticsRequest))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := EventStatisticsHandler(lsclient)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})
	})
})
