package reporting

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	batchv1 "k8s.io/api/batch/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/util/retry"
)

// UpdateGlobalAlertStatusWithRetryOnConflict
func UpdateGlobalAlertStatusWithRetryOnConflict(globalAlert *v3.GlobalAlert, clusterName string, calicoCLI calicoclient.Interface, ctx context.Context) error {

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error { return updateGlobalAlertStatus(globalAlert, clusterName, calicoCLI, ctx) }); err != nil {
		log.WithError(err).Errorf("failed to update status GlobalAlert %s in cluster %s, maximum retries reached", globalAlert.Name, clusterName)
		return err
	}

	return nil
}

// UpdateGlobalAlertStatus gets the latest GlobalAlert and updates its status.
func updateGlobalAlertStatus(globalAlert *v3.GlobalAlert, clusterName string, calicoCLI calicoclient.Interface, ctx context.Context) error {
	log.Debugf("Updating status of GlobalAlert %s in cluster %s", globalAlert.Name, clusterName)
	retrievedAlert, err := calicoCLI.ProjectcalicoV3().GlobalAlerts().Get(ctx, globalAlert.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	retrievedAlert.Status = globalAlert.Status
	_, err = calicoCLI.ProjectcalicoV3().GlobalAlerts().UpdateStatus(ctx, retrievedAlert, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// WatchAndReportJobStatus uses the jobwatcher to retrieve events for a max duration specified by waitTime. Returns a
// GlobalAlertStatus based on the event received, error GlobalAlertStatus is returned if no events are recevied before
// waitTime expires
func WatchAndReportJobStatus(jobWatcher watch.Interface, jobName string, waitTime time.Duration) (v3.GlobalAlertStatus, error) {
	var done <-chan time.Time // signal for when timer us up

	for {
		if done == nil {
			done = time.After(waitTime)
		}

		select {
		case <-done:
			resultChanError := fmt.Errorf("did not receive any events after timeout from Job %s", jobName)
			return GetGlobalAlertErrorStatus(resultChanError), resultChanError
		case event, open := <-jobWatcher.ResultChan():
			if !open {
				continue
			}

			job, ok := event.Object.(*batchv1.Job)

			if !ok || job.Name != jobName {
				log.Error("unexpected type")
				continue
			}

			switch event.Type {
			case watch.Error:
				trainingError := k8serrors.FromObject(job)
				return GetGlobalAlertErrorStatus(trainingError), trainingError
			case watch.Deleted: // deleted event is sent on successful shutdown
				return GetGlobalAlertSuccessStatus(), nil
			case watch.Added:
			case watch.Modified:
			default:
				continue
			}
		}
	}
}

// GetGlobalAlertErrorStatus creates a
func GetGlobalAlertErrorStatus(err error) v3.GlobalAlertStatus {
	return v3.GlobalAlertStatus{
		Healthy:         false,
		Active:          false,
		ErrorConditions: []v3.ErrorCondition{{Message: err.Error()}},
		LastUpdate:      &metav1.Time{Time: time.Now()},
	}
}

func GetGlobalAlertSuccessStatus() v3.GlobalAlertStatus {
	metav1TimeNow := &metav1.Time{Time: time.Now()}

	return v3.GlobalAlertStatus{
		Healthy:         true,
		Active:          true,
		ErrorConditions: nil,
		LastUpdate:      metav1TimeNow,
		LastEvent:       metav1TimeNow,
	}
}
