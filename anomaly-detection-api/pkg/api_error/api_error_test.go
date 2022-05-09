package api_error_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/anomaly-detection-api/pkg/api_error"
)

var _ = Describe("APIError adds Status to header", func() {

	It("WriteAPIErrorToHeader adds Status specified in the APIError", func() {
		w := httptest.NewRecorder()

		testStatus := http.StatusInternalServerError

		apiErr := &api_error.APIError{
			StatusCode: testStatus,
			Err:        fmt.Errorf("test error"),
		}
		api_error.WriteAPIErrorToHeader(w, apiErr)
		Expect(w.Result().StatusCode).To(Equal(testStatus))

		bodyBytes, err := ioutil.ReadAll(w.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(bodyBytes)).To(Equal(http.StatusText(testStatus)))
	})

	It("WriteStatusErrorToHeader adds Status specified to the header", func() {
		w := httptest.NewRecorder()

		testStatus := http.StatusInternalServerError
		api_error.WriteStatusErrorToHeader(w, testStatus)
		Expect(w.Result().StatusCode).To(Equal(testStatus))

		bodyBytes, err := ioutil.ReadAll(w.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(bodyBytes)).To(Equal(http.StatusText(testStatus)))
	})
})
