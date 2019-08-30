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
	Events        []SecurityEventInterface
	value         SecurityEventInterface
}

func (m *MockEvents) PutSecurityEvent(ctx context.Context, l SecurityEventInterface) error {
	if len(m.Events) == m.ErrorIndex && !m.ErrorReturned {
		m.ErrorReturned = true
		return errors.New("PutSecurityEvent error")
	}
	m.Events = append(m.Events, l)
	return nil
}
