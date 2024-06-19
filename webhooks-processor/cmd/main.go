// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/webhooks-processor/pkg/events"
	"github.com/projectcalico/calico/webhooks-processor/pkg/webhooks"
)

func cancelOnSignals(cancel context.CancelFunc, ctrWg, uptWg *sync.WaitGroup, ctrCancel, uptCancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM...)
	<-c
	logrus.Info("signal received")

	// make sure the webhook updater and webhook controller exit in the correct order
	// avoids webhook updater getting stuck writing to a channel, blocking and never terminating
	uptCancel()
	uptWg.Wait()
	ctrCancel()
	ctrWg.Wait()

	cancel()
}

func main() {
	logrus.Info("Starting security events webhook processor...")

	k8sConfig, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		logrus.WithError(err).Fatal("Unable to obtain k8s configuration")
	}

	k8sClient, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		logrus.WithError(err).Fatal("Unable to initialize k8s client")
	}

	calicoClient, err := clientv3.NewFromEnv()
	if err != nil {
		logrus.WithError(err).Fatal("Unable to initialize v3 client")
	}

	config := webhooks.NewControllerConfig(webhooks.DefaultProviders(), events.FetchSecurityEventsFunc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctrCtx, ctrCancel := context.WithCancel(ctx)

	uptCtx, uptCancel := context.WithCancel(ctx)

	k8sEventChan := make(chan watch.Event)

	// init - webhook watcher and updater
	webhookWatcherUpdater := webhooks.NewWebhookWatcherUpdater().
		WithWebhooksClient(calicoClient.SecurityEventWebhook()).
		WithK8sClient(k8sClient)
	// init state
	controllerState := webhooks.NewControllerState().
		WithK8sClient(k8sClient).
		WithConfig(config)
	// init controller that uses state and watcher/updater
	webhookController := webhooks.
		NewWebhookController().
		WithState(controllerState).
		WithUpdater(webhookWatcherUpdater).
		WithK8sEventChan(k8sEventChan)
	// wire up the watcher/updater and controller together
	webhookWatcherUpdater = webhookWatcherUpdater.WithController(webhookController)

	var ctrWg sync.WaitGroup
	var uptWg sync.WaitGroup
	uptWg.Add(1)
	go webhookWatcherUpdater.Run(uptCtx, &uptWg)
	ctrWg.Add(1)
	go webhookController.Run(ctrCtx, &ctrWg)

	go cancelOnSignals(cancel, &ctrWg, &uptWg, ctrCancel, uptCancel)

	// break up wait group to terminate updater first then controller
	// with 2 different child contexts
	logrus.Info("Goodbye!")
}
