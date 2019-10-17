// Copyright 2019 Tigera Inc. All rights reserved.

package watcher

import (
	"context"
	"sync"

	"github.com/tigera/intrusion-detection/controller/pkg/anomaly/filters"
	"github.com/tigera/intrusion-detection/controller/pkg/anomaly/puller"
	"github.com/tigera/intrusion-detection/controller/pkg/anomaly/statser"
	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/health"
)

type Watcher interface {
	health.Pinger

	// Run starts the feed synchronization.
	Run(ctx context.Context)
	Close()
}

type watcher struct {
	events      db.Events
	auditLog    db.AuditLog
	xPack       elastic.XPackAnomalyDetector
	jobWatchers map[string]*jobWatcher
	cancel      context.CancelFunc
	once        sync.Once
	watching    bool
}

type jobWatcher struct {
	name    string
	puller  puller.Puller
	statser statser.Statser
}

func NewWatcher(events db.Events, auditLog db.AuditLog, xPack elastic.XPackAnomalyDetector) Watcher {
	return &watcher{
		events:      events,
		auditLog:    auditLog,
		xPack:       xPack,
		jobWatchers: make(map[string]*jobWatcher),
	}
}

func (w *watcher) Run(ctx context.Context) {
	w.once.Do(func() {
		ctx, w.cancel = context.WithCancel(ctx)

		go func() {
			for jid, info := range Jobs {
				statser := statser.NewStatser(jid)

				w.jobWatchers[jid] = &jobWatcher{
					name: jid,
					puller: puller.NewPuller(
						jid, w.xPack, w.events,
						filters.NewAuditLog(w.auditLog),
						info.Detectors),
					statser: statser,
				}

				statser.Run(ctx)
				w.jobWatchers[jid].puller.Run(ctx, statser)
			}
			w.watching = true
		}()
	})
}

func (w *watcher) Close() {
	w.cancel()
}

func (w *watcher) Ping(context.Context) error {
	return nil
}
