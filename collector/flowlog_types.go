// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	unsetIntField = 0
)

type FlowLogEndpointType string
type FlowLogAction string
type FlowLogDirection string

type EndpointMetadata struct {
	Type      FlowLogEndpointType `json:"type"`
	Namespace string              `json:"namespace"`
	Name      string              `json:"name"`
	Labels    string              `json:"labels"`
}

type FlowMeta struct {
	Tuple     Tuple            `json:"tuple"`
	SrcMeta   EndpointMetadata `json:"sourceMeta"`
	DstMeta   EndpointMetadata `json:"destinationMeta"`
	Action    FlowLogAction    `json:"action"`
	Direction FlowLogDirection `json:"flowDirection"`
}

func newFlowMeta(mu MetricUpdate) (FlowMeta, error) {
	f := FlowMeta{}

	// Extract Tuple Info
	f.Tuple = mu.tuple

	// Extract EndpointMetadata info
	var (
		srcMeta, dstMeta EndpointMetadata
		err              error
	)
	if mu.srcEp != nil {
		srcMeta, err = getFlowLogEndpointMetadata(mu.srcEp)
		if err != nil {
			return FlowMeta{}, fmt.Errorf("Could not extract metadata for source %v", mu.srcEp)
		}
	}
	if mu.dstEp != nil {
		dstMeta, err = getFlowLogEndpointMetadata(mu.dstEp)
		if err != nil {
			return FlowMeta{}, fmt.Errorf("Could not extract metadata for destination %v", mu.dstEp)
		}
	}

	f.SrcMeta = srcMeta
	f.DstMeta = dstMeta

	action, direction := getFlowLogActionAndDirFromRuleID(mu.ruleID)
	f.Action = action
	f.Direction = direction

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

	f.SrcMeta.Name = getEndpointGenerateName(mu.srcEp)
	f.DstMeta.Name = getEndpointGenerateName(mu.dstEp)

	return f, nil
}

func NewFlowMeta(mu MetricUpdate, kind AggregationKind) (FlowMeta, error) {
	switch kind {
	case Default:
		return newFlowMeta(mu)
	case SourcePort:
		return newFlowMetaWithSourcePortAggregation(mu)
	case PrefixName:
		return newFlowMetaWithPrefixNameAggregation(mu)
	}

	return FlowMeta{}, fmt.Errorf("aggregation kind %s not recognized", kind)
}

// FlowStats captures the context and information associated with
// 5-tuple update.
type FlowStats struct {
	PacketsIn          int `json:"packetsIn"`
	PacketsOut         int `json:"packetsOut"`
	BytesIn            int `json:"bytesIn"`
	BytesOut           int `json:"bytesOut"`
	NumFlows           int `json:"numFlows"`
	NumFlowsStarted    int `json:"numFlowsStarted"`
	NumFlowsCompleted  int `json:"numFlowsCompleted"`
	flowsRefs          tupleSet
	flowsStartedRefs   tupleSet
	flowsCompletedRefs tupleSet
}

func NewFlowStats(mu MetricUpdate) FlowStats {
	flowsRefs := NewTupleSet()
	flowsRefs.Add(mu.tuple)
	flowsStartedRefs := NewTupleSet()
	flowsCompletedRefs := NewTupleSet()

	switch mu.updateType {
	case UpdateTypeReport:
		flowsStartedRefs.Add(mu.tuple)
	case UpdateTypeExpire:
		flowsCompletedRefs.Add(mu.tuple)
	}

	return FlowStats{
		NumFlows:           flowsRefs.Len(),
		NumFlowsStarted:    flowsStartedRefs.Len(),
		NumFlowsCompleted:  flowsCompletedRefs.Len(),
		PacketsIn:          mu.inMetric.deltaPackets,
		BytesIn:            mu.inMetric.deltaBytes,
		PacketsOut:         mu.outMetric.deltaPackets,
		BytesOut:           mu.outMetric.deltaBytes,
		flowsRefs:          flowsRefs,
		flowsStartedRefs:   flowsStartedRefs,
		flowsCompletedRefs: flowsCompletedRefs,
	}
}

func (f *FlowStats) aggregateMetricUpdate(mu MetricUpdate) {
	// TODO(doublek): Handle metadata updates.
	switch {
	case mu.updateType == UpdateTypeReport && !f.flowsStartedRefs.Contains(mu.tuple):
		f.flowsStartedRefs.Add(mu.tuple)
	case mu.updateType == UpdateTypeExpire && !f.flowsCompletedRefs.Contains(mu.tuple):
		f.flowsCompletedRefs.Add(mu.tuple)
	}
	// If this is the first time we are seeing this tuple.
	if !f.flowsRefs.Contains(mu.tuple) || (mu.updateType == UpdateTypeReport && f.flowsCompletedRefs.Contains(mu.tuple)) {
		f.flowsRefs.Add(mu.tuple)
	}
	f.NumFlows = f.flowsRefs.Len()
	f.NumFlowsStarted = f.flowsStartedRefs.Len()
	f.NumFlowsCompleted = f.flowsCompletedRefs.Len()
	f.PacketsIn += mu.inMetric.deltaPackets
	f.BytesIn += mu.inMetric.deltaBytes
	f.PacketsOut += mu.outMetric.deltaPackets
	f.BytesOut += mu.outMetric.deltaBytes
}

type FlowLog struct {
	fm FlowMeta
	fs FlowStats
}

func (f FlowLog) Serialize(includeLabels bool) string {
	var srcLabels, dstLabels string
	fm := f.fm
	fs := f.fs

	if includeLabels {
		srcLabels = fm.SrcMeta.Labels
		dstLabels = fm.DstMeta.Labels
	} else {
		srcLabels = flowLogFieldNotIncluded
		dstLabels = flowLogFieldNotIncluded
	}

	srcIP, dstIP, proto, l4Src, l4Dst := extractPartsFromAggregatedTuple(fm.Tuple)

	return fmt.Sprintf("%v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v",
		fm.SrcMeta.Type, fm.SrcMeta.Namespace, fm.SrcMeta.Name, srcLabels,
		fm.DstMeta.Type, fm.DstMeta.Namespace, fm.DstMeta.Name, dstLabels,
		srcIP, dstIP, proto, l4Src, l4Dst,
		fs.NumFlows, fs.NumFlowsStarted, fs.NumFlowsCompleted, fm.Direction,
		fs.PacketsIn, fs.PacketsOut, fs.BytesIn, fs.BytesOut,
		fm.Action)
}

func (f *FlowLog) Deserialize(fl string) error {
	// Format is
	// startTime endTime srcType srcNamespace srcName srcLabels dstType dstNamespace dstName dstLabels srcIP dstIP proto srcPort dstPort numFlows numFlowsStarted numFlowsCompleted flowDirection packetsIn packetsOut bytesIn bytesOut action
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
	case "pvt":
		srcType = FlowLogEndpointTypePvt
	case "pub":
		srcType = FlowLogEndpointTypePub
	}

	srcMeta := EndpointMetadata{
		Type:      srcType,
		Namespace: parts[3],
		Name:      parts[4],
		Labels:    parts[5],
	}
	if strings.Compare(parts[5], "-") == 0 {
		srcMeta.Labels = ""
	}

	switch parts[6] {
	case "wep":
		dstType = FlowLogEndpointTypeWep
	case "hep":
		dstType = FlowLogEndpointTypeHep
	case "ns":
		dstType = FlowLogEndpointTypeNs
	case "pvt":
		dstType = FlowLogEndpointTypePvt
	case "pub":
		dstType = FlowLogEndpointTypePub
	}

	dstMeta := EndpointMetadata{
		Type:      dstType,
		Namespace: parts[7],
		Name:      parts[8],
		Labels:    parts[9],
	}
	if strings.Compare(parts[9], "-") == 0 {
		dstMeta.Labels = ""
	}

	var sip [16]byte
	if parts[10] != "-" {
		sip = ipStrTo16Byte(parts[10])
	}
	dip := ipStrTo16Byte(parts[11])
	p, _ := strconv.Atoi(parts[12])
	sp, _ := strconv.Atoi(parts[13])
	dp, _ := strconv.Atoi(parts[14])
	tuple := *NewTuple(sip, dip, p, sp, dp)

	nf, _ := strconv.Atoi(parts[15])
	nfs, _ := strconv.Atoi(parts[16])
	nfc, _ := strconv.Atoi(parts[17])

	var fd FlowLogDirection
	switch parts[18] {
	case "I":
		fd = FlowLogDirectionIn
	case "O":
		fd = FlowLogDirectionOut
	}

	pi, _ := strconv.Atoi(parts[19])
	po, _ := strconv.Atoi(parts[20])
	bi, _ := strconv.Atoi(parts[21])
	bo, _ := strconv.Atoi(parts[22])

	var a FlowLogAction
	switch parts[23] {
	case "A":
		a = FlowLogActionAllow
	case "D":
		a = FlowLogActionDeny
	}

	*f = FlowLog{
		FlowMeta{
			Tuple:     tuple,
			SrcMeta:   srcMeta,
			DstMeta:   dstMeta,
			Action:    a,
			Direction: fd,
		},
		FlowStats{
			NumFlows:          nf,
			NumFlowsStarted:   nfs,
			NumFlowsCompleted: nfc,
			PacketsIn:         pi,
			PacketsOut:        po,
			BytesIn:           bi,
			BytesOut:          bo,
		},
	}

	return nil
}
