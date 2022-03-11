package anomalydetection

import (
	"context"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"

	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/controllers/controller"
	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/podtemplate"
	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/worker"
	"github.com/tigera/intrusion-detection/controller/pkg/health"
	"github.com/tigera/intrusion-detection/controller/pkg/maputil"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	ADTrainingJobTemplateName       = "tigera.io.adjob.training"
	DefaultADDetectorTrainingPeriod = 24 * time.Hour

	defaultCronJobTrainingSchedule time.Duration = 24 * time.Hour
	defaultTrainingLookback                      = 1000

	trainingCronJobSuffix = "-training"

	ADJobOwnerLabelValue       = "intrusion-detection-controller"
	maxWaitTimeForTrainingJobs = 5 * time.Minute
)

var (
	TrainingJobLabels = func() map[string]string {
		return map[string]string{
			"tigera.io.detector-cycle": "training",
		}
	}
)

type adJobTrainingController struct {
	clusterName          string
	k8sClient            kubernetes.Interface
	cancel               context.CancelFunc
	worker               worker.Worker
	podTemplateQuery     podtemplate.ADPodTemplateQuery
	namespace            string
	adTrainingReconciler *adJobTrainingReconciler
}

type ManagedTrainingJobsState struct {
	ClusterName string
	CronJob     *batchv1.CronJob
}

// NewADJobTrainingController creates and reconciles cycles that train for all the AnomalyDetection models daily
// In a MCM Architecture and Calico cloud it maintains a training cronjob per cluster
func NewADJobTrainingController(k8sClient kubernetes.Interface,
	calicoCLI calicoclient.Interface, podTemplateQuery podtemplate.ADPodTemplateQuery, namespace string,
	clusterName string) (controller.ADJobController, []health.Pinger) {

	adTrainingReconciler := &adJobTrainingReconciler{
		calicoCLI:               calicoCLI,
		k8sClient:               k8sClient,
		podTemplateQuery:        podTemplateQuery,
		namespace:               namespace,
		managedTrainingCronJobs: make(map[string]*batchv1.CronJob),
	}

	adTrainingController := &adJobTrainingController{
		clusterName:          clusterName,
		k8sClient:            k8sClient,
		podTemplateQuery:     podTemplateQuery,
		namespace:            namespace,
		adTrainingReconciler: adTrainingReconciler,
	}

	adTrainingController.worker = worker.New(adTrainingReconciler)

	tainingJobLabelByteStr := maputil.CreateLabelValuePairStr(TrainingJobLabels())

	optionsModifier := func(options *metav1.ListOptions) {
		options.LabelSelector = tainingJobLabelByteStr
	}

	pinger := adTrainingController.worker.AddWatch(
		cache.NewFilteredListWatchFromClient(adTrainingController.k8sClient.BatchV1().RESTClient(), "cronjobs", namespace,
			optionsModifier),
		&batchv1.CronJob{})

	return adTrainingController, []health.Pinger{pinger}
}

// Run intializes the ADJobTrainingController monitoring routine. Initially runs one job that trains all
// AnomalyDetection Jobs and schediles a Training CronJob for training all models that run daily
func (c *adJobTrainingController) Run(parentCtx context.Context) {
	var ctx context.Context
	ctx, c.cancel = context.WithCancel(parentCtx)

	c.adTrainingReconciler.managementClusterCtx = ctx

	// initializing training cronjob and jobs will be done in the reconciler
	err := c.AddToManagedJobs(c.clusterName)

	if err != nil {
		log.WithError(err).Error("Unable to start training cycles for the models, unable to add training cycles to controller")
		return
	}

	log.Infof("Starting AD Job Training controller on cluster %s", c.clusterName)
	go c.worker.Run(ctx.Done())
}

// AddToManagedJobs adds to the list of jobs managed by the training controller
func (c *adJobTrainingController) AddToManagedJobs(resource interface{}) error {
	managedClusterName, ok := resource.(string)

	if !ok {
		return errors.New("unexpected type for an ADJob Training resource")
	}

	err := c.adTrainingReconciler.addToManagedTrainingJobs(managedClusterName)
	if err != nil {
		return err
	}

	return nil
}

// RemoveManagedJob removes from the list of jobs managed by the training controller.
// Usually called when a Done() signal is received from the parent context
func (c *adJobTrainingController) RemoveManagedJob(cronJobName string) {
	c.adTrainingReconciler.removeManagedTrainingJobs(cronJobName)
}

// Close cancels the ADJobController worker context and removes health check for all
//  the objects that worker watches.
func (c *adJobTrainingController) Close() {
	c.worker.Close()
	c.cancel()
}
