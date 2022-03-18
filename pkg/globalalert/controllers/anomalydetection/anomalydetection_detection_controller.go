package anomalydetection

import (
	"context"
	"errors"
	"reflect"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/client-go/kubernetes"

	rcache "github.com/projectcalico/calico/kube-controllers/pkg/cache"

	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/controllers/controller"
	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/podtemplate"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"

	log "github.com/sirupsen/logrus"
)

const (
	// setting as same period as /pkg/alert/worker
	detectionCycleResourceCachePeriod = time.Second
)

var (
	DetectionJobLabels = func() map[string]string {
		return map[string]string{
			"tigera.io.detector-cycle": "detection",
		}
	}
)

type adJobDetectionController struct {
	ctx                         context.Context
	k8sClient                   kubernetes.Interface
	cancel                      context.CancelFunc
	clusterName                 string
	adJobDetectionReconciler    *adDetectionReconciler
	detectionCycleResourceCache rcache.ResourceCache
}

type DetectionCycleRequest struct {
	ClusterName string
	GlobalAlert *v3.GlobalAlert
}

// NewADJobDetectionController creates a controller that manages and reconciles the Detection Cronjobs that are created
// for each GlobalAlert created for AnomalyDetection.  The ADJobDetectionController is referenced by GlobalAlertController
// and ManagedClusterController - to handle GlobalAlert of typed AnomalyDetection in Standalone or Management clusters
// through GlobalAlertController, and Managed Cluster through ManagedClusterController.
func NewADJobDetectionController(ctx context.Context, k8sClient kubernetes.Interface, calicoCLI calicoclient.Interface,
	podTemplateQuery podtemplate.ADPodTemplateQuery, namespace string, clusterName string) controller.AnomalyDetectionController {

	adJobDetectionReconciler := &adDetectionReconciler{
		calicoCLI:                 calicoCLI,
		k8sClient:                 k8sClient,
		podTemplateQuery:          podTemplateQuery,
		detectionADDetectorStates: make(map[string]detectionCycleState),
		clusterName:               clusterName,
		namespace:                 namespace,
	}

	adDetectionController := &adJobDetectionController{
		ctx:                      ctx,
		clusterName:              clusterName,
		k8sClient:                k8sClient,
		adJobDetectionReconciler: adJobDetectionReconciler,
	}

	rcacheArgs := rcache.ResourceCacheArgs{
		ListFunc:    adJobDetectionReconciler.listDetectionCronJobs,
		ObjectType:  reflect.TypeOf(batchv1.CronJob{}),
		LogTypeDesc: "States of the deployed detection CronJob assosciated with the GlobalAlert.",
	}
	detectionCycleResourceCache := rcache.NewResourceCache(rcacheArgs)

	adDetectionController.detectionCycleResourceCache = detectionCycleResourceCache
	adJobDetectionReconciler.detectionCycleResourceCache = detectionCycleResourceCache
	adDetectionController.adJobDetectionReconciler = adJobDetectionReconciler

	return adDetectionController
}

// Run starts the ADJobDetectionController monitoring routine.
func (c *adJobDetectionController) Run(parentCtx context.Context) {
	var ctx context.Context
	ctx, c.cancel = context.WithCancel(parentCtx)
	c.adJobDetectionReconciler.managementClusterCtx = ctx

	log.Infof("Starting AD Detection controller on cluster %s", c.clusterName)

	c.detectionCycleResourceCache.Run(detectionCycleResourceCachePeriod.String())

	go c.adJobDetectionReconciler.Run(ctx.Done())
}

// Close cancels the ADJobController worker context and removes health check for all the objects that worker watches.
func (c *adJobDetectionController) Close() {
	c.adJobDetectionReconciler.Close()
	c.cancel()
}

// AddDetector adds to the list of jobs managed by the detection controller
func (c *adJobDetectionController) AddDetector(resource interface{}) error {
	managedResource, ok := resource.(DetectionCycleRequest)
	if !ok {
		return errors.New("unexpected manageed type for an ADJob Detection resource")
	}
	err := c.adJobDetectionReconciler.addDetector(managedResource)

	if err != nil {
		return err
	}
	return nil
}

// RemoveDetector removes from the list of jobs managed by the detection controller.
// Usually called when a Done() signal is received from the parent context
func (c *adJobDetectionController) RemoveDetector(resource interface{}) error {
	detectionCycle, ok := resource.(DetectionCycleRequest)
	if !ok {
		return errors.New("unexpected type for an ADJob Ddetection resource")
	}

	c.adJobDetectionReconciler.removeDetector(detectionCycle)

	return nil
}
