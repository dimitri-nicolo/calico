// Copyright (c) 2018-2022 Tigera, Inc. All rights reserved.

package collector

import (
	"container/list"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	logutil "github.com/projectcalico/calico/felix/logutils"

	"github.com/projectcalico/calico/libcalico-go/lib/set"
)

const (
	unsetIntField = -1
)

var (
	emptyService = FlowService{"-", "-", "-", 0}
	emptyIP      = [16]byte{}
)

type FlowLogEndpointType string
type FlowLogAction string
type FlowLogReporter string
type FlowLogSubnetType string

type EndpointMetadata struct {
	Type           FlowLogEndpointType `json:"type"`
	Namespace      string              `json:"namespace"`
	Name           string              `json:"name"`
	AggregatedName string              `json:"aggregated_name"`
}

type FlowService struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	PortName  string `json:"port_name"`
	PortNum   int    `json:"port_num"`
}

type FlowMeta struct {
	Tuple      Tuple            `json:"tuple"`
	SrcMeta    EndpointMetadata `json:"sourceMeta"`
	DstMeta    EndpointMetadata `json:"destinationMeta"`
	DstService FlowService      `json:"destinationService"`
	Action     FlowLogAction    `json:"action"`
	Reporter   FlowLogReporter  `json:"flowReporter"`
}

type TCPRtt struct {
	Mean int `json:"mean"`
	Max  int `json:"max"`
}

type TCPWnd struct {
	Mean int `json:"mean"`
	Min  int `json:"min"`
}

type TCPMss struct {
	Mean int `json:"mean"`
	Min  int `json:"min"`
}

func newFlowMeta(mu MetricUpdate, includeService bool) (FlowMeta, error) {
	f := FlowMeta{}

	// Extract Tuple Info
	f.Tuple = mu.tuple

	// Extract EndpointMetadata info
	srcMeta, err := getFlowLogEndpointMetadata(mu.srcEp, mu.tuple.src)
	if err != nil {
		return FlowMeta{}, fmt.Errorf("could not extract metadata for source %v", mu.srcEp)
	}
	dstMeta, err := getFlowLogEndpointMetadata(mu.dstEp, mu.tuple.dst)
	if err != nil {
		return FlowMeta{}, fmt.Errorf("could not extract metadata for destination %v", mu.dstEp)
	}

	f.SrcMeta = srcMeta
	f.DstMeta = dstMeta

	if includeService {
		f.DstService = getFlowLogService(mu.dstService)
	} else {
		f.DstService = emptyService
	}

	lastRuleID := mu.GetLastRuleID()
	if lastRuleID == nil {
		log.WithField("metric update", mu).Error("no rule id present")
		return f, fmt.Errorf("invalid metric update")
	}

	action, direction := getFlowLogActionAndReporterFromRuleID(lastRuleID)
	f.Action = action
	f.Reporter = direction

	return f, nil
}

func newFlowMetaWithSourcePortAggregation(mu MetricUpdate, includeService bool) (FlowMeta, error) {
	f, err := newFlowMeta(mu, includeService)
	if err != nil {
		return FlowMeta{}, err
	}
	f.Tuple.l4Src = unsetIntField

	return f, nil
}

func newFlowMetaWithPrefixNameAggregation(mu MetricUpdate, includeService bool) (FlowMeta, error) {
	f, err := newFlowMeta(mu, includeService)
	if err != nil {
		return FlowMeta{}, err
	}

	f.Tuple.src = emptyIP
	f.Tuple.l4Src = unsetIntField
	f.Tuple.dst = emptyIP
	f.SrcMeta.Name = flowLogFieldNotIncluded
	f.DstMeta.Name = flowLogFieldNotIncluded

	return f, nil
}

func newFlowMetaWithNoDestPortsAggregation(mu MetricUpdate, includeService bool) (FlowMeta, error) {
	f, err := newFlowMeta(mu, includeService)
	if err != nil {
		return FlowMeta{}, err
	}

	f.Tuple.src = emptyIP
	f.Tuple.l4Src = unsetIntField
	f.Tuple.l4Dst = unsetIntField
	f.Tuple.dst = emptyIP
	f.SrcMeta.Name = flowLogFieldNotIncluded
	f.DstMeta.Name = flowLogFieldNotIncluded
	f.DstService.PortName = flowLogFieldNotIncluded

	return f, nil
}

func NewFlowMeta(mu MetricUpdate, kind FlowAggregationKind, includeService bool) (FlowMeta, error) {
	switch kind {
	case FlowDefault:
		return newFlowMeta(mu, includeService)
	case FlowSourcePort:
		return newFlowMetaWithSourcePortAggregation(mu, includeService)
	case FlowPrefixName:
		return newFlowMetaWithPrefixNameAggregation(mu, includeService)
	case FlowNoDestPorts:
		return newFlowMetaWithNoDestPortsAggregation(mu, includeService)
	}

	return FlowMeta{}, fmt.Errorf("aggregation kind %v not recognized", kind)
}

type FlowSpec struct {
	FlowStatsByProcess
	flowExtrasRef
	FlowLabels
	FlowPolicies
	FlowDestDomains

	// Reset aggregated data on the next metric update to ensure we clear out obsolete labels, policies and Domains for
	// connections that are not actively part of the flow during the export interval.
	resetAggrData bool
}

func NewFlowSpec(mu *MetricUpdate, maxOriginalIPsSize, maxDomains int, includeProcess bool, processLimit, processArgsLimit int, displayDebugTraceLogs bool, natOutgoingPortLimit int) *FlowSpec {
	// NewFlowStatsByProcess potentially needs to update fields in mu *MetricUpdate hence passing it by pointer
	// TODO: reconsider/refactor the inner functions called in NewFlowStatsByProcess to avoid above scenario
	return &FlowSpec{
		FlowLabels:         NewFlowLabels(*mu),
		FlowPolicies:       NewFlowPolicies(*mu),
		FlowStatsByProcess: NewFlowStatsByProcess(mu, includeProcess, processLimit, processArgsLimit, displayDebugTraceLogs, natOutgoingPortLimit),
		flowExtrasRef:      NewFlowExtrasRef(*mu, maxOriginalIPsSize),
		FlowDestDomains:    NewFlowDestDomains(*mu, maxDomains),
	}
}

func (f *FlowSpec) ContainsActiveRefs(mu *MetricUpdate) bool {
	return f.FlowStatsByProcess.containsActiveRefs(mu)
}

func (f *FlowSpec) ToFlowLogs(fm FlowMeta, startTime, endTime time.Time, includeLabels bool, includePolicies bool) []*FlowLog {
	stats := f.FlowStatsByProcess.toFlowProcessReportedStats()

	flogs := make([]*FlowLog, 0, len(stats))
	for _, stat := range stats {
		fl := &FlowLog{
			FlowMeta:                 fm,
			StartTime:                startTime,
			EndTime:                  endTime,
			FlowProcessReportedStats: stat,
			FlowDestDomains:          f.FlowDestDomains,
		}
		if f.flowExtrasRef.originalSourceIPs != nil {
			fe := FlowExtras{
				OriginalSourceIPs:    f.flowExtrasRef.originalSourceIPs.ToIPSlice(),
				NumOriginalSourceIPs: f.flowExtrasRef.originalSourceIPs.TotalCount(),
			}
			fl.FlowExtras = fe
		}

		if includeLabels {
			fl.FlowLabels = f.FlowLabels
		}

		if !includePolicies {
			fl.FlowPolicies = nil
		} else {
			fl.FlowPolicies = f.FlowPolicies
		}

		flogs = append(flogs, fl)
	}
	return flogs
}

func (f *FlowSpec) AggregateMetricUpdate(mu *MetricUpdate) {
	if f.resetAggrData {
		// Reset the aggregated data from this metric update.
		f.FlowPolicies = make(FlowPolicies)
		f.FlowLabels.SrcLabels = nil
		f.FlowLabels.DstLabels = nil
		f.FlowDestDomains.reset()
		f.resetAggrData = false
	}
	f.aggregateFlowLabels(*mu)
	f.aggregateFlowPolicies(*mu)
	f.aggregateFlowDestDomains(*mu)
	f.aggregateFlowExtrasRef(*mu)
	f.aggregateFlowStatsByProcess(mu)
}

// MergeWith merges two flow specs. This means copying the flowRefsActive that contains a reference
// to the original tuple that identifies the traffic. This help keeping the same numFlows counts while
// changing aggregation levels
func (f *FlowSpec) MergeWith(mu MetricUpdate, other *FlowSpec) {
	if stats, ok := f.statsByProcessName[mu.processName]; ok {
		if otherStats, ok := other.statsByProcessName[mu.processName]; ok {
			for tuple, _ := range otherStats.flowsRefsActive {
				stats.flowsRefsActive.AddWithValue(tuple, mu.natOutgoingPort)
				stats.flowsRefs.AddWithValue(tuple, mu.natOutgoingPort)
			}
			stats.NumFlows = stats.flowsRefs.Len()
			// TODO(doublek): Merge processIDs.
		}

	}
}

// FlowSpec has FlowStats that are stats assocated with a given FlowMeta
// These stats are to be refreshed everytime the FlowData
// {FlowMeta->FlowStats} is published so as to account
// for correct no. of started flows in a given aggregation
// interval.
//
// This also resets policy and label data which will be re-populated from metric updates for the still active
// flows.
func (f *FlowSpec) Reset() {
	f.FlowStatsByProcess.reset()
	f.flowExtrasRef.reset()

	// Set the reset flag. We'll reset the aggregated data on the next metric update - that way we don't completely
	// zero out the labels and policies if there is no traffic for an export interval.
	f.resetAggrData = true
}

func (f *FlowSpec) GetActiveFlowsCount() int {
	return f.FlowStatsByProcess.getActiveFlowsCount()
}

// GarbageCollect provides a chance to remove process names and corresponding stats that don't have
// any active flows being tracked.
// As an added optimization, we also return the remaining active flows so that we don't have to
// iterate over all the flow stats grouped by processes a second time.
func (f *FlowSpec) GarbageCollect() int {
	return f.FlowStatsByProcess.gc()
}

type FlowLabels struct {
	SrcLabels map[string]string
	DstLabels map[string]string
}

func NewFlowLabels(mu MetricUpdate) FlowLabels {
	return FlowLabels{
		SrcLabels: getFlowLogEndpointLabels(mu.srcEp),
		DstLabels: getFlowLogEndpointLabels(mu.dstEp),
	}
}

func intersectLabels(in, out map[string]string) map[string]string {
	common := map[string]string{}
	for k := range out {
		// Skip Calico labels from the logs
		if strings.HasPrefix(k, "projectcalico.org/") {
			continue
		}
		if v, ok := in[k]; ok && v == out[k] {
			common[k] = v
		}
	}

	return common
}

func (f *FlowLabels) aggregateFlowLabels(mu MetricUpdate) {
	srcLabels := getFlowLogEndpointLabels(mu.srcEp)
	dstLabels := getFlowLogEndpointLabels(mu.dstEp)

	// The flow labels are reset on calibration, so either copy the labels or intersect them.
	if f.SrcLabels == nil {
		f.SrcLabels = srcLabels
	} else {
		f.SrcLabels = intersectLabels(srcLabels, f.SrcLabels)
	}

	if f.DstLabels == nil {
		f.DstLabels = dstLabels
	} else {
		f.DstLabels = intersectLabels(dstLabels, f.DstLabels)
	}
}

type FlowPolicies map[string]empty

func NewFlowPolicies(mu MetricUpdate) FlowPolicies {
	fp := make(FlowPolicies)
	if mu.ruleIDs == nil {
		return fp
	}
	for idx, rid := range mu.ruleIDs {
		if rid == nil {
			continue
		}
		fp[fmt.Sprintf("%d|%s|%s", idx, rid.GetFlowLogPolicyName(), rid.IndexStr)] = emptyValue
	}
	return fp
}

func (fp FlowPolicies) aggregateFlowPolicies(mu MetricUpdate) {
	if mu.ruleIDs == nil {
		return
	}
	for idx, rid := range mu.ruleIDs {
		if rid == nil {
			continue
		}
		fp[fmt.Sprintf("%d|%s|%s", idx, rid.GetFlowLogPolicyName(), rid.IndexStr)] = emptyValue
	}
}

type FlowDestDomains struct {
	maxDomains int
	Domains    map[string]empty
}

func NewFlowDestDomains(mu MetricUpdate, maxDomains int) FlowDestDomains {
	fp := FlowDestDomains{
		maxDomains: maxDomains,
	}
	fp.aggregateFlowDestDomains(mu)
	return fp
}

func (fp *FlowDestDomains) aggregateFlowDestDomains(mu MetricUpdate) {
	if len(mu.dstDomains) == 0 {
		return
	}
	if fp.Domains == nil {
		fp.Domains = make(map[string]empty)
	}
	if len(fp.Domains) >= fp.maxDomains {
		return
	}
	for _, name := range mu.dstDomains {
		fp.Domains[name] = emptyValue
		if len(fp.Domains) >= fp.maxDomains {
			return
		}
	}
}

func (fp *FlowDestDomains) reset() {
	fp.Domains = nil
}

type flowExtrasRef struct {
	originalSourceIPs *boundedSet
}

func NewFlowExtrasRef(mu MetricUpdate, maxOriginalIPsSize int) flowExtrasRef {
	var osip *boundedSet
	if mu.origSourceIPs != nil {
		osip = NewBoundedSetFromSliceWithTotalCount(maxOriginalIPsSize, mu.origSourceIPs.ToIPSlice(), mu.origSourceIPs.TotalCount())
	} else {
		osip = NewBoundedSet(maxOriginalIPsSize)
	}
	return flowExtrasRef{originalSourceIPs: osip}
}

func (fer *flowExtrasRef) aggregateFlowExtrasRef(mu MetricUpdate) {
	if mu.origSourceIPs != nil {
		fer.originalSourceIPs.Combine(mu.origSourceIPs)
	}
}

func (fer *flowExtrasRef) reset() {
	if fer.originalSourceIPs != nil {
		fer.originalSourceIPs.Reset()
	}
}

// FlowExtras contains some additional useful information for flows.
type FlowExtras struct {
	OriginalSourceIPs    []net.IP `json:"originalSourceIPs"`
	NumOriginalSourceIPs int      `json:"numOriginalSourceIPs"`
}

// flowReferences are internal only stats used for computing numbers of flows
type flowReferences struct {
	// The set of unique flows that were started within the reporting interval. This is added to when a new flow
	// (i.e. one that is not currently active) is reported during the reporting interval. It is reset when the
	// flow data is reported.
	flowsStartedRefs tupleSet
	// The set of unique flows that were completed within the reporting interval. This is added to when a flow
	// termination is reported during the reporting interval. It is reset when the flow data is reported.
	flowsCompletedRefs tupleSet
	// The current set of active flows. The set may increase and decrease during the reporting interval.
	flowsRefsActive tupleSet
	// The set of unique flows that have been active at any point during the reporting interval. This is added
	// to during the reporting interval, and is reset to the set of active flows when the flow data is reported.
	flowsRefs tupleSet
}

// FlowReportedStats are the statistics we actually report out in flow logs.
type FlowReportedStats struct {
	PacketsIn             int `json:"packetsIn"`
	PacketsOut            int `json:"packetsOut"`
	BytesIn               int `json:"bytesIn"`
	BytesOut              int `json:"bytesOut"`
	HTTPRequestsAllowedIn int `json:"httpRequestsAllowedIn"`
	HTTPRequestsDeniedIn  int `json:"httpRequestsDeniedIn"`
	NumFlows              int `json:"numFlows"`
	NumFlowsStarted       int `json:"numFlowsStarted"`
	NumFlowsCompleted     int `json:"numFlowsCompleted"`
}

// FlowReportedTCPSocketStats
type FlowReportedTCPStats struct {
	Count             int    `json:"count"`
	SendCongestionWnd TCPWnd `json:"sendCongestionWnd"`
	SmoothRtt         TCPRtt `json:"smoothRtt"`
	MinRtt            TCPRtt `json:"minRtt"`
	Mss               TCPMss `json:"mss"`
	TotalRetrans      int    `json:"totalRetrans"`
	LostOut           int    `json:"lostOut"`
	UnrecoveredRTO    int    `json:"unrecoveredRTO"`
}

func (f *FlowReportedStats) Add(other FlowReportedStats) {
	f.PacketsIn += other.PacketsIn
	f.PacketsOut += other.PacketsOut
	f.BytesIn += other.BytesIn
	f.BytesOut += other.BytesOut
	f.HTTPRequestsAllowedIn += other.HTTPRequestsAllowedIn
	f.HTTPRequestsDeniedIn += other.HTTPRequestsDeniedIn
	f.NumFlows += other.NumFlows
	f.NumFlowsStarted += other.NumFlowsStarted
	f.NumFlowsCompleted += other.NumFlowsCompleted
}

func (f *FlowReportedTCPStats) Add(other FlowReportedTCPStats) {
	if f.Count == 0 {
		f.SendCongestionWnd.Min = other.SendCongestionWnd.Min
		f.SendCongestionWnd.Mean = other.SendCongestionWnd.Mean

		f.SmoothRtt.Max = other.SmoothRtt.Max
		f.SmoothRtt.Mean = other.SmoothRtt.Mean

		f.MinRtt.Max = other.MinRtt.Max
		f.MinRtt.Mean = other.MinRtt.Mean

		f.Mss.Min = other.Mss.Min
		f.Mss.Mean = other.Mss.Mean

		f.LostOut = other.LostOut
		f.TotalRetrans = other.TotalRetrans
		f.UnrecoveredRTO = other.UnrecoveredRTO
		f.Count = 1
		return

	}

	if other.SendCongestionWnd.Min < f.SendCongestionWnd.Min {
		f.SendCongestionWnd.Min = other.SendCongestionWnd.Min
	}
	f.SendCongestionWnd.Mean = ((f.SendCongestionWnd.Mean * f.Count) +
		(other.SendCongestionWnd.Mean * other.Count)) /
		(f.Count + other.Count)

	if f.SmoothRtt.Max < other.SmoothRtt.Max {
		f.SmoothRtt.Max = other.SmoothRtt.Max
	}
	f.SmoothRtt.Mean = ((f.SmoothRtt.Mean * f.Count) +
		(other.SmoothRtt.Mean * other.Count)) /
		(f.Count + other.Count)

	if f.MinRtt.Max < other.MinRtt.Max {
		f.MinRtt.Max = other.MinRtt.Max
	}
	f.MinRtt.Mean = ((f.MinRtt.Mean * f.Count) +
		(other.MinRtt.Mean * other.Count)) /
		(f.Count + other.Count)

	if other.Mss.Min < f.Mss.Min {
		f.Mss.Min = other.Mss.Min
	}
	f.Mss.Mean = ((f.Mss.Mean * f.Count) +
		(other.Mss.Mean * other.Count)) /
		(f.Count + other.Count)

	f.TotalRetrans += other.TotalRetrans
	f.LostOut += other.LostOut
	f.UnrecoveredRTO += other.UnrecoveredRTO
	f.Count += other.Count
}

// FlowStats captures stats associated with a given FlowMeta.
type FlowStats struct {
	FlowReportedStats
	FlowReportedTCPStats
	flowReferences
	processIDs  set.Set[string]
	processArgs set.Set[string]

	// Reset Process IDs  on the next metric update aggregation cycle. this ensures that we only clear
	// process ID information when we receive a new metric update.
	resetProcessIDs bool
}

func NewFlowStats(mu MetricUpdate) FlowStats {
	flowsRefs := NewTupleSet()
	flowsRefs.AddWithValue(mu.tuple, mu.natOutgoingPort)
	flowsStartedRefs := NewTupleSet()
	flowsCompletedRefs := NewTupleSet()
	flowsRefsActive := NewTupleSet()

	switch mu.updateType {
	case UpdateTypeReport:
		flowsStartedRefs.AddWithValue(mu.tuple, mu.natOutgoingPort)
		flowsRefsActive.AddWithValue(mu.tuple, mu.natOutgoingPort)
	case UpdateTypeExpire:
		flowsCompletedRefs.AddWithValue(mu.tuple, mu.natOutgoingPort)
	}

	pids := set.New[string]()
	pids.Add(strconv.Itoa(mu.processID))

	processArgs := set.New[string]()
	if mu.processArgs != "" {
		processArgs.Add(mu.processArgs)
	}

	flowStats := FlowStats{
		FlowReportedStats: FlowReportedStats{
			NumFlows:              flowsRefs.Len(),
			NumFlowsStarted:       flowsStartedRefs.Len(),
			NumFlowsCompleted:     flowsCompletedRefs.Len(),
			PacketsIn:             mu.inMetric.deltaPackets,
			BytesIn:               mu.inMetric.deltaBytes,
			PacketsOut:            mu.outMetric.deltaPackets,
			BytesOut:              mu.outMetric.deltaBytes,
			HTTPRequestsAllowedIn: mu.inMetric.deltaAllowedHTTPRequests,
			HTTPRequestsDeniedIn:  mu.inMetric.deltaDeniedHTTPRequests,
		},
		flowReferences: flowReferences{
			// flowsRefs track the flows that were tracked
			// in the give interval
			flowsRefs:          flowsRefs,
			flowsStartedRefs:   flowsStartedRefs,
			flowsCompletedRefs: flowsCompletedRefs,
			// flowsRefsActive tracks the active (non-completed)
			// flows associated with the flowMeta
			flowsRefsActive: flowsRefsActive,
		},
		processIDs:  pids,
		processArgs: processArgs,
	}
	// Here we check if the metric update has a valid TCP stats.
	// If the TCP stats is not valid (example: config is disabled),
	// it is indicated by one of sendCongestionWnd, smoothRtt, minRtt, Mss
	// being nil. Hence it is enough if we compare one of the above with nil
	if mu.sendCongestionWnd != nil {
		flowStats.FlowReportedTCPStats.SendCongestionWnd = TCPWnd{Mean: *mu.sendCongestionWnd, Min: *mu.sendCongestionWnd}
		flowStats.FlowReportedTCPStats.SmoothRtt = TCPRtt{Mean: *mu.smoothRtt, Max: *mu.smoothRtt}
		flowStats.FlowReportedTCPStats.MinRtt = TCPRtt{Mean: *mu.minRtt, Max: *mu.minRtt}
		flowStats.FlowReportedTCPStats.Mss = TCPMss{Mean: *mu.mss, Min: *mu.mss}
		flowStats.FlowReportedTCPStats.TotalRetrans = mu.tcpMetric.deltaTotalRetrans
		flowStats.FlowReportedTCPStats.LostOut = mu.tcpMetric.deltaLostOut
		flowStats.FlowReportedTCPStats.UnrecoveredRTO = mu.tcpMetric.deltaUnRecoveredRTO
		flowStats.FlowReportedTCPStats.Count = 1
	}
	return flowStats
}

func (f *FlowStats) aggregateFlowTCPStats(mu MetricUpdate, displayDebugTraceLogs bool) {
	logutil.Tracef(displayDebugTraceLogs, "Aggregrate TCP stats %+v with flow %+v", mu, f)
	// Here we check if the metric update has a valid TCP stats.
	// If the TCP stats is not valid (example: config is disabled),
	// it is indicated by one of sendCongestionWnd, smoothRtt, minRtt, Mss
	// being nil. Hence it is enough if we compare one of the above with nil
	if mu.sendCongestionWnd == nil {
		return
	}
	if f.Count == 0 {
		f.SendCongestionWnd.Min = *mu.sendCongestionWnd
		f.SendCongestionWnd.Mean = *mu.sendCongestionWnd
		f.Count = 1

		f.SmoothRtt.Max = *mu.smoothRtt
		f.SmoothRtt.Mean = *mu.smoothRtt
		f.Count = 1

		f.MinRtt.Max = *mu.minRtt
		f.MinRtt.Mean = *mu.minRtt
		f.Count = 1

		f.Mss.Min = *mu.mss
		f.Mss.Mean = *mu.mss
		f.Count = 1

		f.LostOut = mu.tcpMetric.deltaLostOut
		f.TotalRetrans = mu.tcpMetric.deltaTotalRetrans
		f.UnrecoveredRTO = mu.tcpMetric.deltaUnRecoveredRTO
		return
	}
	// Calculate Mean, Min of Send congestion window
	if *mu.sendCongestionWnd < f.SendCongestionWnd.Min {
		f.SendCongestionWnd.Min = *mu.sendCongestionWnd
	}
	f.SendCongestionWnd.Mean = ((f.SendCongestionWnd.Mean * f.Count) +
		*mu.sendCongestionWnd) / (f.Count + 1)

	// Calculate Mean, Max of Smooth Rtt
	if *mu.smoothRtt > f.SmoothRtt.Max {
		f.SmoothRtt.Max = *mu.smoothRtt
	}
	f.SmoothRtt.Mean = ((f.SmoothRtt.Mean * f.Count) +
		*mu.smoothRtt) / (f.Count + 1)

	// Calculate Mean, Max of Min Rtt
	if *mu.minRtt > f.MinRtt.Max {
		f.MinRtt.Max = *mu.minRtt
	}
	f.MinRtt.Mean = ((f.MinRtt.Mean * f.Count) +
		*mu.minRtt) / (f.Count + 1)

	// Calculate Mean,Min of MSS
	if *mu.mss < f.Mss.Min {
		f.Mss.Min = *mu.mss
	}
	f.Mss.Mean = ((f.Mss.Mean * f.Count) +
		*mu.mss) / (f.Count + 1)

	f.TotalRetrans += mu.tcpMetric.deltaTotalRetrans
	f.LostOut += mu.tcpMetric.deltaLostOut
	f.UnrecoveredRTO += mu.tcpMetric.deltaUnRecoveredRTO
	f.Count += 1
}

func (f *FlowStats) aggregateFlowStats(mu MetricUpdate, displayDebugTraceLogs bool) {
	if f.resetProcessIDs {
		// Only clear process IDs when aggregating a new metric update and after
		// a prior export.
		f.processIDs.Clear()
		f.processArgs.Clear()
		f.resetProcessIDs = false
	}
	switch {
	case mu.updateType == UpdateTypeReport:
		// Add / update the flowStartedRefs if we either haven't seen this tuple before OR the tuple is already in the
		// flowStartRefs (we may have an updated value).
		if !f.flowsRefsActive.Contains(mu.tuple) || f.flowsStartedRefs.Contains(mu.tuple) {
			f.flowsStartedRefs.AddWithValue(mu.tuple, mu.natOutgoingPort)
		}

		f.flowsRefsActive.AddWithValue(mu.tuple, mu.natOutgoingPort)
	case mu.updateType == UpdateTypeExpire:
		f.flowsCompletedRefs.AddWithValue(mu.tuple, mu.natOutgoingPort)
		f.flowsRefsActive.Discard(mu.tuple)
	}
	f.flowsRefs.AddWithValue(mu.tuple, mu.natOutgoingPort)
	f.processIDs.Add(strconv.Itoa(mu.processID))
	if mu.processArgs != "" {
		f.processArgs.Add(mu.processArgs)
	}

	f.NumFlows = f.flowsRefs.Len()
	f.NumFlowsStarted = f.flowsStartedRefs.Len()
	f.NumFlowsCompleted = f.flowsCompletedRefs.Len()
	f.PacketsIn += mu.inMetric.deltaPackets
	f.BytesIn += mu.inMetric.deltaBytes
	f.PacketsOut += mu.outMetric.deltaPackets
	f.BytesOut += mu.outMetric.deltaBytes
	f.HTTPRequestsAllowedIn += mu.inMetric.deltaAllowedHTTPRequests
	f.HTTPRequestsDeniedIn += mu.inMetric.deltaDeniedHTTPRequests
	f.aggregateFlowTCPStats(mu, displayDebugTraceLogs)
}

func (f *FlowStats) getActiveFlowsCount() int {
	return len(f.flowsRefsActive)
}

func (f *FlowStats) containsActiveRefs(mu MetricUpdate) bool {
	return f.flowsRefsActive.Contains(mu.tuple)
}

func (f *FlowStats) reset() {
	f.flowsStartedRefs = NewTupleSet()
	f.flowsCompletedRefs = NewTupleSet()
	f.flowsRefs = f.flowsRefsActive.Copy()
	f.FlowReportedStats = FlowReportedStats{
		NumFlows: f.flowsRefs.Len(),
	}
	f.FlowReportedTCPStats = FlowReportedTCPStats{}
	// Signal that the process ID information should be reset prior to
	// aggregating.
	f.resetProcessIDs = true
}

// FlowStatsByProcess collects statistics organized by process names. When process information is not enabled
// this stores the stats in a single entry keyed by a "-".
// Flow logs should be constructed by calling toFlowProcessReportedStats and then flattening the resulting
// slice with FlowMeta and other FlowLog information such as policies and labels.
type FlowStatsByProcess struct {

	// statsByProcessName stores aggregated flow statistics grouped by a process name.
	statsByProcessName map[string]*FlowStats
	// processNames stores the order in which process information is tracked and aggrgated.
	// this is done so that when we export flow logs, we do so in the order they appeared.
	processNames          *list.List
	displayDebugTraceLogs bool
	includeProcess        bool
	processLimit          int
	processArgsLimit      int
	natOutgoingPortLimit  int
	// TODO(doublek): Track the most significant stats and show them as part
	// of the flows that are included in the process limit. Current processNames
	// only tracks insertion order.
}

func NewFlowStatsByProcess(mu *MetricUpdate, includeProcess bool, processLimit, processArgsLimit int,
	displayDebugTraceLogs bool, natOutgoingPortLimit int) FlowStatsByProcess {

	f := FlowStatsByProcess{
		displayDebugTraceLogs: displayDebugTraceLogs,
		statsByProcessName:    make(map[string]*FlowStats),
		processNames:          list.New(),
		includeProcess:        includeProcess,
		processLimit:          processLimit,
		processArgsLimit:      processArgsLimit,
		natOutgoingPortLimit:  natOutgoingPortLimit,
	}
	f.aggregateFlowStatsByProcess(mu)
	return f
}

func (f *FlowStatsByProcess) aggregateFlowStatsByProcess(mu *MetricUpdate) {
	if !f.includeProcess || mu.processName == "" {
		mu.processName = flowLogFieldNotIncluded
		mu.processID = 0
		mu.processArgs = flowLogFieldNotIncluded
	}
	if stats, ok := f.statsByProcessName[mu.processName]; ok {
		logutil.Tracef(f.displayDebugTraceLogs, "Process stats found %+v for metric update %+v", stats, mu)
		stats.aggregateFlowStats(*mu, f.displayDebugTraceLogs)
		logutil.Tracef(f.displayDebugTraceLogs, "Aggregated stats %+v after processing metric update %+v", stats, mu)
		f.statsByProcessName[mu.processName] = stats
	} else {
		logutil.Tracef(f.displayDebugTraceLogs, "Process stats not found for metric update %+v", mu)
		f.processNames.PushBack(mu.processName)
		stats := NewFlowStats(*mu)
		f.statsByProcessName[mu.processName] = &stats
	}
}

func (f *FlowStatsByProcess) getActiveFlowsCount() int {
	activeCount := 0
	for _, stats := range f.statsByProcessName {
		activeCount += stats.getActiveFlowsCount()
	}
	return activeCount
}

func (f *FlowStatsByProcess) containsActiveRefs(mu *MetricUpdate) bool {
	if !f.includeProcess || mu.processName == "" {
		mu.processName = flowLogFieldNotIncluded
	}
	if stats, ok := f.statsByProcessName[mu.processName]; ok {
		return stats.flowsRefsActive.Contains(mu.tuple)
	}
	return false
}

func (f *FlowStatsByProcess) reset() {
	for name, stats := range f.statsByProcessName {
		stats.reset()
		f.statsByProcessName[name] = stats
	}
}

// gc garbage collects any process names and corresponding stats that don't have any active flows.
// This should only be called after stats have been reported.
func (f *FlowStatsByProcess) gc() int {
	var next *list.Element
	remainingActiveFlowsCount := 0
	for e := f.processNames.Front(); e != nil; e = next {
		// Don't lose where we are since we may in-place delete
		// the element.
		next = e.Next()
		name := e.Value.(string)
		stats := f.statsByProcessName[name]
		afc := stats.getActiveFlowsCount()
		if afc == 0 {
			delete(f.statsByProcessName, name)
			f.processNames.Remove(e)
			continue
		}
		remainingActiveFlowsCount += afc
	}
	return remainingActiveFlowsCount
}

// toFlowProcessReportedStats returns atmost processLimit + 1 entry slice containing
// flow stats grouped by process information.
func (f *FlowStatsByProcess) toFlowProcessReportedStats() []FlowProcessReportedStats {
	var pArgs []string
	if !f.includeProcess {
		// If we are not configured to include process information then
		// we expect to only have a single entry with no process information
		// and all stats are already aggregated into a single value.
		reportedStats := make([]FlowProcessReportedStats, 0, 1)
		if stats, ok := f.statsByProcessName[flowLogFieldNotIncluded]; ok {
			s := FlowProcessReportedStats{
				ProcessName:          flowLogFieldNotIncluded,
				NumProcessNames:      0,
				ProcessID:            flowLogFieldNotIncluded,
				NumProcessIDs:        0,
				ProcessArgs:          []string{"-"},
				NumProcessArgs:       0,
				FlowReportedStats:    stats.FlowReportedStats,
				FlowReportedTCPStats: stats.FlowReportedTCPStats,
				NatOutgoingPorts:     f.getNatOutGoingPortsFromStats(stats),
			}
			reportedStats = append(reportedStats, s)
		} else {
			log.Warnf("No flow log status recorded %+v", f)
		}
		return reportedStats
	}

	// Only collect up to process limit stats and one additional entry for rest
	// of the aggregated stats.
	reportedStats := make([]FlowProcessReportedStats, 0, f.processLimit+1)
	numProcessNames := 0
	numPids := 0
	numProcessArgs := 0
	appendAggregatedStats := false
	aggregatedReportedStats := FlowReportedStats{}
	aggregatedReportedTCPStats := FlowReportedTCPStats{}
	var aggregatedNatOutgoingPorts []int
	var next *list.Element
	// Collect in insertion order, the first processLimit entries and then aggregate
	// the remaining statistics in a single aggregated FlowProcessReportedStats entry.
	for e := f.processNames.Front(); e != nil; e = next {
		next = e.Next()
		name := e.Value.(string)
		stats, ok := f.statsByProcessName[name]
		if !ok {
			log.Warnf("Stats not found for process name %v", name)
			f.processNames.Remove(e)
			continue
		}

		natOutGoingPorts := f.getNatOutGoingPortsFromStats(stats)

		// If we didn't receive any process data then the flow stats are
		// aggregated under a "-" which is flowLogFieldNotIncluded. All these
		// This is handled separately here so that we can set numProcessNames
		// and numProcessIDs to 0.
		if name == flowLogFieldNotIncluded {
			s := FlowProcessReportedStats{
				ProcessName:          flowLogFieldNotIncluded,
				NumProcessNames:      0,
				ProcessID:            flowLogFieldNotIncluded,
				NumProcessIDs:        0,
				ProcessArgs:          []string{"-"},
				NumProcessArgs:       0,
				FlowReportedStats:    stats.FlowReportedStats,
				FlowReportedTCPStats: stats.FlowReportedTCPStats,
				NatOutgoingPorts:     natOutGoingPorts,
			}
			reportedStats = append(reportedStats, s)
			// Continue processing in case there are other tuples that did collect
			// process information.
			continue
		}

		// Figure out how PIDs are to be included in a flow log entry.
		// If there are no PIDs then the pid is not included.
		// If there is a singe PID then the pid is added.
		// If there are multiple PIDs for a single process name then
		// the pid field is set to "*" (aggregated) and numProcessIDs is
		// set to the number of PIDs for this process name.
		numPids = stats.processIDs.Len()
		var pid string
		if numPids == 0 {
			pid = flowLogFieldNotIncluded
		} else if numPids == 1 {
			// Get the first and only PID.
			stats.processIDs.Iter(func(p string) error {
				pid = p
				return set.StopIteration
			})
		} else {
			pid = flowLogFieldAggregated
		}

		argList := func(numAllowedArgs int, stats *FlowStats) []string {
			var aList []string
			var tempStr string
			numProcessArgs = stats.processArgs.Len()
			if numProcessArgs == 0 {
				return []string{"-"}
			}
			if numPids == 1 {
				// This is a corner case. Logically there should be a
				// single argument if the numPids is 1. There could be more
				// when aggregating, reason being 1 flow has args from kprobes
				// and other flow has args read from /proc/pid/cmdline. In this
				// we just show a single arg which is longest, with numProcessArgs
				// set to 1.
				stats.processArgs.Iter(func(item string) error {
					if len(item) > len(tempStr) {
						tempStr = item
					}
					return nil
				})
				numProcessArgs = 1
				return []string{tempStr}
			}
			if numProcessArgs == 1 || numAllowedArgs == 1 {
				stats.processArgs.Iter(func(item string) error {
					aList = append(aList, item)
					return set.StopIteration
				})
			} else {
				argCount := 0
				stats.processArgs.Iter(func(item string) error {
					aList = append(aList, item)
					argCount = argCount + 1
					if argCount == numAllowedArgs {
						return set.StopIteration
					}
					return nil
				})
			}
			return aList
		}

		pArgs = argList(f.processArgsLimit, stats)
		// If we've reached the process limit, then start aggregating the remaining
		// stats so that we can add one additional entry containing this information.
		if len(reportedStats) == f.processLimit {
			numProcessNames++
			numPids += numPids
			numProcessArgs += numProcessArgs
			aggregatedReportedStats.Add(stats.FlowReportedStats)
			appendAggregatedStats = true
			aggregatedReportedTCPStats.Add(stats.FlowReportedTCPStats)

			spaceInNatOutGoingPortArray := f.natOutgoingPortLimit - len(aggregatedNatOutgoingPorts)
			if spaceInNatOutGoingPortArray > 0 {
				numIncludedPorts := len(natOutGoingPorts)
				if spaceInNatOutGoingPortArray < len(natOutGoingPorts) {
					numIncludedPorts = spaceInNatOutGoingPortArray
				}
				aggregatedNatOutgoingPorts = append(aggregatedNatOutgoingPorts, natOutGoingPorts[0:numIncludedPorts]...)
			}
		} else {
			s := FlowProcessReportedStats{
				ProcessName:          name,
				NumProcessNames:      1,
				ProcessID:            pid,
				NumProcessIDs:        numPids,
				ProcessArgs:          pArgs,
				NumProcessArgs:       numProcessArgs,
				FlowReportedStats:    stats.FlowReportedStats,
				FlowReportedTCPStats: stats.FlowReportedTCPStats,
				NatOutgoingPorts:     natOutGoingPorts,
			}
			reportedStats = append(reportedStats, s)
		}
	}
	if appendAggregatedStats {
		s := FlowProcessReportedStats{
			ProcessName:          flowLogFieldAggregated,
			NumProcessNames:      numProcessNames,
			ProcessID:            flowLogFieldAggregated,
			NumProcessIDs:        numPids,
			ProcessArgs:          pArgs,
			NumProcessArgs:       numProcessArgs,
			FlowReportedStats:    aggregatedReportedStats,
			FlowReportedTCPStats: aggregatedReportedTCPStats,
		}
		reportedStats = append(reportedStats, s)
	}
	return reportedStats
}

func (f *FlowStatsByProcess) getNatOutGoingPortsFromStats(stats *FlowStats) []int {
	var natOutGoingPorts []int

	numNatOutgoingPorts := 0
	for _, value := range stats.flowsRefsActive {
		if numNatOutgoingPorts >= f.natOutgoingPortLimit {
			break
		}

		if value != 0 {
			natOutGoingPorts = append(natOutGoingPorts, value)
			numNatOutgoingPorts++
		}
	}

	for _, value := range stats.flowsCompletedRefs {
		if numNatOutgoingPorts >= f.natOutgoingPortLimit {
			break
		}

		if value != 0 {
			natOutGoingPorts = append(natOutGoingPorts, value)
			numNatOutgoingPorts++
		}
	}

	return natOutGoingPorts
}

// FlowProcessReportedStats contains FlowReportedStats along with process information.
type FlowProcessReportedStats struct {
	ProcessName      string   `json:"processName"`
	NumProcessNames  int      `json:"numProcessNames"`
	ProcessID        string   `json:"processID"`
	NumProcessIDs    int      `json:"numProcessIDs"`
	ProcessArgs      []string `json:"processArgs"`
	NumProcessArgs   int      `json:"numProcessArgs"`
	NatOutgoingPorts []int
	FlowReportedStats
	FlowReportedTCPStats
}

// FlowLog is a record of flow data (metadata & reported stats) including
// timestamps. A FlowLog is ready to be serialized to an output format.
type FlowLog struct {
	StartTime, EndTime time.Time
	FlowMeta
	FlowLabels
	FlowPolicies
	FlowDestDomains
	FlowExtras
	FlowProcessReportedStats
}

func (f *FlowLog) Deserialize(fl string) error {
	// Format is
	// startTime endTime srcType srcNamespace srcName srcLabels dstType dstNamespace dstName dstLabels srcIP dstIP proto srcPort dstPort numFlows numFlowsStarted numFlowsCompleted flowReporter packetsIn packetsOut bytesIn bytesOut action policies originalSourceIPs numOriginalSourceIPs destServiceNamespace dstServiceName dstServicePort processName numProcessNames processPid numProcessIds
	// Sample entry with no aggregation and no labels.
	// 1529529591 1529529892 wep policy-demo nginx-7d98456675-2mcs4 nginx-7d98456675-* - wep kube-system kube-dns-7cc87d595-pxvxb kube-dns-7cc87d595-* - 192.168.224.225 192.168.135.53 17 36486 53 1 1 1 in 1 1 73 119 allow ["0|tier|namespace/tier.policy|allow|0"] [1.0.0.1] 1 kube-system kube-dns dig 23033 0

	var (
		srcType, dstType FlowLogEndpointType
	)

	parts := strings.Split(fl, " ")
	if len(parts) < 32 {
		return fmt.Errorf("log %v cant be processed", fl)
	}

	switch parts[2] {
	case "wep":
		srcType = FlowLogEndpointTypeWep
	case "hep":
		srcType = FlowLogEndpointTypeHep
	case "ns":
		srcType = FlowLogEndpointTypeNs
	case "net":
		srcType = FlowLogEndpointTypeNet
	}

	f.SrcMeta = EndpointMetadata{
		Type:           srcType,
		Namespace:      parts[3],
		Name:           parts[4],
		AggregatedName: parts[5],
	}
	f.SrcLabels = stringToLabels(parts[6])
	if srcType == FlowLogEndpointTypeNs {
		namespace, name := extractNamespaceFromNetworkSet(f.SrcMeta.AggregatedName)
		f.SrcMeta.Namespace = namespace
		f.SrcMeta.AggregatedName = name
	}

	switch parts[7] {
	case "wep":
		dstType = FlowLogEndpointTypeWep
	case "hep":
		dstType = FlowLogEndpointTypeHep
	case "ns":
		dstType = FlowLogEndpointTypeNs
	case "net":
		dstType = FlowLogEndpointTypeNet
	}

	f.DstMeta = EndpointMetadata{
		Type:           dstType,
		Namespace:      parts[8],
		Name:           parts[9],
		AggregatedName: parts[10],
	}
	f.DstLabels = stringToLabels(parts[11])
	if dstType == FlowLogEndpointTypeNs {
		namespace, name := extractNamespaceFromNetworkSet(f.DstMeta.AggregatedName)
		f.DstMeta.Namespace = namespace
		f.DstMeta.AggregatedName = name
	}

	var sip, dip [16]byte
	if parts[12] != "-" {
		sip = ipStrTo16Byte(parts[12])
	}
	if parts[13] != "-" {
		dip = ipStrTo16Byte(parts[13])
	}
	p, _ := strconv.Atoi(parts[14])
	sp, _ := strconv.Atoi(parts[15])
	dp, _ := strconv.Atoi(parts[16])
	f.Tuple = *NewTuple(sip, dip, p, sp, dp)

	f.NumFlows, _ = strconv.Atoi(parts[17])
	f.NumFlowsStarted, _ = strconv.Atoi(parts[18])
	f.NumFlowsCompleted, _ = strconv.Atoi(parts[19])

	switch parts[20] {
	case "src":
		f.Reporter = FlowLogReporterSrc
	case "dst":
		f.Reporter = FlowLogReporterDst
	}

	f.PacketsIn, _ = strconv.Atoi(parts[21])
	f.PacketsOut, _ = strconv.Atoi(parts[22])
	f.BytesIn, _ = strconv.Atoi(parts[23])
	f.BytesOut, _ = strconv.Atoi(parts[24])

	switch parts[25] {
	case "allow":
		f.Action = FlowLogActionAllow
	case "deny":
		f.Action = FlowLogActionDeny
	}

	// Parse policies, empty ones are just -
	if parts[26] == "-" {
		f.FlowPolicies = make(FlowPolicies)
	} else if len(parts[26]) > 1 {
		f.FlowPolicies = make(FlowPolicies)
		polParts := strings.Split(parts[26][1:len(parts[26])-1], ",")
		for _, p := range polParts {
			f.FlowPolicies[p] = emptyValue
		}
	}

	// Parse original source IPs, empty ones are just -
	if parts[27] == "-" {
		f.FlowExtras = FlowExtras{}
	} else if len(parts[27]) > 1 {
		ips := []net.IP{}
		exParts := strings.Split(parts[27][1:len(parts[27])-1], ",")
		for _, ipStr := range exParts {
			ip := net.ParseIP(ipStr)
			if ip == nil {
				continue
			}
			ips = append(ips, ip)
		}
		f.FlowExtras = FlowExtras{
			OriginalSourceIPs: ips,
		}
		f.FlowExtras.NumOriginalSourceIPs, _ = strconv.Atoi(parts[28])
	}

	svcPortNum, err := strconv.Atoi(parts[32])
	if err != nil {
		svcPortNum = 0
	}

	f.DstService = FlowService{
		Namespace: parts[29],
		Name:      parts[30],
		PortName:  parts[31],
		PortNum:   svcPortNum,
	}

	f.ProcessName = parts[33]
	f.NumProcessNames, _ = strconv.Atoi(parts[34])
	f.ProcessID = parts[35]
	f.NumProcessIDs, _ = strconv.Atoi(parts[36])
	temp, _ := strconv.Atoi(parts[37])
	f.SendCongestionWnd.Mean = temp
	temp, _ = strconv.Atoi(parts[38])
	f.SendCongestionWnd.Min = temp
	temp, _ = strconv.Atoi(parts[39])
	f.SmoothRtt.Mean = temp
	temp, _ = strconv.Atoi(parts[40])
	f.SmoothRtt.Max = temp
	temp, _ = strconv.Atoi(parts[41])
	f.MinRtt.Mean = temp
	temp, _ = strconv.Atoi(parts[42])
	f.MinRtt.Max = temp
	temp, _ = strconv.Atoi(parts[43])
	f.Mss.Mean = temp
	temp, _ = strconv.Atoi(parts[44])
	f.Mss.Min = temp
	temp, _ = strconv.Atoi(parts[45])
	f.TotalRetrans = temp
	temp, _ = strconv.Atoi(parts[46])
	f.LostOut = temp
	temp, _ = strconv.Atoi(parts[47])
	f.UnrecoveredRTO = temp

	return nil
}
