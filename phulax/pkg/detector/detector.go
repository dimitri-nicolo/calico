package detector

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/intrusion-detection/phulax/pkg/db"
)

type Detector interface {
	RunIPSet(ctx context.Context, name string, period time.Duration)
}

type detector struct {
	q db.SuspiciousIP
	p db.Events
}

func NewDetector(q db.SuspiciousIP, p db.Events) Detector {
	return &detector{q, p}
}

func (d *detector) RunIPSet(ctx context.Context, name string, period time.Duration) {
	d.doIPSet(name)
	t := time.NewTicker(period)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			// continue
		}
		d.doIPSet(name)
	}
}

func (d *detector) doIPSet(name string) {
	flows, err := d.q.QueryIPSet(name)
	if err != nil {
		log.WithError(err).Error("suspicious IP query failed")
	}
	log.WithField("num", len(flows)).Info("got flows")
	for _, flow := range flows {
		err := d.p.PutFlowLog(flow)
		if err != nil {
			log.WithError(err).Error("failed to store suspicious flow")
		}
	}
}
