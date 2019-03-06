package searcher

import (
	"context"
	"github.com/tigera/intrusion-detection/controller/pkg/feed"
	"github.com/tigera/intrusion-detection/controller/pkg/statser"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
)

const statserType = "SearchFailed"

type FlowSearcher interface {
	Run(context.Context, statser.Statser)
	Close()
}

type flowSearcher struct {
	feed         feed.Feed
	period       time.Duration
	suspiciousIP db.SuspiciousIP
	events       db.Events
	once         sync.Once
	cancel       context.CancelFunc
}

func NewFlowSearcher(feed feed.Feed, period time.Duration, suspiciousIP db.SuspiciousIP, events db.Events) FlowSearcher {
	return &flowSearcher{feed: feed, period: period, suspiciousIP: suspiciousIP, events: events}
}

func (d *flowSearcher) Run(ctx context.Context, statser statser.Statser) {
	d.once.Do(func() {
		if ctx == nil {
			ctx = context.Background()
		}
		ctx, d.cancel = context.WithCancel(ctx)

		go func() {
			t := time.NewTicker(d.period)
			for {
				d.doIPSet(ctx, d.feed.Name(), statser)
				select {
				case <-ctx.Done():
					t.Stop()
					return
				case <-t.C:
					// continue
				}
			}
		}()
	})
}

func (d *flowSearcher) Close() {
	d.cancel()
}

func (d *flowSearcher) doIPSet(ctx context.Context, name string, statser statser.Statser) {
	flows, err := d.suspiciousIP.QueryIPSet(ctx, name)
	if err != nil {
		log.WithError(err).Error("suspicious IP query failed")
		statser.Error(statserType, err)
		return
	}
	log.WithField("num", len(flows)).Info("got flows")
	var clean = true
	for _, flow := range flows {
		err := d.events.PutFlowLog(ctx, flow)
		if err != nil {
			clean = false
			statser.Error(statserType, err)
			log.WithError(err).Error("failed to store suspicious flow")
		}
	}
	if clean {
		statser.ClearError(statserType)
		statser.SuccessfulSearch()
	}
}
