// Copyright 2019 Tigera Inc. All rights reserved.

package db

import (
	"context"
)

type MockSuspicious struct {
	Error error
	Hits  []SecurityEventInterface
}

func (m *MockSuspicious) QuerySet(ctx context.Context, name string) ([]SecurityEventInterface, error) {
	return m.Hits, m.Error
}
