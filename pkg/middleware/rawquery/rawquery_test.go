package rawquery

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Raw query middleware tests", func() {
	Context("Elasticsearch / request and response validation", func() {
		It("should parse raw query request", func() {
			req, err := http.NewRequest(http.MethodPost, "https://some-url/some-index/_search", nil)
			Expect(err).NotTo(HaveOccurred())

			body := []byte(`{"filter":[],"page_size":100,"page_num":0,"sort_by":[{"field":"time","descending":true}]}`)
			req.Body = ioutil.NopCloser(bytes.NewBuffer(body))

			var w http.ResponseWriter
			index, query, err := parseQueryRequest(w, req)
			Expect(err).NotTo(HaveOccurred())

			Expect(index).To(Equal("some-index"))
			Expect(query).To(Equal(json.RawMessage(body)))
		})

		It("should return error when raw query request method is not POST", func() {
			req, err := http.NewRequest(http.MethodGet, "https://some-url/some-index/_search", nil)
			Expect(err).NotTo(HaveOccurred())
			var w http.ResponseWriter
			index, query, err := parseQueryRequest(w, req)
			Expect(err).To(HaveOccurred())
			Expect(index).To(BeEmpty())
			Expect(query).To(BeNil())
		})

		It("should return error when raw query request url is not _search", func() {
			invalidURLs := []string{
				"https://some-url/some-index",
				"https://some-url/some-index/random-endpoint",
				"https://some-url/some-index/random-endpoint/_search",
			}

			for _, url := range invalidURLs {
				req, err := http.NewRequest(http.MethodPost, url, nil)
				Expect(err).NotTo(HaveOccurred())
				var w http.ResponseWriter
				index, query, err := parseQueryRequest(w, req)
				Expect(err).To(HaveOccurred())
				Expect(index).To(BeEmpty())
				Expect(query).To(BeNil())
			}
		})
	})
})
