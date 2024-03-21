// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package event

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/compliance/pkg/datastore"
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
	var (
		fakeClientSet datastore.ClientSet
		mockDoer      *thirdpartymock.MockDoer
		lsclient      client.MockClient
	)

	BeforeEach(func() {
		fakeClientSet = datastore.NewClientSet(nil, fake.NewSimpleClientset().ProjectcalicoV3())

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
			handler := EventStatisticsHandler(fakeClientSet, lsclient)
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
			handler := EventStatisticsHandler(fakeClientSet, lsclient)
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
			handler := EventStatisticsHandler(fakeClientSet, lsclient)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Context("Elasticsearch /events/statistics and AlertExceptions", func() {
		It("should update linseed request selector to consider AlertExceptions ", func() {
			// Set up responses from Linseed.
			lsclient.SetResults(
				rest.MockResult{Body: lapi.EventStatistics{
					FieldValues: &lapi.FieldValues{
						NameValues: []lapi.FieldValue{
							{Value: "WAF Event", Count: 16},
						},
					},
				}},
			)

			// Setup request
			req, err := http.NewRequest(http.MethodPost, "", strings.NewReader(`{
				"selector": "type = waf and not dismissed = true",
				"field_values": {
					"name": {"count": true}
				}
			}`))
			Expect(err).NotTo(HaveOccurred())

			// create an alert exception
			alertException := v3.AlertException{
				// no expiry
				ObjectMeta: metav1.ObjectMeta{
					Name:              "alert-exception-no-expiry",
					CreationTimestamp: metav1.Now(),
				},
				Spec: v3.AlertExceptionSpec{
					Description: "AlertException no expiry",
					Selector:    "origin = origin1",
					StartTime:   metav1.Time{Time: time.Now().Add(-time.Hour)},
				},
			}
			_, err = fakeClientSet.AlertExceptions().Create(context.Background(), &alertException, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := EventStatisticsHandler(fakeClientSet, lsclient)
			handler.ServeHTTP(rr, req)

			// We don't care whether the request succeeds, all we want is to check that
			// it contains the expected selector from the AlertException (everything else is mocked up)
			Expect(lsclient.Requests()).To(HaveLen(1))

			requestBody := lsclient.Requests()[0].GetBody()
			requestBodyBytes, ok := requestBody.([]byte)
			Expect(ok).To(BeTrue())
			requestSelector := gjson.Get(string(requestBodyBytes), "selector").String()

			Expect(requestSelector).To(Equal("(type = waf and not dismissed = true) AND NOT ( origin = origin1 )"))
		})
	})
})
