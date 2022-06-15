package podtemplate

import (
	"fmt"
	"strconv"
	"time"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	v1 "k8s.io/api/core/v1"
)

const (
	ADJobsContainerName = "adjobs"
	AllADJobsKeyword    = "all"

	ADJobTrainCycleArg  = "train"
	ADJobDetectCycleArg = "detect"

	DefaultADDetectorTrainingPeriod = 24 * time.Hour
	DefaultCronJobDetectionSchedule = 15 * time.Minute

	DefaultDetectionAlertSeverity = 100
)

var (
	ADJobStartupCommand = func() []string {
		return []string{"python3"}
	}

	ADJobStartupArgs = func() []string {
		return []string{"-m", "adj"}
	}
)

func DecoratePodTemplateForTrainingCycle(adJobPT *v1.PodTemplate, clusterName, detectors string) error {
	adJobContainerIndex := GetContainerIndex(adJobPT.Template.Spec.Containers, ADJobsContainerName)

	args := append(ADJobStartupArgs(), ADJobTrainCycleArg)
	adJobPT.Template.Spec.Containers[adJobContainerIndex].Command = ADJobStartupCommand()
	adJobPT.Template.Spec.Containers[adJobContainerIndex].Args = args

	err := decorateBaseADPodTemplate(adJobPT, clusterName, adJobContainerIndex)
	if err != nil {
		return err
	}

	if len(detectors) > 0 {
		adJobPT.Template.Spec.Containers[adJobContainerIndex].Env = append(
			adJobPT.Template.Spec.Containers[adJobContainerIndex].Env,
			v1.EnvVar{
				Name:  "AD_ENABLED_JOBS",
				Value: detectors,
			},
		)
	}

	adJobPT.Template.Spec.Containers[adJobContainerIndex].Env = append(
		adJobPT.Template.Spec.Containers[adJobContainerIndex].Env,
		v1.EnvVar{
			Name:  "AD_train_default_query_time_duration",
			Value: DefaultADDetectorTrainingPeriod.String(),
		},
	)

	return nil

}

func DecoratePodTemplateForDetectionCycle(adJobPT *v1.PodTemplate, clusterName string, globalAlert v3.GlobalAlert) error {
	adJobContainerIndex := GetContainerIndex(adJobPT.Template.Spec.Containers, ADJobsContainerName)

	args := append(ADJobStartupArgs(), ADJobDetectCycleArg)
	adJobPT.Template.Spec.Containers[adJobContainerIndex].Command = ADJobStartupCommand()
	adJobPT.Template.Spec.Containers[adJobContainerIndex].Args = args

	err := decorateBaseADPodTemplate(adJobPT, clusterName, adJobContainerIndex)
	if err != nil {
		return err
	}

	if globalAlert.Spec.Detector != nil && len(globalAlert.Spec.Detector.Name) > 0 {
		adJobPT.Template.Spec.Containers[adJobContainerIndex].Env = append(
			adJobPT.Template.Spec.Containers[adJobContainerIndex].Env,
			v1.EnvVar{
				Name:  "AD_ENABLED_JOBS",
				Value: globalAlert.Spec.Detector.Name,
			},
		)
	}

	detectionSchedule := DefaultCronJobDetectionSchedule
	if globalAlert.Spec.Period != nil {
		detectionSchedule = globalAlert.Spec.Period.Duration
	}

	detectionSeverity := DefaultDetectionAlertSeverity
	if globalAlert.Spec.Severity != 0 {
		detectionSeverity = globalAlert.Spec.Severity
	}

	adJobPT.Template.Spec.Containers[adJobContainerIndex].Env = append(
		adJobPT.Template.Spec.Containers[adJobContainerIndex].Env,
		v1.EnvVar{
			Name:  "AD_detect_default_query_time_duration",
			Value: detectionSchedule.String(),
		},
		v1.EnvVar{
			Name:  "AD_ALERT_SEVERITY",
			Value: strconv.Itoa(detectionSeverity),
		},

		// set in detection cycle to indicate anomaly detection job image to train a model
		// in a detection cycle if one does not already exist for the detectors in AD_ENABLED_JOBS
		v1.EnvVar{
			Name:  "AD_DETECTION_VERIFY_MODEL_EXISTENCE",
			Value: "True",
		},
	)

	return nil
}

// decorateBaseADPodTemplate adds required fields and environment variables for a PodSpec from the provided
// v1.PodTemplate common to both detection and training cycles
func decorateBaseADPodTemplate(adJobPT *v1.PodTemplate, clusterName string, adJobContainerIndex int) error {

	if adJobContainerIndex == -1 {
		return fmt.Errorf("unable to retrtieve container for %s", ADJobsContainerName)
	}

	adJobPT.Template.Spec.Containers[adJobContainerIndex].Env = append(
		adJobPT.Template.Spec.Containers[adJobContainerIndex].Env,
		v1.EnvVar{
			Name:  "CLUSTER_NAME",
			Value: clusterName,
		},
		v1.EnvVar{
			Name:  "AD_USE_INTERNAL_SCHEDULER",
			Value: "False",
		},
	)

	return nil
}

func GetContainerIndex(containers []v1.Container, name string) int {
	for i, container := range containers {
		if container.Name == name {
			return i
		}
	}
	return -1
}
