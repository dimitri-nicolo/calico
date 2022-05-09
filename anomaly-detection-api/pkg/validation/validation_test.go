package validation_test

import (
	"bytes"
	"crypto/rand"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/anomaly-detection-api/pkg/validation"
)

const (
	testBase64FileString = "dGVzdCBjb250ZW50"
)

var _ = Describe("Validation test", func() {

	It("fails validation for a POST method for missing content-type", func() {
		req, _ := http.NewRequest("POST", "/clusters/cluster/models/port_scan", strings.NewReader(testBase64FileString))

		err := validation.ValidateClustersEndpointRequest(req)
		Expect(err).ToNot(BeNil())
		Expect(err.StatusCode).To(Equal(http.StatusBadRequest))
	})

	It("fails validation for a POST method for incorrect content-type", func() {
		req, _ := http.NewRequest("POST", "/clusters/cluster/models/port_scan", strings.NewReader(testBase64FileString))
		req.Header.Add("Content-Type", "application/json")

		err := validation.ValidateClustersEndpointRequest(req)
		Expect(err).ToNot(BeNil())
		Expect(err.StatusCode).To(Equal(http.StatusUnsupportedMediaType))
	})

	It("passes validation for a POST method with correct content-type and path", func() {
		req, _ := http.NewRequest("POST", "/clusters/cluster/models/port_scan", strings.NewReader(testBase64FileString))
		req.Header.Add("Content-Type", "text/plain")

		err := validation.ValidateClustersEndpointRequest(req)
		Expect(err).To(BeNil())
	})

	It("passes validation for a GET method without content-type", func() {
		req, _ := http.NewRequest("GET", "/clusters/cluster/models/port_scan", strings.NewReader(testBase64FileString))

		err := validation.ValidateClustersEndpointRequest(req)
		Expect(err).To(BeNil())
	})

	It("fails validation for a POST method with incorrect path", func() {
		req, _ := http.NewRequest("POST", "/incorrect-path/cluster/port_scan", strings.NewReader(testBase64FileString))
		req.Header.Add("Content-Type", "text/plain")

		err := validation.ValidateClustersEndpointRequest(req)
		Expect(err).ToNot(BeNil())
		Expect(err.StatusCode).To(Equal(http.StatusBadRequest))
	})

	It("fails validation for a POST method with request body too large", func() {
		req, _ := http.NewRequest("POST", "/clusters/cluster/models/port_scan", strings.NewReader(testBase64FileString))

		modelSize := 15730001
		req.Header.Add("Content-Type", "text/plain")
		req.Header.Add("Content-Length", strconv.Itoa(modelSize))
		req.ContentLength = int64(modelSize)

		token := make([]byte, modelSize)
		_, err := rand.Read(token)
		Expect(err).To(BeNil())

		req.Body = ioutil.NopCloser(bytes.NewBuffer(token))

		apiErr := validation.ValidateClustersEndpointRequest(req)
		Expect(apiErr).ToNot(BeNil())
		Expect(apiErr.StatusCode).To(Equal(http.StatusRequestEntityTooLarge))
	})

	It("fails validation for a POST method with request body differing than contentlength", func() {
		req, _ := http.NewRequest("POST", "/clusters/cluster/models/port_scan", strings.NewReader(testBase64FileString))

		modelSize := 15730001
		req.Header.Add("Content-Type", "text/plain")
		req.Header.Add("Content-Length", strconv.Itoa(16))
		req.ContentLength = int64(16)
		token := make([]byte, modelSize)
		_, err := rand.Read(token)
		Expect(err).To(BeNil())

		req.Body = ioutil.NopCloser(bytes.NewBuffer(token))

		apiErr := validation.ValidateClustersEndpointRequest(req)
		Expect(apiErr).ToNot(BeNil())
		Expect(apiErr.StatusCode).To(Equal(http.StatusBadRequest))
	})
})
