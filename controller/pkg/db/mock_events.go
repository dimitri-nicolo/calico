// Copyright 2019 Tigera Inc. All rights reserved.

package db

import (
	"context"
	"errors"
)

type MockEvents struct {
	Error         error
	ErrorIndex    int
	ErrorReturned bool
	FlowLogs      []SecurityEventInterface
	value         SecurityEventInterface
}

func (m *MockEvents) PutSecurityEvent(ctx context.Context, l SecurityEventInterface) error {
	if len(m.FlowLogs) == m.ErrorIndex && !m.ErrorReturned {
		m.ErrorReturned = true
		return errors.New("PutSecurityEvent error")
	}
	m.FlowLogs = append(m.FlowLogs, l)
	return nil
}
