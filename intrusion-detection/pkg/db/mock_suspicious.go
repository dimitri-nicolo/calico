// Copyright 2019 Tigera Inc. All rights reserved.

package db

import (
	"context"
	"time"

	apiV3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
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
