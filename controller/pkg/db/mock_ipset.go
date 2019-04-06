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
	Metas         []IPSetMeta
	Set           IPSetSpec
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

func (m *MockIPSet) ListIPSets(ctx context.Context) ([]IPSetMeta, error) {
	return m.Metas, m.Error
}

func (m *MockIPSet) DeleteIPSet(ctx context.Context, meta IPSetMeta) error {
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

func (m *MockIPSet) GetIPSetModified(ctx context.Context, name string) (time.Time, error) {
	return m.Time, m.Error
}

func (m *MockIPSet) GetIPSet(ctx context.Context, name string) (IPSetSpec, error) {
	return m.Set, m.Error
}

func (m *MockIPSet) PutIPSet(ctx context.Context, name string, set IPSetSpec) error {
	m.m.Lock()
	defer m.m.Unlock()
	m.calls = append(m.calls, Call{Method: "PutIPSet", Name: name, Set: set})
	m.Name = name
	m.Set = set

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
