package clusters_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/anomaly-detection-api/pkg/config"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/handler/clusters"
)

const (
	testBase64FileString = "dGVzdCBjb250ZW50"
	testModelTempDir     = "../../../test-resources"
)

var _ = Describe("Clusters Endpoint test", func() {

	var apiConfig *config.Config
	var modelStorageHandler *clusters.ClustersEndpointHandler

	BeforeEach(func() {
		var err error
		apiConfig, err = config.NewConfigFromEnv()
		apiConfig.StoragePath = testModelTempDir
		Expect(err).NotTo(HaveOccurred())

		modelStorageHandler = clusters.NewClustersEndpointHandler(apiConfig)
	})

	AfterEach(func() {
		err := os.RemoveAll(testModelTempDir)
		Expect(err).NotTo(HaveOccurred())
	})

	It("stores body content as file for a successful POST /models", func() {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/clusters/cluster/models/port_scan", strings.NewReader(testBase64FileString))
		req.Header.Add("Content-Type", "text/plain")
		handler := modelStorageHandler.HandleClusters()

		handler.ServeHTTP(w, req)

		Expect(w.Result().StatusCode).To(Equal(200))
		_, err := os.Stat(apiConfig.StoragePath + "/clusters/cluster/models/port_scan.model")
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns 400 upon failing validation for POST /models", func() {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/incorrect-path/cluster/port_scan", strings.NewReader(testBase64FileString))
		req.Header.Add("Content-Type", "text/plain")
		handler := modelStorageHandler.HandleClusters()

		handler.ServeHTTP(w, req)

		Expect(w.Result().StatusCode).To(Equal(400))
	})

	It("returns 405 for PUT method since it is not accepted right now for PUT /models ", func() {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/clusters/cluster/models/port_scan", strings.NewReader(testBase64FileString))
		req.Header.Add("Content-Type", "text/plain")
		handler := modelStorageHandler.HandleClusters()

		handler.ServeHTTP(w, req)

		Expect(w.Result().StatusCode).To(Equal(405))
	})

	It("file content can be fetched for a successful GET /models", func() {
		postWriter := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/clusters/cluster/models/port_scan", strings.NewReader(testBase64FileString))
		req.Header.Add("Content-Type", "text/plain")
		handler := modelStorageHandler.HandleClusters()

		handler.ServeHTTP(postWriter, req)

		Expect(postWriter.Result().StatusCode).To(Equal(200))
		_, err := os.Stat(apiConfig.StoragePath + "/clusters/cluster/models/port_scan.model")
		Expect(err).NotTo(HaveOccurred())

		getWriter := httptest.NewRecorder()
		getReq, _ := http.NewRequest("GET", "/clusters/cluster/models/port_scan", nil)
		handler.ServeHTTP(getWriter, getReq)

		Expect(getWriter.Result().StatusCode).To(Equal(200))
		bodyBytes, err := ioutil.ReadAll(getWriter.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(bodyBytes)).To(Equal(testBase64FileString))

	})
})
