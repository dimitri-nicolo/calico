// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package webhooks

import (
	"time"

	api "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/webhooks-processor/pkg/providers/generic"
	"github.com/projectcalico/calico/webhooks-processor/pkg/providers/jira"
	"github.com/projectcalico/calico/webhooks-processor/pkg/providers/slack"
)

type ProviderConfiguration struct {
	Provider            ProviderInterface
	RateLimiterDuration time.Duration
	RateLimiterCount    uint
}

func DefaultProviders() map[api.SecurityEventWebhookConsumer]*ProviderConfiguration {
	RegisteredProviders := make(map[api.SecurityEventWebhookConsumer]*ProviderConfiguration)
	RegisteredProviders[api.SecurityEventWebhookConsumerJira] = &ProviderConfiguration{
		Provider:            jira.NewProvider(),
		RateLimiterDuration: 60 * time.Minute,
		RateLimiterCount:    1,
	}
	RegisteredProviders[api.SecurityEventWebhookConsumerSlack] = &ProviderConfiguration{
		Provider:            slack.NewProvider(),
		RateLimiterDuration: 5 * time.Minute,
		RateLimiterCount:    3,
	}
	RegisteredProviders[api.SecurityEventWebhookConsumerGeneric] = &ProviderConfiguration{
		Provider:            generic.NewProvider(),
		RateLimiterDuration: 1 * time.Hour,
		RateLimiterCount:    100,
	}

	return RegisteredProviders
}
