package alert

import (
	"context"
	"fmt"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	adj "github.com/projectcalico/calico/intrusion-detection/controller/pkg/globalalert/anomalydetection"
	es "github.com/projectcalico/calico/intrusion-detection/controller/pkg/globalalert/elastic"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"
)

const (
	alertName = "sample-test"
)

var _ = Describe("GlobalAlert", func() {
	Context("Alert Execution", func() {
		It("on default type UserDefined should happen based on last executed time of alert", func() {
			now := time.Now()
			lastExecutedTime := now.Add(-2 * time.Second)

			// Set the lastExecutedTime to 2s behind current time
			// Set spec.Period to be 5s
			globalAlert := &v3.GlobalAlert{
				ObjectMeta: metav1.ObjectMeta{
					Name: alertName,
				},
				Spec: v3.GlobalAlertSpec{
					Description: fmt.Sprintf("test alert: %s", alertName),
					Severity:    100,
					DataSet:     "flows",
					Metric:      "count",
					Threshold:   100,
					Condition:   "gt",
					Query:       "action=allow",
					Period:      &metav1.Duration{Duration: 5 * time.Second},
				},
				Status: v3.GlobalAlertStatus{
					LastUpdate:   &metav1.Time{Time: now},
					Active:       true,
					Healthy:      true,
					LastExecuted: &metav1.Time{Time: lastExecutedTime},
				},
			}

			fakeClient := fake.NewSimpleClientset(globalAlert)
			mockElasticSvc := &es.MockService{}
			mockADJSvc := &adj.MockService{}
			a := &Alert{
				alert:       globalAlert,
				clusterName: "test-cluster",
				es:          mockElasticSvc,
				adj:         mockADJSvc,
				calicoCLI:   fakeClient,
			}

			ctx, cancelFunc := context.WithCancel(context.Background())
			var wg sync.WaitGroup
			wg.Add(3)

			mockElasticSvc.On("DeleteElasticWatchers", mock.Anything).Return()
			mockElasticSvc.On("ExecuteAlert", mock.Anything).Run(func(args mock.Arguments) {
				for _, c := range mockElasticSvc.ExpectedCalls {
					if c.Method == "ExecuteAlert" {
						wg.Done()
						c.ReturnArguments = mock.Arguments{v3.GlobalAlertStatus{
							LastExecuted: &metav1.Time{Time: time.Now()},
						}}
					}
				}
			})

			// first call to ExecuteAlert func should happen 5 seconds after the current time
			// second call and subsequent call should happen after 10 seconds interval
			go a.Execute(ctx)

			// Calls to onInterval func should happen at 3s, 8s and 13s
			// Wait for 3 calls to onInterval func and cancel the context
			time.AfterFunc(15*time.Second, func() {
				cancelFunc()
			})

			wg.Wait()
			updatedAlert, err := fakeClient.ProjectcalicoV3().GlobalAlerts().Get(ctx, a.alert.Name, metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(updatedAlert.Status.LastExecuted.UnixNano()).To(BeNumerically(">", lastExecutedTime.UnixNano()))
		})

		It("on default type UserDefined should happen immediately if last executed time of alert was before spec.Period duration", func() {
			now := time.Now()
			lastExecutedTime := now.Add(-20 * time.Second)

			// Set the lastExecutedTime to 20s behind current time
			// Set spec.Period to be 10s
			globalAlert := &v3.GlobalAlert{
				ObjectMeta: metav1.ObjectMeta{
					Name: alertName,
				},
				Spec: v3.GlobalAlertSpec{
					Description: fmt.Sprintf("test alert: %s", alertName),
					Severity:    100,
					DataSet:     "flows",
					Metric:      "count",
					Threshold:   100,
					Condition:   "gt",
					Query:       "action=allow",
					Period:      &metav1.Duration{Duration: 10 * time.Second},
				},
				Status: v3.GlobalAlertStatus{
					LastUpdate:   &metav1.Time{Time: now},
					Active:       true,
					Healthy:      true,
					LastExecuted: &metav1.Time{Time: lastExecutedTime},
				},
			}

			fakeClient := fake.NewSimpleClientset(globalAlert)
			mockElasticSvc := &es.MockService{}
			mockADJSvc := &adj.MockService{}

			a := &Alert{
				alert:       globalAlert,
				clusterName: "test-cluster",
				calicoCLI:   fakeClient,
				es:          mockElasticSvc,
				adj:         mockADJSvc,
			}

			var wg sync.WaitGroup
			wg.Add(2)
			firstOnIntervalCall := true
			mockElasticSvc.On("DeleteElasticWatchers", mock.Anything).Return()
			mockElasticSvc.On("ExecuteAlert", mock.Anything).Run(func(args mock.Arguments) {
				for _, c := range mockElasticSvc.ExpectedCalls {
					if c.Method == "ExecuteAlert" {
						wg.Done()
						diff := time.Now().Sub(now)
						if firstOnIntervalCall {
							firstOnIntervalCall = false
							Expect(diff.Seconds()).To(BeNumerically("<", 6))
						} else {
							Expect(diff.Seconds()).To(BeNumerically("<", 16))
						}
						c.ReturnArguments = mock.Arguments{v3.GlobalAlertStatus{
							LastExecuted: &metav1.Time{Time: time.Now()},
						}}
					}
				}
			})

			// first call to onInterval func should happen after 5sec
			// second call and all subsequent call should happen after 10 seconds interval
			ctx, cancelFunc := context.WithCancel(context.Background())
			go a.Execute(ctx)

			// Calls to onInterval func should happen at 5s and 15s
			// Wait for 2 calls to onInterval func and cancel the context
			time.AfterFunc(17*time.Second, func() {
				cancelFunc()
			})

			wg.Wait()
			updatedAlert, err := fakeClient.ProjectcalicoV3().GlobalAlerts().Get(ctx, a.alert.Name, metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(updatedAlert.Status.LastExecuted.UnixNano()).To(BeNumerically(">", lastExecutedTime.UnixNano()))
		})

		It("on type AnomalyDetection should starts the anomaly detection service at time of alert", func() {
			now := time.Now()
			lastExecutedTime := now.Add(-2 * time.Second)

			globalAlert := &v3.GlobalAlert{
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

			fakeClient := fake.NewSimpleClientset(globalAlert)
			mockElasticSvc := &es.MockService{}
			mockADJSvc := &adj.MockService{}
			a := &Alert{
				alert:                  globalAlert,
				clusterName:            "test-cluster",
				es:                     mockElasticSvc,
				adj:                    mockADJSvc,
				calicoCLI:              fakeClient,
				enableAnomalyDetection: true,
			}

			ctx, cancel := context.WithCancel(context.Background())

			mockADJSvc.On("Start", mock.Anything, mock.Anything).Return(
				v3.GlobalAlertStatus{
					LastUpdate:   &metav1.Time{Time: now.Add(2 * time.Second)},
					Active:       true,
					Healthy:      true,
					LastExecuted: &metav1.Time{Time: now.Add(2 * time.Second)},
				},
			)

			go a.Execute(ctx)

			// wait until ExecuteAlert calls Start and returns mock status
			time.Sleep(500 * time.Millisecond)

			updatedAlert, err := fakeClient.ProjectcalicoV3().GlobalAlerts().Get(ctx, a.alert.Name, metav1.GetOptions{})
			cancel()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(updatedAlert.Status.Active).To(BeTrue())
			Expect(updatedAlert.Status.LastExecuted.UnixNano()).To(BeNumerically(">", lastExecutedTime.UnixNano()))
		})
	})
})
