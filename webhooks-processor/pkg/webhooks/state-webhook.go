// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package webhooks

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	api "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/libcalico-go/lib/validator/v3/query"
)

func (s *ControllerState) webhookGoroutine(
	ctx context.Context, // context for the goroutine
	config map[string]string, // configuration for the webhook
	selector *query.Query, // Security Events selector
	processFunc ProcessFunc, // Security Events processing function
	inUpdatesChan chan *api.SecurityEventWebhook, // incoming updates for webhookRef
	webhookRef *api.SecurityEventWebhook, // SecurityEventWebhook from k8s store
	rateLimiter RateLimiterInterface, // RateLimiter for this goroutine
) {
	defer s.wg.Done()
	defer logrus.WithField("uid", webhookRef.UID).Info("Webhook goroutine is terminating")
	logrus.WithField("uid", webhookRef.UID).Info("Webhook goroutine started")

	var processingLock sync.Mutex

	eventProcessing := func() {
		defer processingLock.Unlock()

		thisRunStamp := time.Now().Add(-time.Second).Round(time.Second)
		var previousRunStamp metav1.Time
		for _, condition := range webhookRef.Status {
			if condition.Type != ConditionHealthy {
				continue
			}
			previousRunStamp = condition.LastTransitionTime
			break
		}
		if previousRunStamp.IsZero() {
			s.updateWebhookHealth(webhookRef, "SecurityEventsProcessing", thisRunStamp, errors.New("status corrupted"))
			return
		}

		var err error

		events, err := s.config.EventsFetchFunction(ctx, selector, previousRunStamp.Time, thisRunStamp)
		if err == nil {
			for _, event := range events {
				if err = rateLimiter.Event(); err != nil {
					break
				}
				if err = processFunc(ctx, config, &event); err != nil {
					break
				}
			}
		}

		s.updateWebhookHealth(webhookRef, "SecurityEventsProcessing", thisRunStamp, err)
		logrus.WithField("uid", webhookRef.UID).WithError(err).Info("Iteration completed")
	}

	tick := time.NewTicker(s.config.FetchingInterval)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case webhookRef = <-inUpdatesChan:
			continue
		case <-tick.C:
			if processingLock.TryLock() {
				go eventProcessing()
			} else {
				logrus.WithField("uid", webhookRef.UID).Info("Still processing events")
			}
		}
	}
}
