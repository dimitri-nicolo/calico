// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package webhooks

import (
	"context"
	"sync"

	"github.com/cnf/structhash"
	"github.com/sirupsen/logrus"
	api "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

type webhookState struct {
	specHash       string
	cancelFunc     context.CancelFunc
	webhookUpdates chan *api.SecurityEventWebhook
}

type ControllerState struct {
	outUpdatesChan  chan *api.SecurityEventWebhook
	webhooksTrail   map[types.UID]*webhookState
	preventRestarts map[types.UID]bool
	wg              sync.WaitGroup
	cli             *kubernetes.Clientset
	eventsFetchFunc EventsFetchFunc
}

func NewControllerState() *ControllerState {
	return &ControllerState{
		outUpdatesChan:  make(chan *api.SecurityEventWebhook),
		webhooksTrail:   make(map[types.UID]*webhookState),
		preventRestarts: make(map[types.UID]bool),
	}
}

func (s *ControllerState) WithFetchEventsFunction(fetch EventsFetchFunc) *ControllerState {
	s.eventsFetchFunc = fetch
	return s
}

func (s *ControllerState) IncomingWebhookUpdate(ctx context.Context, webhook *api.SecurityEventWebhook) {
	logrus.WithField("uid", webhook.UID).Info("Processing incoming webhook update")

	if trail, ok := s.webhooksTrail[webhook.UID]; ok {
		specHash := string(structhash.Md5(webhook.Spec, 1))
		if trail.specHash == specHash {
			trail.webhookUpdates <- webhook
			return
		}
		logrus.WithField("uid", webhook.UID).Info("Webhook spec changed")
		s.Stop(ctx, webhook)
	}

	if _, preventRestart := s.preventRestarts[webhook.UID]; preventRestart {
		logrus.WithField("uid", webhook.UID).Info("Webhook restart prevented")
		delete(s.preventRestarts, webhook.UID)
		return
	}

	s.startNewInstance(ctx, webhook)
}

func (s *ControllerState) Stop(ctx context.Context, webhook *api.SecurityEventWebhook) {
	if trail, ok := s.webhooksTrail[webhook.UID]; ok {
		trail.cancelFunc()
		delete(s.webhooksTrail, webhook.UID)
		logrus.WithField("uid", webhook.UID).Info("Webhook stopped")
	}
}

func (s *ControllerState) StopAll() {
	for _, trail := range s.webhooksTrail {
		trail.cancelFunc()
	}
	logrus.Info("Waiting for all webhooks to terminate")
	s.wg.Wait()
	s.webhooksTrail = make(map[types.UID]*webhookState)
}

func (s *ControllerState) OutgoingWebhookUpdates() <-chan *api.SecurityEventWebhook {
	return s.outUpdatesChan
}
