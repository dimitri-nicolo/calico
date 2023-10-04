package util

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

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
