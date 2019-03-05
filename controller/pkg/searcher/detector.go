package searcher

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
)

type FlowSearcher interface {
	Run(ctx context.Context, name string, period time.Duration)
}

type flowSearcher struct {
	q db.SuspiciousIP
	p db.Events
}

func NewFlowSearcher(q db.SuspiciousIP, p db.Events) FlowSearcher {
	return &flowSearcher{q, p}
}

func (d *flowSearcher) Run(ctx context.Context, name string, period time.Duration) {
	t := time.NewTicker(period)
	for {
		d.doIPSet(ctx, name)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			// continue
		}
	}
}

func (d *flowSearcher) doIPSet(ctx context.Context, name string) {
	flows, err := d.q.QueryIPSet(ctx, name)
	if err != nil {
		log.WithError(err).Error("suspicious IP query failed")
	}
	log.WithField("num", len(flows)).Info("got flows")
	for _, flow := range flows {
		err := d.p.PutFlowLog(ctx, flow)
		if err != nil {
			log.WithError(err).Error("failed to store suspicious flow")
		}
	}
}
