// Copyright 2019 Tigera Inc. All rights reserved.

package db

import (
	"context"
	"sync"
	"time"
)

type MockSets struct {
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

func (m *MockSets) ListIPSets(ctx context.Context) ([]Meta, error) {
	return m.Metas, m.Error
}

func (m *MockSets) ListDomainNameSets(ctx context.Context) ([]Meta, error) {
	return m.Metas, m.Error
}

func (m *MockSets) DeleteIPSet(ctx context.Context, meta Meta) error {
	m.m.Lock()
	defer m.m.Unlock()
	m.calls = append(m.calls, Call{Method: "DeleteIPSet", Name: meta.Name, Version: meta.Version})
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

func (m *MockSets) DeleteDomainNameSet(ctx context.Context, meta Meta) error {
	m.m.Lock()
	defer m.m.Unlock()
	m.calls = append(m.calls, Call{Method: "DeleteDomainNameSet", Name: meta.Name, Version: meta.Version})
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

func (m *MockSets) GetIPSetModified(ctx context.Context, name string) (time.Time, error) {
	return m.Time, m.Error
}

func (m *MockSets) GetIPSet(ctx context.Context, name string) (IPSetSpec, error) {
	if m.Value == nil {
		return nil, m.Error
	}
	return m.Value.(IPSetSpec), m.Error
}

func (m *MockSets) PutIPSet(ctx context.Context, name string, set IPSetSpec) error {
	m.m.Lock()
	defer m.m.Unlock()
	m.calls = append(m.calls, Call{Method: "PutIPSet", Name: name, Value: set})
	m.Name = name
	m.Value = set

	if m.PutError == nil {
		m.Time = time.Now()
	}

	return m.PutError
}

func (m *MockSets) PutDomainNameSet(ctx context.Context, name string, set DomainNameSetSpec) error {
	m.m.Lock()
	defer m.m.Unlock()
	m.calls = append(m.calls, Call{Method: "PutDomainNameSet", Name: name, Value: set})
	m.Name = name
	m.Value = set

	if m.PutError == nil {
		m.Time = time.Now()
	}

	return m.PutError
}

func (m *MockSets) Calls() []Call {
	var out []Call
	m.m.Lock()
	defer m.m.Unlock()
	for _, c := range m.calls {
		out = append(out, c)
	}
	return out
}
