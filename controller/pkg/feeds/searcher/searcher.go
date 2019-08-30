// Copyright 2019 Tigera Inc. All rights reserved.

package searcher

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/statser"
	"github.com/tigera/intrusion-detection/controller/pkg/runloop"
)

type Searcher interface {
	Run(context.Context, statser.Statser)
	SetFeed(*v3.GlobalThreatFeed)
	Close()
}

type searcher struct {
	feed   *v3.GlobalThreatFeed
	period time.Duration
	q      db.SuspiciousSet
	events db.Events
	once   sync.Once
	cancel context.CancelFunc
}

func (d *searcher) Run(ctx context.Context, statser statser.Statser) {
	d.once.Do(func() {
		ctx, d.cancel = context.WithCancel(ctx)
		go func() {
			defer d.cancel()
			_ = runloop.RunLoop(ctx, func() { d.doSearch(ctx, statser) }, d.period)
		}()
	})
}

func (d *searcher) SetFeed(f *v3.GlobalThreatFeed) {
	d.feed = f.DeepCopy()
}

func (d *searcher) Close() {
	d.cancel()
}

func (d *searcher) doSearch(ctx context.Context, st statser.Statser) {
	results, err := d.q.QuerySet(ctx, d.feed.Name)
	if err != nil {
		log.WithError(err).Error("query failed")
		st.Error(statser.SearchFailed, err)
		return
	}
	var clean = true
	for _, result := range results {
		err := d.events.PutSecurityEvent(ctx, result)
		if err != nil {
			clean = false
			st.Error(statser.SearchFailed, err)
			log.WithError(err).Error("failed to store event")
		}
	}
	if clean {
		st.ClearError(statser.SearchFailed)
		st.SuccessfulSearch()
	}
}

func NewSearcher(feed *v3.GlobalThreatFeed, period time.Duration, suspiciousSet db.SuspiciousSet, events db.Events) Searcher {
	return &searcher{
		feed:   feed.DeepCopy(),
		period: period,
		q:      suspiciousSet,
		events: events,
	}
}
