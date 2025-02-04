// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package webhooks

import (
	"context"
	"sync"
	"time"

	calicoWatch "github.com/projectcalico/calico/libcalico-go/lib/watch"
	"github.com/sirupsen/logrus"
	api "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
)

const (
	WebhooksWatcherTimeout = 2 * time.Minute
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
	logrus.Info("Webhook updater/watcher started")
	defer wg.Done()
	defer logrus.Info("Webhook updater/watcher is terminating")

	watchGroup := sync.WaitGroup{}
	go w.executeUntilContextIsAlive(ctx, &watchGroup, w.watchWebhooks)
	go w.executeUntilContextIsAlive(ctx, &watchGroup, w.watchCMs)
	go w.executeUntilContextIsAlive(ctx, &watchGroup, w.watchSecrets)
	go w.executeUntilContextIsAlive(ctx, &watchGroup, w.updateWebhooks)
	watchGroup.Wait()
}

func (w *WebhookWatcherUpdater) executeUntilContextIsAlive(ctx context.Context, wg *sync.WaitGroup, f func(context.Context)) {
	wg.Add(1)
	for ctx.Err() == nil {
		f(ctx)
	}
	wg.Done()
}

func (w *WebhookWatcherUpdater) watchCMs(ctx context.Context) {
	var watchRevision string
	if cms, err := w.client.CoreV1().ConfigMaps(ConfigVarNamespace).List(ctx, metav1.ListOptions{}); err != nil {
		logrus.WithError(err).Fatal("unable to list configmaps")
		return
	} else {
		watchRevision = cms.ResourceVersion
		for _, secret := range cms.Items {
			w.controller.K8sEventsChan() <- watch.Event{Type: watch.Added, Object: &secret}
		}
	}
	if watcher, err := w.client.CoreV1().ConfigMaps(ConfigVarNamespace).Watch(ctx, metav1.ListOptions{ResourceVersion: watchRevision}); err != nil {
		logrus.WithError(err).Fatal("unable to watch for configmaps changes")
		return
	} else {
		for event := range watcher.ResultChan() {
			w.controller.K8sEventsChan() <- event
		}
	}
}

func (w *WebhookWatcherUpdater) watchSecrets(ctx context.Context) {
	var watchRevision string
	if secrets, err := w.client.CoreV1().Secrets(ConfigVarNamespace).List(ctx, metav1.ListOptions{}); err != nil {
		logrus.WithError(err).Fatal("unable to list secrets")
		return
	} else {
		watchRevision = secrets.ResourceVersion
		for _, secret := range secrets.Items {
			w.controller.K8sEventsChan() <- watch.Event{Type: watch.Added, Object: &secret}
		}
	}
	if watcher, err := w.client.CoreV1().Secrets(ConfigVarNamespace).Watch(ctx, metav1.ListOptions{ResourceVersion: watchRevision}); err != nil {
		logrus.WithError(err).Fatal("unable to watch for secrets changes")
		return
	} else {
		for event := range watcher.ResultChan() {
			w.controller.K8sEventsChan() <- event
		}
	}
}

func (w *WebhookWatcherUpdater) updateWebhooks(ctx context.Context) {
	select {
	case webhook := <-w.webhookUpdatesChan:
		if _, err := w.whClient.Update(ctx, webhook, options.SetOptions{}); err != nil {
			logrus.WithError(err).Error("unable to update webhook definition")
		}
	case <-ctx.Done():
		return
	}
}

func (w *WebhookWatcherUpdater) watchWebhooks(ctx context.Context) {
	var watchRevision string
	if webhooks, err := w.whClient.List(ctx, options.ListOptions{}); err != nil {
		logrus.WithError(err).Fatal("unable to list webhooks")
		return
	} else {
		watchRevision = webhooks.ResourceVersion
		for _, webhook := range webhooks.Items {
			w.controller.WebhookEventsChan() <- calicoWatch.Event{Type: calicoWatch.Added, Previous: nil, Object: &webhook}
		}
	}

	watcherCtx, watcherCtxCancel := context.WithTimeout(ctx, WebhooksWatcherTimeout)
	defer watcherCtxCancel()

	if watcher, err := w.whClient.Watch(watcherCtx, options.ListOptions{ResourceVersion: watchRevision}); err != nil {
		logrus.WithError(err).Fatal("unable to watch for webhook changes")
		return
	} else {
		for event := range watcher.ResultChan() {
			w.controller.WebhookEventsChan() <- event
		}
	}
}
