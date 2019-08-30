// Copyright 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"context"
	"github.com/tigera/intrusion-detection/controller/pkg/db"

	"github.com/olivere/elastic"
)

type MockSetQuerier struct {
	QueryError error
	Iterator   *MockIterator
	GetError   error
	Set        db.DomainNameSetSpec
}

func (m *MockSetQuerier) GetDomainNameSet(ctx context.Context, name string) (db.DomainNameSetSpec, error) {
	return m.Set, m.GetError
}

func (m *MockSetQuerier) QueryIPSet(ctx context.Context, name string) (Iterator, error) {
	return m.Iterator, m.QueryError
}

func (m *MockSetQuerier) QueryDomainNameSet(ctx context.Context, name string, set db.DomainNameSetSpec) (Iterator, error) {
	return m.Iterator, m.QueryError
}

type MockIterator struct {
	Error      error
	ErrorIndex int
	Keys       []string
	Hits       []*elastic.SearchHit
	next       int
}

func (m *MockIterator) Next() bool {
	cur := m.next
	m.next++
	return cur < len(m.Hits) && cur != m.ErrorIndex
}

func (m *MockIterator) Value() (scrollerName string, hit *elastic.SearchHit) {
	cur := m.next - 1
	return m.Keys[cur], m.Hits[cur]
}

func (m *MockIterator) Err() error {
	cur := m.next - 1
	if cur == m.ErrorIndex {
		return m.Error
	}
	return nil
}
