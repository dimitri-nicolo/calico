// Copyright (c) 2021-2022 Tigera, Inc. All rights reserved.
package anomalydetection

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/globalalert/controllers/controller"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/globalalert/podtemplate"
	rcache "github.com/projectcalico/calico/kube-controllers/pkg/cache"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"
)

const (
	ADTrainingJobTemplateName         = "tigera.io.detectors.training"
	DefaultADDetectorTrainingSchedule = 1 * time.Hour

	initialTrainingJobSuffix = "initial-training"
	trainingCycleSuffix      = "training"

	ADJobOwnerLabelValue = "intrusion-detection-controller"
)

var (
	TrainingJobLabels = func() map[string]string {
		return map[string]string{
			"tigera.io.detector-cycle": "training",
		}
	}

	TrainingCycleLabels = func() map[string]string {
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
	ClusterName      string
	IsManagedCluster bool
	GlobalAlert      *v3.GlobalAlert
}

// NewADJobTrainingController creates and reconciles cycles that train for all the AnomalyDetection models daily
// In a MCM Architecture and Calico cloud it maintains a training cronjob per cluster
func NewADJobTrainingController(k8sClient kubernetes.Interface,
	calicoCLI calicoclient.Interface, podTemplateQuery podtemplate.ADPodTemplateQuery, namespace string,
	clusterName string) controller.AnomalyDetectionController {

	adTrainingReconciler := &adJobTrainingReconciler{
		managementClusterName:       clusterName,
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
		return fmt.Errorf("unexpected type for an ADJob Training resource")
	}

	// Kicks-off an initial training job if the training cycle isn't found or for a first time
	// detector, otherwise returns.
	log.Infof("Run initial training job for cluster: %s",
		trainingDetectorStateForCluster.ClusterName)
	err := c.adTrainingReconciler.runInitialTrainingJob(trainingDetectorStateForCluster)
	if err != nil {
		// No need to continue with training cycles, as the pod template will not available for cronJobs
		// as well.
		var podTemplateError *PodTemplateError
		if errors.As(err, &podTemplateError) {
			log.Error(err)
			return err
		}
		log.Warn("Initial training job cannot complete rely on training model fallbacks")
	}

	log.Infof("Add a training cycle cronJob for cluster: %s",
		trainingDetectorStateForCluster.ClusterName)
	err = c.adTrainingReconciler.addTrainingCycle(trainingDetectorStateForCluster)
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

	if err := c.adTrainingReconciler.removeTrainingCycles(trainingDetectorStateForCluster); err != nil {
		return err
	}

	return nil
}

func (c *adJobTrainingController) StopADForCluster(clusterName string) {
	_ = c.adTrainingReconciler.stopTrainigCycleForCluster(clusterName)
}

// Close cancels the ADJobController worker context and removes health check for all
//
//	the objects that worker watches.
func (c *adJobTrainingController) Close() {
	c.adTrainingReconciler.Close()
	c.cancel()
}
