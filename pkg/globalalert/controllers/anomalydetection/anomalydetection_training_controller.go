package anomalydetection

import (
	"context"
	"errors"
	"reflect"
	"time"

	rcache "github.com/projectcalico/calico/kube-controllers/pkg/cache"

	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"

	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/controllers/controller"
	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/podtemplate"
	batchv1 "k8s.io/api/batch/v1"

	"k8s.io/client-go/kubernetes"
)

const (
	ADTrainingJobTemplateName       = "tigera.io.detectors.training"
	DefaultADDetectorTrainingPeriod = 24 * time.Hour

	defaultCronJobTrainingSchedule time.Duration = 24 * time.Hour
	defaultTrainingLookback                      = 1000

	trainingCronJobSuffix = "training"

	ADJobOwnerLabelValue       = "intrusion-detection-controller"
	maxWaitTimeForTrainingJobs = 5 * time.Minute
)

var (
	TrainingCronJobLabels = func() map[string]string {
		return map[string]string{
			"tigera.io.detector-cycle": "training",
		}
	}
)

type adJobTrainingController struct {
	clusterName                string
	k8sClient                  kubernetes.Interface
	cancel                     context.CancelFunc
	namespace                  string
	adTrainingReconciler       *adJobTrainingReconciler
	trainingCycleResourceCache rcache.ResourceCache
}

type TrainingDetectorsRequest struct {
	ClusterName string
	GlobalAlert *v3.GlobalAlert
}

// NewADJobTrainingController creates and reconciles cycles that train for all the AnomalyDetection models daily
// In a MCM Architecture and Calico cloud it maintains a training cronjob per cluster
func NewADJobTrainingController(k8sClient kubernetes.Interface,
	calicoCLI calicoclient.Interface, podTemplateQuery podtemplate.ADPodTemplateQuery, namespace string,
	clusterName string) controller.AnomalyDetectionController {

	adTrainingReconciler := &adJobTrainingReconciler{
		calicoCLI:                   calicoCLI,
		k8sClient:                   k8sClient,
		podTemplateQuery:            podTemplateQuery,
		namespace:                   namespace,
		trainingDetectorsPerCluster: make(map[string]trainingCycleStatePerCluster),
	}

	adTrainingController := &adJobTrainingController{
		clusterName:          clusterName,
		k8sClient:            k8sClient,
		namespace:            namespace,
		adTrainingReconciler: adTrainingReconciler,
	}

	rcacheArgs := rcache.ResourceCacheArgs{
		ListFunc:    adTrainingReconciler.listTrainingCronJobs,
		ObjectType:  reflect.TypeOf(batchv1.CronJob{}),
		LogTypeDesc: "States of the deployed training CronJob for the multiple GlobalAlerts assosciated with the cluster.",
	}
	trainingCycleResourceCache := rcache.NewResourceCache(rcacheArgs)

	adTrainingController.trainingCycleResourceCache = trainingCycleResourceCache
	adTrainingReconciler.trainingCycleResourceCache = trainingCycleResourceCache

	return adTrainingController
}

// Run intializes the ADJobTrainingController monitoring routine. Initially runs one job that trains all
// AnomalyDetection Jobs and schediles a Training CronJob for training all models that run daily
func (c *adJobTrainingController) Run(parentCtx context.Context) {
	var ctx context.Context
	ctx, c.cancel = context.WithCancel(parentCtx)
	c.adTrainingReconciler.managementClusterCtx = ctx

	log.Infof("Starting AD Training controller on cluster %s", c.clusterName)

	c.trainingCycleResourceCache.Run(detectionCycleResourceCachePeriod.String())

	go c.adTrainingReconciler.Run(ctx.Done())
}

// AddDetector adds to the list of detectors managed by the training controller.
// The managed list is declared in the reconciler.
func (c *adJobTrainingController) AddDetector(resource interface{}) error {
	trainingDetectorStateForCluster, ok := resource.(TrainingDetectorsRequest)

	if !ok {
		return errors.New("unexpected type for an ADJob Training resource")
	}

	err := c.adTrainingReconciler.addTrainingCycle(trainingDetectorStateForCluster)
	if err != nil {
		return err
	}

	return nil
}

// RemoveDetector removes from the list of jobs managed by the training controller.
// Usually called when a Done() signal is received from the parent context
func (c *adJobTrainingController) RemoveDetector(resource interface{}) error {
	trainingDetectorStateForCluster, ok := resource.(TrainingDetectorsRequest)

	if !ok {
		return errors.New("unexpected type for an ADJob Training resource")
	}

	c.adTrainingReconciler.removeTrainingCycles(trainingDetectorStateForCluster)

	return nil
}

// Close cancels the ADJobController worker context and removes health check for all
//  the objects that worker watches.
func (c *adJobTrainingController) Close() {
	c.adTrainingReconciler.Close()
	c.cancel()
}
