// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package webhooks

import (
	"context"
	"time"

	api "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/libcalico-go/lib/validator/v3/query"
	"github.com/projectcalico/calico/libcalico-go/lib/watch"
	lsApi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

type ProcessFunc func(context.Context, map[string]string, map[string]string, *lsApi.Event) error

type EventsFetchFunc func(context.Context, *query.Query, time.Time, time.Time) ([]lsApi.Event, error)

type RateLimiterInterface interface {
	Event() error
}

type WebhookControllerInterface interface {
	EventsChan() chan<- watch.Event
}

type WebhookUpdaterInterface interface {
	UpdatesChan() chan<- *api.SecurityEventWebhook
}

type StateInterface interface {
	OutgoingWebhookUpdates() <-chan *api.SecurityEventWebhook
	IncomingWebhookUpdate(context.Context, *api.SecurityEventWebhook)
	Stop(context.Context, *api.SecurityEventWebhook)
	StopAll()
}
