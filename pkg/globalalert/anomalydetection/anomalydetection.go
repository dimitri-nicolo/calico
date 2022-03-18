package anomalydetection

import (
	"context"
	"errors"
	"time"

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

	DefaultDetectionLookback = 1000
	DefaultTrainingPeriod    = 24 * time.Hour

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
		Kind:    v3.KindGlobalAlert,
		Version: v3.VersionCurrent,
		Group:   v3.Group,
	}
)

// ADService is the AnomalyDetection ADService used to initialize and seize the aanomaly detection
// cycles set for the the received global alert
type ADService interface {
	// Start initializes the detection cycle for the ad job specified by the GlobalAlert param
	Start(parentCtx context.Context) v3.GlobalAlertStatus

	// Stop terminates the detection cycle for the AD Job specified by the GlobalAlert
	Stop() v3.GlobalAlertStatus
}

type adService struct {
	clusterName string
	namespace   string

	// globalAlert has the copy of GlobalAlert, it is updated periodically when AnomalyDetection
	// is queried for alert.
	globalAlert      *v3.GlobalAlert
	calicoCLI        calicoclient.Interface
	k8sClient        kubernetes.Interface
	podTemplateQuery podtemplate.ADPodTemplateQuery

	adDetectionController controller.AnomalyDetectionController
	adTrainingController  controller.AnomalyDetectionController
}

// NewService creates the Anomaly Detection service. Currently initialized per GlobalAlertController
func NewService(calicoCLI calicoclient.Interface, k8sClient kubernetes.Interface,
	podTemplateQuery podtemplate.ADPodTemplateQuery, anomalyDetectionController controller.AnomalyDetectionController,
	anomalyDetectionTrainingController controller.AnomalyDetectionController,
	clusterName string, namespace string, globalAlert *v3.GlobalAlert) (ADService, error) {

	s := &adService{
		clusterName:           clusterName,
		namespace:             namespace,
		calicoCLI:             calicoCLI,
		k8sClient:             k8sClient,
		podTemplateQuery:      podTemplateQuery,
		adDetectionController: anomalyDetectionController,
		adTrainingController:  anomalyDetectionTrainingController,
		globalAlert:           globalAlert,
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
func (s *adService) Start(parentCtx context.Context) v3.GlobalAlertStatus {

	log.Infof("Initialize Anomaly Detection Cycles for %s", s.globalAlert.Name)

	// sets the recurring detection cycle
	err := s.manageDetectionGlobalAlert()
	if err != nil {
		log.WithError(err).Errorf("failed to manage detection cycle for GlobalAlert %s", s.globalAlert.Name)

		s.globalAlert.Status = reporting.GetGlobalAlertErrorStatus(err)
		s.globalAlert.Status.Active = false
		return s.globalAlert.Status
	}

	return reporting.GetGlobalAlertSuccessStatus()
}

// Stop seizes the detection cycle for the AD Job by deleting the CronJob and removing it from the
// AnomalyDetectionJobController as to not be reconciled anymore
func (s *adService) Stop() v3.GlobalAlertStatus {
	if s.adDetectionController == nil || s.adTrainingController == nil {
		log.Warningf("executed stop for an Anomaly Detection Detector: %s that has not been started", s.globalAlert.Name)
	}

	err := s.adDetectionController.RemoveDetector(adjcontroller.DetectionCycleRequest{
		ClusterName: s.clusterName,
		GlobalAlert: s.globalAlert,
	})

	if err != nil {
		log.WithError(err).Errorf("failed to remove detection cycle for GlobalAlert %s", s.globalAlert.Name)

		s.globalAlert.Status = reporting.GetGlobalAlertErrorStatus(err)
		s.globalAlert.Status.Active = false
		return s.globalAlert.Status
	}

	err = s.adTrainingController.RemoveDetector(adjcontroller.TrainingDetectorsRequest{
		ClusterName: s.clusterName,
		GlobalAlert: s.globalAlert,
	})
	if err != nil {
		log.WithError(err).Errorf("failed to remove from training cycle for GlobalAlert %s", s.globalAlert.Name)

		s.globalAlert.Status = reporting.GetGlobalAlertErrorStatus(err)
		s.globalAlert.Status.Active = false
		return s.globalAlert.Status
	}

	s.globalAlert.Status = reporting.GetGlobalAlertSuccessStatus()
	s.globalAlert.Status.Active = false

	return s.globalAlert.Status
}

// manageCronJob adds the CronJob to be maanged by AnomalyDetectionController
func (s *adService) manageDetectionGlobalAlert() error {
	if s.adDetectionController == nil || s.adTrainingController == nil {
		return errors.New("anomaly detection controllers cycle controller not found")
	}

	err := s.adDetectionController.AddDetector(adjcontroller.DetectionCycleRequest{
		ClusterName: s.clusterName,
		GlobalAlert: s.globalAlert,
	})

	if err != nil {
		return err
	}

	err = s.adTrainingController.AddDetector(adjcontroller.TrainingDetectorsRequest{
		ClusterName: s.clusterName,
		GlobalAlert: s.globalAlert,
	})

	if err != nil {
		return err
	}

	return nil
}
