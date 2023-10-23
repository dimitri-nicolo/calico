// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/webhooks-processor/pkg/events"
	"github.com/projectcalico/calico/webhooks-processor/pkg/webhooks"
)

func main() {
	logrus.Info("Starting security events webhook processor...")

	v3Client, err := clientv3.NewFromEnv()
	if err != nil {
		logrus.WithError(err).Fatal("Unable to connect to initialize v3 client")
	}

	webhookWatcherUpdater := webhooks.NewWebhookWatcherUpdater().WithClient(v3Client)
	controllerState := webhooks.NewControllerState().WithFetchEventsFunction(events.FetchSecurityEventsFunc)
	webhookController := webhooks.NewWebhookController().WithState(controllerState)

	ctx, ctxCancel := context.WithCancel(context.Background())
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
