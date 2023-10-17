// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package webhooks

import (
	"time"

	"github.com/kelseyhightower/envconfig"
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

type JiraProviderConfiguration struct {
	RateLimiterDuration time.Duration `envconfig:"WEBHOOKS_JIRA_RATE_LIMITER_DURATION" default:"60m"`
	RateLimiterCount    uint          `envconfig:"WEBHOOKS_JIRA_RATE_LIMITER_COUNT" default:"1"`
}

type SlackProviderConfiguration struct {
	RateLimiterDuration time.Duration `envconfig:"WEBHOOKS_SLACK_RATE_LIMITER_DURATION" default:"5m"`
	RateLimiterCount    uint          `envconfig:"WEBHOOKS_SLACK_RATE_LIMITER_COUNT" default:"3"`
}

type GenericProviderConfiguration struct {
	RateLimiterDuration time.Duration `envconfig:"WEBHOOKS_GENERIC_RATE_LIMITER_DURATION" default:"1h"`
	RateLimiterCount    uint          `envconfig:"WEBHOOKS_GENERIC_RATE_LIMITER_COUNT" default:"100"`
}

func NewJiraProviderConfiguration() *ProviderConfiguration {
	config := new(JiraProviderConfiguration)
	envconfig.MustProcess("webhooks", config)

	return &ProviderConfiguration{
		Provider:            jira.NewProvider(),
		RateLimiterDuration: config.RateLimiterDuration,
		RateLimiterCount:    config.RateLimiterCount,
	}
}

func NewSlackProviderConfiguration() *ProviderConfiguration {
	config := new(SlackProviderConfiguration)
	envconfig.MustProcess("webhooks", config)

	return &ProviderConfiguration{
		Provider:            slack.NewProvider(),
		RateLimiterDuration: config.RateLimiterDuration,
		RateLimiterCount:    config.RateLimiterCount,
	}
}

func NewGenericProviderConfiguration() *ProviderConfiguration {
	config := new(GenericProviderConfiguration)
	envconfig.MustProcess("webhooks", config)

	return &ProviderConfiguration{
		Provider:            generic.NewProvider(),
		RateLimiterDuration: config.RateLimiterDuration,
		RateLimiterCount:    config.RateLimiterCount,
	}
}

func DefaultProviders() map[api.SecurityEventWebhookConsumer]*ProviderConfiguration {
	RegisteredProviders := make(map[api.SecurityEventWebhookConsumer]*ProviderConfiguration)
	RegisteredProviders[api.SecurityEventWebhookConsumerJira] = NewJiraProviderConfiguration()
	RegisteredProviders[api.SecurityEventWebhookConsumerSlack] = NewSlackProviderConfiguration()
	RegisteredProviders[api.SecurityEventWebhookConsumerGeneric] = NewGenericProviderConfiguration()

	return RegisteredProviders
}
