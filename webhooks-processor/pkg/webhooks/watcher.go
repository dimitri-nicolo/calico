// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package webhooks

import (
	"context"
	"sync"

	api "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
)

type WebhookWatcherUpdater struct {
	whClient           clientv3.SecurityEventWebhookInterface
	controller         WebhookControllerInterface
	webhookUpdatesChan chan *api.SecurityEventWebhook
}

func NewWebhookWatcherUpdater() (watcher *WebhookWatcherUpdater) {
	watcher = new(WebhookWatcherUpdater)
	watcher.webhookUpdatesChan = make(chan *api.SecurityEventWebhook)
	return
}

func (w *WebhookWatcherUpdater) WithClient(client clientv3.SecurityEventWebhookInterface) *WebhookWatcherUpdater {
	w.whClient = client
	return w
}

func (w *WebhookWatcherUpdater) WithController(controller WebhookControllerInterface) *WebhookWatcherUpdater {
	w.controller = controller
	return w
}

func (w *WebhookWatcherUpdater) UpdatesChan() chan<- *api.SecurityEventWebhook {
	return w.webhookUpdatesChan
}

func (w *WebhookWatcherUpdater) Run(ctx context.Context, ctxCancel context.CancelFunc, wg *sync.WaitGroup) {
	defer ctxCancel()
	defer wg.Done()
	defer logrus.Info("Webhook watcher is terminating")

	logrus.Info("Watching for webhook definitions")

	go func() {
		for {
			select {
			case webhook := <-w.webhookUpdatesChan:
				logEntry(webhook).Debug("Updating webhook")
				if _, err := w.whClient.Update(ctx, webhook, options.SetOptions{}); err != nil {
					logrus.WithError(err).Warn("Unable to update SecurityEventWebhook definition")
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	for ctx.Err() == nil {
		watcher, err := w.whClient.Watch(ctx, options.ListOptions{})
		if err != nil {
			logrus.WithError(err).Error("Unable to watch for SecurityEventWebhook resources")
			return
		}
		for event := range watcher.ResultChan() {
			w.controller.EventsChan() <- event
		}
	}
}
