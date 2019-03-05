package searcher

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
)

type FlowSearcher interface {
	Run(ctx context.Context, name string, period time.Duration)
	Close()
}

type flowSearcher struct {
	suspiciousIP db.SuspiciousIP
	events       db.Events
	once         sync.Once
	cancel       context.CancelFunc
}

func NewFlowSearcher(suspiciousIP db.SuspiciousIP, events db.Events) FlowSearcher {
	return &flowSearcher{suspiciousIP: suspiciousIP, events: events}
}

func (d *flowSearcher) Run(ctx context.Context, name string, period time.Duration) {
	d.once.Do(func() {
		if ctx == nil {
			ctx = context.Background()
		}
		ctx, d.cancel = context.WithCancel(ctx)

		go func() {
			t := time.NewTicker(period)
			for {
				d.doIPSet(ctx, name)
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

func (d *flowSearcher) doIPSet(ctx context.Context, name string) {
	flows, err := d.suspiciousIP.QueryIPSet(ctx, name)
	if err != nil {
		log.WithError(err).Error("suspicious IP query failed")
	}
	log.WithField("num", len(flows)).Info("got flows")
	for _, flow := range flows {
		err := d.events.PutFlowLog(ctx, flow)
		if err != nil {
			log.WithError(err).Error("failed to store suspicious flow")
		}
	}
}
