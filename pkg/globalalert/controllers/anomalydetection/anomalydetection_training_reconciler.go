package anomalydetection

import (
	"context"
	"sync"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"

	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/podtemplate"
	"github.com/tigera/intrusion-detection/controller/pkg/util"

	log "github.com/sirupsen/logrus"
)

type adJobTrainingReconciler struct {
	managementClusterCtx context.Context

	k8sClient        kubernetes.Interface
	calicoCLI        calicoclient.Interface
	podTemplateQuery podtemplate.ADPodTemplateQuery

	namespace string

	managedTrainingCronJobs map[string]*batchv1.CronJob
	trainingJobsMutex       sync.Mutex
}

// Reconcile is the main reconciliation loop called by the worker of the Training Job Controller.
//
// Reconcialiation for the AD Training CrongJob conducts the verfication of the received resource as follows:
// 	- verfies if the received resource name is within the list of Training CronJobs, ignores if not
//  - verfies that the resource is present, if not found restores it with the initial Training CronJob
// 		configuration it had when it was first added
//  - verrifes that the fields of the received Training CronJob has not been altered,  restores it
// 		with the initial CronJob configuration if found otherwise
func (r *adJobTrainingReconciler) Reconcile(namespacedName types.NamespacedName) error {
	log.Infof("Reconciling job for %s", namespacedName)

	cachedTrainingCronJob, ok := r.managedTrainingCronJobs[namespacedName.Name]

	// ignore cronjobs that ar unmanaged by this controller
	if !ok {
		log.Debugf("Ignore unmanaged Resource: %s", namespacedName)
		return nil
	}

	currentTrainingCronJob, err := r.k8sClient.BatchV1().CronJobs(namespacedName.Namespace).Get(r.managementClusterCtx,
		namespacedName.Name, metav1.GetOptions{})

	if err != nil && !k8serrors.IsNotFound(err) {
		log.WithError(err).Errorf("Unable to retrieve managed CronJob %s", namespacedName.Name)
		return err
	}

	if k8serrors.IsNotFound(err) {

		log.Infof("Recreating removed training cronJob job for %s", cachedTrainingCronJob.Name)

		// create / restore deleted managed cronJobs
		_, err := r.k8sClient.BatchV1().CronJobs(namespacedName.Namespace).Create(r.managementClusterCtx,
			cachedTrainingCronJob, metav1.CreateOptions{})

		// update GlobalAlertStats with events for newly created CronJob
		if err != nil {
			log.WithError(err).Errorf("Unable to restore managed CronJob %s that has been deleted", namespacedName.Name)
			return err
		}

		return nil
	}

	// validate if the expected cronjob fields in the cache (disregarding the status) is equal to the one currently deployed
	baseCachedTrainingCronJob, baseCurrentTrainingCronJob := *cachedTrainingCronJob, *currentTrainingCronJob

	if !util.CronJobDeepEqualsLabelAndSpec(baseCachedTrainingCronJob, baseCurrentTrainingCronJob) {
		log.Infof("Recreating altered training cronJob job for %s", cachedTrainingCronJob.Name)

		// create / restore deleted managed cronJobs
		_, err := r.k8sClient.BatchV1().CronJobs(namespacedName.Namespace).Update(r.managementClusterCtx,
			&baseCachedTrainingCronJob, metav1.UpdateOptions{})

		// update GlobalAlertStats with events for newly created CronJob
		if err != nil {
			log.WithError(err).Errorf("Unable to restore managed CronJob %s that has been altered", namespacedName.Name)

			return err
		}
	}

	return nil
}

// Close cancels the ADJobController worker context and removes health check for all
// the objects that worker watches.
func (r *adJobTrainingReconciler) Close() {
	for name, cachedTrainingCronJob := range r.managedTrainingCronJobs {

		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return r.k8sClient.BatchV1().CronJobs(cachedTrainingCronJob.Namespace).Delete(r.managementClusterCtx,
				cachedTrainingCronJob.Name, metav1.DeleteOptions{})
		}); err != nil {
			log.WithError(err).Errorf("failed to delete CronJob %s, maximum retries reached", name)
		}

		delete(r.managedTrainingCronJobs, name)
	}
}

// addToManagedTrainingJobs adds to the list of cronjobs managed, called by the training controller
func (r *adJobTrainingReconciler) addToManagedTrainingJobs(clusterName string) error {
	r.trainingJobsMutex.Lock()
	defer r.trainingJobsMutex.Unlock()

	adTrainingJobPT, err := r.podTemplateQuery.GetPodTemplate(r.managementClusterCtx, r.namespace, ADTrainingJobTemplateName)
	if err != nil {
		log.WithError(err).Errorf("Unable to start training cycles for on cluster %s, unable to retrieve podtemplate for training cronjobs", clusterName)
		return err
	}

	// add specs for training
	err = podtemplate.DecoratePodTemplateForADDetectorCycle(adTrainingJobPT, clusterName, podtemplate.ADJobTrainCycleArg, podtemplate.AllADJobsKeyword, DefaultADDetectorTrainingPeriod.String())
	if err != nil {
		log.WithError(err).Errorf("Unable to start training cycles for on cluster %s, unable to specify training ADJob run to the ADJob PodTemnplate", clusterName)
		return err
	}

	var trainingCronJob *batchv1.CronJob
	trainingCronJob, err = r.createTrainingCronJobForCluster(clusterName, *adTrainingJobPT)
	if err != nil {
		log.WithError(err).Errorf("Unable to start training cycles for on cluster %s, failed creating training cronjob", clusterName)
		return err
	}

	r.managedTrainingCronJobs[trainingCronJob.Name] = trainingCronJob

	return nil
}

// createTrainingCronJobForCluster creates the training cronjob from the expected podtemplate,  adTrainingJobPT
// is assumed to have the Pod's specs set for training
func (r *adJobTrainingReconciler) createTrainingCronJobForCluster(clusterName string, adTrainingJobPT v1.PodTemplate) (*batchv1.CronJob, error) {
	trainingCronLabels := TrainingJobLabels()
	trainingCronLabels["cluster"] = clusterName

	trainingCronJob := podtemplate.CreateCronJobFromPodTemplate(clusterName+trainingCronJobSuffix, r.namespace,
		defaultCronJobTrainingSchedule, trainingCronLabels, adTrainingJobPT)

	intrusionDetectionDeployment, err := r.k8sClient.AppsV1().Deployments(r.namespace).Get(r.managementClusterCtx, ADJobOwnerLabelValue,
		metav1.GetOptions{})

	if err != nil {
		log.WithError(err).Errorf("Unable to start training cycles for on cluster %s, unable to create training cycles for models", clusterName)
		return nil, err
	}

	blockGarbageCollection := true
	trainingCronJob.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion:         "apps/v1",
			Kind:               "Deployment",
			Name:               intrusionDetectionDeployment.GetName(),
			UID:                intrusionDetectionDeployment.GetUID(),
			BlockOwnerDeletion: &blockGarbageCollection,
			Controller:         &blockGarbageCollection,
		},
	}

	_, err = r.k8sClient.BatchV1().CronJobs(r.namespace).Create(context.Background(),
		trainingCronJob, metav1.CreateOptions{})

	if k8serrors.IsAlreadyExists(err) {
		_, err = r.k8sClient.BatchV1().CronJobs(r.namespace).Update(r.managementClusterCtx,
			trainingCronJob, metav1.UpdateOptions{})
	}

	if err != nil {
		return nil, err
	}

	return trainingCronJob, nil
}

// removeManagedTrainingJobs removes from the list of jobs managed, called by the detection controller.
func (r *adJobTrainingReconciler) removeManagedTrainingJobs(trainingCronJobName string) {
	r.trainingJobsMutex.Lock()
	delete(r.managedTrainingCronJobs, trainingCronJobName)
	r.trainingJobsMutex.Unlock()
}
