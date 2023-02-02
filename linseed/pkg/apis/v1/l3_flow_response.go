// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package v1

// L3FlowKey represents the identifiers for an L3 Flow.
type L3FlowKey struct {
	// Common fields
	Action   string `json:"action"`
	Reporter string `json:"reporter"`
	Protocol string `json:"protocol"`

	// Source and destination information.
	Source      Endpoint `json:"source"`
	Destination Endpoint `json:"destination"`
}

type Results[T any] struct {
	Items    []T         `json:"items"`
	AfterKey interface{} `json:"after_key"`
}

// L3Flow represents a summary of connection and traffic information between two
// endpoints over a given period of time, as reported by one of said endpoints.
type L3Flow struct {
	// Key contains the identifying information for this L3 Flow.
	Key L3FlowKey `json:"key"`

	Process *Process `json:"process,omitempty"`
	Service *Service `json:"dest_service,omitempty"`

	// Policies applied to this flow.
	Policies []string `json:"policies,omitempty"`

	// DestinationLabels are the labels applied to the destination during the lifetime
	// of this flow. Note that a single label may have had multiple values throughout this flow's life.
	DestinationLabels []FlowLabels `json:"destination_labels,omitempty"`

	// SourceLabels are the labels applied to the source during the lifetime
	// of this flow. Note that a single label may have had multiple values throughout this flow's life.
	SourceLabels []FlowLabels `json:"source_labels,omitempty"`

	// TrafficStats contains summarized traffic stats for this flow.
	TrafficStats *TrafficStats `json:"connection_stats,omitempty"`

	// TCPStatsStats are aggregated TCP metrics generated from the traffic described by the L3 flow
	TCPStats *TCPStats `json:"tcp_stats,omitempty"`

	// LogStats are aggregated metrics about the underlying flow logs used to generate this flow.
	LogStats *LogStats `json:"log_stats,omitempty"`

	// ProcessStats are process aggregated metrics generated from the traffic described by the L3 flows.
	ProcessStats *ProcessStats `json:"process_stats,omitempty"`
}

// FlowLabels represents a single label and all of its seen values over the course of
// a flow's life.
type FlowLabels struct {
	Key    string   `json:"key"`
	Values []string `json:"values"`
}

// LogStats represent the number of flows aggregated into this entry
type LogStats struct {
	// LogCount is the total number of raw flow logs - prior to client-side aggregation - that were
	// aggregated into this entry. This is in contrast to FlowLogCount, which is the number of
	// flow log entries from Elasticsearch used to generate this flow.
	LogCount int64 `json:"count"`

	// FlowLogCount is the number of flow logs in Elasticsearch used to generate this flow.
	FlowLogCount int64 `json:"flowLogCount"`

	// Completed is the number of flow logs that finished and aggregated into during this entry.
	Completed int64 `json:"completed"`

	// Started is the number of flow logs that started and aggregated into during this entry.
	Started int64 `json:"started"`
}

// ProcessStats represent the number of processes aggregated into this entry
type ProcessStats struct {
	MinNumNamesPerFlow int `json:"min_num_names_per_flow"`
	MaxNumNamesPerFlow int `json:"max_num_names_per_flow"`
	MinNumIDsPerFlow   int `json:"min_num_ids_per_flow"`
	MaxNumIDsPerFlow   int `json:"max_num_ids_per_flow"`
}

// TrafficStats represent L3 metrics aggregated from flows into this entry
type TrafficStats struct {
	// PacketsIn is the total number of incoming packets aggregated into this entry
	PacketsIn int64 `json:"packets_in"`

	// PacketsOut is the total number of outgoing packets aggregated into this entry
	PacketsOut int64 `json:"packets_out"`

	// BytesIn is the total number of incoming packets aggregated into this entry
	BytesIn int64 `json:"bytes_in"`

	// BytesOut is the total number of outgoing packets aggregated into this entry
	BytesOut int64 `json:"bytes_out"`
}

// TCPStats represent TCP metrics aggregated from flows into this entry
type TCPStats struct {
	// LostPackets is the total number of lost TCP packets aggregated into this entry
	LostPackets int64 `json:"lost_packets"`

	// MaxMinRTT is the maximum value of the lower Round Trip Time for TCP packets aggregated into this entry
	MaxMinRTT float64 `json:"max_min_rtt"`

	// MaxSmoothRTT is the maximum value of the Smoothed Round Trip Time for TCP packets aggregated into this entry
	MaxSmoothRTT float64 `json:"max_smooth_rtt"`

	// MeanMinRTT is the mean value of the lower Round Trip Time for TCP packets aggregated into this entry
	MeanMinRTT float64 `json:"mean_min_rtt"`

	// MeanMSS is the mean value of the Maximum Segment Size for a TCP packet aggregated into this entry
	MeanMSS float64 `json:"mean_mss"`

	// MeanSendCongestionWindow is the mean value of SendCongestionWindow for TCP packets aggregated into this entry
	MeanSendCongestionWindow float64 `json:"mean_send_congestion_window"`

	// MeanSmoothRTT is the mean value of the Smoothed Round Trip Time for TCP packets aggregated into this entry
	MeanSmoothRTT float64 `json:"mean_smooth_rtt"`

	// MinMSS is the min value of the Maximum Segment Size for a TCP packet aggregated into this entry
	MinMSS float64 `json:"min_mss"`

	// MinSendCongestionWindow is the min value of SendCongestionWindow for TCP packets aggregated into this entry
	MinSendCongestionWindow float64 `json:"min_send_congestion_window"`

	// UnrecoveredTo
	UnrecoveredTo int64 `json:"unrecovered_to"`

	// TotalRetransmissions is the total number of retransmitted TCP packets that were lost
	TotalRetransmissions int64 `json:"total_retransmissions"`
}

type Process struct {
	Name string `json:"name"`
}

type Service struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Port      int32  `json:"port"`
	PortName  string `json:"port_name"`
}

type L3FlowResponse struct {
	L3Flows  []L3Flow    `json:"l3_flows,omitempty" validate:"omitempty"`
	AfterKey interface{} `json:"after_key"`
}
