// Copyright 2019 Tigera Inc. All rights reserved.

package db

import (
	"context"
	"errors"
)

type MockSuspiciousIP struct {
	Error         error
	ErrorIndex    int
	ErrorReturned bool
	FlowLogs      []SecurityEventInterface
	value         SecurityEventInterface
}

func (m *MockSuspiciousIP) QueryIPSet(ctx context.Context, name string) (SecurityEventIterator, error) {
	return m, m.Error
}

func (m *MockSuspiciousIP) Next() bool {
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

func (m *MockSuspiciousIP) Value() SecurityEventInterface {
	return m.value
}

func (m *MockSuspiciousIP) Err() error {
	if m.ErrorIndex >= 0 {
		return errors.New("Err error")
	}
	return nil
}
