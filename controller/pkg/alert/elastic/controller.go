// Copyright 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"context"

	"github.com/tigera/intrusion-detection/controller/pkg/alert/statser"
	"github.com/tigera/intrusion-detection/controller/pkg/controller"
	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/elastic"
)

type watchData struct {
	elastic.XPackWatcher
}

func NewAlertController(xPack elastic.XPackWatcher) controller.Controller {
	return controller.NewController(watchData{xPack}, statser.SyncFailed)
}

func (d watchData) Put(ctx context.Context, name string, value interface{}) error {
	return d.XPackWatcher.PutWatch(ctx, name, value.(*elastic.PutWatchBody))
}

func (d watchData) List(ctx context.Context) ([]db.Meta, error) {
	return d.XPackWatcher.ListWatches(ctx)
}

func (d watchData) Delete(ctx context.Context, m db.Meta) error {
	return d.XPackWatcher.DeleteWatch(ctx, m)
}
