// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package webhooks

import (
	"context"
	"sync"
	"time"

	api "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	toolsWatch "k8s.io/client-go/tools/watch"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
)

const (
	WebhooksWatcherTimeout    = 1 * time.Minute
	WebhooksWatcherSleepOnErr = 5 * time.Second
)

type WebhookWatcherUpdater struct {
	client             kubernetes.Interface
	whClient           clientv3.SecurityEventWebhookInterface
	controller         WebhookControllerInterface
	webhookUpdatesChan chan *api.SecurityEventWebhook
}

func NewWebhookWatcherUpdater() (watcher *WebhookWatcherUpdater) {
	watcher = new(WebhookWatcherUpdater)
	watcher.webhookUpdatesChan = make(chan *api.SecurityEventWebhook)
	return
}

func (w *WebhookWatcherUpdater) WithWebhooksClient(client clientv3.SecurityEventWebhookInterface) *WebhookWatcherUpdater {
	w.whClient = client
	return w
}

func (w *WebhookWatcherUpdater) WithK8sClient(client kubernetes.Interface) *WebhookWatcherUpdater {
	w.client = client
	return w
}

func (w *WebhookWatcherUpdater) WithController(controller WebhookControllerInterface) *WebhookWatcherUpdater {
	w.controller = controller
	return w
}

func (w *WebhookWatcherUpdater) UpdatesChan() chan<- *api.SecurityEventWebhook {
	return w.webhookUpdatesChan
}

func (w *WebhookWatcherUpdater) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer logrus.Info("Webhook watcher is terminating")

	for {
		if err := w.run(ctx); err != nil {
			logrus.WithError(err).Error("Webhook watcher encountered an error")
		}
		// delay before retrying
		time.Sleep(WebhooksWatcherSleepOnErr)
	}
}

func (w *WebhookWatcherUpdater) run(ptx context.Context) error {
	// create a new context with cancel
	// to ensure all watchers are stopped when this function exits
	ctx, cancel := context.WithCancel(ptx)
	defer cancel()

	// initialize configmap and secret watchers
	cmWatcher, err := toolsWatch.NewRetryWatcher("1", &cache.ListWatch{
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return w.client.CoreV1().ConfigMaps(ConfigVarNamespace).Watch(ctx, metav1.ListOptions{})
		},
	})
	if err != nil {
		logrus.WithError(err).Error("Unable to watch ConfigMap resources")
		return err
	}
	secretWatcher, err := toolsWatch.NewRetryWatcher("1", &cache.ListWatch{
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return w.client.CoreV1().Secrets(ConfigVarNamespace).Watch(ctx, metav1.ListOptions{})
		},
	})
	if err != nil {
		logrus.WithError(err).Error("Unable to watch Secret resources")
		return err
	}

	// initialize webhook watcher
	watcher, err := w.whClient.Watch(ctx, options.ListOptions{})
	if err != nil {
		logrus.WithError(err).Error("Unable to watch for SecurityEventWebhook resources")
		return err
	}

	logrus.Info("Watching for webhook definitions")
	for {
		select {
		case webhook := <-w.webhookUpdatesChan:
			logEntry(webhook).Debug("Updating webhook")
			if _, err := w.whClient.Update(ctx, webhook, options.SetOptions{}); err != nil {
				logrus.WithError(err).Warn("Unable to update SecurityEventWebhook definition")
			}
		case configMapEvent := <-cmWatcher.ResultChan():
			w.controller.K8sEventsChan() <- configMapEvent
		case secretEvent := <-secretWatcher.ResultChan():
			w.controller.K8sEventsChan() <- secretEvent
		case webhookEvent := <-watcher.ResultChan():
			w.controller.WebhookEventsChan() <- webhookEvent
		case <-ctx.Done():
			return nil
		}
	}
}
