package fv_test

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/anomaly-detection-api/pkg/data"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	caCert      = "./resources/tls/tls.crt"
	ADAPIDomain = "https://localhost:8080"

	clustersEndpoint  = "/clusters"
	testClusterName   = "/test-cluster"
	modelsEndpoint    = "/models"
	metadataEndpoint  = "/metadata"
	dynamicModelsPath = "/dynamic"
	flowModelType     = "/flow"
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
				InsecureSkipVerify: true,
				RootCAs:            caCertPool,
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
			endpointPath := ADAPIDomain + clustersEndpoint + testClusterName + modelsEndpoint + dynamicModelsPath + flowModelType + testModelName
			req, err := http.NewRequest("POST", endpointPath, strings.NewReader(testBase64FileString))
			req.Header.Add("Content-Type", "text/plain")
			Expect(err).NotTo(HaveOccurred())
			resp, err := Client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			_, err = os.Stat(testModelTempFolder + clustersEndpoint + testClusterName + modelsEndpoint + dynamicModelsPath + flowModelType + testModelName + modelExtension)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return 200 ok when retriving a model through GET /clusters/{cluster_name}/models ", func() {
			endpointPath := ADAPIDomain + clustersEndpoint + testClusterName + modelsEndpoint + dynamicModelsPath + flowModelType + testModelName
			req, err := http.NewRequest("POST", endpointPath, strings.NewReader(testBase64FileString))
			req.Header.Add("Content-Type", "text/plain")
			Expect(err).NotTo(HaveOccurred())
			resp, err := Client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			_, err = os.Stat(testModelTempFolder + clustersEndpoint + testClusterName + modelsEndpoint + dynamicModelsPath + flowModelType + testModelName + modelExtension)
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
			endpointPath := ADAPIDomain + clustersEndpoint + testClusterName + modelsEndpoint + dynamicModelsPath + flowModelType + testModelName
			req, err := http.NewRequest("POST", endpointPath, strings.NewReader(testBase64FileString))
			req.Header.Add("Content-Type", "text/plain")
			Expect(err).NotTo(HaveOccurred())
			resp, err := Client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			_, err = os.Stat(testModelTempFolder + clustersEndpoint + testClusterName + modelsEndpoint + dynamicModelsPath + flowModelType + testModelName + modelExtension)
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
			_, err = os.Stat(testModelTempFolder + clustersEndpoint + testClusterName + modelsEndpoint + dynamicModelsPath + flowModelType + testModelName + modelExtension)
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
			endpointPath := ADAPIDomain + clustersEndpoint + testClusterName + modelsEndpoint + dynamicModelsPath + flowModelType + testModelName
			req, err := http.NewRequest("POST", endpointPath, strings.NewReader(testBase64FileString))
			req.Header.Add("Content-Type", "application/json")
			Expect(err).NotTo(HaveOccurred())
			resp, err := Client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(415))
		})

		It("should return 200 ok when retriving a model stat through HEAD /clusters/{cluster_name}/models ", func() {
			endpointPath := ADAPIDomain + clustersEndpoint + testClusterName + modelsEndpoint + dynamicModelsPath + flowModelType + testModelName
			req, err := http.NewRequest("POST", endpointPath, strings.NewReader(testBase64FileString))
			req.Header.Add("Content-Type", "text/plain")
			Expect(err).NotTo(HaveOccurred())
			resp, err := Client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			_, err = os.Stat(testModelTempFolder + clustersEndpoint + testClusterName + modelsEndpoint + dynamicModelsPath + flowModelType + testModelName + modelExtension)
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
			endpointPath := ADAPIDomain + clustersEndpoint + testClusterName + modelsEndpoint + dynamicModelsPath + flowModelType + testModelName
			req, err := http.NewRequest("PUT", endpointPath, strings.NewReader(testBase64FileString))
			req.Header.Add("Content-Type", "text/plain")
			Expect(err).NotTo(HaveOccurred())
			resp, err := Client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(405))
			Expect(resp.StatusCode).To(Equal(405))
		})

		It("should return 413 request too large with invalid content type on PUT /clusters/{cluster_name}/models ", func() {
			endpointPath := ADAPIDomain + clustersEndpoint + testClusterName + modelsEndpoint + dynamicModelsPath + flowModelType + testModelName
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
			endpointPath := ADAPIDomain + clustersEndpoint + testClusterName + modelsEndpoint + dynamicModelsPath + flowModelType + testModelName
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

	Describe("/clusters/{cluster_name}/{log_type}/metadata endpoint FV", func() {

		It("returns 405 for a nethod (DELETE) that is not Handled", func() {
			endpointPath := ADAPIDomain + clustersEndpoint + testClusterName + flowModelType + metadataEndpoint
			req, err := http.NewRequest("DELETE", endpointPath, nil)

			Expect(err).NotTo(HaveOccurred())
			resp, err := Client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(405))
		})

		It("returns 400 upon failing validation for PUT /{log_type}/metadata without content-type header", func() {
			endpointPath := ADAPIDomain + clustersEndpoint + testClusterName + flowModelType + metadataEndpoint

			logTypeMetadataRequest := data.LogTypeMetadata{
				LastUpdated: strconv.FormatInt(time.Now().Unix(), 10),
			}

			j, _ := json.Marshal(logTypeMetadataRequest)

			req, err := http.NewRequest("PUT", endpointPath, bytes.NewBuffer(j))

			Expect(err).NotTo(HaveOccurred())
			resp, err := Client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))
		})

		It("returns 415 upon failing validation for PUT /{log_type}/metadata with unsupported content-type header", func() {
			endpointPath := ADAPIDomain + clustersEndpoint + testClusterName + flowModelType + metadataEndpoint

			logTypeMetadataRequest := data.LogTypeMetadata{
				LastUpdated: strconv.FormatInt(time.Now().Unix(), 10),
			}

			j, _ := json.Marshal(logTypeMetadataRequest)

			req, err := http.NewRequest("PUT", endpointPath, bytes.NewBuffer(j))
			req.Header.Add("Content-Type", "text/plain")

			Expect(err).NotTo(HaveOccurred())
			resp, err := Client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(415))
		})

		It("should return 404 not found when attempting to retrieve a non existing metadata", func() {
			endpointPath := ADAPIDomain + clustersEndpoint + testClusterName + flowModelType + metadataEndpoint

			req, err := http.NewRequest("GET", endpointPath, nil)

			Expect(err).NotTo(HaveOccurred())
			resp, err := Client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(404))
		})

		It("should return 200 ok when retriving metadata for an exising cluter flow type", func() {
			endpointPath := ADAPIDomain + clustersEndpoint + testClusterName + flowModelType + metadataEndpoint

			logTypeMetadataRequest := data.LogTypeMetadata{
				LastUpdated: strconv.FormatInt(time.Now().Unix(), 10),
			}

			j, _ := json.Marshal(logTypeMetadataRequest)

			req, err := http.NewRequest("PUT", endpointPath, bytes.NewBuffer(j))
			req.Header.Add("Content-Type", "application/json")

			Expect(err).NotTo(HaveOccurred())
			resp, err := Client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			getReq, err := http.NewRequest("GET", endpointPath, nil)
			Expect(err).NotTo(HaveOccurred())
			getResp, err := Client.Do(getReq)
			Expect(err).NotTo(HaveOccurred())
			Expect(getResp.StatusCode).To(Equal(200))

			result := data.LogTypeMetadata{}
			err = json.NewDecoder(getResp.Body).Decode(&result)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.LastUpdated).Should(Equal(logTypeMetadataRequest.LastUpdated))
		})

		It("should return 200 ok and the metadata value for the clusters flow type of the most updated lastUpdated entry", func() {
			endpointPath := ADAPIDomain + clustersEndpoint + testClusterName + flowModelType + metadataEndpoint

			logTypeMetadataRequest := data.LogTypeMetadata{
				LastUpdated: strconv.FormatInt(time.Now().Unix(), 10),
			}

			j, _ := json.Marshal(logTypeMetadataRequest)

			req, err := http.NewRequest("PUT", endpointPath, bytes.NewBuffer(j))
			req.Header.Add("Content-Type", "application/json")

			Expect(err).NotTo(HaveOccurred())
			go func() {
				_, _ = Client.Do(req)

			}()

			updatedTime := time.Now().Add(1 * time.Hour)
			go func() {
				logTypeMetadataRequest := data.LogTypeMetadata{
					LastUpdated: strconv.FormatInt(updatedTime.Unix(), 10),
				}

				j, _ := json.Marshal(logTypeMetadataRequest)
				req, _ := http.NewRequest("PUT", endpointPath, bytes.NewBuffer(j))
				req.Header.Add("Content-Type", "application/json")

				_, _ = Client.Do(req)
			}()

			time.Sleep(1 * time.Second)

			getReq, err := http.NewRequest("GET", endpointPath, nil)
			Expect(err).NotTo(HaveOccurred())
			getResp, err := Client.Do(getReq)
			Expect(err).NotTo(HaveOccurred())
			Expect(getResp.StatusCode).To(Equal(200))

			result := data.LogTypeMetadata{}
			err = json.NewDecoder(getResp.Body).Decode(&result)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.LastUpdated).Should(Equal(strconv.FormatInt(updatedTime.Unix(), 10)))
		})
	})
})
