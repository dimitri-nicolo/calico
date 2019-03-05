package sync

import (
	"context"
	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/feed"
	"sync"

	log "github.com/sirupsen/logrus"
)

type Syncer interface {
	Run(context.Context, <-chan feed.IPSet)
	Close()
}

type syncer struct {
	feed   feed.Feed
	ipSet  db.IPSet
	cancel context.CancelFunc
	once   sync.Once
}

func NewSyncer(feed feed.Feed, ipSet db.IPSet) Syncer {
	return &syncer{feed: feed, ipSet: ipSet}
}

func (s *syncer) Run(ctx context.Context, c <-chan feed.IPSet) {
	s.once.Do(func() {
		if ctx == nil {
			ctx = context.Background()
		}
		ctx, s.cancel = context.WithCancel(ctx)

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case set, ok := <-c:
					if ok {
						err := s.ipSet.PutIPSet(ctx, s.feed.Name(), set)
						if err != nil {
							log.WithError(err).Error("could not put FeedPuller set from feed")
						}
					} else {
						// channel closed
						return
					}
				}
			}
		}()
	})
}

func (s *syncer) Close() {
	s.cancel()
}
