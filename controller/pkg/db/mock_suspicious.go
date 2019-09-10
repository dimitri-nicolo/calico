// Copyright 2019 Tigera Inc. All rights reserved.

package db

import (
	"context"
)

type MockSuspicious struct {
	Error  error
	Events []SecurityEventInterface
}

func (m *MockSuspicious) QuerySet(ctx context.Context, name string) ([]SecurityEventInterface, error) {
	return m.Events, m.Error
}
