// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

type FlowLogAction string
type FlowLogDirection string
type FlowLogEndpointType string

const (
	flowLogBufferSize = 1000

	flowLogNamespaceGlobal  = "_G_"
	flowLogFieldNotIncluded = "-"

	FlowLogActionAllow FlowLogAction = "A"
	FlowLogActionDeny  FlowLogAction = "D"

	FlowLogDirectionIn  FlowLogDirection = "I"
	FlowLogDirectionOut FlowLogDirection = "O"

	FlowLogEndpointTypeWep FlowLogEndpointType = "wep"
	FlowLogEndpointTypeHep FlowLogEndpointType = "hep"
	FlowLogEndpointTypeNs  FlowLogEndpointType = "ns"
	FlowLogEndpointTypePvt FlowLogEndpointType = "pvt"
	FlowLogEndpointTypePub FlowLogEndpointType = "pub"
)

type EndpointMetadata struct {
	Type      FlowLogEndpointType `json:"type,"`
	Namespace string              `json:"namespace,"`
	Name      string              `json:"name,"`
	Labels    map[string]string   `json:"labels,"`
}

// FlowLog captures the context and information associated with
// 5-tuple update.
type FlowLog struct {
	Tuple             Tuple            `json:"srcType"`
	SrcMeta           EndpointMetadata `json:"srcMeta"`
	DstMeta           EndpointMetadata `json:"dstMeta"`
	NumFlows          int              `json:"numFlows"`
	NumFlowsStarted   int              `json:"numFlowsStarted"`
	NumFlowsCompleted int              `json:"numFlowsCompleted"`
	PacketsIn         int              `json:"packetsIn"`
	PacketsOut        int              `json:"packetsOut"`
	BytesIn           int              `json:"bytesIn"`
	BytesOut          int              `json:"bytesOut"`
	Action            FlowLogAction    `json:"action"`
	FlowDirection     FlowLogDirection `json:"flowDirection"`
}

func (f FlowLog) ToString(startTime, endTime time.Time, includeLabels bool) string {
	var (
		srcLabels, dstLabels string
		l4Src, l4Dst         string
	)

	if includeLabels {
		sl, err := json.Marshal(f.SrcMeta.Labels)
		if err == nil {
			srcLabels = string(sl)
		}
		dl, err := json.Marshal(f.DstMeta.Labels)
		if err == nil {
			dstLabels = string(dl)
		}
	} else {
		srcLabels = flowLogFieldNotIncluded
		dstLabels = flowLogFieldNotIncluded
	}

	srcIP := net.IP(f.Tuple.src[:16]).String()
	dstIP := net.IP(f.Tuple.dst[:16]).String()

	if f.Tuple.proto != 1 {
		l4Src = fmt.Sprintf("%d", f.Tuple.l4Src)
		l4Dst = fmt.Sprintf("%d", f.Tuple.l4Dst)
	} else {
		l4Src = flowLogFieldNotIncluded
		l4Dst = flowLogFieldNotIncluded
	}

	// Format is
	// startTime endTime srcType srcNamespace srcName srcLabels dstType dstNamespace dstName dstLabels srcIP dstIP proto srcPort dstPort numFlows numFlowsStarted numFlowsCompleted flowDirection packetsIn packetsOut bytesIn bytesOut action
	return fmt.Sprintf("%v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v",
		startTime.Unix(), endTime.Unix(),
		f.SrcMeta.Type, f.SrcMeta.Namespace, f.SrcMeta.Name, srcLabels,
		f.DstMeta.Type, f.DstMeta.Namespace, f.DstMeta.Name, dstLabels,
		srcIP, dstIP, f.Tuple.proto, l4Src, l4Dst,
		f.NumFlows, f.NumFlowsStarted, f.NumFlowsCompleted, f.FlowDirection,
		f.PacketsIn, f.PacketsOut, f.BytesIn, f.BytesOut,
		f.Action)
}

func (f *FlowLog) aggregateMetricUpdate(mu MetricUpdate) error {
	// TODO(doublek): Handle metadata updates.
	var nfc int
	if mu.updateType == UpdateTypeExpire {
		nfc = 1
	}
	f.NumFlowsCompleted += nfc
	f.PacketsIn += mu.inMetric.deltaPackets
	f.BytesIn += mu.inMetric.deltaBytes
	f.PacketsOut += mu.outMetric.deltaPackets
	f.BytesOut += mu.outMetric.deltaBytes
	return nil
}
