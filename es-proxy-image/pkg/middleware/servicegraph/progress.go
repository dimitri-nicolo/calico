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

// Create a new zeroed elasticProgress.
func newElasticProgress(name string, tr lmav1.TimeRange) *elasticProgress {
	logCxt := log.WithField("name", name+"("+tr.String()+")")
	logCxt.Info("Starting background enumeration")
	p := &elasticProgress{
		start:  time.Now(),
		logCxt: logCxt,
	}
	return p
}

// elasticProgress is used to track and log elasticsearch query progress. The various elastic queries update the
// progress incrementing for each enumerated raw flow and aggregated entry.  The helper logs every `logEvery` raw flows
// and when the enumeration is complete.
type elasticProgress struct {
	start      time.Time
	logCxt     *log.Entry
	raw        int
	aggregated int
}

func (p *elasticProgress) IncRaw() {
	p.raw++
	if p.raw%logEvery == 0 {
		p.logCxt.Infof("Enumeration update: %d raw logs aggregated to %d cache entries", p.raw, p.aggregated)
	}
}

func (p *elasticProgress) IncAggregated() {
	p.aggregated++
}

func (p *elasticProgress) SetAggregated(num int) {
	p.aggregated = num
}

func (p *elasticProgress) Complete(err error) {
	if err == nil {
		p.logCxt.Infof("Enumeration complete: %d raw logs aggregated to %d cache entries", p.raw, p.aggregated)
	} else {
		p.logCxt.WithError(err).Infof("Enumeration failed: %d raw logs aggregated to %d cache entries", p.raw, p.aggregated)
	}
}
