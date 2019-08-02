// Copyright (c) 2018-2019 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	unsetIntField = -1
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

type FlowMeta struct {
	Tuple    Tuple            `json:"tuple"`
	SrcMeta  EndpointMetadata `json:"sourceMeta"`
	DstMeta  EndpointMetadata `json:"destinationMeta"`
	Action   FlowLogAction    `json:"action"`
	Reporter FlowLogReporter  `json:"flowReporter"`
}

func newFlowMeta(mu MetricUpdate) (FlowMeta, error) {
	f := FlowMeta{}

	// Extract Tuple Info
	f.Tuple = mu.tuple

	// Extract EndpointMetadata info
	srcMeta, err := getFlowLogEndpointMetadata(mu.srcEp, mu.tuple.src)
	if err != nil {
		return FlowMeta{}, fmt.Errorf("Could not extract metadata for source %v", mu.srcEp)
	}
	dstMeta, err := getFlowLogEndpointMetadata(mu.dstEp, mu.tuple.dst)
	if err != nil {
		return FlowMeta{}, fmt.Errorf("Could not extract metadata for destination %v", mu.dstEp)
	}

	f.SrcMeta = srcMeta
	f.DstMeta = dstMeta

	lastRuleID := mu.GetLastRuleID()
	if lastRuleID == nil {
		log.WithField("metric update", mu).Error("no rule id present")
		return f, fmt.Errorf("Invalid metric update")
	}

	action, direction := getFlowLogActionAndReporterFromRuleID(lastRuleID)
	f.Action = action
	f.Reporter = direction

	return f, nil
}

func newFlowMetaWithSourcePortAggregation(mu MetricUpdate) (FlowMeta, error) {
	f, err := newFlowMeta(mu)
	if err != nil {
		return FlowMeta{}, err
	}
	f.Tuple.l4Src = unsetIntField

	return f, nil
}

func newFlowMetaWithPrefixNameAggregation(mu MetricUpdate) (FlowMeta, error) {
	f, err := newFlowMeta(mu)
	if err != nil {
		return FlowMeta{}, err
	}

	f.Tuple.src = [16]byte{}
	f.Tuple.l4Src = unsetIntField
	f.Tuple.dst = [16]byte{}
	f.SrcMeta.Name = flowLogFieldNotIncluded
	f.DstMeta.Name = flowLogFieldNotIncluded

	return f, nil
}

func NewFlowMeta(mu MetricUpdate, kind FlowAggregationKind) (FlowMeta, error) {
	switch kind {
	case FlowDefault:
		return newFlowMeta(mu)
	case FlowSourcePort:
		return newFlowMetaWithSourcePortAggregation(mu)
	case FlowPrefixName:
		return newFlowMetaWithPrefixNameAggregation(mu)
	}

	return FlowMeta{}, fmt.Errorf("aggregation kind %v not recognized", kind)
}

type FlowSpec struct {
	FlowLabels
	FlowPolicies
	FlowStats
	flowExtrasRef
}

func NewFlowSpec(mu MetricUpdate, maxOriginalIPsSize int) FlowSpec {
	return FlowSpec{
		FlowLabels:    NewFlowLabels(mu),
		FlowPolicies:  NewFlowPolicies(mu),
		FlowStats:     NewFlowStats(mu),
		flowExtrasRef: NewFlowExtrasRef(mu, maxOriginalIPsSize),
	}
}

func (f *FlowSpec) aggregateMetricUpdate(mu MetricUpdate) {
	f.aggregateFlowLabels(mu)
	f.FlowPolicies.aggregateMetricUpdate(mu)
	f.aggregateFlowStats(mu)
	f.aggregateFlowExtrasRef(mu)
}

// FlowSpec has FlowStats that are stats assocated with a given FlowMeta
// These stats are to be refreshed everytime the FlowData
// {FlowMeta->FlowStats} is published so as to account
// for correct no. of started flows in a given aggregation
// interval.
func (f FlowSpec) reset() FlowSpec {
	f.flowsStartedRefs = NewTupleSet()
	f.flowsCompletedRefs = NewTupleSet()
	f.flowsRefs = f.flowsRefsActive.Copy()
	f.FlowReportedStats = FlowReportedStats{
		NumFlows: f.flowsRefs.Len(),
	}
	f.flowExtrasRef.reset()
	return f
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

	f.SrcLabels = intersectLabels(srcLabels, f.SrcLabels)
	f.DstLabels = intersectLabels(dstLabels, f.DstLabels)
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
		fp[fmt.Sprintf("%d|%s", idx, rid.GetFlowLogPolicyName())] = emptyValue
	}
	return fp
}

func (fp FlowPolicies) aggregateMetricUpdate(mu MetricUpdate) {
	if mu.ruleIDs == nil {
		return
	}
	for idx, rid := range mu.ruleIDs {
		if rid == nil {
			continue
		}
		fp[fmt.Sprintf("%d|%s", idx, rid.GetFlowLogPolicyName())] = emptyValue
	}
}

// FlowStats captures stats associated with a given FlowMeta
type FlowStats struct {
	FlowReportedStats
	flowReferences
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

func NewFlowStats(mu MetricUpdate) FlowStats {
	flowsRefs := NewTupleSet()
	flowsRefs.Add(mu.tuple)
	flowsStartedRefs := NewTupleSet()
	flowsCompletedRefs := NewTupleSet()
	flowsRefsActive := NewTupleSet()

	switch mu.updateType {
	case UpdateTypeReport:
		flowsStartedRefs.Add(mu.tuple)
		flowsRefsActive.Add(mu.tuple)
	case UpdateTypeExpire:
		flowsCompletedRefs.Add(mu.tuple)
	}

	return FlowStats{
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
	}
}

func (f *FlowStats) aggregateFlowStats(mu MetricUpdate) {
	switch {
	case mu.updateType == UpdateTypeReport && !f.flowsRefsActive.Contains(mu.tuple):
		f.flowsStartedRefs.Add(mu.tuple)
		f.flowsRefsActive.Add(mu.tuple)
	case mu.updateType == UpdateTypeExpire:
		f.flowsCompletedRefs.Add(mu.tuple)
		f.flowsRefsActive.Discard(mu.tuple)
	}
	f.flowsRefs.Add(mu.tuple)
	f.NumFlows = f.flowsRefs.Len()
	f.NumFlowsStarted = f.flowsStartedRefs.Len()
	f.NumFlowsCompleted = f.flowsCompletedRefs.Len()
	f.PacketsIn += mu.inMetric.deltaPackets
	f.BytesIn += mu.inMetric.deltaBytes
	f.PacketsOut += mu.outMetric.deltaPackets
	f.BytesOut += mu.outMetric.deltaBytes
	f.HTTPRequestsAllowedIn += mu.inMetric.deltaAllowedHTTPRequests
	f.HTTPRequestsDeniedIn += mu.inMetric.deltaDeniedHTTPRequests
}

func (f FlowStats) getActiveFlowsCount() int {
	return len(f.flowsRefsActive)
}

// FlowData is metadata and stats about a flow (or aggregated group of flows).
// This is an internal structure for book keeping; FlowLog is what actually gets
// passed to dispatchers or serialized.
type FlowData struct {
	FlowMeta
	FlowSpec
}

// FlowLog is a record of flow data (metadata & reported stats) including
// timestamps. A FlowLog is ready to be serialized to an output format.
type FlowLog struct {
	StartTime, EndTime time.Time
	FlowMeta
	FlowLabels
	FlowPolicies
	FlowReportedStats
	FlowExtras
}

// ToFlowLog converts a FlowData to a FlowLog
func (f FlowData) ToFlowLog(startTime, endTime time.Time, includeLabels bool, includePolicies bool) FlowLog {
	var fl FlowLog
	fl.FlowMeta = f.FlowMeta
	fl.FlowReportedStats = f.FlowReportedStats
	fl.StartTime = startTime
	fl.EndTime = endTime
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

	return fl
}

func (f *FlowLog) Deserialize(fl string) error {
	// Format is
	// startTime endTime srcType srcNamespace srcName srcLabels dstType dstNamespace dstName dstLabels srcIP dstIP proto srcPort dstPort numFlows numFlowsStarted numFlowsCompleted flowReporter packetsIn packetsOut bytesIn bytesOut action policies
	// Sample entry with no aggregation and no labels.
	// 1529529591 1529529892 wep policy-demo nginx-7d98456675-2mcs4 nginx-7d98456675-* - wep kube-system kube-dns-7cc87d595-pxvxb kube-dns-7cc87d595-* - 192.168.224.225 192.168.135.53 17 36486 53 1 1 1 in 1 1 73 119 allow ["0|tier|namespace/tier.policy|allow"] [1.0.0.1] 1

	var (
		srcType, dstType FlowLogEndpointType
	)

	parts := strings.Split(fl, " ")
	if len(parts) < 24 {
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

	return nil
}
