// Copyright 2019 Tigera Inc. All rights reserved.

package db

import (
	"context"
	"sync"
	"time"
)

type MockIPSet struct {
	Name          string
	Version       interface{}
	Metas         []Meta
	Value         interface{}
	Time          time.Time
	Error         error
	DeleteCalled  bool
	DeleteName    string
	DeleteVersion *int64
	DeleteError   error
	PutError      error

	m     sync.Mutex
	calls []Call
}

func (m *MockIPSet) ListSets(ctx context.Context, kind Kind) ([]Meta, error) {
	return m.Metas, m.Error
}

func (m *MockIPSet) DeleteSet(ctx context.Context, meta Meta) error {
	m.m.Lock()
	defer m.m.Unlock()
	m.calls = append(m.calls, Call{Method: "DeleteSet", Name: meta.Name, Version: meta.Version, Kind: meta.Kind})
	m.DeleteCalled = true
	m.DeleteName = meta.Name
	if meta.Version == nil {
		m.DeleteVersion = nil
	} else {
		i := struct{ i int64 }{*meta.Version}
		m.DeleteVersion = &i.i
	}
	return m.DeleteError
}

func (m *MockIPSet) GetIPSetModified(ctx context.Context, name string) (time.Time, error) {
	return m.Time, m.Error
}

func (m *MockIPSet) GetIPSet(ctx context.Context, name string) (IPSetSpec, error) {
	if m.Value == nil {
		return nil, m.Error
	}
	return m.Value.(IPSetSpec), m.Error
}

func (m *MockIPSet) PutSet(ctx context.Context, meta Meta, value interface{}) error {
	m.m.Lock()
	defer m.m.Unlock()
	m.calls = append(m.calls, Call{Method: "PutSet", Name: meta.Name, Value: value, Kind: meta.Kind})
	m.Name = m.Name
	m.Value = value

	if m.PutError == nil {
		m.Time = time.Now()
	}

	return m.PutError
}

func (m *MockIPSet) Calls() []Call {
	var out []Call
	m.m.Lock()
	defer m.m.Unlock()
	for _, c := range m.calls {
		out = append(out, c)
	}
	return out
}
