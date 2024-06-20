// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/webhooks-processor/pkg/events"
	"github.com/projectcalico/calico/webhooks-processor/pkg/webhooks"
)

func cancelOnSignals(cleanup func()) {
	c := make(chan os.Signal, 1)
	syscalls := []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	signal.Notify(c, syscalls...)
	<-c
	logrus.Info("signal received")

	// make sure the webhook updater and webhook controller exit in the correct order
	// avoids webhook updater getting stuck writing to a channel, blocking and never terminating
	cleanup()
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

	ctx := context.Background()

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

	// setup webhookController, webhookWatcherUpdater gorountines
	// returns cleanup function to handle graceful termintation of gorountines
	cleanup := webhooks.SetUp(ctx, webhookController, webhookWatcherUpdater)

	go cancelOnSignals(cleanup)

	// break up wait group to terminate updater first then controller
	// with 2 different child contexts
	logrus.Info("Goodbye!")
}
