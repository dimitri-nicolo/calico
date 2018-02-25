// Copyright (c) 2017-2018 Tigera, Inc. All rights reserved.

package collector

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/rules"
)

const (
	CheckInterval = time.Duration(1) * time.Second
	bufferSize    = 1000
)

type UpdateType string

const (
	UpdateTypeReport UpdateType = "report"
	UpdateTypeExpire UpdateType = "expire"
)

type MetricValue struct {
	deltaPackets int
	deltaBytes   int
}

type MetricUpdate struct {
	updateType UpdateType

	// Tuple key
	tuple Tuple

	// isConnection is true if this update is from an active connection.
	isConnection bool

	// Rule identification
	ruleIDs *rules.RuleIDs

	inMetric  MetricValue
	outMetric MetricValue
}

type MetricsReporter interface {
	Start()
	Report(mu *MetricUpdate) error
}

type ReporterManager struct {
	ReportChan chan *MetricUpdate
	reporters  []MetricsReporter
}

func NewReporterManager() *ReporterManager {
	return &ReporterManager{
		ReportChan: make(chan *MetricUpdate, bufferSize),
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
		}
	}
}
