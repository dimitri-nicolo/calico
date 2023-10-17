// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package webhooks

import (
	"time"

	"github.com/kelseyhightower/envconfig"
	api "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/webhooks-processor/pkg/providers"
)

// The ControllerConfig captures various dependencies that provide IO capabilities
// to process webhooks, along with additional parameters that can tune aspects of
// how webhooks are processed. These parameters can be set explicitly or through
// environment variables, so that we could tune things in production should we need to.
//
// Dependencies:
//   - ClientV3: provides access to webhooks objects in k8s API
//   - EventsFetchFunction: provides a way to query (security) events from Linseed
//   - Providers: map of providers that webhooks can use (Slack, Jira, etc...)
type ControllerConfig struct {
	ClientV3            clientv3.SecurityEventWebhookInterface
	EventsFetchFunction EventsFetchFunc
	Providers           map[api.SecurityEventWebhookConsumer]providers.Provider
	FetchingInterval    time.Duration `envconfig:"WEBHOOKS_FETCHING_INTERVAL" default:"10s"`
}

func NewControllerConfig(clientV3 clientv3.SecurityEventWebhookInterface, providers map[api.SecurityEventWebhookConsumer]providers.Provider, getEvents EventsFetchFunc) *ControllerConfig {
	config := new(ControllerConfig)
	envconfig.MustProcess("webhooks", config)

	config.ClientV3 = clientV3
	config.Providers = providers
	config.EventsFetchFunction = getEvents
	return config
}
