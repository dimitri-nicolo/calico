// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package webhooks

import (
	"time"

	"github.com/kelseyhightower/envconfig"
	api "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/webhooks-processor/pkg/providers"
	"github.com/projectcalico/calico/webhooks-processor/pkg/providers/alertmanager"
	"github.com/projectcalico/calico/webhooks-processor/pkg/providers/generic"
	"github.com/projectcalico/calico/webhooks-processor/pkg/providers/jira"
	"github.com/projectcalico/calico/webhooks-processor/pkg/providers/slack"
)

type ProvidersConfig struct {
	RequestTimeout                          time.Duration `default:"5s"`
	RetryDuration                           time.Duration `default:"2s"`
	RetryTimes                              uint          `default:"5"`
	GenericProviderRateLimiterDuration      time.Duration `default:"1s"`
	GenericProviderRateLimiterCount         uint          `default:"1000"`
	JiraProviderRateLimiterDuration         time.Duration `default:"1m"`
	JiraProviderRateLimiterCount            uint          `default:"10"`
	SlackProviderRateLimiterDuration        time.Duration `default:"1m"`
	SlackProviderRateLimiterCount           uint          `default:"10"`
	AlertManagerProviderRateLimiterDuration time.Duration `default:"1s"`
	AlertManagerProviderRateLimiterCount    uint          `default:"1000"`
}

func DefaultProviders() map[api.SecurityEventWebhookConsumer]providers.Provider {
	c := new(ProvidersConfig)
	envconfig.MustProcess("webhooks", c)

	RegisteredProviders := make(map[api.SecurityEventWebhookConsumer]providers.Provider)
	RegisteredProviders[api.SecurityEventWebhookConsumerJira] = jira.NewProvider(providers.Config{
		RequestTimeout:      c.RequestTimeout,
		RetryDuration:       c.RetryDuration,
		RetryTimes:          c.RetryTimes,
		RateLimiterDuration: c.JiraProviderRateLimiterDuration,
		RateLimiterCount:    c.JiraProviderRateLimiterCount,
	})
	RegisteredProviders[api.SecurityEventWebhookConsumerSlack] = slack.NewProvider(providers.Config{
		RequestTimeout:      c.RequestTimeout,
		RetryDuration:       c.RetryDuration,
		RetryTimes:          c.RetryTimes,
		RateLimiterDuration: c.SlackProviderRateLimiterDuration,
		RateLimiterCount:    c.SlackProviderRateLimiterCount,
	})
	RegisteredProviders[api.SecurityEventWebhookConsumerGeneric] = generic.NewProvider(providers.Config{
		RequestTimeout:      c.RequestTimeout,
		RetryDuration:       c.RetryDuration,
		RetryTimes:          c.RetryTimes,
		RateLimiterDuration: c.GenericProviderRateLimiterDuration,
		RateLimiterCount:    c.GenericProviderRateLimiterCount,
	})
	RegisteredProviders[api.SecurityEventWebhookConsumerAlertManager] = alertmanager.NewProvider(providers.Config{
		RequestTimeout:      c.RequestTimeout,
		RetryDuration:       c.RetryDuration,
		RetryTimes:          c.RetryTimes,
		RateLimiterDuration: c.AlertManagerProviderRateLimiterDuration,
		RateLimiterCount:    c.AlertManagerProviderRateLimiterCount,
	})

	return RegisteredProviders
}
