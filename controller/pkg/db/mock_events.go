// Copyright 2019 Tigera Inc. All rights reserved.

package db

import (
	"context"
	"errors"

	"github.com/tigera/intrusion-detection/controller/pkg/events"
)

type MockEvents struct {
	Error         error
	ErrorIndex    int
	ErrorReturned bool
	FlowLogs      []events.SecurityEvent
	value         events.SecurityEvent
}

func (m *MockEvents) PutSecurityEvent(ctx context.Context, l events.SecurityEvent) error {
	if len(m.FlowLogs) == m.ErrorIndex && !m.ErrorReturned {
		m.ErrorReturned = true
		return errors.New("PutSecurityEvent error")
	}
	m.FlowLogs = append(m.FlowLogs, l)
	return nil
}
