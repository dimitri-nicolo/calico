// Copyright (c) 2017-2018 Tigera, Inc. All rights reserved.

package collector

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/calc"
)

const (
	CheckInterval          = time.Duration(1) * time.Second
	reporterChanBufferSize = 1000
)

type UpdateType int

const (
	UpdateTypeReport UpdateType = iota
	UpdateTypeExpire
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
	ruleID *calc.RuleID

	inMetric  MetricValue
	outMetric MetricValue
}

type MetricsReporter interface {
	Start()
	Report(mu MetricUpdate) error
}

type ReporterManager struct {
	ReportChan chan MetricUpdate
	reporters  []MetricsReporter
}

func NewReporterManager() *ReporterManager {
	return &ReporterManager{
		ReportChan: make(chan MetricUpdate, reporterChanBufferSize),
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
			for _, reporter := range r.reporters {
				reporter.Report(mu)
			}
		}
	}
}
