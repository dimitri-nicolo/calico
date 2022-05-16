// Copyright (c) 2017-2022 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"
	"net"
	"time"

	log "github.com/sirupsen/logrus"

	"k8s.io/kubernetes/pkg/proxy"

	"github.com/projectcalico/calico/felix/calc"
	logutil "github.com/projectcalico/calico/felix/logutils"
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

type TCPMetricValue struct {
	deltaTotalRetrans   int
	deltaLostOut        int
	deltaUnRecoveredRTO int
}

func (mv MetricValue) String() string {
	return fmt.Sprintf("delta=%v deltaBytes=%v deltaAllowedHTTPReq=%v deltaDeniedHTTPReq=%v",
		mv.deltaPackets, mv.deltaBytes, mv.deltaAllowedHTTPRequests, mv.deltaDeniedHTTPRequests)
}

// Increments adds delta values for all counters using another MetricValue
func (mv *MetricValue) Increment(other MetricValue) {
	mv.deltaBytes += other.deltaBytes
	mv.deltaPackets += other.deltaPackets
	mv.deltaAllowedHTTPRequests += other.deltaAllowedHTTPRequests
	mv.deltaDeniedHTTPRequests += other.deltaDeniedHTTPRequests
}

// Reset will set all the counters stored to 0
func (mv *MetricValue) Reset() {
	mv.deltaBytes = 0
	mv.deltaPackets = 0
	mv.deltaAllowedHTTPRequests = 0
	mv.deltaDeniedHTTPRequests = 0
}

func (tm *TCPMetricValue) Increment(other TCPMetricValue) {
	tm.deltaTotalRetrans += other.deltaTotalRetrans
	tm.deltaLostOut += other.deltaLostOut
	tm.deltaUnRecoveredRTO += other.deltaUnRecoveredRTO
}

func (tm *TCPMetricValue) Reset() {
	tm.deltaTotalRetrans = 0
	tm.deltaLostOut = 0
	tm.deltaUnRecoveredRTO = 0
}

// MetricServiceInfo holds information of a service for a MetricUpdate
type MetricServiceInfo struct {
	proxy.ServicePortName
	// the preDNATPort used to query from the Service info in dstService
	PortNum int
}

type MetricUpdate struct {
	updateType UpdateType

	// Tuple key
	tuple           Tuple
	natOutgoingPort int
	origSourceIPs   *boundedSet

	// Endpoint information.
	srcEp      *calc.EndpointData
	dstEp      *calc.EndpointData
	dstService MetricServiceInfo

	// Top level egress Domains.
	dstDomains []string

	// isConnection is true if this update is from an active connection.
	isConnection bool

	// Rules identification
	ruleIDs []*calc.RuleID

	// Whether the rules IDs contains a deny rule.
	hasDenyRule bool

	// Sometimes we may need to send updates without having all the rules
	// in place. This field will help aggregators determine if they need
	// to handle this update or not. Typically this is used when we receive
	// HTTP Data updates after the connection itself has closed.
	unknownRuleID *calc.RuleID

	// Inbound/Outbound packet/byte counts.
	inMetric  MetricValue
	outMetric MetricValue

	// Optional process info
	processName string
	processID   int
	processArgs string

	//optional Tcp v4 socket stats
	sendCongestionWnd *int
	smoothRtt         *int
	minRtt            *int
	mss               *int
	tcpMetric         TCPMetricValue
}

func (mu MetricUpdate) String() string {
	var (
		srcName, dstName string
		numOrigIPs       int
		origIPs          []net.IP
	)
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
	if mu.origSourceIPs != nil {
		numOrigIPs = mu.origSourceIPs.TotalCount()
		origIPs = mu.origSourceIPs.ToIPSlice()
	} else {
		numOrigIPs = 0
		origIPs = []net.IP{}
	}
	if mu.sendCongestionWnd != nil && mu.smoothRtt != nil && mu.minRtt != nil && mu.mss != nil {
		return fmt.Sprintf("MetricUpdate: type=%s tuple={%v}, srcEp={%v} dstEp={%v} isConnection={%v}, ruleID={%v}, unknownRuleID={%v} inMetric={%s} outMetric={%s} origIPs={%v} numOrigIPs={%d} processInfo={%s, %d, %s} tcpSocketStats={%d, %d, %d, %d, %d, %d, %d}",
			mu.updateType, &(mu.tuple), srcName, dstName, mu.isConnection, mu.ruleIDs, mu.unknownRuleID, mu.inMetric, mu.outMetric, origIPs, numOrigIPs, mu.processName, mu.processID, mu.processArgs, *mu.sendCongestionWnd, *mu.smoothRtt, *mu.minRtt, *mu.mss, mu.tcpMetric.deltaTotalRetrans, mu.tcpMetric.deltaLostOut, mu.tcpMetric.deltaUnRecoveredRTO)
	} else {
		return fmt.Sprintf("MetricUpdate: type=%s tuple={%v}, srcEp={%v} dstEp={%v} isConnection={%v}, ruleID={%v}, unknownRuleID={%v} inMetric={%s} outMetric={%s} origIPs={%v} numOrigIPs={%d} processInfo={%s, %d, %s} tcpSocketStats={0, 0, 0, 0, 0, 0, 0}",
			mu.updateType, &(mu.tuple), srcName, dstName, mu.isConnection, mu.ruleIDs, mu.unknownRuleID, mu.inMetric, mu.outMetric, origIPs, numOrigIPs, mu.processName, mu.processID, mu.processArgs)
	}
}

func (mu MetricUpdate) GetLastRuleID() *calc.RuleID {
	if mu.ruleIDs != nil && len(mu.ruleIDs) > 0 {
		return mu.ruleIDs[len(mu.ruleIDs)-1]
	} else if mu.unknownRuleID != nil {
		return mu.unknownRuleID
	}
	return nil
}

type MetricsReporter interface {
	Start()
	Report(mu MetricUpdate) error
}

type ReporterManager struct {
	ReportChan            chan MetricUpdate
	reporters             []MetricsReporter
	displayDebugTraceLogs bool
}

func NewReporterManager(displayDebugTraceLogs bool) *ReporterManager {
	return &ReporterManager{
		displayDebugTraceLogs: displayDebugTraceLogs,
		ReportChan:            make(chan MetricUpdate, reporterChanBufferSize),
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
			logutil.Tracef(r.displayDebugTraceLogs, "Received metric update %v", mu)
			for _, reporter := range r.reporters {
				reporter.Report(mu)
			}
		}
	}
}
