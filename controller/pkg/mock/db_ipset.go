// Copyright 2019 Tigera Inc. All rights reserved.

package mock

import (
	"context"
	"sync"
	"time"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
)

type IPSet struct {
	Name          string
	Version       interface{}
	Metas         []db.IPSetMeta
	Set           db.IPSetSpec
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

func (m *IPSet) ListIPSets(ctx context.Context) ([]db.IPSetMeta, error) {
	return m.Metas, m.Error
}

func (m *IPSet) DeleteIPSet(ctx context.Context, meta db.IPSetMeta) error {
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

func (m *IPSet) GetIPSetModified(ctx context.Context, name string) (time.Time, error) {
	return m.Time, m.Error
}

func (m *IPSet) GetIPSet(ctx context.Context, name string) (db.IPSetSpec, error) {
	return m.Set, m.Error
}

func (m *IPSet) PutIPSet(ctx context.Context, name string, set db.IPSetSpec) error {
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

func (m *IPSet) Calls() []Call {
	var out []Call
	m.m.Lock()
	defer m.m.Unlock()
	for _, c := range m.calls {
		out = append(out, c)
	}
	return out
}
