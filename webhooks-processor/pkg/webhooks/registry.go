// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package webhooks

import (
	"github.com/projectcalico/calico/webhooks-processor/pkg/providers"
	"github.com/projectcalico/calico/webhooks-processor/pkg/providers/generic"
	"github.com/projectcalico/calico/webhooks-processor/pkg/providers/jira"
	"github.com/projectcalico/calico/webhooks-processor/pkg/providers/slack"
	api "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

func DefaultProviders() map[api.SecurityEventWebhookConsumer]providers.Provider {
	RegisteredProviders := make(map[api.SecurityEventWebhookConsumer]providers.Provider)
	RegisteredProviders[api.SecurityEventWebhookConsumerJira] = jira.NewProvider()
	RegisteredProviders[api.SecurityEventWebhookConsumerSlack] = slack.NewProvider()
	RegisteredProviders[api.SecurityEventWebhookConsumerGeneric] = generic.NewProvider()

	return RegisteredProviders
}
