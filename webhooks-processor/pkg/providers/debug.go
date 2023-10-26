// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package providers

import (
	"context"

	lsApi "github.com/projectcalico/calico/linseed/pkg/apis/v1"

	"github.com/sirupsen/logrus"
)

type DebugProvider struct {
	webhookUID string
}

func NewDebugProvider(webhookUID string) Provider {
	return &DebugProvider{webhookUID: webhookUID}
}

func (p *DebugProvider) Validate(config map[string]string) error {
	return nil
}

func (p *DebugProvider) Process(ctx context.Context, config map[string]string, event *lsApi.Event) (err error) {
	logrus.WithField("uid", p.webhookUID).Info("Processing Security Events for a webhook in 'Debug' state")
	return nil
}
func (p *DebugProvider) Config() Config {
	return Config{}
}
