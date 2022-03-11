package util

import (
	"reflect"

	batchv1 "k8s.io/api/batch/v1"
)

// CronJobDeepEqualsLabelAndSpec compares the labels abd specs of two CronJob objects
func CronJobDeepEqualsLabelAndSpec(cronJob0 batchv1.CronJob, cronJob1 batchv1.CronJob) bool {
	return reflect.DeepEqual(cronJob0.GetLabels(), cronJob1.GetLabels()) &&
		reflect.DeepEqual(cronJob0.Spec, cronJob1.Spec)
}
