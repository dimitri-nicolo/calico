// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package v1

type L3Flow struct {
	Source      Endpoint `json:"source"`
	Destination Endpoint `json:"destination"`
	Actions     []string `json:"actions"`
	Reporter    string   `json:"reporter"`
	Protocol    string   `json:"protocol,omitempty"`
	Process     *Process `json:"process,omitempty"`
	Service     *Service `json:"dest_service,omitempty"`

	Policies          []string        `json:"policies,omitempty"`
	DestinationLabels []LabelSelector `json:"destination_labels,omitempty"`
	SourceLabels      []LabelSelector `json:"source_labels,omitempty"`

	// ConnectionStats are aggregated metrics generated from the traffic described by the L3 flow
	ConnectionStats *ConnectionStats `json:"connection_stats,omitempty"`

	// TCPStatsStats are aggregated TCP metrics generated from the traffic described by the L3 flow
	TCPStats *TCPStats `json:"tcp_stats,omitempty"`

	// Stats are aggregated metrics generated from the traffic described by the L3 flows
	Stats *Stats `json:"stats,omitempty"`

	// ProcessStats are process aggregated metrics generated from the traffic described by the L3 flows
	ProcessStats *ProcessStats `json:"process_stats,omitempty"`
}

// Stats represent the number of flows aggregated into this entry
type Stats struct {
	// Count is the number of flows aggregated into this entry.
	Count int64 `json:"count"`

	// Completed is the number of flows that finished and aggregated into during this entry.
	Completed int64 `json:"completed"`

	// Started is the number of flows that started and aggregated into during this entry.
	Started int64 `json:"started"`
}

// ProcessStats represent the number of processes aggregated into this entry
type ProcessStats struct {
	// NamesCount is the total process names aggregated into this entry
	NamesCount int64 `json:"names_count"`

	// IdsCount is the total number of ids for a process aggregated into this entry
	IdsCount int64 `json:"ids_count"`
}

// ConnectionStats represent L3 metrics aggregated from flows into this entry
type ConnectionStats struct {
	//PacketsIn is the total number of incoming packets aggregated into this entry
	PacketsIn int64 `json:"packets_in"`

	//PacketsOut is the total number of outgoing packets aggregated into this entry
	PacketsOut int64 `json:"packets_out"`

	//BytesIn is the total number of incoming packets aggregated into this entry
	BytesIn int64 `json:"bytes_in"`

	//BytesOut is the total number of outgoing packets aggregated into this entry
	BytesOut int64 `json:"bytes_out"`
}

// TCPStats represent TCP metrics aggregated from flows into this entry
type TCPStats struct {
	// LostPackets is the total number of lost TCP packets aggregated into this entry
	LostPackets int64 `json:"lost_packets"`

	// MaxMinRTT is the maximum value of the lower Round Trip Time for TCP packets aggregated into this entry
	MaxMinRTT int64 `json:"max_min_rtt"`

	// MaxSmoothRTT is the maximum value of the Smoothed Round Trip Time for TCP packets aggregated into this entry
	MaxSmoothRTT int64 `json:"max_smooth_rtt"`

	// MeanMinRTT is the mean value of the lower Round Trip Time for TCP packets aggregated into this entry
	MeanMinRTT int64 `json:"mean_min_rtt"`

	// MeanMSS is the mean value of the Maximum Segment Size for a TCP packet aggregated into this entry
	MeanMSS int64 `json:"mean_mss"`

	// MeanSendCongestionWindow is the mean value of SendCongestionWindow for TCP packets aggregated into this entry
	MeanSendCongestionWindow int64 `json:"mean_send_congestion_window"`

	// MeanSmoothRTT is the mean value of the Smoothed Round Trip Time for TCP packets aggregated into this entry
	MeanSmoothRTT int64 `json:"mean_smooth_rtt"`

	// MinMSS is the min value of the Maximum Segment Size for a TCP packet aggregated into this entry
	MinMSS int64 `json:"min_mss"`

	// MinSendCongestionWindow is the min value of SendCongestionWindow for TCP packets aggregated into this entry
	MinSendCongestionWindow int64 `json:"min_send_congestion_window"`

	// TotalRetransmissions is the total number of retransmitted TCP packets that were lost
	UnRecoveredTo int64 `json:"unrecovered_to"`

	// TotalRetransmissions is the total number of retransmitted TCP packets that were lost
	TotalRetransmissions int64 `json:"total_retransmissions"`
}

type Process struct {
	Name string `json:"name"`
}

type Service struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Port      string `json:"port"`
}
