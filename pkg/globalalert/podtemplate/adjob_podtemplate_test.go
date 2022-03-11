package podtemplate

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testPeriod = 15 * time.Minute
)

var _ = Describe("ADJob PodTemplate", func() {
	defaultPodTemplate := &v1.PodTemplate{
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

	Context("DecorateADJobPodTemplate", func() {
		It("adds ADJob specific commands, args and envVar to the training podtemplate", func() {
			testPT := *defaultPodTemplate
			clusterName := "testCluster"
			cycle := ADJobTrainCycleArg
			job := "test-job"

			err := DecoratePodTemplateForADDetectorCycle(&testPT, clusterName, cycle, job, testPeriod.String())
			Expect(err).To(BeNil())
			containerResultIndex := getContainerIndex(testPT.Template.Spec.Containers, ADJobsContainerName)

			Expect(testPT.Template.Spec.Containers[containerResultIndex].Env).To(ContainElements(
				v1.EnvVar{
					Name:      "CLUSTER_NAME",
					Value:     clusterName,
					ValueFrom: nil,
				},
				v1.EnvVar{
					Name:      "AD_ENABLED_JOBS",
					Value:     job,
					ValueFrom: nil,
				},
				v1.EnvVar{
					Name:      "AD_train_default_query_time_duration",
					Value:     testPeriod.String(),
					ValueFrom: nil,
				},
			))
			Expect(testPT.Template.Spec.Containers[containerResultIndex].Command).To(Equal(ADJobStartupCommand()))
			Expect(testPT.Template.Spec.Containers[containerResultIndex].Args).To(Equal(append(ADJobStartupArgs(), cycle)))
		})

		It("adds ADJob specific commands, args and envVar to the detection podtemplate", func() {
			testPT := *defaultPodTemplate
			clusterName := "testCluster"
			cycle := ADJobDetectCycleArg
			job := "test-job"

			err := DecoratePodTemplateForADDetectorCycle(&testPT, clusterName, cycle, job, testPeriod.String())
			Expect(err).To(BeNil())
			containerResultIndex := getContainerIndex(testPT.Template.Spec.Containers, ADJobsContainerName)

			Expect(testPT.Template.Spec.Containers[containerResultIndex].Env).To(ContainElements(
				v1.EnvVar{
					Name:      "CLUSTER_NAME",
					Value:     clusterName,
					ValueFrom: nil,
				},
				v1.EnvVar{
					Name:      "AD_ENABLED_JOBS",
					Value:     job,
					ValueFrom: nil,
				},
				v1.EnvVar{
					Name:      "AD_detect_default_query_time_duration",
					Value:     testPeriod.String(),
					ValueFrom: nil,
				},
				v1.EnvVar{
					Name:  "AD_DETECTION_VERIFY_MODEL_EXISTENCE",
					Value: "True",
				},
			))
			Expect(testPT.Template.Spec.Containers[containerResultIndex].Command).To(Equal(ADJobStartupCommand()))
			Expect(testPT.Template.Spec.Containers[containerResultIndex].Args).To(Equal(append(ADJobStartupArgs(), cycle)))
		})

		It("errors when conainer is not found", func() {
			testPT := *defaultPodTemplate
			testPT.Template.Spec.Containers[0].Name = "unreconized name"
			clusterName := "testCluster"
			cycle := ADJobDetectCycleArg
			job := "test-job"

			err := DecoratePodTemplateForADDetectorCycle(&testPT, clusterName, cycle, job, testPeriod.String())
			Expect(err).ToNot(BeNil())
		})

		It("errors when cycle is not recognized as detection or training", func() {
			testPT := *defaultPodTemplate
			clusterName := "testCluster"
			cycle := "unrecognized"
			job := "test-job"

			err := DecoratePodTemplateForADDetectorCycle(&testPT, clusterName, cycle, job, testPeriod.String())
			Expect(err).ToNot(BeNil())
		})
	})
})
