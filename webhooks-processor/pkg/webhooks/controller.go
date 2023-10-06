// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package webhooks

import (
	"context"
	"sync"

	"github.com/projectcalico/calico/libcalico-go/lib/watch"
	"github.com/sirupsen/logrus"
	api "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

type WebhookController struct {
	eventsChan chan watch.Event
	updater    WebhookUpdaterInterface
	state      StateInterface
}

func NewWebhookController() *WebhookController {
	watcher := new(WebhookController)
	watcher.eventsChan = make(chan watch.Event)
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

func (c *WebhookController) EventsChan() chan<- watch.Event {
	return c.eventsChan
}

func (c *WebhookController) Run(ctx context.Context, ctxCancel context.CancelFunc, wg *sync.WaitGroup) {
	defer ctxCancel()
	defer wg.Done()
	defer logrus.Info("Webhook controller is terminating")

	logrus.Info("Webhook controller started")

	for {
		select {
		case event := <-c.eventsChan:
			switch event.Type {
			case watch.Added, watch.Modified:
				if webhook, ok := event.Object.(*api.SecurityEventWebhook); ok {
					c.state.IncomingWebhookUpdate(ctx, webhook)
				}
			case watch.Deleted:
				if webhook, ok := event.Previous.(*api.SecurityEventWebhook); ok {
					c.state.Stop(ctx, webhook)
				}
			}
		case webhook := <-c.state.OutgoingWebhookUpdates():
			c.updater.UpdatesChan() <- webhook
		case <-ctx.Done():
			c.state.StopAll()
			return
		}
	}
}
