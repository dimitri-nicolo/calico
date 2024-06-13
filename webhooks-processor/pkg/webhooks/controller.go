// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package webhooks

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/watch"

	calicoWatch "github.com/projectcalico/calico/libcalico-go/lib/watch"
	api "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

type WebhookController struct {
	webhookEventsChan chan calicoWatch.Event
	k8sEventsChan     chan watch.Event
	updater           WebhookUpdaterInterface
	state             StateInterface
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
			}
		case webhook := <-c.state.OutgoingWebhookUpdates():
			c.updater.UpdatesChan() <- webhook
		case event := <-c.k8sEventsChan:
			switch event.Type {
			case watch.Modified, watch.Deleted:
				c.state.CheckDependencies(event.Object)
			}
		case <-ctx.Done():
			c.state.StopAll()
			return
		}
	}
}
