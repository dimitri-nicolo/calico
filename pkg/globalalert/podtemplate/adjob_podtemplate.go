package podtemplate

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
)

const (
	ADJobsContainerName = "adjobs"
	AllADJobsKeyword    = "all"

	ADJobTrainCycleArg  = "train"
	ADJobDetectCycleArg = "detect"
)

var (
	ADJobStartupCommand = func() []string {
		return []string{"python3"}
	}

	ADJobStartupArgs = func() []string {
		return []string{"-m", "adj"}
	}
)

// DecoratePodTemplateForADDetectorCycle adds required fields and environment variables for a PodSpec from the provided
// v1.PodTemplate for a training or detection AD cycle.
func DecoratePodTemplateForADDetectorCycle(adJobPT *v1.PodTemplate, clusterName, cycle, detector, period string) error {

	adJobContainerIndex := getContainerIndex(adJobPT.Template.Spec.Containers, ADJobsContainerName)

	if adJobContainerIndex == -1 {
		return fmt.Errorf("unable to retrtieve container for %s", ADJobsContainerName)
	}

	if !(cycle == ADJobTrainCycleArg || cycle == ADJobDetectCycleArg) {
		return fmt.Errorf("unaceepted run type arg %s", cycle)
	}

	args := ADJobStartupArgs()
	args = append(args, cycle)

	adJobPT.Template.Spec.Containers[adJobContainerIndex].Command = ADJobStartupCommand()
	adJobPT.Template.Spec.Containers[adJobContainerIndex].Args = args

	adJobPT.Template.Spec.Containers[adJobContainerIndex].Env = append(
		adJobPT.Template.Spec.Containers[adJobContainerIndex].Env,
		v1.EnvVar{
			Name:  "CLUSTER_NAME",
			Value: clusterName,
		},
	)

	if detector != AllADJobsKeyword {
		adJobPT.Template.Spec.Containers[adJobContainerIndex].Env = append(
			adJobPT.Template.Spec.Containers[adJobContainerIndex].Env,
			v1.EnvVar{
				Name:  "AD_ENABLED_JOBS",
				Value: detector,
			},
		)
	}

	if cycle == ADJobTrainCycleArg {
		adJobPT.Template.Spec.Containers[adJobContainerIndex].Env = append(
			adJobPT.Template.Spec.Containers[adJobContainerIndex].Env,
			v1.EnvVar{
				Name:  "AD_train_default_query_time_duration",
				Value: period,
			},
		)
	} else if cycle == ADJobDetectCycleArg {
		adJobPT.Template.Spec.Containers[adJobContainerIndex].Env = append(
			adJobPT.Template.Spec.Containers[adJobContainerIndex].Env,
			v1.EnvVar{
				Name:  "AD_detect_default_query_time_duration",
				Value: period,
			},
			// set in detection cycle to indicate anomaly detection job image to train a model
			// in a detection cycle if one does not already exist for the detectors in AD_ENABLED_JOBS
			v1.EnvVar{
				Name:  "AD_DETECTION_VERIFY_MODEL_EXISTENCE",
				Value: "True",
			},
		)
	}

	return nil
}

func getContainerIndex(containers []v1.Container, name string) int {
	for i, container := range containers {
		if container.Name == name {
			return i
		}
	}
	return -1
}
