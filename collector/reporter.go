// Copyright (c) 2017-2018 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"
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

const (
	UpdateTypeReportStr = "report"
	UpdateTypeExpireStr = "expire"
)

func (ut UpdateType) String() string {
	if ut == UpdateTypeReport {
		return UpdateTypeReportStr
	}
	return UpdateTypeExpireStr
}

type MetricValue struct {
	deltaPackets             int
	deltaBytes               int
	deltaAllowedHTTPRequests int
	deltaDeniedHTTPRequests  int
}

func (mv MetricValue) String() string {
	return fmt.Sprintf("delta=%v deltaBytes=%v deltaAllowedHTTPReq=%v deltaDeniedHTTPReq=%v",
		mv.deltaPackets, mv.deltaBytes, mv.deltaAllowedHTTPRequests, mv.deltaDeniedHTTPRequests)
}

type MetricUpdate struct {
	updateType UpdateType

	// Tuple key
	tuple Tuple

	// Endpoint information.
	srcEp *calc.EndpointData
	dstEp *calc.EndpointData

	// isConnection is true if this update is from an active connection.
	isConnection bool

	// Rules identification
	ruleIDs []*calc.RuleID

	// Inbound/Outbound packet/byte counts.
	inMetric  MetricValue
	outMetric MetricValue
}

func (mu MetricUpdate) String() string {
	var srcName, dstName string
	if mu.srcEp != nil {
		srcName = endpointName(mu.srcEp.Key)
	} else {
		srcName = "<unknown>"
	}
	if mu.dstEp != nil {
		dstName = endpointName(mu.dstEp.Key)
	} else {
		dstName = "<unknown>"
	}
	return fmt.Sprintf("MetricUpdate: type=%s tuple={%v}, srcEp={%v} dstEp={%v} isConnection={%v}, ruleID={%v}, inMetric={%s} outMetric={%s}",
		mu.updateType, &(mu.tuple), srcName, dstName, mu.isConnection, mu.ruleIDs, mu.inMetric, mu.outMetric)
}

func (mu MetricUpdate) GetLastRuleID() *calc.RuleID {
	if mu.ruleIDs == nil || len(mu.ruleIDs) == 0 {
		return nil
	}
	return mu.ruleIDs[len(mu.ruleIDs)-1]
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
