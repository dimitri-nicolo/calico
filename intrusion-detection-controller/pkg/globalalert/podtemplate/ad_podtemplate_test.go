package podtemplate

import (
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("AD PodTemplate", func() {

	adTestGlobalAlert := v3.GlobalAlert{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mock-adjob-podtemplate",
		},
		Spec: v3.GlobalAlertSpec{
			Detector: &v3.DetectorParams{
				Name: "test-detector",
			},
		},
	}
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

	Context("DecoratePodTemplateForTrainingCycle", func() {
		It("adds ADJob specific commands, args and envVar to the training podtemplate", func() {
			testPT := *defaultPodTemplate
			clusterName := "testCluster"
			cycle := ADJobTrainCycleArg
			detectors := "test-job0, test-job1"

			err := DecoratePodTemplateForTrainingCycle(&testPT, clusterName, detectors)
			Expect(err).To(BeNil())
			adContainer, err := findContainer(&(testPT.Template.Spec.Containers), ADJobsContainerName)
			Expect(err).To(BeNil())

			Expect(adContainer.Env).To(ContainElements(
				v1.EnvVar{
					Name:      "CLUSTER_NAME",
					Value:     clusterName,
					ValueFrom: nil,
				},
				v1.EnvVar{
					Name:      "AD_ENABLED_JOBS",
					Value:     detectors,
					ValueFrom: nil,
				},
				v1.EnvVar{
					Name:      "AD_train_default_query_time_duration",
					Value:     DefaultADDetectorTrainingPeriod.String(),
					ValueFrom: nil,
				},
				v1.EnvVar{
					Name:  "AD_USE_INTERNAL_SCHEDULER",
					Value: "False",
				},
			))
			Expect(adContainer.Command).To(Equal(ADJobStartupCommand()))
			Expect(adContainer.Args).To(Equal(append(ADJobStartupArgs(), cycle)))
		})
	})

	Context("DecoratePodTemplateForDetectionCycle", func() {
		It("adds ADJob specific commands, args and envVar to the detection podtemplate", func() {
			testPT := *defaultPodTemplate
			clusterName := "testCluster"
			cycle := ADJobDetectCycleArg

			testGlobalAlert := adTestGlobalAlert.DeepCopy()
			testGlobalAlert.Spec.Severity = 95
			testGlobalAlert.Spec.Period = &metav1.Duration{Duration: 20 * time.Minute}

			err := DecoratePodTemplateForDetectionCycle(&testPT, clusterName, *testGlobalAlert)
			Expect(err).To(BeNil())
			adContainer, err := findContainer(&(testPT.Template.Spec.Containers), ADJobsContainerName)
			Expect(err).To(BeNil())

			Expect(adContainer.Env).To(ContainElements(
				v1.EnvVar{
					Name:      "CLUSTER_NAME",
					Value:     clusterName,
					ValueFrom: nil,
				},
				v1.EnvVar{
					Name:      "AD_ENABLED_JOBS",
					Value:     testGlobalAlert.Spec.Detector.Name,
					ValueFrom: nil,
				},
				v1.EnvVar{
					Name:      "AD_detect_default_query_time_duration",
					Value:     testGlobalAlert.Spec.Period.Duration.String(),
					ValueFrom: nil,
				},
				v1.EnvVar{
					Name:      "AD_ALERT_SEVERITY",
					Value:     strconv.Itoa(testGlobalAlert.Spec.Severity),
					ValueFrom: nil,
				},
				v1.EnvVar{
					Name:  "AD_DETECTION_VERIFY_MODEL_EXISTENCE",
					Value: "True",
				},
				v1.EnvVar{
					Name:  "AD_USE_INTERNAL_SCHEDULER",
					Value: "False",
				},
			))
			Expect(adContainer.Command).To(Equal(ADJobStartupCommand()))
			Expect(adContainer.Args).To(Equal(append(ADJobStartupArgs(), cycle)))
		})

		It("adds ADJob specific commands, args and default envVar to the detection podtemplate if certain fields are not specified in the GlobalAlert", func() {
			testPT := *defaultPodTemplate
			clusterName := "testCluster"
			cycle := ADJobDetectCycleArg

			err := DecoratePodTemplateForDetectionCycle(&testPT, clusterName, adTestGlobalAlert)
			Expect(err).To(BeNil())
			adContainer, err := findContainer(&(testPT.Template.Spec.Containers), ADJobsContainerName)
			Expect(err).To(BeNil())

			Expect(adContainer.Env).To(ContainElements(
				v1.EnvVar{
					Name:      "CLUSTER_NAME",
					Value:     clusterName,
					ValueFrom: nil,
				},
				v1.EnvVar{
					Name:      "AD_ENABLED_JOBS",
					Value:     adTestGlobalAlert.Spec.Detector.Name,
					ValueFrom: nil,
				},
				v1.EnvVar{
					Name:      "AD_detect_default_query_time_duration",
					Value:     DefaultCronJobDetectionSchedule.String(),
					ValueFrom: nil,
				},
				v1.EnvVar{
					Name:      "AD_ALERT_SEVERITY",
					Value:     strconv.Itoa(DefaultDetectionAlertSeverity),
					ValueFrom: nil,
				},
				v1.EnvVar{
					Name:  "AD_DETECTION_VERIFY_MODEL_EXISTENCE",
					Value: "True",
				},
				v1.EnvVar{
					Name:  "AD_USE_INTERNAL_SCHEDULER",
					Value: "False",
				},
			))
			Expect(adContainer.Command).To(Equal(ADJobStartupCommand()))
			Expect(adContainer.Args).To(Equal(append(ADJobStartupArgs(), cycle)))
		})
	})

	Context("decorateBaseADPodTemplate", func() {
		It("errors when conainer is not found", func() {
			testPT := *defaultPodTemplate
			testPT.Template.Spec.Containers[0].Name = "unrecognized name"
			clusterName := "testCluster"

			err := decorateBaseADPodTemplate(clusterName, nil)
			Expect(err).ToNot(BeNil())
		})
	})
})
