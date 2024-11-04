// Copyright (c) 2021-2022 Tigera, Inc. All rights reserved.
package anomalydetection

import (
	"context"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/globalalert/controllers/controller"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/maputil"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/util"
)

var (
	TrainingCycleLabels = func() map[string]string {
		return map[string]string{
			"tigera.io.detector-cycle": "training",
		}
	}
)

type adJobTrainingController struct {
	kubeClientSet kubernetes.Interface
	namespace     string
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewADJobTrainingController creates a controller that cleans up any AD training cron jobs.
func NewADJobTrainingController(kubeClientSet kubernetes.Interface, namespace string) controller.Controller {
	c := &adJobTrainingController{
		kubeClientSet: kubeClientSet,
		namespace:     namespace,
	}
	return c
}

func (c *adJobTrainingController) Run(parentCtx context.Context) {
	c.ctx, c.cancel = context.WithCancel(parentCtx)

	log.Info("Starting AD training controller")

	go c.cleanup()
}

func (c *adJobTrainingController) Close() {
	if c.ctx != nil {
		c.cancel()
	}
}

// Cleanup AD training cron jobs.
func (c *adJobTrainingController) cleanup() {

	// List all Tigera-created AD detection cron jobs.
	trainingCycleLabelByteStr := maputil.CreateLabelValuePairStr(TrainingCycleLabels())
	trainingCronJobList, err := c.kubeClientSet.BatchV1().CronJobs(c.namespace).List(c.ctx,
		metav1.ListOptions{
			LabelSelector: trainingCycleLabelByteStr,
		})
	if err != nil {
		log.WithError(err).Errorf("failed to list training cronjobs")
		return
	}

	for _, trainingCronJob := range trainingCronJobList.Items {
		// Check in case context has been cancelled.
		if c.ctx.Err() != nil {
			break
		}

		// Delete next cron job in the list.
		log.Infof("Deleting AD training cronJob job %s", trainingCronJob.Name)
		util.EmptyCronJobResourceValues(&trainingCronJob)

		err := util.DeleteCronJobWithRetry(c.ctx, c.kubeClientSet, c.namespace, trainingCronJob.Name)
		if err != nil && !errors.IsNotFound(err) {
			log.WithError(err).Errorf("Unable to delete stored training CronJob %s", trainingCronJob.Name)
		}
	}

	// Wait for context to be cancelled before returning.
	<-c.ctx.Done()
}
