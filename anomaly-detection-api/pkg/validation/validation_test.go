package validation_test

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/anomaly-detection-api/pkg/data"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/httputils"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/validation"
)

const (
	testBase64FileString = "dGVzdCBjb250ZW50"
)

var _ = Describe("Validation test", func() {
	Context("ValidateClustersEndpointModelStorageHandlerRequest", func() {

		It("fails validation for a POST method for missing content-type", func() {
			req, _ := http.NewRequest("POST", "/clusters/cluster/models/dynamic/flow/port_scan", strings.NewReader(testBase64FileString))

			err := validation.ValidateClustersEndpointModelStorageHandlerRequest(req)
			Expect(err).ToNot(BeNil())
			Expect(err.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("fails validation for a POST method for incorrect content-type", func() {
			req, _ := http.NewRequest("POST", "/clusters/cluster/models/dynamic/flow/port_scan", strings.NewReader(testBase64FileString))
			req.Header.Add("Content-Type", "application/json")

			err := validation.ValidateClustersEndpointModelStorageHandlerRequest(req)
			Expect(err).ToNot(BeNil())
			Expect(err.StatusCode).To(Equal(http.StatusUnsupportedMediaType))
		})

		It("passes validation for a POST method with correct content-type and path", func() {
			req, _ := http.NewRequest("POST", "/clusters/cluster/models/dynamic/flow/port_scan", strings.NewReader(testBase64FileString))
			req.Header.Add("Content-Type", httputils.StringMIME)

			err := validation.ValidateClustersEndpointModelStorageHandlerRequest(req)
			Expect(err).To(BeNil())
		})

		It("passes validation for a GET method without content-type", func() {
			req, _ := http.NewRequest("GET", "/clusters/cluster/models/dynamic/flow/port_scan", strings.NewReader(testBase64FileString))

			err := validation.ValidateClustersEndpointModelStorageHandlerRequest(req)
			Expect(err).To(BeNil())
		})

		It("fails validation for a POST method with request body too large", func() {
			req, _ := http.NewRequest("POST", "/clusters/cluster/models/dynamic/flow/port_scan", strings.NewReader(testBase64FileString))

			modelSize := 100000001
			req.Header.Add("Content-Type", httputils.StringMIME)
			req.Header.Add("Content-Length", strconv.Itoa(modelSize))
			req.ContentLength = int64(modelSize)

			token := make([]byte, modelSize)
			_, err := rand.Read(token)
			Expect(err).To(BeNil())

			req.Body = ioutil.NopCloser(bytes.NewBuffer(token))

			apiErr := validation.ValidateClustersEndpointModelStorageHandlerRequest(req)
			Expect(apiErr).ToNot(BeNil())
			Expect(apiErr.StatusCode).To(Equal(http.StatusRequestEntityTooLarge))
		})

		It("fails validation for a POST method with request body differing than contentlength", func() {
			req, _ := http.NewRequest("POST", "/clusters/cluster/models/dynamic/flow/port_scan", strings.NewReader(testBase64FileString))

			modelSize := 15730001
			req.Header.Add("Content-Type", httputils.StringMIME)
			req.Header.Add("Content-Length", strconv.Itoa(16))
			req.ContentLength = int64(16)
			token := make([]byte, modelSize)
			_, err := rand.Read(token)
			Expect(err).To(BeNil())

			req.Body = ioutil.NopCloser(bytes.NewBuffer(token))

			apiErr := validation.ValidateClustersEndpointModelStorageHandlerRequest(req)
			Expect(apiErr).ToNot(BeNil())
			Expect(apiErr.StatusCode).To(Equal(http.StatusBadRequest))
		})
	})

	Context("ValidateClustersLogTypeMetadataRequest", func() {

		It("passes validation for a GET method without content-type", func() {
			req, _ := http.NewRequest("GET", "/clusters/cluster/flow/metadata", nil)

			err := validation.ValidateClustersLogTypeMetadataRequest(req)
			Expect(err).To(BeNil())
		})

		It("fails validation for a PUT method with incorrect content-type", func() {
			logTypeMetadataRequest := data.LogTypeMetadata{
				LastUpdated: strconv.FormatInt(time.Now().Unix(), 10),
			}

			j, _ := json.Marshal(logTypeMetadataRequest)

			req, _ := http.NewRequest("PUT", "/clusters/cluster/flow/metadata", bytes.NewBuffer(j))
			req.Header.Add("Content-Type", httputils.StringMIME)

			err := validation.ValidateClustersLogTypeMetadataRequest(req)
			Expect(err).ToNot(BeNil())
			Expect(err.StatusCode).To(Equal(http.StatusUnsupportedMediaType))
		})

		It("passes validation for a PUT method with incorrect non unix timestamp  for lastupdated", func() {
			logTypeMetadataRequest := data.LogTypeMetadata{
				LastUpdated: "bad time string",
			}

			j, _ := json.Marshal(logTypeMetadataRequest)

			req, _ := http.NewRequest("PUT", "/clusters/cluster/flow/metadata", bytes.NewBuffer(j))
			req.Header.Add("Content-Type", httputils.JSSONMIME)

			err := validation.ValidateClustersLogTypeMetadataRequest(req)
			Expect(err).ToNot(BeNil())
			Expect(err.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("passes validation for a PUT method with correct content-type and request body", func() {
			logTypeMetadataRequest := data.LogTypeMetadata{
				LastUpdated: strconv.FormatInt(time.Now().Unix(), 10),
			}

			j, _ := json.Marshal(logTypeMetadataRequest)

			req, _ := http.NewRequest("PUT", "/clusters/cluster/flow/metadata", bytes.NewBuffer(j))
			req.Header.Add("Content-Type", httputils.JSSONMIME)

			err := validation.ValidateClustersLogTypeMetadataRequest(req)
			Expect(err).To(BeNil())
		})

	})

})
