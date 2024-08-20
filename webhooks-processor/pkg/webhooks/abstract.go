// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package webhooks

import (
	"context"
	"time"

	api "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/projectcalico/calico/libcalico-go/lib/validator/v3/query"
	calicoWatch "github.com/projectcalico/calico/libcalico-go/lib/watch"
	"github.com/projectcalico/calico/webhooks-processor/pkg/providers"

	lsApi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

type ProcessFunc func(context.Context, map[string]string, map[string]string, *lsApi.Event) (providers.ProviderResponse, error)

type EventsFetchFunc func(context.Context, *query.Query, time.Time, time.Time) ([]lsApi.Event, error)

type RateLimiterInterface interface {
	Event() error
}

type WebhookControllerInterface interface {
	WebhookEventsChan() chan<- calicoWatch.Event
	K8sEventsChan() chan<- watch.Event
}

type WebhookUpdaterInterface interface {
	UpdatesChan() chan<- *api.SecurityEventWebhook
}

type StateInterface interface {
	OutgoingWebhookUpdates() <-chan *api.SecurityEventWebhook
	IncomingWebhookUpdate(context.Context, *api.SecurityEventWebhook)
	CheckDependencies(runtime.Object)
	Stop(context.Context, *api.SecurityEventWebhook)
	StopAll()
}
