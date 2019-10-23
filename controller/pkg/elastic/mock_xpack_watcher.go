// Copyright (c) 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"context"

	"github.com/olivere/elastic/v7"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
)

type MockXPackWatcher struct {
	Metas       []db.Meta
	WatchRecord *elastic.XPackWatchRecord
	Status      *elastic.XPackWatchStatus
	Err         error
}

func (m MockXPackWatcher) ListWatches(ctx context.Context) ([]db.Meta, error) {
	return m.Metas, m.Err
}

func (m MockXPackWatcher) ExecuteWatch(ctx context.Context, body *ExecuteWatchBody) (*elastic.XPackWatchRecord, error) {
	return m.WatchRecord, m.Err
}

func (m MockXPackWatcher) PutWatch(ctx context.Context, name string, body *PutWatchBody) error {
	return m.Err
}

func (m MockXPackWatcher) GetWatchStatus(ctx context.Context, name string) (*elastic.XPackWatchStatus, error) {
	return m.Status, m.Err
}

func (m MockXPackWatcher) DeleteWatch(ctx context.Context, meta db.Meta) error {
	return m.Err
}
