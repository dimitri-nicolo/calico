package podtemplate

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	TrainingJobLabels = func() map[string]string {
		return map[string]string{
			"tigera.io.detector-cycle": "training",
		}
	}
)

var _ = Describe("AD PodTemplate", func() {
	var (
		defaultPodTemplate *v1.PodTemplate
	)

	BeforeEach(func() {
		defaultPodTemplate = &v1.PodTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mock-adjob-podtemplate",
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mock-adjob-podtemplate-spec",
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name: ADJobsContainerName,
						},
					},
				},
			},
		}
	})

	Context("CreateJobFromPodTemplate", func() {
		It("create a job ", func() {
			name := "job-name1"
			namespace := "job-namespace1"
			clusterName := "cluster-name1"
			trainingLabels := TrainingJobLabels()
			trainingLabels["cluster"] = clusterName
			bfl := int32(10)
			detectorList := "dector1,detector2,detector3"

			expectedBackoffLimit := int32(10)
			expectedJob := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "job-name1",
					Namespace: "job-namespace1",
					Labels: map[string]string{
						"tigera.io.detector-cycle": "training",
						"cluster":                  "cluster-name1",
					},
				},
				Spec: batchv1.JobSpec{
					BackoffLimit: &expectedBackoffLimit,
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "job-name1",
							Namespace: "job-namespace1",
							Labels: map[string]string{
								"tigera.io.detector-cycle": "training",
								"cluster":                  "cluster-name1",
							},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name:    "adjobs",
									Command: []string{"python3"},
									Args:    []string{"-m", "adj", "train"},
									Env: []v1.EnvVar{
										{
											Name:      "CLUSTER_NAME",
											Value:     "cluster-name1",
											ValueFrom: nil,
										},
										{
											Name:      "AD_ENABLED_DETECTORS",
											Value:     "dector1,detector2,detector3",
											ValueFrom: nil,
										},
									},
								},
							},
						},
					},
				},
			}
			// add specs for training cycle
			err := DecoratePodTemplateForTrainingCycle(defaultPodTemplate, clusterName, detectorList)
			Expect(err).To(BeNil())

			job := CreateJobFromPodTemplate(name, namespace, trainingLabels, *defaultPodTemplate, &bfl)
			Expect(job).To(Equal(expectedJob))
		})
	})
})
