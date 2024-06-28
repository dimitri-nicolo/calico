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

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
)

const (
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

	webhookUpdates := make(chan *api.SecurityEventWebhook)

	go func() {
		for {
			select {
			case webhook := <-webhookUpdates:
				logEntry(webhook).Debug("Updating webhook")
				if _, err := w.whClient.Update(ctx, webhook, options.SetOptions{}); err != nil {
					logrus.WithError(err).Warn("Unable to update SecurityEventWebhook definition")
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// initialize configmap and secret watchers
	var cmWatcher watch.Interface
	var secretWatcher watch.Interface
	var err error
	var useSecret, useConfigmaps bool

	errorCh := make(chan error, 1)
	// start webhook watcher in its own retry loop goroutine
	go w.webhookRetryWatcher(ctx, errorCh)

	for {

		select {
		case webhook := <-w.webhookUpdatesChan:
			webhookUpdates <- webhook
			useConfigmaps, useSecret = w.checkWebhooksForConfigmapsAndSecret(ctx)
		case <-ctx.Done():
			return
		case err = <-errorCh:
			logrus.Debug(err)
		}

		if useConfigmaps && cmWatcher == nil {
			cmWatcher, err = w.client.CoreV1().ConfigMaps(ConfigVarNamespace).Watch(ctx, metav1.ListOptions{})

			if err != nil {
				logrus.WithError(err).Error("Unable to watch ConfigMap resources")
				return
			}
		}

		if useSecret && secretWatcher == nil {
			secretWatcher, err = w.client.CoreV1().Secrets(ConfigVarNamespace).Watch(ctx, metav1.ListOptions{})

			if err != nil {
				logrus.WithError(err).Error("Unable to watch Secret resources")
				return
			}
		}

		if cmWatcher != nil {
			configMapEvent, ok := <-cmWatcher.ResultChan()
			if ok {
				w.controller.K8sEventsChan() <- configMapEvent
			}
		}

		if secretWatcher != nil {
			secretEvent, ok := <-secretWatcher.ResultChan()
			if ok {
				w.controller.K8sEventsChan() <- secretEvent
			}
		}

		// disable configmap & secret watchers if no webhooks use configmaps or secrets
		if !useConfigmaps && cmWatcher != nil {
			cmWatcher.Stop()
			cmWatcher = nil
		}

		if !useSecret && secretWatcher != nil {
			secretWatcher.Stop()
			secretWatcher = nil
		}

	}
}

func (w *WebhookWatcherUpdater) webhookRetryWatcher(ptx context.Context, errorCh chan<- error) {
	logrus.Info("webhook watcher is starting")
	defer logrus.Info("webhook watcher is terminating")

	ctx, cancel := context.WithCancel(ptx)
	defer cancel()
	watcher, err := w.whClient.Watch(ctx, options.ListOptions{})
	if err != nil {
		logrus.WithError(err).Error("Unable to watch for SecurityEventWebhook resources")
		errorCh <- err
		return
	}
	// if watcher.Stop is called or if watcher encounters the error, ResultChan will be closed
	// so it's fine to range over it
	for webhookEvent := range watcher.ResultChan() {
		w.controller.WebhookEventsChan() <- webhookEvent
	}
}

func (w *WebhookWatcherUpdater) checkWebhooksForConfigmapsAndSecret(ctx context.Context) (bool, bool) {
	useConfigmaps, useSecret := false, false
	listopts := options.ListOptions{}
	webhooks, err := w.whClient.List(ctx, listopts)
	if err != nil {
		logrus.Debugf("error while getting webhooks : %v", err)
	} else if webhooks != nil {
		for _, hook := range webhooks.Items {
			for _, config := range hook.Spec.Config {
				if config.ValueFrom.ConfigMapKeyRef != nil {
					useConfigmaps = true
				}

				if config.ValueFrom.SecretKeyRef != nil {
					useSecret = true
				}
			}
		}
	}
	return useConfigmaps, useSecret
}
