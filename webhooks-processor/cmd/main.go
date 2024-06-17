// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/sirupsen/logrus"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/webhooks-processor/pkg/events"
	"github.com/projectcalico/calico/webhooks-processor/pkg/webhooks"
)

func cancelOnSignals(cancel context.CancelFunc, signals ...os.Signal) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, signals...)
	<-c
	logrus.Info("signal received")
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
		WithUpdater(webhookWatcherUpdater)
	// wire up the watcher/updater and controller together
	webhookWatcherUpdater = webhookWatcherUpdater.WithController(webhookController)

	var wg sync.WaitGroup
	wg.Add(2)
	go webhookWatcherUpdater.Run(ctx, &wg)
	go webhookController.Run(ctx, &wg)
	go cancelOnSignals(cancel, syscall.SIGINT, syscall.SIGTERM)
	wg.Wait()
	logrus.Info("Goodbye!")
}
