// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package v1

import "net"

type FlowLog struct {
	// Destination fields.
	DestType             string        `json:"dest_type"`
	DestIP               net.IP        `json:"dest_ip"`
	DestNamespace        string        `json:"dest_namespace"`
	DestNameAggr         string        `json:"dest_name_aggr"`
	DestPort             int           `json:"dest_port"`
	DestLabels           FlowLogLabels `json:"dest_labels"`
	DestServiceNamespace string        `json:"dest_service_namespace"`
	DestServiceName      string        `json:"dest_service_name"`
	DestServicePort      string        `json:"dest_service_port"` // Deprecated
	DestServicePortNum   int           `json:"dest_service_port_num"`
	DestDomains          string        `json:"dest_domains"`

	// Source fields.
	SourceType           string        `json:"source_type"`
	SourceIP             net.IP        `json:"source_ip"`
	SourceNamespace      string        `json:"source_namespace"`
	SourceNameAggr       string        `json:"source_name_aggr"`
	SourcePort           int           `json:"source_port"`
	SourceLabels         FlowLogLabels `json:"source_labels"`
	OriginalSourceIPs    net.IP        `json:"original_source_i_ps"`
	NumOriginalSourceIPs int           `json:"num_original_source_ips"`

	// Reporter is src or dest - the location where this flowlog was generated.
	Reporter         string          `json:"reporter"`
	Protocol         string          `json:"proto"`
	Action           string          `json:"action"`
	NATOutgoingPorts int             `json:"nat_outgoing_ports"`
	Policies         []FlowLogPolicy `json:"policies"`

	// HTTP fields.
	HTTPRequestsAllowedIn int `json:"http_requests_allowed_in"`
	HTTPRequestsDeniedIn  int `json:"http_requests_denied_in"`

	// Traffic stats.
	PacketsIn  int `json:"packets_in"`
	PacketsOut int `json:"packets_out"`
	BytesIn    int `json:"bytes_in"`
	BytesOut   int `json:"bytes_out"`

	// Stats from the original logs used to generate this flow log.
	// Felix aggregates multiple flow logs into a single FlowLog.
	NumFlows          int `json:"num_flows"`
	NumFlowsStarted   int `json:"num_flows_started"`
	NumFlowsCompleted int `json:"num_flows_completed"`

	// Process stats.
	NumProcessNames int    `json:"num_process_names"`
	NumProcessIDs   int    `json:"num_process_ids"`
	ProcessName     string `json:"process_name"`
	NumProcessArgs  int    `json:"num_process_args"`
	ProcessArgs     string `json:"process_args"`

	// TCP stats.
	TCPMinSendCongestionWindow  int `json:"tcp_min_send_congestion_window"`
	TCPMinMSS                   int `json:"tcp_min_mss"`
	TCPMaxSmoothRTT             int `json:"tcp_max_smooth_rtt"`
	TCPMaxMinRTT                int `json:"tcp_max_min_rtt"`
	TCPMeanSendCongestionWindow int `json:"tcp_mean_send_congestion_window"`
	TCPMeanMSS                  int `json:"tcp_mean_mss"`
	TCPMeanMinRTT               int `json:"tcp_mean_min_rtt"`
	TCPMeanSmoothRTT            int `json:"tcp_mean_smooth_rtt"`
	TCPTotalRetransmissions     int `json:"tcp_total_retransmissions"`
	TCPLostPackets              int `json:"tcp_lost_packets"`
	TCPUnrecoveredTo            int `json:"tcp_unrecovered_to"`

	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type FlowLogPolicy struct {
	AllPolicies string `json:"all_policies"`
}

type FlowLogLabels struct {
	Labels []string `json:"labels"`
}

type BulkError struct {
	Message string `json:"message"`
}

type BulkResponse struct {
	Total     int `json:"total"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`

	Errors []BulkError `json:"errors,omitempty"`
}
