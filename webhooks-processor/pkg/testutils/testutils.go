// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package testutils

import (
	"fmt"

	api "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/types"
)

const (
	testUrl = "https://test-hook"
)

func NewTestWebhook(name string) *api.SecurityEventWebhook {
	wh := api.NewSecurityEventWebhook()
	wh.Name = name
	wh.Spec.Consumer = api.SecurityEventWebhookConsumerSlack
	wh.Spec.State = api.SecurityEventWebhookStateEnabled
	wh.Spec.Query = "type = runtime_security"
	wh.Spec.Config = []api.SecurityEventWebhookConfigVar{{
		Name:  "url",
		Value: testUrl,
	}}
	wh.UID = types.UID(fmt.Sprintf("%s-uid", name))
	return wh
}
