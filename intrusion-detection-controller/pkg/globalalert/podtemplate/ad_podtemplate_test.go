package podtemplate

import (
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("AD PodTemplate", func() {
	const (
		testClusterName = "testCluster"
		noTenant        = ""
		testTenant      = "testTenant"
	)

	var (
		adTestGlobalAlert  v3.GlobalAlert
		defaultPodTemplate *v1.PodTemplate
	)

	BeforeEach(func() {
		adTestGlobalAlert = v3.GlobalAlert{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mock-adjob-podtemplate",
			},
			Spec: v3.GlobalAlertSpec{
				Detector: &v3.DetectorParams{
					Name: "test-detector",
				},
			},
		}
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

	Context("DecoratePodTemplateForTrainingCycle", func() {
		DescribeTable("errors when container is not found", func(tenant string) {
			testPT := *defaultPodTemplate
			testPT.Template.Spec.Containers[0].Name = "unreconized name"
			detectors := "test-job0, test-job1"

			err := DecoratePodTemplateForTrainingCycle(&testPT, testClusterName, tenant, detectors)
			Expect(err).ToNot(BeNil())
			Expect(err).To(Equal(ErrADContainerNotFound))
		},
			Entry("no tenant", noTenant),
			Entry("with tenant", testTenant),
		)

		DescribeTable("adds ADJob specific commands, args and envVar to the training podtemplate", func(tenant string) {
			testPT := *defaultPodTemplate
			cycle := ADJobTrainCycleArg
			detectors := "test-job0, test-job1"

			err := DecoratePodTemplateForTrainingCycle(&testPT, testClusterName, tenant, detectors)
			Expect(err).To(BeNil())
			adContainer, err := findContainer(&(testPT.Template.Spec.Containers), ADJobsContainerName)
			Expect(err).To(BeNil())

			Expect(adContainer.Env).To(ContainElements(
				v1.EnvVar{
					Name:      "CLUSTER_NAME",
					Value:     testClusterName,
					ValueFrom: nil,
				},
				v1.EnvVar{
					Name:      "TENANT_ID",
					Value:     tenant,
					ValueFrom: nil,
				},
				v1.EnvVar{
					Name:      "AD_ENABLED_DETECTORS",
					Value:     detectors,
					ValueFrom: nil,
				},
			))
			Expect(adContainer.Command).To(Equal(ADJobStartupCommand()))
			Expect(adContainer.Args).To(Equal(append(ADJobStartupArgs(), cycle)))
		},
			Entry("no tenant", noTenant),
			Entry("with tenant", testTenant),
		)
	})

	Context("DecoratePodTemplateForDetectionCycle", func() {
		DescribeTable("errors when container is not found", func(tenant string) {
			testPT := *defaultPodTemplate
			testPT.Template.Spec.Containers[0].Name = "unreconized name"

			err := DecoratePodTemplateForDetectionCycle(&testPT, testClusterName, tenant, adTestGlobalAlert)
			Expect(err).ToNot(BeNil())
			Expect(err).To(Equal(ErrADContainerNotFound))
		},
			Entry("no tenant", noTenant),
			Entry("with tenant", testTenant),
		)

		DescribeTable("adds ADJob specific commands, args and envVar to the detection podtemplate", func(tenant string) {
			testPT := *defaultPodTemplate
			cycle := ADJobDetectCycleArg

			testGlobalAlert := adTestGlobalAlert.DeepCopy()
			testGlobalAlert.Spec.Severity = 95
			testGlobalAlert.Spec.Period = &metav1.Duration{Duration: 20 * time.Minute}

			err := DecoratePodTemplateForDetectionCycle(&testPT, testClusterName, tenant, *testGlobalAlert)
			Expect(err).To(BeNil())
			adContainer, err := findContainer(&(testPT.Template.Spec.Containers), ADJobsContainerName)
			Expect(err).To(BeNil())

			Expect(adContainer.Env).To(ContainElements(
				v1.EnvVar{
					Name:      "CLUSTER_NAME",
					Value:     testClusterName,
					ValueFrom: nil,
				},
				v1.EnvVar{
					Name:      "TENANT_ID",
					Value:     tenant,
					ValueFrom: nil,
				},

				v1.EnvVar{
					Name:      "AD_ENABLED_DETECTORS",
					Value:     testGlobalAlert.Spec.Detector.Name,
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
			))
			Expect(adContainer.Command).To(Equal(ADJobStartupCommand()))
			Expect(adContainer.Args).To(Equal(append(ADJobStartupArgs(), cycle)))
		},
			Entry("no tenant", noTenant),
			Entry("with tenant", testTenant),
		)

		DescribeTable("adds ADJob specific commands, args and default envVar to the detection podtemplate if certain fields are not specified in the GlobalAlert", func(tenant string) {
			testPT := *defaultPodTemplate
			cycle := ADJobDetectCycleArg

			err := DecoratePodTemplateForDetectionCycle(&testPT, testClusterName, tenant, adTestGlobalAlert)
			Expect(err).To(BeNil())
			adContainer, err := findContainer(&(testPT.Template.Spec.Containers), ADJobsContainerName)
			Expect(err).To(BeNil())

			Expect(adContainer.Env).To(ContainElements(
				v1.EnvVar{
					Name:      "CLUSTER_NAME",
					Value:     testClusterName,
					ValueFrom: nil,
				},
				v1.EnvVar{
					Name:      "TENANT_ID",
					Value:     tenant,
					ValueFrom: nil,
				},

				v1.EnvVar{
					Name:      "AD_ENABLED_DETECTORS",
					Value:     adTestGlobalAlert.Spec.Detector.Name,
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
			))
			Expect(adContainer.Command).To(Equal(ADJobStartupCommand()))
			Expect(adContainer.Args).To(Equal(append(ADJobStartupArgs(), cycle)))
		},
			Entry("no tenant", noTenant),
			Entry("with tenant", testTenant),
		)
	})

	Context("decorateBaseADPodTemplate", func() {
		DescribeTable("errors when container is not found", func(tenant string) {
			testPT := *defaultPodTemplate
			testPT.Template.Spec.Containers[0].Name = "unrecognized name"
			clusterName := "testCluster"

			err := decorateBaseADPodTemplate(clusterName, tenant, nil)
			Expect(err).ToNot(BeNil())
		},
			Entry("no tenant", noTenant),
			Entry("with tenant", testTenant),
		)
	})
})
