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

	ErrADContainerNotFound = fmt.Errorf("unable to retrieve container for %s", ADJobsContainerName)
)

// DecoratePodTemplateForTrainingCycle adds the appropriate labels and env_vars to setup the v1.PodTemplate for
// an AD training cycle
func DecoratePodTemplateForTrainingCycle(adJobPT *v1.PodTemplate, clusterName, detectors string) error {
	adContainer, err := getContainer(adJobPT.Template.Spec.Containers, ADJobsContainerName)
	if err != nil {
		return err
	}

	err = decorateBaseADPodTemplate(clusterName, &adContainer)
	if err != nil {
		return err
	}

	args := append(ADJobStartupArgs(), ADJobTrainCycleArg)
	adContainer.Command = ADJobStartupCommand()
	adContainer.Args = args

	if len(detectors) > 0 {
		adContainer.Env = append(
			adContainer.Env,
			v1.EnvVar{
				Name:  "AD_ENABLED_JOBS",
				Value: detectors,
			},
		)
	}

	adContainer.Env = append(
		adContainer.Env,
		v1.EnvVar{
			Name:  "AD_train_default_query_time_duration",
			Value: DefaultADDetectorTrainingPeriod.String(),
		},
	)

	adContainers := []v1.Container{adContainer}
	adJobPT.Template.Spec.Containers = adContainers

	return nil
}

// DecoratePodTemplateForDetectionCycle adds the appropriate labels and env_vars to setup the v1.PodTemplate for
// an AD detection cycle
func DecoratePodTemplateForDetectionCycle(adJobPT *v1.PodTemplate, clusterName string, globalAlert v3.GlobalAlert) error {
	adContainer, err := getContainer(adJobPT.Template.Spec.Containers, ADJobsContainerName)
	if err != nil {
		return err
	}

	err = decorateBaseADPodTemplate(clusterName, &adContainer)
	if err != nil {
		return err
	}

	args := append(ADJobStartupArgs(), ADJobDetectCycleArg)
	adContainer.Command = ADJobStartupCommand()
	adContainer.Args = args

	if globalAlert.Spec.Detector != nil && len(globalAlert.Spec.Detector.Name) > 0 {
		adContainer.Env = append(
			adContainer.Env,
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

	adContainer.Env = append(
		adContainer.Env,
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

	adContainers := []v1.Container{adContainer}
	adJobPT.Template.Spec.Containers = adContainers

	return nil
}

// decorateBaseADPodTemplate adds required fields and environment variables for ADContainer
// common to both detection and training cycles found in the v1.PodTemplate
func decorateBaseADPodTemplate(clusterName string, adContainer *v1.Container) error {
	if adContainer == nil {
		return ErrADContainerNotFound
	}

	adContainer.Env = append(
		adContainer.Env,
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

// getContainer returns the container specified by name in the Container slice
func getContainer(containers []v1.Container, name string) (v1.Container, error) {
	for _, container := range containers {
		if container.Name == name {
			return container, nil
		}
	}
	return v1.Container{}, ErrADContainerNotFound
}
