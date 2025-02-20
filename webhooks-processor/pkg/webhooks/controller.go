// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package webhooks

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
	api "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/watch"

	calicoWatch "github.com/projectcalico/calico/libcalico-go/lib/watch"
)

type WebhookController struct {
	webhookEventsChan chan calicoWatch.Event
	k8sEventsChan     chan watch.Event
	updater           WebhookUpdaterInterface
	state             StateInterface
}

func SetUp(ctx context.Context, webhookController *WebhookController, webhookWatcherUpdater *WebhookWatcherUpdater) func() {
	// break up wait group to terminate updater first then controller
	// with 2 different child contexts
	ctrCtx, ctrCancel := context.WithCancel(ctx)

	uptCtx, uptCancel := context.WithCancel(ctx)

	var ctrWg sync.WaitGroup
	var uptWg sync.WaitGroup
	uptWg.Add(1)
	go webhookWatcherUpdater.Run(uptCtx, &uptWg)
	ctrWg.Add(1)
	go webhookController.Run(ctrCtx, &ctrWg)

	return func() {
		uptCancel()
		uptWg.Wait()
		ctrCancel()
		ctrWg.Wait()
	}
}

func NewWebhookController() *WebhookController {
	watcher := new(WebhookController)
	watcher.webhookEventsChan = make(chan calicoWatch.Event)
	watcher.k8sEventsChan = make(chan watch.Event)
	return watcher
}

func (c *WebhookController) WithUpdater(updater WebhookUpdaterInterface) *WebhookController {
	c.updater = updater
	return c
}

func (c *WebhookController) WithState(state StateInterface) *WebhookController {
	c.state = state
	return c
}

func (c *WebhookController) WebhookEventsChan() chan<- calicoWatch.Event {
	return c.webhookEventsChan
}

func (c *WebhookController) K8sEventsChan() chan<- watch.Event {
	return c.k8sEventsChan
}

func (c *WebhookController) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer close(c.k8sEventsChan)
	defer close(c.webhookEventsChan)
	defer wg.Done()
	defer logrus.Info("Webhook controller is terminating")

	logrus.Info("Webhook controller started")

	for {
		select {
		case event := <-c.webhookEventsChan:
			switch event.Type {
			case calicoWatch.Added, calicoWatch.Modified:
				if webhook, ok := event.Object.(*api.SecurityEventWebhook); ok {
					c.state.IncomingWebhookUpdate(ctx, webhook)
				}
			case calicoWatch.Deleted:
				if webhook, ok := event.Previous.(*api.SecurityEventWebhook); ok {
					c.state.Stop(ctx, webhook)
				}
			default:
				logrus.Debug("webhook controller k8s events unhandled event type: ", event.Type)
			}
		case webhook := <-c.state.OutgoingWebhookUpdates():
			c.updater.UpdatesChan() <- webhook
		case event := <-c.k8sEventsChan:
			switch event.Type {
			case watch.Added, watch.Modified, watch.Deleted:
				c.state.CheckDependencies(event.Object)
			default:
				logrus.Debug("webhook controller k8s events unhandled event type: ", event.Type)
			}
		case <-ctx.Done():
			c.state.StopAll()
			return
		}
	}
}
