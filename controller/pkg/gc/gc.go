// Copyright 2019 Tigera Inc. All rights reserved.

package gc

import (
	"context"
	"sync"
	"time"

	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"

	"github.com/tigera/intrusion-detection/controller/pkg/statser"
)

type GarbageCollector interface {
	Run(context.Context, statser.Statser)
	SetFeed(*v3.GlobalThreatFeed)
	Close()
}

type garbageCollector struct {
	feed   *v3.GlobalThreatFeed
	period time.Duration
	cancel context.CancelFunc
	once   sync.Once
}

func NewGarbageCollector(feed *v3.GlobalThreatFeed, period time.Duration) GarbageCollector {
	return &garbageCollector{feed: feed.DeepCopy(), period: period}
}

func (g *garbageCollector) Run(ctx context.Context, statser statser.Statser) {
	g.once.Do(func() {
		ctx, g.cancel = context.WithCancel(ctx)

		go func() {
			for {
				// garbage collect

				t := time.NewTicker(g.period)
				select {
				case <-ctx.Done():
					return
				case <-t.C:
					break
				}
			}
		}()
	})
}

func (g *garbageCollector) SetFeed(f *v3.GlobalThreatFeed) {
	g.feed = f.DeepCopy()
}

func (g *garbageCollector) Close() {
	g.cancel()
}
