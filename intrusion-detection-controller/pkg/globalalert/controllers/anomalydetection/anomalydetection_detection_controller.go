// Copyright (c) 2021-2023 Tigera, Inc. All rights reserved.
package anomalydetection

import (
	"context"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/controller"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/maputil"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/util"
)

var (
	DetectionJobLabels = func() map[string]string {
		return map[string]string{
			"tigera.io.detector-cycle": "detection",
		}
	}
)

type adJobDetectionController struct {
	kubeClientSet kubernetes.Interface
	namespace     string
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewADJobDetectionController creates a controller that cleans up any AD detection cron jobs.
func NewADJobDetectionController(kubeClientSet kubernetes.Interface, namespace string) controller.Controller {
	c := &adJobDetectionController{
		kubeClientSet: kubeClientSet,
		namespace:     namespace,
	}
	return c
}

func (c *adJobDetectionController) Run(parentCtx context.Context) {
	c.ctx, c.cancel = context.WithCancel(parentCtx)

	log.Info("Starting AD detection controller")

	go c.cleanup()
}

func (c *adJobDetectionController) Close() {
	if c.ctx != nil {
		c.cancel()
	}
}

func (c *adJobDetectionController) Ping(ctx context.Context) error {
	return nil
}

// Cleanup AD detection cron jobs.
func (c *adJobDetectionController) cleanup() {

	// List all Tigera-created AD detection cron jobs.
	detectionJobLabelByteStr := maputil.CreateLabelValuePairStr(DetectionJobLabels())
	detectionCronJobList, err := c.kubeClientSet.BatchV1().CronJobs(c.namespace).List(c.ctx,
		metav1.ListOptions{
			LabelSelector: detectionJobLabelByteStr,
		})
	if err != nil {
		log.WithError(err).Errorf("failed to list detection cronjobs")
		return
	}

	for _, detectionCronJob := range detectionCronJobList.Items {
		// Check in case context has been cancelled.
		if c.ctx.Err() != nil {
			break
		}

		// Delete next cron job in the list.
		log.Infof("Deleting AD detection cronJob job %s", detectionCronJob.Name)
		util.EmptyCronJobResourceValues(&detectionCronJob)

		err := util.DeleteCronJobWithRetry(c.ctx, c.kubeClientSet, c.namespace, detectionCronJob.Name)
		if err != nil && !errors.IsNotFound(err) {
			log.WithError(err).Errorf("Unable to delete stored detection CronJob %s", detectionCronJob.Name)
		}
	}

	// Wait for context to be cancelled before returning.
	<-c.ctx.Done()
}
