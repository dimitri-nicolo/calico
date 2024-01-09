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

	ctx, ctxCancel := context.WithCancel(context.Background())
	webhookWatcherUpdater := webhooks.NewWebhookWatcherUpdater().
		WithWebhooksClient(calicoClient.SecurityEventWebhook()).
		WithK8sClient(k8sClient)
	controllerState := webhooks.NewControllerState().
		WithK8sClient(k8sClient).
		WithConfig(config)
	webhookController := webhooks.NewWebhookController().WithState(controllerState)

	var wg sync.WaitGroup
	wg.Add(2)
	go webhookController.WithUpdater(webhookWatcherUpdater).Run(ctx, ctxCancel, &wg)
	go webhookWatcherUpdater.WithController(webhookController).Run(ctx, ctxCancel, &wg)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	select {
	case s := <-sigChan:
		logrus.WithField("signal", s).Info("OS signal received")
		ctxCancel()
	case <-ctx.Done():
	}

	logrus.Info("Waiting for all components to terminate...")
	wg.Wait()

	logrus.Info("Goodbye!")
}
