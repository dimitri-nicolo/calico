package usage

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	k8sFake "k8s.io/client-go/kubernetes/fake"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	usagev1 "github.com/projectcalico/calico/libcalico-go/lib/apis/usage.tigera.io/v1"
	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
)

// These UTs validate that the reportWriter performs retries as expected, and does not write incomplete reports.
// The validation of report enrichment and writing to the datastore is handled by the FVs.
var _ = Describe("Usage Writer UTs", func() {
	var fakeK8sClient kubernetes.Interface
	var fakeV3Client clientv3.Interface
	var stopCh chan struct{}
	reportWriterForTest := func(usageClient runtimeClient.Client) reportWriter {
		return newReportWriter(
			make(chan basicLicenseUsageReport),
			stopCh,
			context.Background(),
			fakeK8sClient,
			fakeV3Client,
			usageClient,
		)
	}

	BeforeEach(func() {
		// Fake kubernetes client with no objects associated with it.
		fakeK8sClient = k8sFake.NewSimpleClientset()
		// Fake calico client that will only respond to License GET with a 404.
		fakeV3Client = fakeCalicoClient{}
		// Stop channel used for tests where we run the main loop.
		stopCh = make(chan struct{})
	})

	Context("retries", func() {
		It("should fail immediately if writing to storage results in a non-retryable error", func() {
			fakeRuntimeClient := &errorReturningFakeRuntimeClient{
				err: errors.NewBadRequest("you know what you did wrong"),
			}
			writer := reportWriterForTest(fakeRuntimeClient)

			err := writer.writeDatastoreReport(&usagev1.LicenseUsageReport{
				ObjectMeta: metav1.ObjectMeta{Name: "report"},
				Spec:       usagev1.LicenseUsageReportSpec{},
			})

			Expect(err).To(HaveOccurred())
			Expect(fakeRuntimeClient.requestCount).To(Equal(1))
		})

		It("should retry and succeed if a transient error eventually resolves", func() {
			count2 := 2
			fakeRuntimeClient := &errorReturningFakeRuntimeClient{
				err:                   errors.NewTooManyRequests("chill", 1),
				resolveAtRequestCount: &count2,
			}
			writer := reportWriterForTest(fakeRuntimeClient)

			err := writer.writeDatastoreReport(&usagev1.LicenseUsageReport{
				ObjectMeta: metav1.ObjectMeta{Name: "report"},
				Spec:       usagev1.LicenseUsageReportSpec{},
			})

			Expect(err).To(Not(HaveOccurred()))
			Expect(fakeRuntimeClient.requestCount).To(BeNumerically(">", 1))
		})

		It("should retry and eventually fail if a transient error never resolves", func() {
			fakeRuntimeClient := &errorReturningFakeRuntimeClient{
				err: errors.NewTooManyRequests("chill", 1),
			}
			writer := reportWriterForTest(fakeRuntimeClient)

			err := writer.writeDatastoreReport(&usagev1.LicenseUsageReport{
				ObjectMeta: metav1.ObjectMeta{Name: "report"},
				Spec:       usagev1.LicenseUsageReportSpec{},
			})

			Expect(err).To(HaveOccurred())
			Expect(fakeRuntimeClient.requestCount).To(BeNumerically(">", 1))
		})

		It("should fail immediately if writing to storage results in an error indicating the CRD isn't present", func() {
			fakeRuntimeClient := &errorReturningFakeRuntimeClient{
				err: &discovery.ErrGroupDiscoveryFailed{
					Groups: map[schema.GroupVersion]error{
						{Group: "foo", Version: "v1"}: fmt.Errorf("discovery failure"),
					},
				},
			}
			writer := reportWriterForTest(fakeRuntimeClient)

			err := writer.writeDatastoreReport(&usagev1.LicenseUsageReport{
				ObjectMeta: metav1.ObjectMeta{Name: "report"},
				Spec:       usagev1.LicenseUsageReportSpec{},
			})

			Expect(err).To(HaveOccurred())
			Expect(fakeRuntimeClient.requestCount).To(Equal(1))
		})
	})

	Context("incomplete reports", func() {
		var tracker callCountingObjectTracker
		var fakeRuntimeClient runtimeClient.WithWatch
		var writer reportWriter
		BeforeEach(func() {
			tracker = callCountingObjectTracker{}
			scheme := createScheme()
			fakeRuntimeClient = fake.NewClientBuilder().WithObjectTracker(&tracker).WithScheme(scheme).Build()
			writer = reportWriterForTest(fakeRuntimeClient)
			go writer.startWriting()
		})

		AfterEach(func() {
			stopCh <- struct{}{}
		})

		It("should skip writing incomplete reports", func() {
			report := basicLicenseUsageReport{
				minCounts: counts{
					vCPU:  1,
					nodes: 1,
				},
				maxCounts: counts{
					vCPU:  2,
					nodes: 2,
				},
				complete: true,
			}

			// Verify that the report writes to the datastore when it's complete.
			writer.reports <- report
			Eventually(tracker.noCallsMade).WithTimeout(5 * time.Second).Should(BeFalse())

			// Verify that the report does not write to the datastore when it's incomplete.
			tracker.clear()
			report.complete = false
			writer.reports <- report
			Consistently(tracker.noCallsMade).WithTimeout(5 * time.Second).Should(BeTrue())
		})
	})
})
