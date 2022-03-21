package anomalydetection

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"

	rcache "github.com/projectcalico/calico/kube-controllers/pkg/cache"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"

	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/podtemplate"
	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/reporting"
	"github.com/tigera/intrusion-detection/controller/pkg/maputil"
	"github.com/tigera/intrusion-detection/controller/pkg/util"

	log "github.com/sirupsen/logrus"
)

const (
	ADDetectionJobTemplateName      = "tigera.io.detectors.detection"
	DefaultCronJobDetectionSchedule = 15 * time.Minute
	maxCronJobNameLen               = 52

	// Non RFC1123 compliant characters
	nonRFCCompliantRegexDef            = `[^a-z0-9\.\-]+`
	nameRFC1123LabelFmt                = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"
	nameRFC1123SubdomainFmt            = nameRFC1123LabelFmt + "(\\." + nameRFC1123LabelFmt + ")*"
	nameContainsHashLikeSuffixRegexDef = `[-][a-z0-9]{5}$`

	// Matching period and hyphen regex defs constants
	charactersConvertedToPeriodRegex      = `[.-]*[.][.-]*`
	charactersMatchingPrefixOrSuffixRegex = `^[.-]*|[.-]*$`

	// Hash constants.
	hashShortenedPrefix          = "-"
	numHashChars                 = 5
	rfcNonAlphaCharPeriod        = "."
	rfcWildcard                  = "z"
	acceptableRFCGlobalAlertName = maxCronJobNameLen - len(detectionCronJobSuffix) - numHashChars - 2

	ClusterKey = "cluster"

	detectionCronJobSuffix = "detection"
)

var (
	// controllerKind refers to the GlobalAlert kind that the resources created / reconciled
	// by ths controller will refer to
	GlobalAlertGroupVersionKind = schema.GroupVersionKind{
		Kind:    v3.KindGlobalAlert,
		Version: v3.VersionCurrent,
		Group:   v3.Group,
	}
)

type adDetectionReconciler struct {
	managementClusterCtx context.Context

	k8sClient                   kubernetes.Interface
	calicoCLI                   calicoclient.Interface
	podTemplateQuery            podtemplate.ADPodTemplateQuery
	detectionCycleResourceCache rcache.ResourceCache

	clusterName string
	namespace   string

	detectionJobsMutex        sync.Mutex
	detectionADDetectorStates map[string]detectionCycleState
}

type detectionCycleState struct {
	ClusterName string
	CronJob     *batchv1.CronJob
	GlobalAlert *v3.GlobalAlert
}

// listDetectionCronJobs called by r.detectionCycleResourceCache (rcache.ResourceCache) to poll the current
// deployed Cronjobs relating to the DetectionCycleState controlled by the Detection Controller
func (r *adDetectionReconciler) listDetectionCronJobs() (map[string]interface{}, error) {
	detectionCronJobs := make(map[string]interface{})
	detectionJobLabelByteStr := maputil.CreateLabelValuePairStr(DetectionJobLabels())

	detectionCronJobList, err := r.k8sClient.BatchV1().CronJobs(r.namespace).List(r.managementClusterCtx,
		metav1.ListOptions{
			LabelSelector: detectionJobLabelByteStr,
		})

	if err != nil {
		log.WithError(err).Errorf("failed to list detection cronjobs")
		return nil, err
	}

	for _, detectionCronJob := range detectionCronJobList.Items {
		util.EmptyCronJobResourceValues(&detectionCronJob)

		detectionCronJobs[detectionCronJob.Name] = detectionCronJob
	}

	return detectionCronJobs, nil
}

func (r *adDetectionReconciler) Run(stop <-chan struct{}) {
	log.Infof("Starting detection reconciler")
	for r.reconcile() {
	}

	<-stop
	r.detectionCycleResourceCache.GetQueue().ShutDown()
}

// reconcile is the main reconciliation loop called by the worker of the Detection Job Controller. It receives State
// updates to the CronJob from the resource cache and conducts the verfication of the received resource as follows:
// 	- verfies if the received resource name is within the list of Detection Jobs, ignores if not
//  - verfies that the resource is present, if not found restores it with the initial CronJob
// 		configuration it had when it was first added
//  - verrifes that the fields of the received Detection CronJob has not been altered,  restores it
// 		with the initial CronJob configuration if found otherwise
// Error Statuses are reported to the assosciated GlobalAlert for each Detection CronJob during each verfication step
func (r *adDetectionReconciler) reconcile() bool {
	workqueue := r.detectionCycleResourceCache.GetQueue()

	key, shutdown := workqueue.Get()
	// set key to done as to not keep key marked as dirty in the queue
	defer workqueue.Done(key)

	if shutdown {
		log.Infof("Shutting down detection reconciler")
		return false
	}

	detectionCronJobNameKey, ok := key.(string)
	if !ok {
		log.Debugf("Ignoring unamanged resource key type %s", reflect.TypeOf(key))
		return true
	}

	log.Debugf("Reconciling AD detection cronjob %s", detectionCronJobNameKey)

	detectionJobState, stored := r.detectionADDetectorStates[detectionCronJobNameKey]
	if !stored {
		log.Debugf("Ignoring request to reconcile an uncached object %s, ignoring", detectionCronJobNameKey)
		return true
	}

	detectionCronJobNameSpacedName := types.NamespacedName{
		Name:      detectionCronJobNameKey,
		Namespace: r.namespace,
	}

	detectionCronJobToReconcileInterface, found := r.detectionCycleResourceCache.Get(detectionCronJobNameKey)

	if !found {
		// not found in the resource cache, check the state if it is marked for deletion

		// Handle Deletion
		// nil GlobalAlert and Cronjob indicates entry is indicated for deletion
		if detectionJobState.GlobalAlert == nil && detectionJobState.CronJob == nil {
			// at this point the CronJob as a value in the ResourceCache has been removed, we can only deal with data in the
			// reconciler's detection state map
			log.Infof("Deleting detection cronJob job for %s", detectionCronJobNameKey)

			err := util.DeleteCronJobWithRetry(r.managementClusterCtx, r.k8sClient, r.namespace,
				detectionCronJobNameKey)

			if err != nil && !errors.IsNotFound(err) { // do not report error if it's not found as it is already deleted
				log.WithError(err).Errorf("Unable to delete stored detection CronJob %s", detectionCronJobNameKey)
				r.reportErrorStatus(detectionJobState.GlobalAlert, detectionCronJobNameSpacedName, err)
			}

			delete(r.detectionADDetectorStates, detectionCronJobNameKey)
			return true
		} else { // not in resource cache and state does not think it's for delete, throw error
			log.Errorf("invalid cache, received request to delete detection cronjob %s, not marked for deletion",
				detectionCronJobNameKey)
			return true
		}
	}

	detectionCronJobToReconcile, ok := detectionCronJobToReconcileInterface.(batchv1.CronJob)
	if !ok {
		log.Warnf("Received request to reconcile an expected type %s", reflect.TypeOf(detectionCronJobToReconcile))
		return true
	}

	foundDetectionJob, err := r.k8sClient.BatchV1().CronJobs(detectionJobState.CronJob.Namespace).Get(
		r.managementClusterCtx, detectionJobState.CronJob.Name, metav1.GetOptions{})

	if err != nil && !errors.IsNotFound(err) {
		r.reportErrorStatus(detectionJobState.GlobalAlert, detectionCronJobNameSpacedName, err)
	}

	// Handle Create
	// kubernetes has indicated there is no detection CronJob we are expecting on the cluster, deploy one created
	if errors.IsNotFound(err) {
		log.Infof("Creating detection cronJob job for %s", detectionJobState.GlobalAlert.Name)

		// safety measures to set these to nil values before creating to avoid error
		util.EmptyCronJobResourceValues(&detectionCronJobToReconcile)
		// create / restore deleted managed cronJobs
		createdDetectionCycle, err := r.k8sClient.BatchV1().CronJobs(detectionJobState.CronJob.Namespace).Create(r.managementClusterCtx,
			detectionJobState.CronJob, metav1.CreateOptions{})

		// update GlobalAlertStats with events for newly created CronJob
		if err != nil {
			r.reportErrorStatus(detectionJobState.GlobalAlert, detectionCronJobNameSpacedName, err)
			return true
		}

		detectionJobState.GlobalAlert.Status = reporting.GetGlobalAlertSuccessStatus()

		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return reporting.UpdateGlobalAlertStatus(detectionJobState.GlobalAlert, r.clusterName, r.calicoCLI,
				r.managementClusterCtx)
		}); err != nil && !errors.IsNotFound(err) {
			log.WithError(err).Errorf("failed to update status GlobalAlert %s in cluster %s, maximum retries reached",
				detectionJobState.GlobalAlert.Name, detectionJobState.ClusterName)
			return true
		}

		util.EmptyCronJobResourceValues(createdDetectionCycle)
		r.detectionCycleResourceCache.Set(detectionCronJobNameKey, *createdDetectionCycle)
		return true
	}

	// Handle Update
	// At this point the Detection CronJob for the GlobalAlert already exists update it with the CronJob contents
	// stored by r.detectionADDetectorStates
	// validate if the expected cronjob fields in the cache (disregarding the status) is equal to the one currently deployed
	if util.CronJobDeepEqualsIgnoreStatus(*detectionJobState.CronJob, *foundDetectionJob) {
		log.Debugf("Ignoring resource specific updates to %s", detectionCronJobNameKey)
		return true
	}

	log.Infof("Updating detection cronJob job %s", detectionJobState.GlobalAlert.Name)
	updatedDetectionCronJob, err := r.k8sClient.BatchV1().CronJobs(detectionJobState.CronJob.Namespace).Update(r.managementClusterCtx,
		&detectionCronJobToReconcile, metav1.UpdateOptions{})
	// update GlobalAlertStats with events for newly created CronJob
	if err != nil {
		r.reportErrorStatus(detectionJobState.GlobalAlert, detectionCronJobNameSpacedName, err)
		return true
	}

	util.EmptyCronJobResourceValues(updatedDetectionCronJob)
	r.detectionCycleResourceCache.Set(detectionCronJobNameKey, *updatedDetectionCronJob)

	// Handle GlobalAlert Success Reporting
	// report attached GlobalAlert status of reconcialiation loop
	detectionJobState.GlobalAlert.Status = reporting.GetGlobalAlertSuccessStatus()
	if len(foundDetectionJob.Status.Active) > 0 {
		detectionJobState.GlobalAlert.Status = r.getLatestJobStatusOfCronJob(r.managementClusterCtx,
			foundDetectionJob)
	}
	detectionJobState.GlobalAlert.Status.LastExecuted = foundDetectionJob.Status.LastSuccessfulTime
	detectionJobState.GlobalAlert.Status.LastEvent = foundDetectionJob.Status.LastScheduleTime

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return reporting.UpdateGlobalAlertStatus(detectionJobState.GlobalAlert, r.clusterName, r.calicoCLI,
			r.managementClusterCtx)
	}); err != nil {
		log.WithError(err).Errorf("failed to update status GlobalAlert %s, maximum retries reached",
			detectionJobState.GlobalAlert.Name)
	}

	return true
}

// getLatestJobStatusOfCronJob retrieves the status of the latest run job managed by the cronjob
func (r *adDetectionReconciler) getLatestJobStatusOfCronJob(ctx context.Context,
	cronjob *batchv1.CronJob) v3.GlobalAlertStatus {

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

	// no currently running jobs return
	if len(childJobs.Items) < 1 {
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
		if condition.Type == batchv1.JobFailed && condition.Status == "True" {
			jobError := fmt.Errorf("failed Job %s on CronJob %s error: %s", latestChildJob.Name, cronjob.Name,
				condition.Message)
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
func (r *adDetectionReconciler) reportErrorStatus(alert *v3.GlobalAlert, namespacedName types.NamespacedName, err error) {
	if alert == nil {
		return
	}

	log.WithError(err).Errorf("failed to reconcile detection cycle for %s", namespacedName.Name)

	formattedError := fmt.Errorf("unhealthy CronJob %s, with error: %s", namespacedName.Name, err.Error())
	globalAlertErrorStatus := reporting.GetGlobalAlertErrorStatus(formattedError)

	alert.Status = globalAlertErrorStatus

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return reporting.UpdateGlobalAlertStatus(alert, r.clusterName, r.calicoCLI, r.managementClusterCtx)
	}); err != nil {
		log.WithError(err).Errorf("failed to update status GlobalAlert %s in cluster %s, maximum retries reached",
			namespacedName.Name, r.clusterName)
	}
}

// Close cancels the ADJobController worker context and removes for all resources/objects that worker watches.
func (r *adDetectionReconciler) Close() {
	r.detectionJobsMutex.Lock()
	defer r.detectionJobsMutex.Unlock()

	for detectionStateKey := range r.detectionADDetectorStates {
		r.removeDetectionCycleFromResourceCache(detectionStateKey)
	}
}

// addDetector adds to the list of jobs stored for each GlobalAlert, called by the detection controller and creates the
// cronjob reference based on the AnomalyDetection GlobalAlert deployed.  The updated detection cronjob
// will be deployed next iteration of the Reconcile() loop.
func (r *adDetectionReconciler) addDetector(detectionResource DetectionCycleRequest) error {
	r.detectionJobsMutex.Lock()
	defer r.detectionJobsMutex.Unlock()
	log.Infof("Initializing the detection cycle for Anomaly Detection for %s", detectionResource.GlobalAlert.Name)

	adJobPT, err := r.podTemplateQuery.GetPodTemplate(r.managementClusterCtx, r.namespace,
		ADDetectionJobTemplateName)
	if err != nil {
		return err
	}

	detectionResourceCronJob, err := r.createDetectionCycle(adJobPT, detectionResource)
	if err != nil {
		log.WithError(err).Errorf("failed to create detection cycle for GlobalAlert %s", detectionResource.GlobalAlert.Name)
		return err
	}

	r.detectionADDetectorStates[detectionResourceCronJob.Name] = detectionCycleState{
		ClusterName: detectionResource.ClusterName,
		GlobalAlert: detectionResource.GlobalAlert,
		CronJob:     detectionResourceCronJob,
	}
	r.detectionCycleResourceCache.Set(detectionResourceCronJob.Name, *detectionResourceCronJob)
	return nil
}

// createDetectionCycle creates the CronJob from the podTemplate for the AD Job and adds it to AnomalyDetectionController
func (r *adDetectionReconciler) createDetectionCycle(podTemplate *v1.PodTemplate, detectionResource DetectionCycleRequest) (*batchv1.CronJob, error) {

	globalAlert := detectionResource.GlobalAlert

	detectionSchedule := DefaultCronJobDetectionSchedule
	if globalAlert.Spec.Period != nil {
		detectionSchedule = globalAlert.Spec.Period.Duration
	}

	err := podtemplate.DecoratePodTemplateForADDetectorCycle(podTemplate, detectionResource.ClusterName,
		podtemplate.ADJobDetectCycleArg, globalAlert.Spec.Detector, detectionSchedule.String())

	if err != nil {
		return nil, err
	}

	detectionCronJobName := r.getDetectionCycleCronJobNameForGlobaAlert(globalAlert.Name)
	detectionLabels := DetectionJobLabels()
	detectionLabels[ClusterKey] = detectionResource.ClusterName

	detectionCycleCronJob := podtemplate.CreateCronJobFromPodTemplate(detectionCronJobName, r.namespace,
		detectionSchedule, detectionLabels, *podTemplate)

	// attached detection cronjob to GlobalAlert so it will be garbage collected if GlobalAlert is deleted
	detectionCycleCronJob.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(globalAlert, GlobalAlertGroupVersionKind),
	}

	detectionCycleCronJob.ResourceVersion = ""
	detectionCycleCronJob.UID = ""
	detectionCycleCronJob.Status = batchv1.CronJobStatus{}

	if err != nil {
		return nil, err
	}

	return detectionCycleCronJob, nil
}

// getDetectionCycleCronJobNameForGlobaAlert creates a shortned RFC1123 compliant name for the detection cronjob
// based on the globalalert name
func (r *adDetectionReconciler) getDetectionCycleCronJobNameForGlobaAlert(globaAlertName string) string {
	// Convert all uppercase to lower case, in order to preserve as many characters as possible.
	// Remove each non-RFC compliant character.
	rfcGlobalAlertName := strings.ToLower(globaAlertName)

	// Remove all characters that are not RFC1123.
	regexInvalidChars := regexp.MustCompile(nonRFCCompliantRegexDef)
	rfcGlobalAlertName = regexInvalidChars.ReplaceAllString(rfcGlobalAlertName, "")
	// Replace '-.', '.-' or consecutive '.' with a single '.'.
	regexPeriods := regexp.MustCompile(charactersConvertedToPeriodRegex)
	rfcGlobalAlertName = regexPeriods.ReplaceAllString(rfcGlobalAlertName, rfcNonAlphaCharPeriod)
	// Remove all '.' or '-' from the prefix and suffix of the name.
	regexPrefixSuffix := regexp.MustCompile(charactersMatchingPrefixOrSuffixRegex)
	rfcGlobalAlertName = regexPrefixSuffix.ReplaceAllString(rfcGlobalAlertName, "")

	// If all characters have been removed, replace the empty string with a 'z'.
	if len(rfcGlobalAlertName) == 0 {
		rfcGlobalAlertName = rfcWildcard
	}

	if len(rfcGlobalAlertName) > acceptableRFCGlobalAlertName {
		if rfcGlobalAlertName[acceptableRFCGlobalAlertName-1] == '.' {
			// If the last character of the substring of rfcName is '.', remove it, to avoid introducing
			// an invalid string of the form ".-" into the name.
			rfcGlobalAlertName = rfcGlobalAlertName[:acceptableRFCGlobalAlertName-1]
		} else {
			rfcGlobalAlertName = rfcGlobalAlertName[:acceptableRFCGlobalAlertName]
		}
	}

	return fmt.Sprintf("%s-%s-%s", rfcGlobalAlertName, detectionCronJobSuffix, util.ComputeSha256HashWithLimit(globaAlertName, numHashChars))
}

// removeDetector removes from the GlobalAlert from the detection state for the cluster and signals for the detection CronJob to be delete.
// It is called by the detection controller. The deletion of the detection cronjob will happen on the next iteration of the Reconcile() loop.
func (r *adDetectionReconciler) removeDetector(detectionState DetectionCycleRequest) {
	r.detectionJobsMutex.Lock()
	defer r.detectionJobsMutex.Unlock()

	detectionStateKey := r.getDetectionCycleCronJobNameForGlobaAlert(detectionState.GlobalAlert.Name)
	r.removeDetectionCycleFromResourceCache(detectionStateKey)
}

func (r *adDetectionReconciler) removeDetectionCycleFromResourceCache(key string) {
	detectorState, found := r.detectionADDetectorStates[key]

	log.Infof("Stopping the detection cycle for Anomaly Detection for %s", detectorState.GlobalAlert.Name)

	if !found {
		log.Infof("Ignoring deleting unmanaged detection resource %s", key)
		return
	}

	detectorState.GlobalAlert = nil
	detectorState.CronJob = nil

	r.detectionCycleResourceCache.Delete(key)
	r.detectionADDetectorStates[key] = detectorState
}
