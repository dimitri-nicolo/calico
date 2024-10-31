// Copyright 2019 Tigera Inc. All rights reserved.

package storage

import (
	"context"
	"errors"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client"
	lmaAPI "github.com/projectcalico/calico/lma/pkg/api"
)

type MockEvents struct {
	Error      error
	ErrorIndex int
	Events     []v1.Event
}

func (m *MockEvents) PutSecurityEventWithID(ctx context.Context, l []v1.Event) error {
	if m.ErrorIndex >= 0 {
		m.Events = copyWithSkip(m.ErrorIndex, l)
		return errors.New("PutSecurityEventWithID error")
	}

	m.Events = l
	return nil
}

func copyWithSkip(index int, values []v1.Event) []v1.Event {
	var copyOfEvents []v1.Event
	if index < 0 {
		return values
	}

	if index >= len(values) {
		return values
	}

	copyOfEvents = append(copyOfEvents, values[:index]...)
	copyOfEvents = append(copyOfEvents, values[index+1:]...)

	return copyOfEvents
}

func (m *MockEvents) GetSecurityEvents(ctx context.Context, p client.ListPager[v1.Event]) <-chan *lmaAPI.EventResult {
	return nil
}

func (m *MockEvents) PutForwarderConfig(ctx context.Context, f *ForwarderConfig) error {
	return nil
}

func (m *MockEvents) GetForwarderConfig(ctx context.Context) (*ForwarderConfig, error) {
	return nil, nil
}
