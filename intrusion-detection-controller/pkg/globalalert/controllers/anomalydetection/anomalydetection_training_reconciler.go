// Copyright (c) 2021-2022 Tigera, Inc. All rights reserved.
package anomalydetection

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/types"

	log "github.com/sirupsen/logrus"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/globalalert/podtemplate"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/maputil"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/util"
	rcache "github.com/projectcalico/calico/kube-controllers/pkg/cache"
	"github.com/projectcalico/calico/libcalico-go/lib/set"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"
)

var (
	// due to empty TypeMeta issue: https://github.com/kubernetes/client-go/issues/308
	// Deployment GVK is manually declared
	DeploymentGroupVersionKind = schema.GroupVersionKind{
		Kind:    "Deployment",
		Version: "v1",
		Group:   "apps",
	}
)

type adJobTrainingReconciler struct {
	managementClusterCtx  context.Context
	managementClusterName string
	tenantID              string

	kubeClientSet   kubernetes.Interface
	calicoClientSet calicoclient.Interface

	client ctrlclient.WithWatch

	podTemplateQuery           podtemplate.ADPodTemplateQuery
	trainingCycleResourceCache rcache.ResourceCache

	namespace string

	// key: tenant_id.cluster_name
	trainingDetectorsPerCluster map[string]trainingCycleStatePerCluster
	trainingJobsMutex           sync.Mutex
	tenantNamespace             string
}

type trainingCycleStatePerCluster struct {
	ClusterName  string
	TenantID     string
	CronJob      *batchv1.CronJob
	GlobalAlerts []*v3.GlobalAlert
}

// listTrainingCronJobs called by r.trainingCycleStatePerCluster (rcache.ResourceCache) to poll the current
// deployed Cronjobs relating to the DetectionCycleState controlled by the Detection Controller
func (r *adJobTrainingReconciler) listTrainingCronJobs() (map[string]interface{}, error) {
	trainingCronJobs := make(map[string]interface{})
	trainingJobLabelByteStr := maputil.CreateLabelValuePairStr(TrainingCycleLabels())

	trainingCronJobList, err := r.kubeClientSet.BatchV1().CronJobs(r.namespace).List(r.managementClusterCtx,
		metav1.ListOptions{
			LabelSelector: trainingJobLabelByteStr,
		})

	if err != nil {
		log.WithError(err).Errorf("failed to list training cronjobs")
		return nil, err
	}

	for _, trainingCronJob := range trainingCronJobList.Items {
		util.EmptyCronJobResourceValues(&trainingCronJob)

		trainingCronJobs[trainingCronJob.Name] = trainingCronJob
	}

	return trainingCronJobs, nil
}

func (r *adJobTrainingReconciler) Run(stop <-chan struct{}) {
	log.Infof("Starting training reconciler")
	for r.reconcile() {
	}

	<-stop
	r.trainingCycleResourceCache.GetQueue().ShutDown()
}

// Reconcile is the main reconciliation loop called by the worker of the Training Job Controller.
//
// Reconcialiation for the AD Training CrongJob conducts the verfication of the received resource as follows:
//   - verfies if the received resource name is within the list of Training CronJobs, ignores if not
//   - if it is indicated that all AD GlobalAlerts assosciated with the cluster has been removed, remove also
//     the cronjob that was training the detectors assosciated with them
//   - verfies that the resource is present,
//     -if not found restores it with the initial Training CronJob
//     configuration it had when it was first added
//   - verrifes that the fields of the received Training CronJob has not been altered, restore it
//     with the initial CronJob configuration if found otherwise
func (r *adJobTrainingReconciler) reconcile() bool {
	workqueue := r.trainingCycleResourceCache.GetQueue()

	key, shutdown := workqueue.Get()
	// set key to done as to not keep key marked as dirty in the queue
	defer workqueue.Done(key)

	if shutdown {
		log.Infof("Shutting down training reconciler")
		return false
	}

	trainingCronJobNameKey, ok := key.(string)
	if !ok {
		log.Debugf("Received unamanged resource key type %s, ignoring", reflect.TypeOf(key))
		return true
	}

	log.Debugf("Reconciling AD training cronjob for %s", trainingCronJobNameKey)

	trainingCronJobStaterForCluster, ok := r.trainingDetectorsPerCluster[trainingCronJobNameKey]
	if !ok {
		// ignore cronjobs that are unmanaged by this controller
		log.Debugf("Ignore unmanaged Resource: %s", trainingCronJobNameKey)
		return true
	}

	forceDelete := false
	// skip the check if it is an udpate to a detector for the management cluster
	if trainingCronJobStaterForCluster.ClusterName != r.managementClusterName {
		clusterNameToQuery := trainingCronJobStaterForCluster.ClusterName

		managedCluster := &v3.ManagedCluster{}
		err := r.client.Get(context.Background(), types.NamespacedName{Name: clusterNameToQuery, Namespace: r.tenantNamespace}, managedCluster)
		if err != nil && !k8serrors.IsNotFound(err) {
			log.Errorf("unable to query for required managed cluster info for %s",
				trainingCronJobNameKey)
			return true
		}

		forceDelete = k8serrors.IsNotFound(err) || !clusterConnected(managedCluster)
	}
	trainingCronJobToReconcileInterface, found := r.trainingCycleResourceCache.Get(trainingCronJobNameKey)

	if forceDelete || (!found && trainingCronJobStaterForCluster.GlobalAlerts == nil &&
		trainingCronJobStaterForCluster.CronJob == nil) {
		// forcedelete or not found in the resource cache and check the state if it is marked for deletion

		// Handle Deletion
		// only remove the training cronjob for the cluster if there are no globalalerts deployed for the cluster

		// at this point the CronJob as a value in the ResourceCache has been removed, we can only deal with data in the
		// reconciler's training state map
		log.Infof("Deleting training cronJob job for %s", trainingCronJobNameKey)

		err := util.DeleteCronJobWithRetry(r.managementClusterCtx, r.kubeClientSet, r.namespace,
			trainingCronJobNameKey)

		if err != nil && !k8serrors.IsNotFound(err) { // do not report error if it's not found as it is already deleted
			log.WithError(err).Errorf("Unable to delete stored training CronJob %s", trainingCronJobNameKey)
			return true
		}
		r.trainingCycleResourceCache.Delete(trainingCronJobNameKey)
		delete(r.trainingDetectorsPerCluster, trainingCronJobNameKey)

		return true
	}

	trainingCronJobToReconcile, ok := trainingCronJobToReconcileInterface.(batchv1.CronJob)
	if !ok {
		log.Warnf("Received request to reconcile an expected type %s", reflect.TypeOf(trainingCronJobToReconcile))
		return true
	}

	deployedTrainingCronJob, err := r.kubeClientSet.BatchV1().CronJobs(trainingCronJobStaterForCluster.CronJob.Namespace).
		Get(r.managementClusterCtx, trainingCronJobStaterForCluster.CronJob.Name, metav1.GetOptions{})

	if err != nil && !k8serrors.IsNotFound(err) {
		log.WithError(err).Errorf("Unable to retrieve managed CronJob %s", trainingCronJobStaterForCluster.CronJob.Name)
		return true
	}

	// Handle Create
	// kubernetes has indicated there is no training CronJob we are expecting on the cluster, deploy one created
	if k8serrors.IsNotFound(err) {
		log.Infof("Recreating deleted training cronJob job for %s", trainingCronJobStaterForCluster.CronJob.Name)

		util.EmptyCronJobResourceValues(trainingCronJobStaterForCluster.CronJob)

		// restore deleted cronJob for the cluster
		createdTrainingCycle, err := r.kubeClientSet.BatchV1().CronJobs(trainingCronJobStaterForCluster.CronJob.Namespace).
			Create(r.managementClusterCtx,
				trainingCronJobStaterForCluster.CronJob, metav1.CreateOptions{})

		if err != nil {
			log.WithError(err).Errorf("Unable to restore training CronJob %s that has been deleted",
				trainingCronJobToReconcile.Name)
			return true
		}

		util.EmptyCronJobResourceValues(createdTrainingCycle)
		r.trainingCycleResourceCache.Set(trainingCronJobNameKey, *createdTrainingCycle)
		return true
	}

	// Handle Update
	// At this point the Training CronJob for the GlobalAlert already exists update it with the CronJob contents
	// stored by r.trainingDetectorsPerCluster

	// validate if the expected cronjob fields in the cache (disregarding the status) is equal to the one currently deployed
	if util.CronJobDeepEqualsIgnoreStatus(*trainingCronJobStaterForCluster.CronJob, *deployedTrainingCronJob) {
		log.Debugf("Ignoring resource specific updates to %s", trainingCronJobNameKey)
		return true
	}

	log.Infof("Updating training cronJob job %s", trainingCronJobStaterForCluster.CronJob.Name)

	// restore the altered cronjob for the cluster
	updatedTrainingCronJob, err := r.kubeClientSet.BatchV1().CronJobs(trainingCronJobStaterForCluster.CronJob.Namespace).
		Update(r.managementClusterCtx,
			trainingCronJobStaterForCluster.CronJob, metav1.UpdateOptions{})

	// update GlobalAlertStats with events for newly created CronJob
	if err != nil {
		log.WithError(err).Errorf("Unable to restore managed CronJob %s that has been altered", trainingCronJobToReconcile.Name)

		return true
	}

	util.EmptyCronJobResourceValues(updatedTrainingCronJob)
	r.trainingCycleResourceCache.Set(trainingCronJobNameKey, *updatedTrainingCronJob)
	return true
}

// Close cancels the ADJobController worker context and removes health check for all
// the objects that worker watches.
func (r *adJobTrainingReconciler) Close() {
	r.trainingJobsMutex.Lock()
	defer r.trainingJobsMutex.Unlock()

	for name, cachedTrainingCronJob := range r.trainingDetectorsPerCluster {
		r.removeTrainingCycleFromResourceCache(name, &cachedTrainingCronJob)
	}
}

// addTrainingCycle adds to the list of cronjobs state for the cluster and creates the cronjob
// reference based on the list of AnomalyDetection GlobalAlerts deployed by the cluster. The updated
// training cronjob will be deployed next iteration of the Reconcile() loop.
func (r *adJobTrainingReconciler) addTrainingCycle(mcs TrainingDetectorsRequest) error {
	clusterName := mcs.ClusterName
	trainingCronJobStateNameKey := r.getTrainingCycleJobNameForCluster(clusterName)

	r.trainingJobsMutex.Lock()
	defer r.trainingJobsMutex.Unlock()

	trainingCycle, found := r.trainingDetectorsPerCluster[trainingCronJobStateNameKey]

	// No existing training cycle for the cluster.
	if !found {
		trainingCycle = trainingCycleStatePerCluster{
			ClusterName:  mcs.ClusterName,
			TenantID:     mcs.TenantID,
			GlobalAlerts: []*v3.GlobalAlert{},
		}
	}

	trainingCycle.GlobalAlerts = append(trainingCycle.GlobalAlerts, mcs.GlobalAlert)

	// Add specs for training cycle.
	detectorList := collectDetectorsFromGlobalAlerts(trainingCycle.GlobalAlerts)
	adTrainingJobPT, err := r.getADPodTemplateWithEnabledDecorators(clusterName, detectorList)
	if err != nil {
		log.WithError(err).
			Errorf("Unable to start training cycles for on cluster %s, unable to retrieve podtemplate for training cronjobs",
				clusterName)
		return err
	}

	var trainingCronJob *batchv1.CronJob
	trainingCronJob, err = r.createTrainingCronJobForCluster(clusterName, trainingCronJobStateNameKey, *adTrainingJobPT)
	if err != nil {
		log.WithError(err).Errorf("Unable to create training cycles for on cluster %s", clusterName)
		return err
	}

	// Update store entry.
	trainingCycle.CronJob = trainingCronJob
	r.trainingDetectorsPerCluster[trainingCronJob.Name] = trainingCycle
	r.trainingCycleResourceCache.Set(trainingCronJobStateNameKey, *trainingCronJob)

	return nil
}

// runInitialTrainingJob kicks-off an initial training job if the training cycle isn't found or for
// a first time detector.
func (r *adJobTrainingReconciler) runInitialTrainingJob(mcs TrainingDetectorsRequest) error {
	clusterName := mcs.ClusterName
	trainingCycleJobStateNameKey := r.getTrainingCycleJobNameForCluster(clusterName)

	r.trainingJobsMutex.Lock()
	defer r.trainingJobsMutex.Unlock()

	trainingCycle, found := r.trainingDetectorsPerCluster[trainingCycleJobStateNameKey]

	// kick-off an initial training job if there is not existing training cycle or for a first time
	// detector.
	detector := mcs.GlobalAlert.Spec.Detector.Name
	if !found || !collectDetectorsSetFromGlobalAlerts(trainingCycle.GlobalAlerts).Contains(detector) {
		trainingJobStateNameKey := r.getInitialTrainingJobNameForCluster(clusterName, detector)
		adTrainingJobPT, err := r.getADPodTemplateWithEnabledDecorator(clusterName, detector)
		if err != nil {
			log.WithError(err).
				Errorf("Unable to start initial training pod for on cluster %s, unable to retrieve podtemplate for training job",
					clusterName)
			return err
		}

		adInitialTrainingJob, err :=
			r.createInitialTrainingJobForCluster(clusterName, trainingJobStateNameKey, *adTrainingJobPT)
		if err != nil {
			log.WithError(err).
				Errorf("Unable to create initial training pod for on cluster %s", clusterName)
			return err
		}

		// Create an initial training job.
		_, err = r.kubeClientSet.BatchV1().Jobs(r.namespace).
			Create(r.managementClusterCtx, adInitialTrainingJob, metav1.CreateOptions{})
		if err != nil {
			log.WithError(err).
				Errorf("Unable to create initial training jod for on cluster %s", clusterName)
			return err
		}
	}

	return nil
}

// getInitialTrainingJobNameForCluster creates a standardized string from the cluster's name to be
// used as the initial training job name created for the cluster.
func (r *adJobTrainingReconciler) getInitialTrainingJobNameForCluster(cluster, detector string) string {
	return util.GetValidInitialTrainingJobName(cluster, r.tenantID, detector, initialTrainingJobSuffix)
}

// getTrainingCycleCronJobNameForCluster creates a standardized string from the cluster's name to be
// used as the cronjob name created for the cluster.
func (r *adJobTrainingReconciler) getTrainingCycleJobNameForCluster(clusterName string) string {
	// We need to take into account Calico Cloud setup that functions in a multi-tenant flavour
	// In order to keep backwards compatibility, a job name will have <tenant_id.cluster_name-training-cycle>
	// in multi-tenant setup and <cluster_name-training-cycle> for Enterprise
	return fmt.Sprintf("%s-%s-cycle", util.Unify(r.tenantID, clusterName), trainingCycleSuffix)
}

// collectDetectorsSetFromGlobalAlerts collects and returns the comma delimited string of detectors
// of the global alerts.
func collectDetectorsFromGlobalAlerts(globalAlerts []*v3.GlobalAlert) string {
	var detectorList []string
	for _, ga := range globalAlerts {
		detectorList = append(detectorList, ga.Spec.Detector.Name)
	}

	return strings.Join(detectorList, ",")
}

// collectDetectorsSetFromGlobalAlerts collects and returns the set of detectors of the global
// alerts.
func collectDetectorsSetFromGlobalAlerts(globalAlerts []*v3.GlobalAlert) set.Set[string] {
	detectors := set.New[string]()
	for _, ga := range globalAlerts {
		if ga.Spec.Detector != nil {
			detectors.Add(ga.Spec.Detector.Name)
		}
	}

	return detectors
}

// getADPodTemplateWithEnabledDecorator returns a pod template with enabled detector for an individual detector.
func (r *adJobTrainingReconciler) getADPodTemplateWithEnabledDecorator(
	clusterName string, detector string,
) (*v1.PodTemplate, error) {
	adTrainingJobPT, err := r.podTemplateQuery.GetPodTemplate(r.managementClusterCtx, r.namespace, ADTrainingJobTemplateName)
	if err != nil {
		log.WithError(err).
			Errorf("Unable to start initial training pod for on cluster %s, unable to specify training ADJob run to the ADJob PodTemnplate",
				clusterName)
		return nil, err
	}

	// Add specs for training cycle.
	err = podtemplate.DecoratePodTemplateForTrainingCycle(adTrainingJobPT, clusterName, r.tenantID, detector)
	if err != nil {
		return nil, err
	}

	return adTrainingJobPT, nil
}

// getADPodTemplateWithEnabledDecorators returns a pod template with enabled detector for a list of detectors.
func (r *adJobTrainingReconciler) getADPodTemplateWithEnabledDecorators(
	clusterName string, detectorList string,
) (*v1.PodTemplate, error) {
	adTrainingJobPT, err := r.podTemplateQuery.GetPodTemplate(r.managementClusterCtx, r.namespace, ADTrainingJobTemplateName)
	if err != nil {
		log.WithError(err).Errorf("Unable to start training cycles for on cluster %s, unable to specify training ADJob run to the ADJob PodTemnplate",
			clusterName)
		return nil, err
	}

	// add specs for training cycle
	err = podtemplate.DecoratePodTemplateForTrainingCycle(adTrainingJobPT, clusterName, r.tenantID, detectorList)
	if err != nil {
		return nil, err
	}

	return adTrainingJobPT, nil
}

// createInitialTrainingJobForCluster creates an initial training job from the expected podtemplate,
// adTrainingJobPT is assumed to have the Pod's specs set for training.
func (r *adJobTrainingReconciler) createInitialTrainingJobForCluster(
	clusterName string, cronJobName string, adTrainingJobPT v1.PodTemplate,
) (*batchv1.Job, error) {
	trainingLabels := TrainingJobLabels()
	trainingLabels[clusterKey] = util.Unify(r.tenantID, clusterName)

	// Restart policy set to 'Never' and a backoffLimit of zero means that in the event that it
	// results in an error, the initial training job would not be put in a crashloop since we have the
	// fallback of the pre-trained model in the AD Pods themselves.
	adTrainingJobPT.Template.Spec.RestartPolicy = v1.RestartPolicyNever
	backoffLimit := int32(0)

	trainingJob := podtemplate.CreateJobFromPodTemplate(
		cronJobName, r.namespace, trainingLabels, adTrainingJobPT, &backoffLimit)

	// Attach this IDS controller as owner
	intrusionDetectionDeployment, err := r.kubeClientSet.AppsV1().Deployments(r.namespace).
		Get(r.managementClusterCtx, ADJobOwnerLabelValue, metav1.GetOptions{})

	if err != nil {
		log.WithError(err).
			Errorf("Unable to start initial training for on cluster %s, unable to create training cycles for models",
				clusterName)
		return nil, err
	}

	trainingJob.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(intrusionDetectionDeployment, DeploymentGroupVersionKind),
	}

	return trainingJob, nil
}

// createTrainingCronJobForCluster creates the training cronjob from the expected podtemplate, adTrainingJobPT
// is assumed to have the Pod's specs set for training
func (r *adJobTrainingReconciler) createTrainingCronJobForCluster(clusterName string, cronJobName string, adTrainingJobPT v1.PodTemplate) (*batchv1.CronJob, error) {
	trainingCronLabels := TrainingCycleLabels()
	trainingCronLabels["cluster"] = clusterName

	trainingCronJob := podtemplate.CreateCronJobFromPodTemplate(cronJobName, r.namespace,
		DefaultADDetectorTrainingSchedule, trainingCronLabels, adTrainingJobPT)

	// attach this IDS controller as owner
	intrusionDetectionDeployment, err := r.kubeClientSet.AppsV1().Deployments(r.namespace).Get(r.managementClusterCtx, ADJobOwnerLabelValue,
		metav1.GetOptions{})

	if err != nil {
		log.WithError(err).Errorf("Unable to start training cycles for on cluster %s, unable to create training cycles for models", clusterName)
		return nil, err
	}

	trainingCronJob.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(intrusionDetectionDeployment, DeploymentGroupVersionKind),
	}

	return trainingCronJob, nil
}

// removeTrainingCycles removes the specified GlobalAlert from the stored TrainingDetectorState for the Cluster and signals an update for the training CronJob.
// TrainingDetectorState's CronJob and GlobalAlerts will be set to nil if no GlobalAlerts are deployed by the the cluster and signals a deletion for the training
// CronJob. It is called by the training controller. The deletion of the training cronjob will happen on the next iteration of the Reconcile() loop.
func (r *adJobTrainingReconciler) removeTrainingCycles(mcs TrainingDetectorsRequest) error {
	r.trainingJobsMutex.Lock()
	defer r.trainingJobsMutex.Unlock()

	trainingCycleCronJobNameKey := r.getTrainingCycleJobNameForCluster(mcs.ClusterName)
	managedTrainingDetectorsForCluster, found := r.trainingDetectorsPerCluster[trainingCycleCronJobNameKey]

	if !found {
		log.Debugf("Ignore unmanaged Resource: %s", mcs.GlobalAlert.Name)
		return nil
	}

	managedTrainingDetectorsForCluster.GlobalAlerts, found = removeGlobalAlertFromSlice(managedTrainingDetectorsForCluster.GlobalAlerts,
		mcs.GlobalAlert)
	if !found {
		log.Warnf("unable to find expected managed resource: %s, ", mcs.GlobalAlert.Name)
		return nil
	}

	// if no more AD globalAlerts exists, remove the whole training cronjob and exit as there is nothing else to update
	if len(managedTrainingDetectorsForCluster.GlobalAlerts) == 0 {
		r.removeTrainingCycleFromResourceCache(trainingCycleCronJobNameKey, &managedTrainingDetectorsForCluster)
		r.trainingDetectorsPerCluster[trainingCycleCronJobNameKey] = managedTrainingDetectorsForCluster
		return nil
	}

	// else update AD_ENABLED_DETECTORS to exclude detector with deleted GlobalAlert
	detectorList := collectDetectorsFromGlobalAlerts(managedTrainingDetectorsForCluster.GlobalAlerts)

	adTrainingJobPT, err := r.getADPodTemplateWithEnabledDecorators(managedTrainingDetectorsForCluster.ClusterName, detectorList)
	if err != nil {
		log.WithError(err).Errorf("Unable to update training cycles for on cluster %s, unable to retrieve podtemplate for training cronjobs",
			managedTrainingDetectorsForCluster.ClusterName)
		return err
	}

	trainingCronJob, err := r.createTrainingCronJobForCluster(managedTrainingDetectorsForCluster.ClusterName, trainingCycleCronJobNameKey, *adTrainingJobPT)
	if err != nil {
		log.WithError(err).Errorf("Unable to update training cycles for on cluster %s, unable to retrieve podtemplate for training cronjobs",
			managedTrainingDetectorsForCluster.ClusterName)
		return err
	}

	managedTrainingDetectorsForCluster.CronJob = trainingCronJob
	r.trainingDetectorsPerCluster[trainingCycleCronJobNameKey] = managedTrainingDetectorsForCluster
	r.trainingCycleResourceCache.Set(trainingCycleCronJobNameKey, *managedTrainingDetectorsForCluster.CronJob)
	return nil
}

func (r *adJobTrainingReconciler) stopTrainigCycleForCluster(clusterName string) error {
	trainingCycleCronJobNameKey := r.getTrainingCycleJobNameForCluster(clusterName)
	managedTrainingDetectorsForCluster, found := r.trainingDetectorsPerCluster[trainingCycleCronJobNameKey]

	if !found {
		log.Debugf("Ignore unmanaged Resource: %s", trainingCycleCronJobNameKey)
		return nil
	}

	r.removeTrainingCycleFromResourceCache(trainingCycleCronJobNameKey, &managedTrainingDetectorsForCluster)
	r.trainingDetectorsPerCluster[trainingCycleCronJobNameKey] = managedTrainingDetectorsForCluster

	return nil
}

func (r *adJobTrainingReconciler) removeTrainingCycleFromResourceCache(key string, trainingState *trainingCycleStatePerCluster) {
	trainingState.CronJob = nil
	trainingState.GlobalAlerts = nil
	r.trainingCycleResourceCache.Delete(key)
}

// removeGlobalAlertFromSlice remove the specified GlobalAlert toRemove from the specified GlobalAlerts slice, a found
// boolean returns true if toRemove is found in globalAlerts and removed false o/w
func removeGlobalAlertFromSlice(globalAlerts []*v3.GlobalAlert, toRemove *v3.GlobalAlert) ([]*v3.GlobalAlert, bool) {
	for i, ga := range globalAlerts {
		if ga.Name == toRemove.Name {
			return append(globalAlerts[:i], globalAlerts[i+1:]...), true
		}
	}

	return nil, false
}
