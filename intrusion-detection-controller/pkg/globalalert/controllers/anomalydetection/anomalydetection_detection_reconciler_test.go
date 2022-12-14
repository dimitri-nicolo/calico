package anomalydetection

import (
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AnomalyDetection Detection Reconciler", func() {
	var (
		reconciler adDetectionReconciler
	)

	BeforeEach(func() {
		reconciler = adDetectionReconciler{}

	})

	Context("getDetectionCycleCronJobNameForGlobaAlert", func() {
		It("creates a RFC1123 Valid name given a long globalalert name", func() {
			cluster := "cluster"
			globalAlertName := "tigera.io.detectors.port-scan"
			result := reconciler.getDetectionCycleCronJobNameForGlobaAlert(cluster, globalAlertName)
			// IsDNS1123Subdomain returns list of error strings
			Expect(validation.IsDNS1123Subdomain(result)).To(HaveLen(0))
			parts := strings.Split(result, "-detection")
			Expect(parts[0]).To(HaveLen(acceptableRFCGlobalAlertNameLen))
		})

		It("creates a RFC1123 Valid name an long globalalert and cluster name from a CC management cluster", func() {
			// bug where the CC cloud management / managed cluster name creates a bib rfc1123 compliant name
			// since acceptableRFCGlobalAlertNameLen will create the cronjob name bh0iuz1z.cluster-tigera.io.detectors.-detection...
			cluster := "bh0iuz1z.cluster"
			globalAlertName := "tigera.io.detectors.port-scan"
			result := reconciler.getDetectionCycleCronJobNameForGlobaAlert(cluster, globalAlertName)
			// IsDNS1123Subdomain returns list of error strings
			Expect(validation.IsDNS1123Subdomain(result)).To(HaveLen(0))
			parts := strings.Split(result, "-detection")
			Expect(parts[0]).To(HaveLen(acceptableRFCGlobalAlertNameLen))
		})

		It("creates a RFC1123 Valid name short globalalert and cluster name", func() {
			cluster := "cster"
			globalAlertName := "port-scan"
			result := reconciler.getDetectionCycleCronJobNameForGlobaAlert(cluster, globalAlertName)
			// IsDNS1123Subdomain returns list of error strings
			Expect(validation.IsDNS1123Subdomain(result)).To(HaveLen(0))
			parts := strings.Split(result, "-detection")
			// should include the full cluster-globalalertname if they are less than acceptableRFCGlobalAlertNameLen
			Expect(parts[0]).To(HaveLen(len(cluster + "-" + globalAlertName)))
		})
	})
})
