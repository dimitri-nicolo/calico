package anomalydetection

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"

	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/reporting"

	log "github.com/sirupsen/logrus"
)

type adJobDetectionReconciler struct {
	managementClusterCtx context.Context

	k8sClient   kubernetes.Interface
	calicoCLI   calicoclient.Interface
	clusterName string

	managedDetectionJobs map[string]ManagedADDetectionJobsState
	detectionJobsMutex   sync.Mutex
}

// Reconcile is the main reconciliation loop called by the worker of the Detection Job Controller.
//
// Reconciliation for the AD Detecion CronJobs conducts the verfication of the received resource as follows:
// 	- verfies if the received resource name is within the list of Detection Jobs, ignores if not
//  - verfies that the resource is present, if not found restores it with the initial CronJob
// 		configuration it had when it was first added
//  - verrifes that the fields of the received Detection CronJob has not been altered,  restores it
// 		with the initial CronJob configuration if found otherwise
// Error Statuses are reported to the assosciated GlobalAlert for each Detection CronJob during each verfication step
func (r *adJobDetectionReconciler) Reconcile(namespacedName types.NamespacedName) error {
	log.Infof("Reconciling ad detection job for %s", namespacedName)
	cachedDetectionJobState, ok := r.managedDetectionJobs[namespacedName.Name]

	// ignore cronjobs that ar unmanaged by this controller
	if !ok {
		log.Debugf("Ignore unmanaged Resource: %s", namespacedName)
		return nil
	}

	currentDetectionCronJob, err := r.k8sClient.BatchV1().CronJobs(namespacedName.Namespace).Get(r.managementClusterCtx,
		namespacedName.Name, metav1.GetOptions{})

	if err != nil && !errors.IsNotFound(err) {
		r.reportErrorStatus(cachedDetectionJobState.GlobalAlert, namespacedName, err)
		return err
	}

	if errors.IsNotFound(err) {

		log.Infof("Recreating deleted job for %s", cachedDetectionJobState.CronJob.Name)

		// create / restore deleted managed cronJobs
		_, err := r.k8sClient.BatchV1().CronJobs(namespacedName.Namespace).Create(r.managementClusterCtx,
			cachedDetectionJobState.CronJob, metav1.CreateOptions{})

		// update GlobalAlertStats with events for newly created CronJob
		if err != nil {
			r.reportErrorStatus(cachedDetectionJobState.GlobalAlert, namespacedName, err)
			return err
		}

		cachedDetectionJobState.GlobalAlert.Status.Healthy = true
		cachedDetectionJobState.GlobalAlert.Status.Active = true
		cachedDetectionJobState.GlobalAlert.Status.ErrorConditions = nil
		cachedDetectionJobState.GlobalAlert.Status.LastUpdate = &metav1.Time{Time: time.Now()}

		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return reporting.UpdateGlobalAlertStatus(cachedDetectionJobState.GlobalAlert, r.clusterName, r.calicoCLI,
				r.managementClusterCtx)
		}); err != nil {
			log.WithError(err).Errorf("failed to update status GlobalAlert %s in cluster %s, maximum retries reached",
				namespacedName.Name, r.clusterName)
		}
		return nil
	}

	// validate if the expected cronjob fields in the cahe minus the status is equal to the one currently deployed
	baseCachedDetectionCronJob, baseCurrentDetectionCronJob := *cachedDetectionJobState.CronJob, *currentDetectionCronJob

	if !reflect.DeepEqual(baseCachedDetectionCronJob.Spec, baseCurrentDetectionCronJob.Spec) {
		log.Infof("Recreating altered training cronJob job for %s", r.clusterName)

		// create / restore deleted managed cronJobs
		_, err := r.k8sClient.BatchV1().CronJobs(namespacedName.Namespace).Update(r.managementClusterCtx,
			&baseCachedDetectionCronJob, metav1.UpdateOptions{})

		// update GlobalAlertStats with events for newly created CronJob
		if err != nil {
			r.reportErrorStatus(cachedDetectionJobState.GlobalAlert, namespacedName, err)
			return err
		}
	}

	adDetectionCronJobState := r.managedDetectionJobs[currentDetectionCronJob.Name]

	if len(currentDetectionCronJob.Status.Active) > 0 {
		adDetectionCronJobState.GlobalAlert.Status = r.getLatestJobStatusOfCronJob(r.managementClusterCtx, currentDetectionCronJob)
	}

	adDetectionCronJobState.GlobalAlert.Status.LastExecuted = currentDetectionCronJob.Status.LastSuccessfulTime
	adDetectionCronJobState.GlobalAlert.Status.LastEvent = currentDetectionCronJob.Status.LastScheduleTime

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return reporting.UpdateGlobalAlertStatus(adDetectionCronJobState.GlobalAlert, r.clusterName, r.calicoCLI, r.managementClusterCtx)
	}); err != nil {
		log.WithError(err).Errorf("failed to update status GlobalAlert %s in cluster %s, maximum retries reached", namespacedName.Name, r.clusterName)
	}

	return nil
}

// getLatestJobStatusOfCronJob retrieves the status of the latest run job managed by the cronjob
func (r *adJobDetectionReconciler) getLatestJobStatusOfCronJob(ctx context.Context, cronjob *batchv1.CronJob) v3.GlobalAlertStatus {

	resultantGlobalAlertStatus := reporting.GetGlobalAlertSuccessStatus()

	childJobs, err := r.k8sClient.BatchV1().Jobs(cronjob.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "owner=" + cronjob.Name,
	})

	if err != nil {

		resultantGlobalAlertStatus = reporting.GetGlobalAlertErrorStatus(err)
		resultantGlobalAlertStatus.LastExecuted = cronjob.Status.LastSuccessfulTime
		resultantGlobalAlertStatus.LastEvent = cronjob.Status.LastScheduleTime

		return resultantGlobalAlertStatus
	}

	// sort the jobs managed by start time to get the laatest Job triggered by the running detection job.
	// Only the latest run Job is relevant for GlbalAlertStatus reporting, as it is more concerned with health
	// and status of the most current run to show the health of the GlobalAlert hence why only the latest job
	// run is retrieved.
	sort.Slice(childJobs.Items, func(i, j int) bool {
		return childJobs.Items[i].Status.StartTime.After(childJobs.Items[j].Status.StartTime.Time)
	})
	latestChildJob := childJobs.Items[0]

	// return an error to report to the GlobalAlert.Status if the latest Job Run from the CronJob reports an error
	for _, condition := range latestChildJob.Status.Conditions {
		if condition.Type == v1.JobFailed && condition.Status == "True" {
			jobError := fmt.Errorf("failed Job %s on CronJob %s error: %s", latestChildJob.Name, cronjob.Name, condition.Message)
			resultantGlobalAlertStatus = reporting.GetGlobalAlertErrorStatus(jobError)
			break
		}
	}

	// success status o/w
	resultantGlobalAlertStatus.LastExecuted = cronjob.Status.LastSuccessfulTime
	resultantGlobalAlertStatus.LastEvent = cronjob.Status.LastScheduleTime

	return resultantGlobalAlertStatus
}

// reportErrorStatus sets the status of alert with the error param
func (r *adJobDetectionReconciler) reportErrorStatus(alert *v3.GlobalAlert, namespacedName types.NamespacedName, err error) {
	if alert == nil {
		return
	}

	formattedError := fmt.Errorf("unhealthy CronJob %s, with error: %s", namespacedName.Name, err.Error())
	globalAlertErrorStatus := reporting.GetGlobalAlertErrorStatus(formattedError)

	alert.Status = globalAlertErrorStatus

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return reporting.UpdateGlobalAlertStatus(alert, r.clusterName, r.calicoCLI, r.managementClusterCtx)
	}); err != nil {
		log.WithError(err).Errorf("failed to update status GlobalAlert %s in cluster %s, maximum retries reached", namespacedName.Name, r.clusterName)
	}
}

// addToManagedDetectionJobs adds to the list of jobs managed, called by the detection controller
func (r *adJobDetectionReconciler) addToManagedDetectionJobs(detectionResource ManagedADDetectionJobsState) {
	r.detectionJobsMutex.Lock()
	r.managedDetectionJobs[detectionResource.CronJob.Name] = detectionResource
	r.detectionJobsMutex.Unlock()
}

// removeManagedDetectionJobs removes from the list of jobs managed, called by the detection controller.
func (r *adJobDetectionReconciler) removeManagedDetectionJobs(cronJobName string) {
	r.detectionJobsMutex.Lock()
	delete(r.managedDetectionJobs, cronJobName)
	r.detectionJobsMutex.Unlock()
}

// Close cancels the ADJobController worker context and removes for all resources/objects that worker watches.
func (r *adJobDetectionReconciler) Close() {
	for name, adCronJobState := range r.managedDetectionJobs {

		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return r.k8sClient.BatchV1().CronJobs(adCronJobState.CronJob.Namespace).Delete(r.managementClusterCtx,
				adCronJobState.CronJob.Name, metav1.DeleteOptions{})
		}); err != nil {
			log.WithError(err).Errorf("failed to delete CronJob %s in cluster %s, maximum retries reached", name, r.clusterName)
		}

		delete(r.managedDetectionJobs, name)
	}
}
