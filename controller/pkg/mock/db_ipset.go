// Copyright 2019 Tigera Inc. All rights reserved.

package mock

import (
	"context"
	"time"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
)

type IPSet struct {
	Name  string
	Set   db.IPSetSpec
	Time  time.Time
	Error error
}

func (m *IPSet) GetIPSetModified(ctx context.Context, name string) (time.Time, error) {
	return m.Time, m.Error
}

func (m *IPSet) GetIPSet(ctx context.Context, name string) (db.IPSetSpec, error) {
	return m.Set, m.Error
}

func (m *IPSet) PutIPSet(ctx context.Context, name string, set db.IPSetSpec) error {
	m.Name = name
	m.Set = set

	if m.Error == nil {
		m.Time = time.Now()
	}

	return m.Error
}
