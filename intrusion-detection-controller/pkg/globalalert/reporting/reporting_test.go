package reporting

import (
	"context"
	"errors"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	"github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

const (
	alertName   = "sample-test"
	clusterName = "ut-cluster"
	namespace   = "ut-namespace"
)

var (
	mockCalicoCLI calicoclient.Interface
	ctx           context.Context
	cancel        context.CancelFunc
)

var _ = Describe("Reporting", func() {
	now := time.Now()
	lastExecutedTime := now.Add(-2 * time.Second)

	defaultSampleGlobalAlert := v3.GlobalAlert{
		ObjectMeta: metav1.ObjectMeta{
			Name: alertName,
		},
		Spec: v3.GlobalAlertSpec{
			Type: v3.GlobalAlertTypeAnomalyDetection,
			Detector: &v3.DetectorParams{
				Name: "port-scan",
			},
			Description: fmt.Sprintf("test anomalyDetection alert: %s", alertName),
			Severity:    100,
			Period:      &metav1.Duration{Duration: 5 * time.Second},
			Lookback:    &metav1.Duration{Duration: 1 * time.Second},
		},
		Status: v3.GlobalAlertStatus{
			LastUpdate:   &metav1.Time{Time: now},
			Active:       false,
			Healthy:      true,
			LastExecuted: &metav1.Time{Time: lastExecutedTime},
		},
	}

	BeforeEach(func() {
		mockCalicoCLI = fake.NewSimpleClientset(&defaultSampleGlobalAlert)

		ctx, cancel = context.WithCancel(context.Background())
	})

	Context("UpdateAlertStatus", func() {
		It("retrieves currently deployed GlobalAlertStatus", func() {
			testGlobalAlert := defaultSampleGlobalAlert
			prevExecTime := now.Add(-10 * time.Second)
			statusToUpdate := v3.GlobalAlertStatus{
				LastUpdate:   &metav1.Time{Time: prevExecTime},
				Active:       true,
				Healthy:      false,
				LastExecuted: &metav1.Time{Time: prevExecTime},
			}
			testGlobalAlert.Status = statusToUpdate

			err := UpdateGlobalAlertStatusWithRetryOnConflict(&testGlobalAlert, clusterName, mockCalicoCLI, ctx)
			Expect(err).To(BeNil())

			updatedAlert, err := mockCalicoCLI.ProjectcalicoV3().GlobalAlerts().Get(ctx, testGlobalAlert.Name, metav1.GetOptions{})
			Expect(err).To(BeNil())
			Expect(updatedAlert.Status).To(Equal(statusToUpdate))
		})
	})

	Context("WatchAndReportJobStatus", func() {
		It("returns GlobalAlertStatus with no error for deleted events", func() {

			fakewatcher := watch.NewFake()
			testJobName := "testJobName"
			testJob := batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testJobName,
					Namespace: namespace,
				},
			}

			result := v3.GlobalAlertStatus{}
			go func() {
				var err error
				result, err = WatchAndReportJobStatus(fakewatcher, testJobName, 5*time.Second)
				Expect(err).To(BeNil())
			}()
			fakewatcher.Delete(testJob.DeepCopyObject())
			time.Sleep(1 * time.Second)

			Expect(result.Healthy).To(BeTrue())
			Expect(result.Active).To(BeTrue())
		})

		It("returns GlobalAlertStatus as an error for an error event", func() {

			fakewatcher := watch.NewFake()
			testJobName := "testJobName"
			testJob := batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testJobName,
					Namespace: namespace,
				},
			}

			result := v3.GlobalAlertStatus{}
			var err error
			go func() {
				result, err = WatchAndReportJobStatus(fakewatcher, testJobName, 5*time.Second)
			}()
			fakewatcher.Error(testJob.DeepCopyObject())
			time.Sleep(1 * time.Second)

			Expect(result.Healthy).To(BeFalse())
			Expect(result.Active).To(BeFalse())
			Expect(result.ErrorConditions[0].Message).To(Equal(err.Error()))
		})

		It("returns GlobalAlertStatus as an error if it did not receive even within timeout", func() {

			fakewatcher := watch.NewFake()
			testJobName := "testJobName"
			result := v3.GlobalAlertStatus{}
			var err error
			go func() {
				result, err = WatchAndReportJobStatus(fakewatcher, testJobName, 100*time.Millisecond)
			}()
			time.Sleep(1 * time.Second)

			Expect(result.Healthy).To(BeFalse())
			Expect(result.Active).To(BeFalse())
			Expect(result.ErrorConditions[0].Message).To(Equal(err.Error()))
		})
	})

	Context("GetGlobalAlertStatus", func() {
		It("GetGlobalAlertErrorStatus", func() {
			testError := errors.New("testError")

			now := time.Now()
			result := GetGlobalAlertErrorStatus(testError)

			Expect(result.Healthy).To(BeFalse())
			Expect(result.Active).To(BeFalse())
			Expect(result.LastUpdate.Time.After(now)).To(BeTrue())
			Expect(result.ErrorConditions[0].Message).To(Equal(testError.Error()))
		})

		It("GetGlobalAlertSuccessStatus", func() {

			now := time.Now()
			result := GetGlobalAlertSuccessStatus()

			Expect(result.Healthy).To(BeTrue())
			Expect(result.Active).To(BeTrue())
			Expect(result.LastUpdate.Time.After(now)).To(BeTrue())
			Expect(result.LastEvent.Time.After(now)).To(BeTrue())

			Expect(result.ErrorConditions).To(BeNil())
		})
	})
})
