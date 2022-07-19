package log_type_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/anomaly-detection-api/pkg/data"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/handler/clusters/log_type"
)

var _ = Describe("Log Type Metadata Endpoint test", func() {

	var modelStorageHandler *log_type.LogTypeEndpointHandler

	BeforeEach(func() {
		modelStorageHandler = log_type.NewLogTypeHandler()
	})

	It("returns 405 for that is not Handled", func() {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("HEAD", "/clusters/cluster/flow/metadata", nil)

		modelStorageHandler.HandleLogTypeMetaData(w, req)

		Expect(w.Result().StatusCode).To(Equal(405))
	})

	It("returns 400 upon failing validation for PUT /models without content-type header", func() {
		logTypeMetadataRequest := data.LogTypeMetadata{
			LastUpdated: strconv.FormatInt(time.Now().Unix(), 10),
		}

		j, _ := json.Marshal(logTypeMetadataRequest)

		req, _ := http.NewRequest("PUT", "/clusters/cluster/flow/metadata", bytes.NewBuffer(j))
		w := httptest.NewRecorder()

		modelStorageHandler.HandleLogTypeMetaData(w, req)

		Expect(w.Result().StatusCode).To(Equal(400))
	})

	It("GET for a non-existing key returns 404 - not found", func() {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/clusters/cluster/flow/metadata", nil)

		modelStorageHandler.HandleLogTypeMetaData(w, req)

		Expect(w.Result().StatusCode).To(Equal(404))
	})

	It("GET returns the metadata value for the clusters flow type when it exists", func() {
		logTypeMetadataRequest := data.LogTypeMetadata{
			LastUpdated: strconv.FormatInt(time.Now().Unix(), 10),
		}

		j, _ := json.Marshal(logTypeMetadataRequest)

		req, _ := http.NewRequest("PUT", "/clusters/cluster/flow/metadata", bytes.NewBuffer(j))
		req.Header.Add("Content-Type", "application/json")
		putWriter := httptest.NewRecorder()

		modelStorageHandler.HandleLogTypeMetaData(putWriter, req)

		Expect(putWriter.Result().StatusCode).To(Equal(200))

		getWriter := httptest.NewRecorder()
		getReq, err := http.NewRequest("GET", "/clusters/cluster/flow/metadata", nil)
		Expect(err).NotTo(HaveOccurred())

		modelStorageHandler.HandleLogTypeMetaData(getWriter, getReq)

		Expect(getWriter.Result().StatusCode).To(Equal(200))
		result := data.LogTypeMetadata{}

		err = json.NewDecoder(getWriter.Result().Body).Decode(&result)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.LastUpdated).Should(Equal(logTypeMetadataRequest.LastUpdated))
	})

	It("GET returns the metadata value for the clusters flow type of the most updated lastUpdated entry", func() {
		now := time.Now()
		logTypeMetadataRequest := data.LogTypeMetadata{
			LastUpdated: strconv.FormatInt(now.Unix(), 10),
		}
		j, _ := json.Marshal(logTypeMetadataRequest)
		req, _ := http.NewRequest("PUT", "/clusters/cluster/flow/metadata", bytes.NewBuffer(j))
		req.Header.Add("Content-Type", "application/json")
		putWriter := httptest.NewRecorder()

		modelStorageHandler.HandleLogTypeMetaData(putWriter, req)

		updatedTime := now.Add(1 * time.Hour)
		go func() {
			logTypeMetadataRequest := data.LogTypeMetadata{
				LastUpdated: strconv.FormatInt(updatedTime.Unix(), 10),
			}

			j, _ := json.Marshal(logTypeMetadataRequest)
			req, _ := http.NewRequest("PUT", "/clusters/cluster/flow/metadata", bytes.NewBuffer(j))
			req.Header.Add("Content-Type", "application/json")
			putWriter := httptest.NewRecorder()

			modelStorageHandler.HandleLogTypeMetaData(putWriter, req)
		}()

		time.Sleep(500 * time.Millisecond)

		getWriter := httptest.NewRecorder()
		getReq, err := http.NewRequest("GET", "/clusters/cluster/flow/metadata", nil)
		Expect(err).NotTo(HaveOccurred())

		modelStorageHandler.HandleLogTypeMetaData(getWriter, getReq)

		Expect(getWriter.Result().StatusCode).To(Equal(200))
		result := data.LogTypeMetadata{}

		err = json.NewDecoder(getWriter.Result().Body).Decode(&result)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.LastUpdated).Should(Equal(strconv.FormatInt(updatedTime.Unix(), 10)))
	})
})
