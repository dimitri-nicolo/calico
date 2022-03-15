package anomalydetection

import (
	"context"
	"errors"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"

	adjcontroller "github.com/tigera/intrusion-detection/controller/pkg/globalalert/controllers/anomalydetection"
	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/controllers/controller"
	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/podtemplate"
	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/reporting"

	log "github.com/sirupsen/logrus"
)

const (
	GlobalAlertAnomalyDetectionDataSource = "logs"
	ADDetectionJobTemplateName            = "tigera.io.detectors.detection"

	DefaultDetectionLookback = 1000
	DefaultTrainingPeriod    = 24 * time.Hour

	ADDetectorDetection = "detect"

	DefaultCronJobDetectionSchedule time.Duration = 15 * time.Minute

	ADJobOwnerLabelValue = "intrusion-detection-controller"

	MaxWaitTimeForTrainingJobs  = 5 * time.Minute
	MaxWaitTimeForDetectionJobs = 5 * time.Minute

	ClusterKey = "cluster"
)

var (
	// controllerKind refers to the GlobalAlert kind that the resources created / reconciled
	// by ths controller will refer to
	GlobalAlertGroupVersionKind = schema.GroupVersionKind{
		Kind:    "GlobalAlert",
		Version: v3.VersionCurrent,
		Group:   v3.Group,
	}
)

// Service is the AnomalyDetection Service used to initialize and seize the aanomaly detection
// cycles set for the the received global alert
type Service interface {
	// Start initializes the detection cycle for the ad job specified by the GlobalAlert param
	Start(parentCtx context.Context) v3.GlobalAlertStatus

	// Stop terminates the detection cycle for the AD Job specified by the GlobalAlert
	Stop() v3.GlobalAlertStatus
}

type service struct {
	clusterName string
	namespace   string

	// globalAlert has the copy of GlobalAlert, it is updated periodically when AnomalyDetection
	// is queried for alert.
	globalAlert      *v3.GlobalAlert
	calicoCLI        calicoclient.Interface
	k8sClient        kubernetes.Interface
	podTemplateQuery podtemplate.ADPodTemplateQuery

	detectionCronJobName     string
	adJobDetectionController controller.ADJobController
}

// NewService creates the Anomaly Detection service. Currently initialized per GlobalAlertController
func NewService(calicoCLI calicoclient.Interface, k8sClient kubernetes.Interface,
	podTemplateQuery podtemplate.ADPodTemplateQuery, anomalyDetectionController controller.ADJobController,
	clusterName string, namespace string, globalAlert *v3.GlobalAlert) (Service, error) {

	s := &service{
		clusterName:              clusterName,
		namespace:                namespace,
		calicoCLI:                calicoCLI,
		k8sClient:                k8sClient,
		podTemplateQuery:         podTemplateQuery,
		adJobDetectionController: anomalyDetectionController,
		globalAlert:              globalAlert,
	}

	return s, nil
}

// Start initializes the detection cycle as a CronJob for the GlobalAlert and all preliminary steps
// required which follows:
// 	1. retrieves the expected PodTemplate for the AnomalyDetection jobs
//     - does not initialize the service if the PodTemplate does not exist
//  2. verifies the existence that the model for the specified AD Job exists
//     - runs an initial training job if no models found
//  3. initialize a CronJob using the retrieved PodTemplate modified with the AnomalyDetection
//     fields specified in the GlobalAlert
//  4. the created cronjob is then added to be managed by AnomalyDetectionController
//
// Returns the GlobalAlertStatus sucessfully if the cronjob dor the detection cycle is successfully
// initialized with all the pre-requisite steps.  Reports a GlobalAlertStatus with an error and exits
// if any steps are found as unsuccessful
func (s *service) Start(parentCtx context.Context) v3.GlobalAlertStatus {

	log.Infof("Initialize Anomaly Detection Cycles for for %s", s.globalAlert.Name)

	s.detectionCronJobName = s.clusterName + "-" + s.globalAlert.Name + "-" + ADDetectorDetection

	adJobPT, err := s.podTemplateQuery.GetPodTemplate(parentCtx, s.namespace, ADDetectionJobTemplateName)

	if err != nil {
		log.WithError(err).Errorf("Omit starting Anomaly Detection service for %s, Error retrieving PodTemplate for AD Job", s.globalAlert.Name)
		s.globalAlert.Status.Active = false
		s.globalAlert.Status = reporting.GetGlobalAlertErrorStatus(err)

		return s.globalAlert.Status
	}

	// sets the recurring detection cycle
	s.globalAlert.Status = s.createDetectionCycle(parentCtx, *adJobPT, s.globalAlert.Spec.Period.Duration.String())

	return s.globalAlert.Status
}

// Stop seizes the detection cycle for the AD Job by deleting the CronJob and removing it from the
// AnomalyDetectionJobController as to not be reconciled anymore
func (s *service) Stop() v3.GlobalAlertStatus {
	if s.adJobDetectionController != nil {
		s.adJobDetectionController.RemoveManagedJob(s.detectionCronJobName)
	} else {
		log.Warningf("execute stop for an Anomaly Detection Detector: %s that has not been started", s.globalAlert.Name)
	}

	s.globalAlert.Status = reporting.GetGlobalAlertSuccessStatus()
	s.globalAlert.Status.Active = false

	return s.globalAlert.Status
}

// createDetectionCycle cretes the CronJob from the podTemplate for the AD Job and adds it to AnomalyDetectionController
func (s *service) createDetectionCycle(parentCtx context.Context, podTemplate v1.PodTemplate, period string) v3.GlobalAlertStatus {
	log.Infof("Initializing the detection cycle for Anomaly Detection for %s", s.globalAlert.Name)

	err := podtemplate.DecoratePodTemplateForADDetectorCycle(&podTemplate, s.clusterName, podtemplate.ADJobDetectCycleArg, s.globalAlert.Spec.Detector, period)
	if err != nil {
		errorAlertStatus := reporting.GetGlobalAlertErrorStatus(err)

		return errorAlertStatus
	}

	detectionCycleSchedule := DefaultCronJobDetectionSchedule

	detectionLabels := adjcontroller.DetectionJobLabels()
	detectionLabels[ClusterKey] = s.clusterName

	detectionCycleCronJob := podtemplate.CreateCronJobFromPodTemplate(s.detectionCronJobName, s.namespace,
		detectionCycleSchedule, detectionLabels, podTemplate)

	// attached detection cronjob to GlobalAlert so it will be garbage collected if GlobalAlert is deleted
	detectionCycleCronJob.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(s.globalAlert, GlobalAlertGroupVersionKind),
	}

	detectionCycleCronJob, err = s.k8sClient.BatchV1().CronJobs(s.namespace).Create(parentCtx,
		detectionCycleCronJob, metav1.CreateOptions{})

	if err != nil {
		log.WithError(err).Errorf("failed to initialize detection cycle for GlobalAlert %s", s.globalAlert.Name)

		errorAlertStatus := reporting.GetGlobalAlertErrorStatus(err)

		return errorAlertStatus
	}

	// empty resource versioning and status before saving
	detectionCycleCronJob.ResourceVersion = ""
	detectionCycleCronJob.UID = ""
	detectionCycleCronJob.Status = batchv1.CronJobStatus{}

	err = s.manageCronJob(detectionCycleCronJob)
	if err != nil {
		log.WithError(err).Errorf("failed to initialize detection cycle for GlobalAlert %s", s.globalAlert.Name)

		errorAlertStatus := reporting.GetGlobalAlertErrorStatus(err)

		return errorAlertStatus
	}

	return reporting.GetGlobalAlertSuccessStatus()
}

// manageCronJob adds the CronJob to be maanged by AnomalyDetectionController
func (s *service) manageCronJob(detectionCronJob *batchv1.CronJob) error {
	if s.adJobDetectionController == nil {
		return errors.New("detection cycle controller not found")
	}

	s.adJobDetectionController.AddToManagedJobs(adjcontroller.ManagedADDetectionJobsState{
		CronJob:     detectionCronJob,
		GlobalAlert: s.globalAlert,
	})

	return nil
}
