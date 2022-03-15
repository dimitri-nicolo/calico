package anomalydetection

import (
	"context"
	"errors"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/controllers/controller"
	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/worker"
	"github.com/tigera/intrusion-detection/controller/pkg/maputil"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"

	log "github.com/sirupsen/logrus"
)

var (
	DetectionJobLabels = func() map[string]string {
		return map[string]string{
			"tigera.io.detector-cycle": "detection",
		}
	}
)

type adJobDetectionController struct {
	clusterName              string
	k8sClient                kubernetes.Interface
	cancel                   context.CancelFunc
	worker                   worker.Worker
	adJobDetectionReconciler *adJobDetectionReconciler
}

type ManagedADDetectionJobsState struct {
	CronJob     *batchv1.CronJob
	GlobalAlert *v3.GlobalAlert
}

// NewADJobDetectionController creates a controller that manages and reconciles the Detection Cronjobs that are created
// for each GlobalAlert created for AnomalyDetection.  The ADJobDetectionController is referenced by GlobalAlertController
// and ManagedClusterController - to handle GlobalAlert of typed AnomalyDetection in Standalone or Management clusters
// through GlobalAlertController, and Managed Cluster through ManagedClusterController.
func NewADJobDetectionController(k8sClient kubernetes.Interface, calicoCLI calicoclient.Interface, namespace string,
	clusterName string) controller.ADJobController {

	adJobReconciler := &adJobDetectionReconciler{
		calicoCLI:            calicoCLI,
		k8sClient:            k8sClient,
		managedDetectionJobs: make(map[string]ManagedADDetectionJobsState),
		clusterName:          clusterName,
	}

	adDetectionController := &adJobDetectionController{
		clusterName:              clusterName,
		k8sClient:                k8sClient,
		adJobDetectionReconciler: adJobReconciler,
	}

	adDetectionController.worker = worker.New(adJobReconciler)

	detectionJobLabelByteStr := maputil.CreateLabelValuePairStr(DetectionJobLabels())

	optionsModifier := func(options *metav1.ListOptions) {
		options.LabelSelector = detectionJobLabelByteStr
	}

	adDetectionController.worker = worker.New(adJobReconciler)
	adDetectionController.worker.AddWatch(
		cache.NewFilteredListWatchFromClient(adDetectionController.k8sClient.BatchV1().RESTClient(), "cronjobs", namespace,
			optionsModifier),
		&batchv1.CronJob{})

	return adDetectionController
}

// Run starts the ADJobDetectionController monitoring routine.
func (c *adJobDetectionController) Run(parentCtx context.Context) {
	var ctx context.Context
	ctx, c.cancel = context.WithCancel(parentCtx)

	c.adJobDetectionReconciler.managementClusterCtx = ctx

	log.Infof("Starting AD Job Detection controller on cluster %s", c.clusterName)
	go c.worker.Run(ctx.Done())
}

// Close cancels the ADJobController worker context and removes health check for all the objects that worker watches.
func (c *adJobDetectionController) Close() {
	c.worker.Close()
	c.cancel()
}

// AddToManagedJobs adds to the list of jobs managed by the detection controller
func (c *adJobDetectionController) AddToManagedJobs(resource interface{}) error {
	managedResource, ok := resource.(ManagedADDetectionJobsState)

	if !ok {
		return errors.New("unexpected manageed type for an ADJob Detection resource")
	}
	c.adJobDetectionReconciler.addToManagedDetectionJobs(managedResource)

	return nil
}

// RemoveManagedJob removes from the list of jobs managed by the detection controller.
// Usually called when a Done() signal is received from the parent context
func (c *adJobDetectionController) RemoveManagedJob(cronJobName string) {
	c.adJobDetectionReconciler.removeManagedDetectionJobs(cronJobName)
}
