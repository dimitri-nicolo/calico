package fv_test

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	caCert      = "./resources/tls/tls.crt"
	ADAPIDomain = "https://localhost:8080"

	clustersEndpoint  = "/clusters"
	testClusterName   = "/test-cluster"
	modelsEndpoint    = "/models"
	dynamicModelsPath = "/dynamic"
	flowsModelType    = "/flows"
	testModelName     = "/port-scan"

	testBase64FileString = "dGVzdCBjb250ZW50"

	testModelTempFolder = "./temp"
	modelExtension      = ".model"

	healthEndpoint = "/health"
)

var (
	Client *http.Client
)

var _ = BeforeSuite(func() {
	caCert, err := ioutil.ReadFile(caCert)
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	Client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}

})

var _ = Describe("FV", func() {
	Describe("/health endpoint FV", func() {
		It("should return 200 ok", func() {
			req, err := http.NewRequest("GET", ADAPIDomain+healthEndpoint, nil)
			Expect(err).NotTo(HaveOccurred())
			resp, err := Client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))
		})
	})
	Describe("/clusters/{cluster_name}/models endpoint FV", func() {
		AfterEach(func() {
			err := os.RemoveAll(testModelTempFolder)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return 200 ok when sending a model to save through POST /clusters/{cluster_name}/models ", func() {
			endpointPath := ADAPIDomain + clustersEndpoint + testClusterName + modelsEndpoint + dynamicModelsPath + flowsModelType + testModelName
			req, err := http.NewRequest("POST", endpointPath, strings.NewReader(testBase64FileString))
			req.Header.Add("Content-Type", "text/plain")
			Expect(err).NotTo(HaveOccurred())
			resp, err := Client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			_, err = os.Stat(testModelTempFolder + clustersEndpoint + testClusterName + modelsEndpoint + dynamicModelsPath + flowsModelType + testModelName + modelExtension)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return 200 ok when retriving a model through GET /clusters/{cluster_name}/models ", func() {
			endpointPath := ADAPIDomain + clustersEndpoint + testClusterName + modelsEndpoint + dynamicModelsPath + flowsModelType + testModelName
			req, err := http.NewRequest("POST", endpointPath, strings.NewReader(testBase64FileString))
			req.Header.Add("Content-Type", "text/plain")
			Expect(err).NotTo(HaveOccurred())
			resp, err := Client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			_, err = os.Stat(testModelTempFolder + clustersEndpoint + testClusterName + modelsEndpoint + dynamicModelsPath + flowsModelType + testModelName + modelExtension)
			Expect(err).NotTo(HaveOccurred())

			req, err = http.NewRequest("GET", endpointPath, nil)
			Expect(err).NotTo(HaveOccurred())
			resp, err = Client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))
			Expect(err).NotTo(HaveOccurred())

			bodyBytes, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(bodyBytes)).To(Equal(testBase64FileString))
		})

		It("should return 200 ok when updating a model through calling POST /clusters/{cluster_name}/models twice", func() {
			endpointPath := ADAPIDomain + clustersEndpoint + testClusterName + modelsEndpoint + dynamicModelsPath + flowsModelType + testModelName
			req, err := http.NewRequest("POST", endpointPath, strings.NewReader(testBase64FileString))
			req.Header.Add("Content-Type", "text/plain")
			Expect(err).NotTo(HaveOccurred())
			resp, err := Client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			_, err = os.Stat(testModelTempFolder + clustersEndpoint + testClusterName + modelsEndpoint + dynamicModelsPath + flowsModelType + testModelName + modelExtension)
			Expect(err).NotTo(HaveOccurred())

			updatedTestFileContent := "dGVzdCBhbm90aGVyIGNvbnRlbnQ="
			req, err = http.NewRequest("POST", endpointPath, strings.NewReader(updatedTestFileContent))
			req.Header.Add("Content-Type", "text/plain")
			Expect(err).NotTo(HaveOccurred())
			resp, err = Client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))
			Expect(err).NotTo(HaveOccurred())

			// file should still exist
			_, err = os.Stat(testModelTempFolder + clustersEndpoint + testClusterName + modelsEndpoint + dynamicModelsPath + flowsModelType + testModelName + modelExtension)
			Expect(err).NotTo(HaveOccurred())

			// refetching the model should fetch the new context
			req, err = http.NewRequest("GET", endpointPath, nil)
			Expect(err).NotTo(HaveOccurred())
			resp, err = Client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))
			Expect(err).NotTo(HaveOccurred())

			bodyBytes, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(bodyBytes)).To(Equal(updatedTestFileContent))
		})

		It("should return 400 bad request with invalid content type on POST /clusters/{cluster_name}/models ", func() {
			endpointPath := ADAPIDomain + clustersEndpoint + testClusterName + modelsEndpoint + dynamicModelsPath + flowsModelType + testModelName
			req, err := http.NewRequest("POST", endpointPath, strings.NewReader(testBase64FileString))
			req.Header.Add("Content-Type", "application/json")
			Expect(err).NotTo(HaveOccurred())
			resp, err := Client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(415))
		})

		It("should return 200 ok when retriving a model stat through HEAD /clusters/{cluster_name}/models ", func() {
			endpointPath := ADAPIDomain + clustersEndpoint + testClusterName + modelsEndpoint + dynamicModelsPath + flowsModelType + testModelName
			req, err := http.NewRequest("POST", endpointPath, strings.NewReader(testBase64FileString))
			req.Header.Add("Content-Type", "text/plain")
			Expect(err).NotTo(HaveOccurred())
			resp, err := Client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			_, err = os.Stat(testModelTempFolder + clustersEndpoint + testClusterName + modelsEndpoint + dynamicModelsPath + flowsModelType + testModelName + modelExtension)
			Expect(err).NotTo(HaveOccurred())

			req, err = http.NewRequest("HEAD", endpointPath, nil)
			Expect(err).NotTo(HaveOccurred())
			resp, err = Client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))
			Expect(err).NotTo(HaveOccurred())

			bodyBytes, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(bodyBytes)).To(Equal(0))
		})

		It("should return 405 invalid method with invalid content type on PUT /clusters/{cluster_name}/models ", func() {
			endpointPath := ADAPIDomain + clustersEndpoint + testClusterName + modelsEndpoint + dynamicModelsPath + flowsModelType + testModelName
			req, err := http.NewRequest("PUT", endpointPath, strings.NewReader(testBase64FileString))
			req.Header.Add("Content-Type", "text/plain")
			Expect(err).NotTo(HaveOccurred())
			resp, err := Client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(405))
			Expect(resp.StatusCode).To(Equal(405))
		})

		It("should return 413 request too large with invalid content type on PUT /clusters/{cluster_name}/models ", func() {
			endpointPath := ADAPIDomain + clustersEndpoint + testClusterName + modelsEndpoint + dynamicModelsPath + flowsModelType + testModelName
			req, err := http.NewRequest("POST", endpointPath, strings.NewReader(testBase64FileString))
			Expect(err).NotTo(HaveOccurred())

			modelSize := 15730001
			req.Header.Add("Content-Type", "text/plain")
			req.ContentLength = int64(modelSize)
			token := make([]byte, modelSize)
			_, err = rand.Read(token)
			Expect(err).To(BeNil())

			req.Body = ioutil.NopCloser(bytes.NewBuffer(token))

			Expect(err).NotTo(HaveOccurred())
			resp, err := Client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(413))
		})

		It("should return 413 request too large with invalide content type on PUT /clusters/{cluster_name}/models ", func() {
			endpointPath := ADAPIDomain + clustersEndpoint + testClusterName + modelsEndpoint + dynamicModelsPath + flowsModelType + testModelName
			req, err := http.NewRequest("POST", endpointPath, strings.NewReader(testBase64FileString))
			Expect(err).NotTo(HaveOccurred())

			modelSize := 15730001
			req.Header.Add("Content-Type", "text/plain")
			req.ContentLength = int64(modelSize)
			token := make([]byte, modelSize)
			_, err = rand.Read(token)
			Expect(err).To(BeNil())

			req.Body = ioutil.NopCloser(bytes.NewBuffer(token))

			Expect(err).NotTo(HaveOccurred())
			resp, err := Client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(413))
		})

	})
})
