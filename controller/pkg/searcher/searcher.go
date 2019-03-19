// Copyright 2019 Tigera Inc. All rights reserved.

package searcher

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/runloop"
	"github.com/tigera/intrusion-detection/controller/pkg/statser"
)

type FlowSearcher interface {
	Run(context.Context, statser.Statser)
	SetFeed(*v3.GlobalThreatFeed)
	Close()
}

type flowSearcher struct {
	feed         *v3.GlobalThreatFeed
	period       time.Duration
	suspiciousIP db.SuspiciousIP
	events       db.Events
	once         sync.Once
	cancel       context.CancelFunc
}

func NewFlowSearcher(feed *v3.GlobalThreatFeed, period time.Duration, suspiciousIP db.SuspiciousIP, events db.Events) FlowSearcher {
	return &flowSearcher{feed: feed.DeepCopy(), period: period, suspiciousIP: suspiciousIP, events: events}
}

func (d *flowSearcher) Run(ctx context.Context, statser statser.Statser) {
	d.once.Do(func() {
		ctx, d.cancel = context.WithCancel(ctx)
		go func() {
			defer d.cancel()
			_ = runloop.RunLoop(ctx, func() { d.doIPSet(ctx, statser) }, d.period)
		}()
	})
}

func (d *flowSearcher) SetFeed(f *v3.GlobalThreatFeed) {
	d.feed = f.DeepCopy()
}

func (d *flowSearcher) Close() {
	d.cancel()
}

func (d *flowSearcher) doIPSet(ctx context.Context, st statser.Statser) {
	flowIterator, err := d.suspiciousIP.QueryIPSet(ctx, d.feed.Name)
	if err != nil {
		log.WithError(err).Error("suspicious IP query failed")
		st.Error(statser.SearchFailed, err)
		return
	}
	c := 0
	var clean = true
	for flowIterator.Next() {
		c++
		err := d.events.PutSecurityEvent(ctx, flowIterator.Value())
		if err != nil {
			clean = false
			st.Error(statser.SearchFailed, err)
			log.WithError(err).Error("failed to store suspicious flow")
		}
	}
	log.WithField("num", c).Debug("got events")
	if flowIterator.Err() != nil {
		log.WithError(flowIterator.Err()).Error("suspicious IP iteration failed")
		st.Error(statser.SearchFailed, flowIterator.Err())
		return
	}
	if clean {
		st.ClearError(statser.SearchFailed)
		st.SuccessfulSearch()
	}
}
