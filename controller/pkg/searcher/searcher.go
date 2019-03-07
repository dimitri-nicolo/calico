package searcher

import (
	"context"
	"sync"
	"time"

	"github.com/tigera/intrusion-detection/controller/pkg/feed"
	"github.com/tigera/intrusion-detection/controller/pkg/runloop"
	"github.com/tigera/intrusion-detection/controller/pkg/statser"

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
		runloop.RunLoop(ctx, func() { d.doIPSet(ctx, statser) }, d.period)
	})
}

func (d *flowSearcher) Close() {
	d.cancel()
}

func (d *flowSearcher) doIPSet(ctx context.Context, statser statser.Statser) {
	flowIterator, err := d.suspiciousIP.QueryIPSet(ctx, d.feed.Name())
	if err != nil {
		log.WithError(err).Error("suspicious IP query failed")
		statser.Error(statserType, err)
		return
	}
	c := 0
	var clean = true
	for flowIterator.Next() {
		c++
		err := d.events.PutFlowLog(ctx, flowIterator.Value())
		if err != nil {
			clean = false
			statser.Error(statserType, err)
			log.WithError(err).Error("failed to store suspicious flow")
		}
	}
	log.WithField("num", c).Debug("got flows")
	if flowIterator.Err() != nil {
		log.WithError(flowIterator.Err()).Error("suspicious IP iteration failed")
		statser.Error(statserType, flowIterator.Err())
		return
	}
	if clean {
		statser.ClearError(statserType)
		statser.SuccessfulSearch()
	}
}
