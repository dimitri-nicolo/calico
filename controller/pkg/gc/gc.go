package gc

import (
	"context"
	"github.com/tigera/intrusion-detection/controller/pkg/feed"
	"github.com/tigera/intrusion-detection/controller/pkg/statser"
	"sync"
	"time"
)

const statserType = "GarbageCollectionFailed"

type GarbageCollector interface {
	Run(context.Context, statser.Statser)
	Close()
}

type garbageCollector struct {
	feed   feed.Feed
	period time.Duration
	cancel context.CancelFunc
	once   sync.Once
}

func NewGarbageCollector(feed feed.Feed, period time.Duration) GarbageCollector {
	return &garbageCollector{feed: feed, period: period}
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

func (g *garbageCollector) Close() {
	g.cancel()
}
