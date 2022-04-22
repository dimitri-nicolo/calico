// Copyright 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"context"

	"github.com/olivere/elastic/v7"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/db"

	apiV3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
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

func (m *MockSetQuerier) QueryIPSet(ctx context.Context, feed *apiV3.GlobalThreatFeed) (Iterator, string, error) {
	return m.Iterator, "", m.QueryError
}

func (m *MockSetQuerier) QueryDomainNameSet(ctx context.Context, set db.DomainNameSetSpec, feed *apiV3.GlobalThreatFeed) (Iterator, string, error) {
	return m.Iterator, "", m.QueryError
}

type MockIterator struct {
	Error      error
	ErrorIndex int
	Keys       []db.QueryKey
	Hits       []*elastic.SearchHit
	next       int
}

func (m *MockIterator) Next() bool {
	cur := m.next
	m.next++
	return cur < len(m.Hits) && cur != m.ErrorIndex
}

func (m *MockIterator) Value() (key db.QueryKey, hit *elastic.SearchHit) {
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
