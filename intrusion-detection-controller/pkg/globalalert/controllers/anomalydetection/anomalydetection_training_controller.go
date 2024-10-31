// Copyright (c) 2021-2022 Tigera, Inc. All rights reserved.
package anomalydetection

import (
	"context"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/controller"
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

	fifo *cache.DeltaFIFO
	ping chan struct{}
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
	c.pong()
}

func (c *adJobTrainingController) Close() {
	if c.ctx != nil {
		c.cancel()
	}
}

func (c *adJobTrainingController) Ping(ctx context.Context) error {
	// Enqueue a ping
	err := c.fifo.Update(util.Ping{})
	if err != nil {
		// Local fifo & cache should never error.
		panic(err)
	}

	// Wait for the ping to be processed, or context to expire.
	select {
	case <-ctx.Done():
		return ctx.Err()

	// Since this channel is unbuffered, this will block if the main loop is not
	// running, or has itself blocked.
	case <-c.ping:
		return nil
	}
}

// pong is called from the main processing loop to reply to a ping.
func (c *adJobTrainingController) pong() {
	// Nominally, a sync.Cond would work nicely here rather than a channel,
	// which would allow us to wake up all pingers at once. However, sync.Cond
	// doesn't allow timeouts, so we stick with channels and one pong() per ping.
	c.ping <- struct{}{}
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
