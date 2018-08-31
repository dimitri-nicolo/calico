// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

import (
	"encoding/json"
	"net"
	"strconv"
	"time"
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

	// Some empty values should be json marshalled as null and NOT with golang null values such as "" for
	// a empty string
	// Having such values as pointers ensures that json marshalling will render it as such.
	SourceIP        *string `json:"source_ip"`
	SourceName      string  `json:"source_name"`
	SourceNamespace string  `json:"source_namespace"`
	SourcePort      *int64  `json:"source_port"`
	SourceType      string  `json:"source_type"`
	SourceLabels    string  `json:"source_labels,omitempty"`
	DestIP          *string `json:"dest_ip"`
	DestName        string  `json:"dest_name"`
	DestNamespace   string  `json:"dest_namespace"`
	DestPort        int64   `json:"dest_port"`
	DestType        string  `json:"dest_type"`
	DestLabels      string  `json:"dest_labels,omitempty"`
	Proto           string  `json:"proto"`

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
		s := ip.String()
		out.SourceIP = &s
	}
	if l.Tuple.l4Src != unsetIntField {
		t := int64(l.Tuple.l4Src)
		out.SourcePort = &t
	}
	out.SourceName = l.SrcMeta.Name
	out.SourceNamespace = l.SrcMeta.Namespace
	out.SourceType = string(l.SrcMeta.Type)
	srcLabels, _ := json.Marshal(l.SrcLabels)
	out.SourceLabels = string(srcLabels)

	ip = net.IP(l.Tuple.dst[:16])
	if !ip.IsUnspecified() {
		s := ip.String()
		out.DestIP = &s
	}
	if l.Tuple.l4Dst != unsetIntField {
		out.DestPort = int64(l.Tuple.l4Dst)
	}
	out.DestName = l.DstMeta.Name
	out.DestNamespace = l.DstMeta.Namespace
	out.DestType = string(l.DstMeta.Type)
	dstLabels, _ := json.Marshal(l.DstLabels)
	out.DestLabels = string(dstLabels)

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

func stringToProto(s string) int {
	for i, st := range protoNames {
		if s == st {
			return i
		}
	}
	p, _ := strconv.Atoi(s)
	return p
}

func (o FlowLogJSONOutput) ToFlowLog() (FlowLog, error) {
	fl := FlowLog{}
	fl.StartTime = time.Unix(o.StartTime, 0)
	fl.EndTime = time.Unix(o.EndTime, 0)

	var sip, dip [16]byte
	if o.SourceIP != nil && *o.SourceIP != "" {
		sip = ipStrTo16Byte(*o.SourceIP)
	}
	if o.DestIP != nil && *o.DestIP != "" {
		dip = ipStrTo16Byte(*o.DestIP)
	}
	p := stringToProto(o.Proto)
	var sPort int
	if o.SourcePort != nil {
		sPort = int(*o.SourcePort)
	}
	fl.Tuple = *NewTuple(sip, dip, p, sPort, int(o.DestPort))

	var srcType, dstType FlowLogEndpointType
	switch o.SourceType {
	case "wep":
		srcType = FlowLogEndpointTypeWep
	case "hep":
		srcType = FlowLogEndpointTypeHep
	case "ns":
		srcType = FlowLogEndpointTypeNs
	case "net":
		srcType = FlowLogEndpointTypeNet
	}

	fl.SrcMeta = EndpointMetadata{
		Type:      srcType,
		Namespace: o.SourceNamespace,
		Name:      o.SourceName,
	}

	switch o.DestType {
	case "wep":
		dstType = FlowLogEndpointTypeWep
	case "hep":
		dstType = FlowLogEndpointTypeHep
	case "ns":
		dstType = FlowLogEndpointTypeNs
	case "net":
		dstType = FlowLogEndpointTypeNet
	}

	fl.DstMeta = EndpointMetadata{
		Type:      dstType,
		Namespace: o.DestNamespace,
		Name:      o.DestName,
	}

	fl.Action = FlowLogAction(o.Action)
	fl.Reporter = FlowLogReporter(o.Reporter)
	fl.BytesIn = int(o.BytesIn)
	fl.BytesOut = int(o.BytesOut)
	fl.PacketsIn = int(o.PacketsIn)
	fl.PacketsOut = int(o.PacketsOut)
	fl.NumFlows = int(o.NumFlows)
	fl.NumFlowsStarted = int(o.NumFlowsStarted)
	fl.NumFlowsCompleted = int(o.NumFlowsCompleted)

	return fl, nil
}
