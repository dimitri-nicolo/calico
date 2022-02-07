// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package event

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/mock"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	"github.com/tigera/es-proxy/test/thirdpartymock"
	lmaelastic "github.com/tigera/lma/pkg/elastic"
)

var (
	//go:embed event_bulk_delete_request.json
	eventBulkDeleteRequest string
	//go:embed event_bulk_dismiss_request.json
	eventBulkDismissRequest string
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
		It("should parse event delete request", func() {
			req, err := http.NewRequest(http.MethodDelete, "https://some-url", nil)
			Expect(err).NotTo(HaveOccurred())

			buf := []byte(`{"cluster":"test-cluster-name","items":[{"id":"id1"},{"id":"id2"},{"id":"id3"}]}`)
			req.Body = ioutil.NopCloser(bytes.NewBuffer(buf))

			var w http.ResponseWriter
			deleteRequest, err := parseDeleteRequest(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(deleteRequest.ClusterName).To(Equal("test-cluster-name"))
			Expect(deleteRequest.Items).To(HaveLen(3))
			Expect(deleteRequest.Items[0].ID).To(Equal("id1"))
			Expect(deleteRequest.Items[1].ID).To(Equal("id2"))
			Expect(deleteRequest.Items[2].ID).To(Equal("id3"))
			Expect(deleteRequest.Timeout.Duration).To(Equal(60 * time.Second))
		})

		It("should parse event dismiss request", func() {
			req, err := http.NewRequest(http.MethodPost, "https://some-url", nil)
			Expect(err).NotTo(HaveOccurred())

			buf := []byte(`{"cluster":"test-cluster-name","items":[{"id":"id1"},{"id":"id2"},{"id":"id3"}]}`)
			req.Body = ioutil.NopCloser(bytes.NewBuffer(buf))

			var w http.ResponseWriter
			deleteRequest, err := parseDismissRequest(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(deleteRequest.ClusterName).To(Equal("test-cluster-name"))
			Expect(deleteRequest.Items).To(HaveLen(3))
			Expect(deleteRequest.Items[0].ID).To(Equal("id1"))
			Expect(deleteRequest.Items[1].ID).To(Equal("id2"))
			Expect(deleteRequest.Items[2].ID).To(Equal("id3"))
			Expect(deleteRequest.Timeout.Duration).To(Equal(60 * time.Second))
		})

		It("should return error when event delete request method is not DELETE", func() {
			req, err := http.NewRequest(http.MethodGet, "https://some-url", nil)
			Expect(err).NotTo(HaveOccurred())
			var w http.ResponseWriter
			resp, err := parseDeleteRequest(w, req)
			Expect(err).To(HaveOccurred())
			Expect(resp).To(BeNil())
		})

		It("should return error when event dismiss request method is not POST", func() {
			req, err := http.NewRequest(http.MethodGet, "https://some-url", nil)
			Expect(err).NotTo(HaveOccurred())
			var w http.ResponseWriter
			resp, err := parseDismissRequest(w, req)
			Expect(err).To(HaveOccurred())
			Expect(resp).To(BeNil())
		})

		It("should return a valid event delete response for multiple doc IDs", func() {
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
				Body: responseToBody(elastic.BulkResponse{
					Took:   123,
					Errors: true,
					Items: []map[string]*elastic.BulkResponseItem{
						{
							"delete": &elastic.BulkResponseItem{
								Index:  "some-index-1",
								Id:     "id1",
								Result: "deleted",
								Status: http.StatusOK,
								Type:   "_doc",
							},
						},
						{
							"delete": &elastic.BulkResponseItem{
								Index:  "some-index-2",
								Id:     "id2",
								Status: http.StatusNotFound,
								Type:   "_doc",
								Error: &elastic.ErrorDetails{
									Type:   "document_missing_exception",
									Reason: "[_doc][1]: document missing",
								},
							},
						},
						{
							"delete": &elastic.BulkResponseItem{
								Index:  "some-index-3",
								Id:     "id3",
								Result: "deleted",
								Status: http.StatusOK,
								Type:   "_doc",
							},
						},
						{
							"updated": &elastic.BulkResponseItem{
								Index:  "some-index-4",
								Id:     "id4",
								Result: "updated",
								Status: http.StatusOK,
								Type:   "_doc",
							},
						},
					},
				}),
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

			// validate
			bodyJSON := []byte(`{"cluster":"test-cluster-name","items":[{"id":"id1"},{"id":"id2"},{"id":"id3"}],"timeout":"60s"}`)
			r, err := http.NewRequest(http.MethodDelete, "", bytes.NewReader(bodyJSON))
			Expect(err).NotTo(HaveOccurred())

			var params v1.BulkRequest
			err = json.Unmarshal(bodyJSON, &params)
			Expect(err).NotTo(HaveOccurred())

			resp, err := delete(mockESFactory, &params, r)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Took).To(Equal(123))
			Expect(resp.Errors).To(BeTrue())
			Expect(resp.Items).To(HaveLen(3))

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

		It("should return a valid event dismiss response for multiple doc IDs", func() {
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
				Body: responseToBody(elastic.BulkResponse{
					Took:   123,
					Errors: true,
					Items: []map[string]*elastic.BulkResponseItem{
						{
							"update": &elastic.BulkResponseItem{
								Index:  "some-index-1",
								Id:     "id1",
								Result: "updated",
								Status: http.StatusOK,
								Type:   "_doc",
							},
						},
						{
							"update": &elastic.BulkResponseItem{
								Index:  "some-index-2",
								Id:     "id2",
								Status: http.StatusNotFound,
								Type:   "_doc",
								Error: &elastic.ErrorDetails{
									Type:   "document_missing_exception",
									Reason: "[_doc][1]: document missing",
								},
							},
						},
						{
							"update": &elastic.BulkResponseItem{
								Index:  "some-index-3",
								Id:     "id3",
								Result: "updated",
								Status: http.StatusOK,
								Type:   "_doc",
							},
						},
						{
							"delete": &elastic.BulkResponseItem{
								Index:  "some-index-4",
								Id:     "id4",
								Result: "deleted",
								Status: http.StatusOK,
								Type:   "_doc",
							},
						},
					},
				}),
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

			// validate
			bodyJSON := []byte(`{"cluster":"test-cluster-name","items":[{"id":"id1"},{"id":"id2"},{"id":"id3"}],"timeout":"60s"}`)
			r, err := http.NewRequest(http.MethodDelete, "", bytes.NewReader(bodyJSON))
			Expect(err).NotTo(HaveOccurred())

			var params v1.BulkRequest
			err = json.Unmarshal(bodyJSON, &params)
			Expect(err).NotTo(HaveOccurred())

			resp, err := dismiss(mockESFactory, &params, r)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Took).To(Equal(123))
			Expect(resp.Errors).To(BeTrue())
			Expect(resp.Items).To(HaveLen(3))

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
	})
})

func responseToBody(resp interface{}) io.ReadCloser {
	buf, err := json.Marshal(resp)
	if err != nil {
		panic(err)
	}
	return ioutil.NopCloser(bytes.NewBuffer(buf))
}
