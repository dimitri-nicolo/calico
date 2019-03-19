// Copyright 2019 Tigera Inc. All rights reserved.

package mock

import (
	"context"
	"errors"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/events"
)

type SuspiciousIP struct {
	Error         error
	ErrorIndex    int
	ErrorReturned bool
	FlowLogs      []events.SecurityEvent
	value         events.SecurityEvent
}

func (m *SuspiciousIP) QueryIPSet(ctx context.Context, name string) (db.SecurityEventIterator, error) {
	return m, m.Error
}

func (m *SuspiciousIP) Next() bool {
	if len(m.FlowLogs) == m.ErrorIndex {
		return false
	}
	if len(m.FlowLogs) > 0 {
		m.value = m.FlowLogs[0]
		m.FlowLogs = m.FlowLogs[1:]
		return true
	}
	return false
}

func (m *SuspiciousIP) Value() events.SecurityEvent {
	return m.value
}

func (m *SuspiciousIP) Err() error {
	if m.ErrorIndex >= 0 {
		return errors.New("Err error")
	}
	return nil
}
