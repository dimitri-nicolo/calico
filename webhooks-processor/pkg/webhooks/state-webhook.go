// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package webhooks

import (
	"context"
	"errors"
	"sync"
	"time"

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
	defer logEntry(webhookRef).Info("Webhook goroutine is terminating")
	logEntry(webhookRef).Info("Webhook goroutine started")

	var processingLock sync.Mutex

	var previousError error

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

		events, err := s.config.EventsFetchFunction(ctx, selector, previousRunStamp.Time, thisRunStamp)
		switch {
		case err == nil && len(events) > 0:
			labels := s.extractLabels(*webhookRef)
			for _, event := range events {
				if err = rateLimiter.Event(); err != nil {
					break
				}
				if _, err = processFunc(ctx, config, labels, &event); err != nil {
					break
				}
			}
			// we have now processed events - either with or without processing error;
			// we store the previous error value and update webhoook health:
			previousError = err
			s.updateWebhookHealth(webhookRef, "SecurityEventsProcessing", thisRunStamp, err)
		case err == nil && len(events) == 0:
			// we have no error and there is nothing to process so we keep the previousError value
			// and update the timestamp; the value of previousError is not important to us, we just keep it:
			s.updateWebhookHealth(webhookRef, "SecurityEventsProcessing", thisRunStamp, previousError)
		default:
			// we have encountered a Linseed fetch error - we log it and keep the previous timestamp value:
			s.updateWebhookHealth(webhookRef, "SecurityEventsProcessing", previousRunStamp.Time, err)
		}

		logEntry(webhookRef).WithError(err).Info("Iteration completed")
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
				logEntry(webhookRef).Info("Still processing events")
			}
		}
	}
}
