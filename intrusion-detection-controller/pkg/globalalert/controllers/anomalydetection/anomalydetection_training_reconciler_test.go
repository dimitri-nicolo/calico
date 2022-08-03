// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package anomalydetection

import (
	"context"
	"fmt"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	"github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/globalalert/podtemplate"
	rcache "github.com/projectcalico/calico/kube-controllers/pkg/cache"

	apps "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	fakeK8s "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/util/workqueue"
)

const (
	alertName   = "sample-test"
	clusterName = "ut-cluster"
	namespace   = "ut-namespace"
)

var (
	mockCalicoCLI        calicoclient.Interface
	mockK8sClient        kubernetes.Interface
	mockPodTemplateQuery *podtemplate.MockPodTemplateQuery
	mockrc               rcache.ResourceCache

	ctx    context.Context
	cancel context.CancelFunc
)

var _ = Describe("AnomalyDetection Reconciler", func() {

	now := time.Now()
	lastExecutedTime := now.Add(-2 * time.Second)

	defaultSampleGlobalAlert := &v3.GlobalAlert{
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
	mockDeployment := &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "intrusion-detection-controller",
			Namespace: namespace,
		},
		Spec: apps.DeploymentSpec{
			Template: v1.PodTemplateSpec{},
		},
	}

	BeforeEach(func() {
		mockCalicoCLI = fake.NewSimpleClientset(defaultSampleGlobalAlert)
		mockK8sClient = fakeK8s.NewSimpleClientset(mockDeployment)
		mockPodTemplateQuery = &podtemplate.MockPodTemplateQuery{}
		mockrc = mockResourceCache{}

		ctx, cancel = context.WithCancel(context.Background())
	})

	Context("runInitialTrainingJob", func() {
		It("does not create an initial training job, as training cycle found and detector present", func() {
			adJobTr := adJobTrainingReconciler{
				managementClusterCtx:       ctx,
				k8sClient:                  mockK8sClient,
				calicoCLI:                  mockCalicoCLI,
				podTemplateQuery:           mockPodTemplateQuery,
				trainingCycleResourceCache: mockrc,

				namespace: namespace,

				// key: cluster name
				trainingDetectorsPerCluster: map[string]trainingCycleStatePerCluster{
					"ut-cluster-training-cycle": {
						ClusterName: clusterName,
						CronJob:     nil,
						GlobalAlerts: []*v3.GlobalAlert{
							{
								Spec: v3.GlobalAlertSpec{
									Detector: &v3.DetectorParams{
										Name: "detector1",
									},
								},
							},
							{
								Spec: v3.GlobalAlertSpec{
									Detector: &v3.DetectorParams{
										Name: "detector2",
									},
								},
							},
						},
					},
				}, trainingJobsMutex: sync.Mutex{},
			}

			mcs := TrainingDetectorsRequest{
				ClusterName: clusterName,
				GlobalAlert: &v3.GlobalAlert{
					Spec: v3.GlobalAlertSpec{
						Detector: &v3.DetectorParams{
							Name: "detector1",
						},
					},
				},
			}

			By("attempting to create an initial training job")
			err := adJobTr.runInitialTrainingJob(mcs)
			Expect(err).To(BeNil())

			By("listing the mock jobs present")
			list, err := mockK8sClient.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
			Expect(err).To(BeNil())
			Expect(list.Items).To(BeNil())
		})

		It("create an initial training job for 'detector1'", func() {
			adJobTr := adJobTrainingReconciler{
				managementClusterCtx:       ctx,
				k8sClient:                  mockK8sClient,
				calicoCLI:                  mockCalicoCLI,
				podTemplateQuery:           mockPodTemplateQuery,
				trainingCycleResourceCache: mockrc,

				namespace: namespace,

				// key: cluster name
				trainingDetectorsPerCluster: map[string]trainingCycleStatePerCluster{
					"clusterKey1": {
						ClusterName: clusterName,
						CronJob:     nil,
						GlobalAlerts: []*v3.GlobalAlert{
							{
								Spec: v3.GlobalAlertSpec{
									Detector: &v3.DetectorParams{
										Name: "detector1",
									},
								},
							},
						},
					},
				},
				trainingJobsMutex: sync.Mutex{},
			}

			mcs := TrainingDetectorsRequest{
				ClusterName: clusterName,
				GlobalAlert: &v3.GlobalAlert{
					Spec: v3.GlobalAlertSpec{
						Detector: &v3.DetectorParams{
							Name: "detector1",
						},
					},
				},
			}

			expectedControllerOwner := true
			expectedBlockOwnerDeletion := true
			expectedbackoffLimit := int32(0)
			expectedJob := batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-detector1-initial-training", clusterName),
					Namespace: namespace,
					Labels: map[string]string{
						"tigera.io.detector-cycle": "training",
						"cluster":                  clusterName,
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "apps/v1",
							Kind:               "Deployment",
							Name:               "intrusion-detection-controller",
							UID:                "",
							Controller:         &expectedControllerOwner,
							BlockOwnerDeletion: &expectedBlockOwnerDeletion,
						},
					},
				},
				Spec: batchv1.JobSpec{
					BackoffLimit: &expectedbackoffLimit,
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("%s-detector1-initial-training", clusterName),
							Namespace: namespace,
							Labels: map[string]string{
								"tigera.io.detector-cycle": "training",
								"cluster":                  clusterName,
							},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name:       "adjobs",
									Image:      "",
									Command:    []string{"python3"},
									Args:       []string{"-m", "adj", "train"},
									WorkingDir: "",
									Ports:      nil,
									EnvFrom:    nil,
									Env: []v1.EnvVar{
										{
											Name:      "CLUSTER_NAME",
											Value:     clusterName,
											ValueFrom: nil},
										{
											Name:      "AD_ENABLED_DETECTORS",
											Value:     "detector1",
											ValueFrom: nil,
										},
									},
								},
							},
							RestartPolicy: "Never",
						},
					},
				},
			}

			By("mocking GetPodTemplate output")
			mockPodTemplateQuery.On("GetPodTemplate", ctx, namespace, "tigera.io.detectors.training").Return(&v1.PodTemplate{Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{Containers: []v1.Container{{Name: "adjobs"}}},
			}}, nil)

			By("attempting to create an initial training job")
			err := adJobTr.runInitialTrainingJob(mcs)
			Expect(err).To(BeNil())

			By("listing the mock jobs present")
			list, err := mockK8sClient.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
			Expect(err).To(BeNil())
			Expect(len(list.Items)).To(Equal(1))

			By("verifying the new job in the list is expected")
			job := list.Items[0]
			Expect(job).To(Equal(expectedJob))
		})
	})
})

// Mocks the resource cache.
type mockResourceCache struct {
}

func (rc mockResourceCache) Set(key string, value interface{}) {
	// Not yet implemented.
}

func (rc mockResourceCache) Get(key string) (interface{}, bool) {
	// Not yet implemented.
	return nil, false
}

func (rc mockResourceCache) Prime(key string, value interface{}) {
	// Not yet implemented.
}

func (rc mockResourceCache) Delete(key string) {
	// Not yet implemented.
}

func (rc mockResourceCache) Clean(key string) {
	// Not yet implemented.
}

func (rc mockResourceCache) ListKeys() []string {
	// Not yet implemented.
	return nil
}

func (rc mockResourceCache) Run(reconcilerPeriod string) {
	// Not yet implemented.
}

func (rc mockResourceCache) GetQueue() workqueue.RateLimitingInterface {
	// Not yet implemented.
	return nil
}
