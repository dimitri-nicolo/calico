// Copyright 2019 Tigera Inc. All rights reserved.

package db

import (
	"context"
	apiV3 "github.com/projectcalico/apiserver/pkg/apis/projectcalico/v3"
	"time"
)

type MockSuspicious struct {
	Error                error
	Events               []SecurityEventInterface
	LastSuccessfulSearch time.Time
	SetHash              string
}

func (m *MockSuspicious) QuerySet(ctx context.Context, feed *apiV3.GlobalThreatFeed) ([]SecurityEventInterface, time.Time, string, error) {
	return m.Events, m.LastSuccessfulSearch, m.SetHash, m.Error
}
