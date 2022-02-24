// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package event

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/mock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"
	"github.com/tigera/compliance/pkg/datastore"
	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	"github.com/tigera/es-proxy/test/thirdpartymock"
	lmaelastic "github.com/tigera/lma/pkg/elastic"
	lmaindex "github.com/tigera/lma/pkg/elastic/index"
)

var (
	// requests from manager to es-proxy
	//go:embed testdata/event_delete_request_from_manager.json
	eventDeleteRequest string
	//go:embed testdata/event_dismiss_request_from_manager.json
	eventDismissRequest string
	//go:embed testdata/event_mixed_request_from_manager.json
	eventMixedRequest string
	//go:embed testdata/event_search_request_from_manager.json
	eventSearchRequestFromManager string

	// requests from es-proxy to elastic
	//go:embed testdata/event_bulk_delete_request.json
	eventBulkDeleteRequest string
	//go:embed testdata/event_bulk_dismiss_request.json
	eventBulkDismissRequest string
	//go:embed testdata/event_bulk_mixed_request.json
	eventBulkMixedRequest string
	//go:embed testdata/event_search_request.json
	eventSearchRequest string
	//go:embed testdata/event_search_request_selector.json
	eventSearchRequestSelector string
	//go:embed testdata/event_search_request_selector_invalid.json
	eventSearchRequestSelectorInvalid string

	// responses from elastic to es-proxy
	//go:embed testdata/event_bulk_delete_response.json
	eventBulkDeleteResponse string
	//go:embed testdata/event_bulk_dismiss_response.json
	eventBulkDismissResponse string
	//go:embed testdata/event_bulk_mixed_response.json
	eventBulkMixedResponse string
	//go:embed testdata/event_search_response.json
	eventSearchResponse string
)

var _ = Describe("Event middleware tests", func() {
	var (
		fakeClientSet datastore.ClientSet
		mockDoer      *thirdpartymock.MockDoer
		mockESFactory *lmaelastic.MockClusterContextClientFactory
	)

	BeforeEach(func() {
		fakeClientSet = datastore.NewClientSet(nil, fake.NewSimpleClientset().ProjectcalicoV3())
		mockDoer = new(thirdpartymock.MockDoer)
		mockESFactory = new(lmaelastic.MockClusterContextClientFactory)
	})

	AfterEach(func() {
		mockDoer.AssertExpectations(GinkgoT())
		mockESFactory.AssertExpectations(GinkgoT())
	})

	Context("Elasticsearch /events request and response validation", func() {
		It("should return a valid event bulk delete response", func() {
			// mock http client
			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
				defer GinkgoRecover()
				req := args.Get(0).(*http.Request)

				// Elastic _bluk request
				Expect(req.Method).To(Equal(http.MethodPost))
				Expect(req.URL.Path).To(Equal("/_bulk"))

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).NotTo(HaveOccurred())
				// Elastic bulk request NDJSON
				Expect(body).To(Equal([]byte(eventBulkDeleteRequest)))

				Expect(req.Body.Close()).NotTo(HaveOccurred())
				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			}).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBuffer([]byte(eventBulkDeleteResponse))),
			}, nil)

			// mock lma es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())
			lmaClient := lmaelastic.NewWithClient(client)
			mockESFactory.On("ClientForCluster", mock.Anything).Return(lmaClient, nil)

			// validate responses
			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(eventDeleteRequest)))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := EventBulkHandler(mockESFactory)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))

			var resp v1.BulkEventResponse
			err = json.Unmarshal(rr.Body.Bytes(), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Took).To(Equal(123))
			Expect(resp.Errors).To(BeTrue())

			Expect(resp.Items[0].Index).To(Equal("some-index-1"))
			Expect(resp.Items[0].ID).To(Equal("id1"))
			Expect(resp.Items[0].Result).To(Equal("deleted"))
			Expect(resp.Items[0].Status).To(Equal(http.StatusOK))
			Expect(resp.Items[0].Error).To(BeNil())

			Expect(resp.Items[1].Index).To(Equal("some-index-2"))
			Expect(resp.Items[1].ID).To(Equal("id2"))
			Expect(resp.Items[1].Status).To(Equal(http.StatusNotFound))
			Expect(resp.Items[1].Error).NotTo(BeNil())
			Expect(resp.Items[1].Error.Type).To(Equal("document_missing_exception"))
			Expect(resp.Items[1].Error.Reason).To(Equal("[_doc][1]: document missing"))

			Expect(resp.Items[2].Index).To(Equal("some-index-3"))
			Expect(resp.Items[2].ID).To(Equal("id3"))
			Expect(resp.Items[2].Result).To(Equal("deleted"))
			Expect(resp.Items[2].Status).To(Equal(http.StatusOK))
			Expect(resp.Items[2].Error).To(BeNil())
		})

		It("should return a valid event bulk dismiss response", func() {
			// mock http client
			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
				defer GinkgoRecover()
				req := args.Get(0).(*http.Request)

				// Elastic _bluk request
				Expect(req.Method).To(Equal(http.MethodPost))
				Expect(req.URL.Path).To(Equal("/_bulk"))

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).NotTo(HaveOccurred())
				// Elastic bulk request NDJSON
				Expect(body).To(Equal([]byte(eventBulkDismissRequest)))

				Expect(req.Body.Close()).NotTo(HaveOccurred())
				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			}).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBuffer([]byte(eventBulkDismissResponse))),
			}, nil)

			// mock lma es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())
			lmaClient := lmaelastic.NewWithClient(client)
			mockESFactory.On("ClientForCluster", mock.Anything).Return(lmaClient, nil)

			// validate responses
			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(eventDismissRequest)))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := EventBulkHandler(mockESFactory)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))

			var resp v1.BulkEventResponse
			err = json.Unmarshal(rr.Body.Bytes(), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Took).To(Equal(456))
			Expect(resp.Errors).To(BeTrue())

			Expect(resp.Items[0].Index).To(Equal("some-index-1"))
			Expect(resp.Items[0].ID).To(Equal("id1"))
			Expect(resp.Items[0].Result).To(Equal("updated"))
			Expect(resp.Items[0].Status).To(Equal(http.StatusOK))
			Expect(resp.Items[0].Error).To(BeNil())

			Expect(resp.Items[1].Index).To(Equal("some-index-2"))
			Expect(resp.Items[1].ID).To(Equal("id2"))
			Expect(resp.Items[1].Status).To(Equal(http.StatusNotFound))
			Expect(resp.Items[1].Error).NotTo(BeNil())
			Expect(resp.Items[1].Error.Type).To(Equal("document_missing_exception"))
			Expect(resp.Items[1].Error.Reason).To(Equal("[_doc][1]: document missing"))

			Expect(resp.Items[2].Index).To(Equal("some-index-3"))
			Expect(resp.Items[2].ID).To(Equal("id3"))
			Expect(resp.Items[2].Result).To(Equal("updated"))
			Expect(resp.Items[2].Status).To(Equal(http.StatusOK))
			Expect(resp.Items[2].Error).To(BeNil())
		})

		It("should return a valid event bulk mixed response", func() {
			// mock http client
			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
				defer GinkgoRecover()
				req := args.Get(0).(*http.Request)

				// Elastic _bluk request
				Expect(req.Method).To(Equal(http.MethodPost))
				Expect(req.URL.Path).To(Equal("/_bulk"))

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).NotTo(HaveOccurred())
				// Elastic bulk request NDJSON
				Expect(body).To(Equal([]byte(eventBulkMixedRequest)))

				Expect(req.Body.Close()).NotTo(HaveOccurred())
				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			}).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBuffer([]byte(eventBulkMixedResponse))),
			}, nil)

			// mock lma es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())
			lmaClient := lmaelastic.NewWithClient(client)
			mockESFactory.On("ClientForCluster", mock.Anything).Return(lmaClient, nil)

			// validate responses
			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(eventMixedRequest)))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := EventBulkHandler(mockESFactory)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))

			var resp v1.BulkEventResponse
			err = json.Unmarshal(rr.Body.Bytes(), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Took).To(Equal(789))
			Expect(resp.Errors).To(BeTrue())

			Expect(resp.Items[0].Index).To(Equal("some-index-1"))
			Expect(resp.Items[0].ID).To(Equal("id1"))
			Expect(resp.Items[0].Result).To(Equal("deleted"))
			Expect(resp.Items[0].Status).To(Equal(http.StatusOK))
			Expect(resp.Items[0].Error).To(BeNil())

			Expect(resp.Items[1].Index).To(Equal("some-index-2"))
			Expect(resp.Items[1].ID).To(Equal("id3"))
			Expect(resp.Items[1].Status).To(Equal(http.StatusNotFound))
			Expect(resp.Items[1].Error).NotTo(BeNil())
			Expect(resp.Items[1].Error.Type).To(Equal("document_missing_exception"))
			Expect(resp.Items[1].Error.Reason).To(Equal("[_doc][1]: document missing"))

			Expect(resp.Items[2].Index).To(Equal("some-index-3"))
			Expect(resp.Items[2].ID).To(Equal("id5"))
			Expect(resp.Items[2].Result).To(Equal("deleted"))
			Expect(resp.Items[2].Status).To(Equal(http.StatusOK))
			Expect(resp.Items[2].Error).To(BeNil())

			Expect(resp.Items[3].Index).To(Equal("some-index-4"))
			Expect(resp.Items[3].ID).To(Equal("id2"))
			Expect(resp.Items[3].Result).To(Equal("updated"))
			Expect(resp.Items[3].Status).To(Equal(http.StatusOK))
			Expect(resp.Items[3].Error).To(BeNil())

			Expect(resp.Items[4].Index).To(Equal("some-index-5"))
			Expect(resp.Items[4].ID).To(Equal("id4"))
			Expect(resp.Items[4].Status).To(Equal(http.StatusNotFound))
			Expect(resp.Items[4].Error).NotTo(BeNil())
			Expect(resp.Items[4].Error.Type).To(Equal("document_missing_exception"))
			Expect(resp.Items[4].Error.Reason).To(Equal("[_doc][4]: document missing"))

			Expect(resp.Items[5].Index).To(Equal("some-index-6"))
			Expect(resp.Items[5].ID).To(Equal("id6"))
			Expect(resp.Items[5].Result).To(Equal("updated"))
			Expect(resp.Items[5].Status).To(Equal(http.StatusOK))
			Expect(resp.Items[5].Error).To(BeNil())
		})

		It("should return error when request is not POST", func() {
			req, err := http.NewRequest(http.MethodGet, "", bytes.NewReader([]byte("any")))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := EventBulkHandler(mockESFactory)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusMethodNotAllowed))
		})

		It("should return error when request body is not valid", func() {
			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte("invalid-json-body")))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := EventBulkHandler(mockESFactory)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("should return error when failed to gat Elastic client from factory", func() {
			mockESFactory.On("ClientForCluster", mock.Anything).Return(nil, errors.New("some error"))

			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(eventDismissRequest)))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := EventBulkHandler(mockESFactory)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Context("Elasticsearch /events/search request and response validation", func() {
		It("should inject alert exceptions in search request", func() {
			// mock http client
			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
				defer GinkgoRecover()
				req := args.Get(0).(*http.Request)

				// Elastic _search request
				Expect(req.Method).To(Equal(http.MethodPost))
				Expect(req.URL.Path).To(Equal("/tigera_secure_ee_events.cluster*/_search"))

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).NotTo(HaveOccurred())
				// Elastic search request json
				Expect(body).To(Equal([]byte(eventSearchRequest)))

				Expect(req.Body.Close()).NotTo(HaveOccurred())
				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			}).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBuffer([]byte(eventSearchResponse))),
			}, nil)

			// create some alert exceptions
			alertExceptions := []*v3.AlertException{
				// no expiry
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "alert-exception-no-expiry",
						CreationTimestamp: metav1.Now(),
					},
					Spec: v3.AlertExceptionSpec{
						Description: "AlertException no expiry",
						Selector:    "origin = origin1",
					},
				},
				// not expired
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "alert-exception-not-expired",
						CreationTimestamp: metav1.Now(),
					},
					Spec: v3.AlertExceptionSpec{
						Description: "AlertException not expired",
						Selector:    "origin = origin2",
						Period:      &metav1.Duration{Duration: time.Hour},
					},
				},
				// expired
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "alert-exception-expired",
						CreationTimestamp: metav1.Time{Time: metav1.Now().Add(-2 * time.Hour)}, // make this one expire
					},
					Spec: v3.AlertExceptionSpec{
						Description: "AlertException expired",
						Selector:    "origin = origin3",
						Period:      &metav1.Duration{Duration: time.Hour},
					},
				},
			}
			for _, alertException := range alertExceptions {
				_, err := fakeClientSet.AlertExceptions().Create(context.Background(), alertException, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
			}

			// mock lma es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())
			lmaClient := lmaelastic.NewWithClient(client)

			// validate responses
			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(eventSearchRequestFromManager)))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := EventSearchHandler(lmaindex.Alerts(), fakeClientSet, lmaClient.Backend())
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))

			var resp v1.SearchResponse
			err = json.Unmarshal(rr.Body.Bytes(), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Hits).To(HaveLen(2))
			Expect(resp.NumPages).To(Equal(1))
			Expect(resp.TimedOut).To(BeFalse())
			Expect(resp.TotalHits).To(Equal(2))
		})

		It("should handle alert exceptions selector AND/OR conditions", func() {
			// mock http client
			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
				defer GinkgoRecover()
				req := args.Get(0).(*http.Request)

				// Elastic _search request
				Expect(req.Method).To(Equal(http.MethodPost))
				Expect(req.URL.Path).To(Equal("/tigera_secure_ee_events.cluster*/_search"))

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).NotTo(HaveOccurred())
				// Elastic search request json
				Expect(body).To(Equal([]byte(eventSearchRequestSelector)))

				Expect(req.Body.Close()).NotTo(HaveOccurred())
				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			}).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBuffer([]byte(eventSearchResponse))),
			}, nil)

			// create some alert exceptions
			alertExceptions := []*v3.AlertException{
				// AND
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "alert-exception-and",
						CreationTimestamp: metav1.Now(),
					},
					Spec: v3.AlertExceptionSpec{
						Description: "AlertException all AND",
						Selector:    "origin = origin1 AND type = global_alert",
					},
				},
				// OR
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "alert-exception-or",
						CreationTimestamp: metav1.Now(),
					},
					Spec: v3.AlertExceptionSpec{
						Description: "AlertException OR",
						Selector:    "origin = origin2 OR type = honeypod",
					},
				},
				// mixed AND / OR
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "alert-exception-and-or",
						CreationTimestamp: metav1.Now(),
					},
					Spec: v3.AlertExceptionSpec{
						Description: "AlertException AND OR",
						Selector:    "origin = origin3 AND type = alert OR source_namespace = ns3",
					},
				},
			}
			for _, alertException := range alertExceptions {
				_, err := fakeClientSet.AlertExceptions().Create(context.Background(), alertException, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
			}

			// mock lma es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())
			lmaClient := lmaelastic.NewWithClient(client)

			// validate responses
			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(eventSearchRequestFromManager)))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := EventSearchHandler(lmaindex.Alerts(), fakeClientSet, lmaClient.Backend())
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))

			var resp v1.SearchResponse
			err = json.Unmarshal(rr.Body.Bytes(), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Hits).To(HaveLen(2))
			Expect(resp.NumPages).To(Equal(1))
			Expect(resp.TimedOut).To(BeFalse())
			Expect(resp.TotalHits).To(Equal(2))
		})

		It("should skip invalid alert exceptions selector", func() {
			// mock http client
			mockDoer.On("Do", mock.AnythingOfType("*http.Request")).Run(func(args mock.Arguments) {
				defer GinkgoRecover()
				req := args.Get(0).(*http.Request)

				// Elastic _search request
				Expect(req.Method).To(Equal(http.MethodPost))
				Expect(req.URL.Path).To(Equal("/tigera_secure_ee_events.cluster*/_search"))

				body, err := ioutil.ReadAll(req.Body)
				Expect(err).NotTo(HaveOccurred())
				// Elastic search request json
				Expect(body).To(Equal([]byte(eventSearchRequestSelectorInvalid)))

				Expect(req.Body.Close()).NotTo(HaveOccurred())
				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			}).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBuffer([]byte(eventSearchResponse))),
			}, nil)

			// create some alert exceptions
			alertExceptions := []*v3.AlertException{
				// valid selector
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "alert-exception-valid-selector",
						CreationTimestamp: metav1.Now(),
					},
					Spec: v3.AlertExceptionSpec{
						Description: "AlertException valid selector",
						Selector:    "origin = origin1",
					},
				},
				// invalid selector
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "alert-exception-invalid-selector",
						CreationTimestamp: metav1.Now(),
					},
					Spec: v3.AlertExceptionSpec{
						Description: "AlertException invalid selector",
						Selector:    "invalid selector",
					},
				},
			}
			for _, alertException := range alertExceptions {
				_, err := fakeClientSet.AlertExceptions().Create(context.Background(), alertException, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
			}

			// mock lma es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())
			lmaClient := lmaelastic.NewWithClient(client)

			// validate responses
			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(eventSearchRequestFromManager)))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := EventSearchHandler(lmaindex.Alerts(), fakeClientSet, lmaClient.Backend())
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))

			var resp v1.SearchResponse
			err = json.Unmarshal(rr.Body.Bytes(), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Hits).To(HaveLen(2))
			Expect(resp.NumPages).To(Equal(1))
			Expect(resp.TimedOut).To(BeFalse())
			Expect(resp.TotalHits).To(Equal(2))
		})

		It("should return error when request is not GET or POST", func() {
			// mock lma es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())
			lmaClient := lmaelastic.NewWithClient(client)

			req, err := http.NewRequest(http.MethodPatch, "", bytes.NewReader([]byte("any")))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := EventSearchHandler(lmaindex.Alerts(), fakeClientSet, lmaClient.Backend())
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusMethodNotAllowed))
		})

		It("should return error when request body is not valid", func() {
			// mock lma es client
			client, err := elastic.NewClient(
				elastic.SetHttpClient(mockDoer),
				elastic.SetSniff(false),
				elastic.SetHealthcheck(false),
			)
			Expect(err).NotTo(HaveOccurred())
			lmaClient := lmaelastic.NewWithClient(client)

			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte("invalid-json-body")))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := EventSearchHandler(lmaindex.Alerts(), fakeClientSet, lmaClient.Backend())
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})
	})
})
