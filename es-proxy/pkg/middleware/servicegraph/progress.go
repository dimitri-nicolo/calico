// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"time"

	log "github.com/sirupsen/logrus"

	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
)

const (
	logEvery = 5000
)

// Create a new zeroed progressMeter.
func newProgress(name string, tr lmav1.TimeRange) *progressMeter {
	logCxt := log.WithField("name", name+"("+tr.String()+")")
	logCxt.Info("Starting background enumeration")
	p := &progressMeter{
		start:  time.Now(),
		logCxt: logCxt,
	}
	return p
}

// progressMeter is used to track and log query progress. The various queries update the
// progress incrementing for each enumerated raw flow and aggregated entry.  The helper logs every `logEvery` raw flows
// and when the enumeration is complete.
type progressMeter struct {
	start      time.Time
	logCxt     *log.Entry
	raw        int
	aggregated int
}

func (p *progressMeter) IncRaw() {
	p.raw++
	if p.raw%logEvery == 0 {
		p.logCxt.Infof("Enumeration update: %d raw logs aggregated to %d cache entries", p.raw, p.aggregated)
	}
}

func (p *progressMeter) IncAggregated() {
	p.aggregated++
}

func (p *progressMeter) SetAggregated(num int) {
	p.aggregated = num
}

func (p *progressMeter) Complete(err error) {
	if err == nil {
		p.logCxt.Infof("Enumeration complete: %d raw logs aggregated to %d cache entries", p.raw, p.aggregated)
	} else {
		p.logCxt.WithError(err).Infof("Enumeration failed: %d raw logs aggregated to %d cache entries", p.raw, p.aggregated)
	}
}
