// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package v1

import (
	"net"
)

// FlowLogParams define querying parameters to retrieve flow logs
type FlowLogParams struct {
	QueryParams        `json:",inline" validate:"required"`
	LogSelectionParams `json:",inline"`
}

// FlowLog is the input format to ingest flow logs
// Some empty values should be json marshalled as null and NOT with golang null values such as "" for
// an empty string
// Having such values as pointers ensures that json marshalling will render it as such.
type FlowLog struct {
	// Destination fields.
	DestType             string         `json:"dest_type"`
	DestIP               *string        `json:"dest_ip"`
	DestName             string         `json:"dest_name"`
	DestNamespace        string         `json:"dest_namespace"`
	DestNameAggr         string         `json:"dest_name_aggr"`
	DestPort             *int64         `json:"dest_port"`
	DestLabels           *FlowLogLabels `json:"dest_labels"`
	DestServiceNamespace string         `json:"dest_service_namespace"`
	DestServiceName      string         `json:"dest_service_name"`
	DestServicePortName  string         `json:"dest_service_port"`
	DestServicePortNum   *int64         `json:"dest_service_port_num"`
	DestDomains          []string       `json:"dest_domains"`

	// Source fields.
	SourceType       string         `json:"source_type"`
	SourceIP         *string        `json:"source_ip"`
	SourceName       string         `json:"source_name"`
	SourceNamespace  string         `json:"source_namespace"`
	SourceNameAggr   string         `json:"source_name_aggr"`
	SourcePort       *int64         `json:"source_port"`
	SourceLabels     *FlowLogLabels `json:"source_labels"`
	OrigSourceIPs    []net.IP       `json:"original_source_ips"`
	NumOrigSourceIPs int64          `json:"num_original_source_ips"`

	// Reporter is src or dest - the location where this flowlog was generated.
	Reporter         string         `json:"reporter"`
	Protocol         string         `json:"proto"`
	Action           string         `json:"action"`
	NatOutgoingPorts []int          `json:"nat_outgoing_ports"`
	Policies         *FlowLogPolicy `json:"policies"`

	// HTTP fields.
	HTTPRequestsAllowedIn int64 `json:"http_requests_allowed_in"`
	HTTPRequestsDeniedIn  int64 `json:"http_requests_denied_in"`

	// Traffic stats.
	PacketsIn  int64 `json:"packets_in"`
	PacketsOut int64 `json:"packets_out"`
	BytesIn    int64 `json:"bytes_in"`
	BytesOut   int64 `json:"bytes_out"`

	// Stats from the original logs used to generate this flow log.
	// Felix aggregates multiple flow logs into a single FlowLog.
	NumFlows          int64 `json:"num_flows"`
	NumFlowsStarted   int64 `json:"num_flows_started"`
	NumFlowsCompleted int64 `json:"num_flows_completed"`

	// Process stats.
	NumProcessNames int64    `json:"num_process_names"`
	NumProcessIDs   int64    `json:"num_process_ids"`
	ProcessName     string   `json:"process_name"`
	NumProcessArgs  int64    `json:"num_process_args"`
	ProcessArgs     []string `json:"process_args"`
	ProcessID       string   `json:"process_id"`

	// TCP stats.
	TCPMinSendCongestionWindow  int64 `json:"tcp_min_send_congestion_window"`
	TCPMinMSS                   int64 `json:"tcp_min_mss"`
	TCPMaxSmoothRTT             int64 `json:"tcp_max_smooth_rtt"`
	TCPMaxMinRTT                int64 `json:"tcp_max_min_rtt"`
	TCPMeanSendCongestionWindow int64 `json:"tcp_mean_send_congestion_window"`
	TCPMeanMSS                  int64 `json:"tcp_mean_mss"`
	TCPMeanMinRTT               int64 `json:"tcp_mean_min_rtt"`
	TCPMeanSmoothRTT            int64 `json:"tcp_mean_smooth_rtt"`
	TCPTotalRetransmissions     int64 `json:"tcp_total_retransmissions"`
	TCPLostPackets              int64 `json:"tcp_lost_packets"`
	TCPUnrecoveredTo            int64 `json:"tcp_unrecovered_to"`

	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
	Timestamp int64  `json:"@timestamp"`
	Host      string `json:"host"`
}

type FlowLogPolicy struct {
	AllPolicies []string `json:"all_policies"`
}

type FlowLogLabels struct {
	Labels []string `json:"labels"`
}
