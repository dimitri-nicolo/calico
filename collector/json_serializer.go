// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

import (
	"net"
	"strconv"
)

var protoNames = map[int]string{
	1:  "icmp",
	6:  "tcp",
	17: "udp",
	4:  "ipip",
	50: "esp",
	58: "icmp6",
}

// FlowLogJSONOutput represents the JSON representation of a flow log.
type FlowLogJSONOutput struct {
	StartTime int64 `json:"start_time"`
	EndTime   int64 `json:"end_time"`

	SourceIP        string `json:"source_ip"`
	SourceName      string `json:"source_name"`
	SourceNamespace string `json:"source_namespace"`
	SourcePort      int64  `json:"source_port"`
	SourceType      string `json:"source_type"`
	SourceLabels    string `json:"source_labels,omitempty"`
	DestIP          string `json:"dest_ip"`
	DestName        string `json:"dest_name"`
	DestNamespace   string `json:"dest_namespace"`
	DestPort        int64  `json:"dest_port"`
	DestType        string `json:"dest_type"`
	DestLabels      string `json:"dest_labels,omitempty"`
	Proto           string `json:"proto"`

	Action   string `json:"action"`
	Reporter string `json:"reporter"`

	BytesIn           int64 `json:"bytes_in"`
	BytesOut          int64 `json:"bytes_out"`
	NumFlows          int64 `json:"num_flows"`
	NumFlowsStarted   int64 `json:"num_flows_started"`
	NumFlowsCompleted int64 `json:"num_flows_completed"`
	PacketsIn         int64 `json:"packets_in"`
	PacketsOut        int64 `json:"packets_out"`
}

func toOutput(l *FlowLog) FlowLogJSONOutput {
	var out FlowLogJSONOutput

	out.StartTime = l.StartTime.Unix()
	out.EndTime = l.EndTime.Unix()

	ip := net.IP(l.Tuple.src[:16])
	if !ip.IsUnspecified() {
		out.SourceIP = ip.String()
	}
	if l.Tuple.l4Src != unsetIntField {
		out.SourcePort = int64(l.Tuple.l4Src)
	}
	out.SourceName = l.SrcMeta.Name
	out.SourceNamespace = l.SrcMeta.Namespace
	out.SourceType = string(l.SrcMeta.Type)
	out.SourceLabels = l.SrcMeta.Labels

	ip = net.IP(l.Tuple.dst[:16])
	if !ip.IsUnspecified() {
		out.DestIP = ip.String()
	}
	if l.Tuple.l4Dst != unsetIntField {
		out.DestPort = int64(l.Tuple.l4Dst)
	}
	out.DestName = l.DstMeta.Name
	out.DestNamespace = l.DstMeta.Namespace
	out.DestType = string(l.DstMeta.Type)
	out.DestLabels = l.DstMeta.Labels

	out.Proto = protoToString(l.Tuple.proto)

	out.Action = string(l.Action)
	out.Reporter = string(l.Reporter)

	out.BytesIn = int64(l.BytesIn)
	out.BytesOut = int64(l.BytesOut)
	out.PacketsIn = int64(l.PacketsIn)
	out.PacketsOut = int64(l.PacketsOut)
	out.NumFlows = int64(l.NumFlows)
	out.NumFlowsCompleted = int64(l.NumFlowsCompleted)
	out.NumFlowsStarted = int64(l.NumFlowsStarted)
	return out
}

func protoToString(p int) string {
	s, ok := protoNames[p]
	if ok {
		return s
	}
	return strconv.Itoa(p)
}
