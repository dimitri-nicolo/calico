// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package event

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/mock"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	"github.com/tigera/es-proxy/test/thirdpartymock"
	lmaelastic "github.com/tigera/lma/pkg/elastic"
)

var (
	// requests from manager to es-proxy
	//go:embed testdata/event_delete_request_from_manager.json
	eventDeleteRequest string
	//go:embed testdata/event_dismiss_request_from_manager.json
	eventDismissRequest string
	//go:embed testdata/event_mixed_request_from_manager.json
	eventMixedRequest string

	// requests from es-proxy to elastic
	//go:embed testdata/event_bulk_delete_request.json
	eventBulkDeleteRequest string
	//go:embed testdata/event_bulk_dismiss_request.json
	eventBulkDismissRequest string
	//go:embed testdata/event_bulk_mixed_request.json
	eventBulkMixedRequest string

	// responses from elastic to es-proxy
	//go:embed testdata/event_bulk_delete_response.json
	eventBulkDeleteResponse string
	//go:embed testdata/event_bulk_dismiss_response.json
	eventBulkDismissResponse string
	//go:embed testdata/event_bulk_mixed_response.json
	eventBulkMixedResponse string
)

var _ = Describe("Event middleware tests", func() {
	var (
		mockDoer      *thirdpartymock.MockDoer
		mockESFactory *lmaelastic.MockClusterContextClientFactory
	)

	BeforeEach(func() {
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
			handler := EventHandler(mockESFactory)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))

			var resp v1.BulkResponse
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
			handler := EventHandler(mockESFactory)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))

			var resp v1.BulkResponse
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
			handler := EventHandler(mockESFactory)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))

			var resp v1.BulkResponse
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
			handler := EventHandler(mockESFactory)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusMethodNotAllowed))
		})

		It("should return error when request body is not valid", func() {
			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte("invalid-json-body")))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := EventHandler(mockESFactory)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("should return error when failed to gat Elastic client from factory", func() {
			mockESFactory.On("ClientForCluster", mock.Anything).Return(nil, errors.New("some error"))

			req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader([]byte(eventDismissRequest)))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := EventHandler(mockESFactory)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})
	})
})
