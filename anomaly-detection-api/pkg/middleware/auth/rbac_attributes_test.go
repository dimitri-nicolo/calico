package auth_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/anomaly-detection-api/pkg/middleware/auth"
)

var _ = Describe("GetRBACResoureAttribute", func() {

	const (
		testModelStorageEndpoint    = "/clusters/cluster_name/models/dynamic/flow/port_scan.models"
		testLogTypeMetaDataEndpoint = "/clusters/cluster_name/flow/metadata"
		testNamespace               = "namespace"
	)

	It("returns 404 api error when trying to retrieve and rbac resource for an unregistered path", func() {
		req, _ := http.NewRequest("GET", "unregistered", nil)

		result, apiErr := auth.GetRBACResoureAttribute(testNamespace, req)

		Expect(result).To(BeNil())
		Expect(apiErr).ToNot(BeNil())
		Expect(apiErr.StatusCode).To(Equal(http.StatusNotFound))
	})

	It("an RBAC attribute for a Model Storage endpoint ", func() {
		req, _ := http.NewRequest("GET", testModelStorageEndpoint, nil)

		results, apiErr := auth.GetRBACResoureAttribute(testNamespace, req)

		Expect(apiErr).To(BeNil())
		Expect(len(results)).To(BeNumerically(">", 0))
		for _, result := range results {
			Expect(result.Namespace).To(Equal(testNamespace))
			Expect(result.Group).To(Equal("detectors.tigera.io"))
			Expect(result.Resource).To(Equal("models"))
		}
	})

	It("an RBAC attribute for a LogType Metadata endpoint ", func() {
		req, _ := http.NewRequest("GET", testLogTypeMetaDataEndpoint, nil)

		results, apiErr := auth.GetRBACResoureAttribute(testNamespace, req)

		Expect(apiErr).To(BeNil())
		Expect(len(results)).To(BeNumerically(">", 0))
		for _, result := range results {
			Expect(result.Namespace).To(Equal(testNamespace))
			Expect(result.Group).To(Equal("detectors.tigera.io"))
			Expect(result.Resource).To(Equal("metadata"))
		}
	})
})
