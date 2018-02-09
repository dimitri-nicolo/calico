// Copyright (c) 2017-2018 Tigera, Inc. All rights reserved.

package collector

import (
	"time"

	log "github.com/sirupsen/logrus"
)

const CheckInterval = time.Duration(1) * time.Second

type MetricUpdate struct {
	// Tuple key
	tuple Tuple

	// Rule identification
	ruleIDs *RuleIDs

	// Traffic direction.  For NFLOG entries, the traffic direction will always
	// be "outbound" since the direction is already defined by the source and
	// destination.
	trafficDir TrafficDirection

	// isConnection is true if this update is from an active connection (i.e. a conntrack
	// update compared to an NFLOG update).
	isConnection bool

	// Metric values
	packets      int
	bytes        int
	deltaPackets int
	deltaBytes   int
}

type MetricsReporter interface {
	Start()
	Report(mu *MetricUpdate) error
	Expire(mu *MetricUpdate) error
}

//TODO: RLB: I think expiration events should only be provided for connections and not for
// NFLOG rule events.  It feels like the expiration of a statistic is the responsibility of the
// reporter.  However, I think this requires additional changes:
// -  We should only provide deltas in the MetricUpdate and not actual values (that way the
//    higher layers can expire data before the reporter does (if it desires).
// -  We should use a single channel to report all updates rather than split between a report
//    and expiration metric.  We can just have a field in the metric indicating whether this
//    is a connection close event.  So maybe have an event type:  conn-active, conn-inactive, rule-update.
type ReporterManager struct {
	ReportChan chan *MetricUpdate
	ExpireChan chan *MetricUpdate
	reporters  []MetricsReporter
}

func NewReporterManager() *ReporterManager {
	return &ReporterManager{
		// TODO: RLB: This is a blocking channel, should we give it some buffer?
		ReportChan: make(chan *MetricUpdate),
		ExpireChan: make(chan *MetricUpdate),
	}
}

func (r *ReporterManager) RegisterMetricsReporter(mr MetricsReporter) {
	r.reporters = append(r.reporters, mr)
}

func (r *ReporterManager) Start() {
	for _, reporter := range r.reporters {
		reporter.Start()
	}
	go r.startManaging()
}

func (r *ReporterManager) startManaging() {
	log.Info("Starting ReporterManager")
	for {
		// TODO(doublek): Channel for stopping the reporter.
		select {
		case mu := <-r.ReportChan:
			log.Debugf("Reporting metric update %+v", mu)
			for _, reporter := range r.reporters {
				reporter.Report(mu)
			}
		case mu := <-r.ExpireChan:
			log.Debugf("Expiring metric update %+v", mu)
			for _, reporter := range r.reporters {
				reporter.Expire(mu)
			}
		}
	}
}
