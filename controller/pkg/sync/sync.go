package sync

import (
	"context"
	"sync"
	"time"

	retry "github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/feed"
	"github.com/tigera/intrusion-detection/controller/pkg/puller"
	"github.com/tigera/intrusion-detection/controller/pkg/runloop"
	"github.com/tigera/intrusion-detection/controller/pkg/statser"
)

const (
	statserType   = "ElasticSyncFailed"
	retryAttempts = 3
	retryDelay    = 5 * time.Second
)

type Syncer interface {
	Run(context.Context, <-chan feed.IPSet, puller.SyncFailFunction, statser.Statser)
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

func (s *syncer) Run(ctx context.Context, c <-chan feed.IPSet, failFunc puller.SyncFailFunction, st statser.Statser) {
	s.once.Do(func() {
		ctx, s.cancel = context.WithCancel(ctx)

		_ = runloop.RunLoopRecvChannel(ctx, func(x interface{}) {
			s.sync(ctx, x.(feed.IPSet), failFunc, st, retryAttempts, retryDelay)
		}, c)
	})
}

func (s *syncer) sync(ctx context.Context, set feed.IPSet, failFunction puller.SyncFailFunction, st statser.Statser, attempts uint, delay time.Duration) {
	err := retry.Do(func() error {
		return s.ipSet.PutIPSet(ctx, s.feed.Name(), set)
	}, retry.Attempts(attempts), retry.Delay(delay))
	if err != nil {
		failFunction()
		log.WithError(err).Error("could not put FeedPuller set from feed")
		st.Error(statserType, err)
	} else {
		st.ClearError(statserType)
		st.SuccessfulSync()
	}
}

func (s *syncer) Close() {
	s.cancel()
}
