package storage_test

import (
	"net/http"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/anomaly-detection-api/pkg/storage"
)

var _ = Describe("File Storage test", func() {

	const (
		testBase64FileString = "dGVzdCBjb250ZW50"
		testModelTempDir     = "../../../test-resources"
	)

	var fileHandler *storage.FileModelStorageHandler

	BeforeEach(func() {
		fileHandler = &storage.FileModelStorageHandler{
			FileStoragePath: testModelTempDir,
		}
	})

	AfterEach(func() {
		err := os.RemoveAll(testModelTempDir)
		Expect(err).NotTo(HaveOccurred())
	})

	It("stores on the filepath string content for Save", func() {
		req, _ := http.NewRequest("POST", "/clusters/cluster/models/port_scan", strings.NewReader(testBase64FileString))

		err := fileHandler.Save(req)
		Expect(err).NotTo(HaveOccurred())

		_, err = os.Stat(testModelTempDir + "/clusters/cluster/models/port_scan.model")
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns file content as base64 string for Load", func() {
		req, _ := http.NewRequest("POST", "/clusters/cluster/models/port_scan", strings.NewReader(testBase64FileString))

		err := fileHandler.Save(req)
		Expect(err).NotTo(HaveOccurred())

		_, err = os.Stat(testModelTempDir + "/clusters/cluster/models/port_scan.model")
		Expect(err).NotTo(HaveOccurred())

		getReq, _ := http.NewRequest("GET", "/clusters/cluster/models/port_scan", nil)
		content, apiErr := fileHandler.Load(getReq)
		Expect(apiErr).NotTo(HaveOccurred())
		Expect(content).To(Equal(testBase64FileString))
	})

	It("returns 404 error if requested on a cluster that does not exist for Load", func() {
		getReq, _ := http.NewRequest("GET", "/bad-path/cluster/models/port_scan", nil)
		content, apiErr := fileHandler.Load(getReq)
		Expect(content).To(Equal(""))
		Expect(apiErr.StatusCode).To(Equal(http.StatusNotFound))
	})
})
