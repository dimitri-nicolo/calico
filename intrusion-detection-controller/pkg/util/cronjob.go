package util

import (
	"context"
	"reflect"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

// CronJobDeepEqualsIgnoreStatus compares the non resource specific fields
func CronJobDeepEqualsIgnoreStatus(cronJob0 batchv1.CronJob, cronJob1 batchv1.CronJob) bool {
	return cronJob0.Name == cronJob1.Name && cronJob0.Namespace == cronJob1.Namespace &&
		reflect.DeepEqual(cronJob0.GetLabels(), cronJob1.GetLabels()) &&
		reflect.DeepEqual(cronJob0.Spec, cronJob1.Spec)
}

func EmptyCronJobResourceValues(cronJob *batchv1.CronJob) {
	cronJob.ResourceVersion = ""
	cronJob.UID = ""
	cronJob.CreationTimestamp = metav1.Time{}
	cronJob.Status = batchv1.CronJobStatus{}
}

func DeleteCronJobWithRetry(ctx context.Context, k8sClient kubernetes.Interface, namespace, cronJobName string) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return k8sClient.BatchV1().CronJobs(namespace).Delete(ctx, cronJobName, metav1.DeleteOptions{})
	})

	if err != nil {
		return err
	}

	return nil
}
