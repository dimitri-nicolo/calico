// Copyright 2019 Tigera Inc. All rights reserved.

package events

// From github.com/tigera/felix-private/convert/json_serializer.go

// FlowLogJSONOutput represents the JSON representation of a flow log.
type FlowLogJSONOutput struct {
	StartTime int64 `json:"start_time"`
	EndTime   int64 `json:"end_time"`

	// Some empty values should be json marshalled as null and NOT with golang null values such as "" for
	// a empty string
	// Having such values as pointers ensures that json marshalling will render it as such.
	SourceIP        *string                  `json:"source_ip"`
	SourceName      string                   `json:"source_name"`
	SourceNameAggr  string                   `json:"source_name_aggr"`
	SourceNamespace string                   `json:"source_namespace"`
	SourcePort      *int64                   `json:"source_port"`
	SourceType      string                   `json:"source_type"`
	SourceLabels    *FlowLogLabelsJSONOutput `json:"source_labels"`
	DestIP          *string                  `json:"dest_ip"`
	DestName        string                   `json:"dest_name"`
	DestNameAggr    string                   `json:"dest_name_aggr"`
	DestNamespace   string                   `json:"dest_namespace"`
	DestPort        *int64                   `json:"dest_port"`
	DestType        string                   `json:"dest_type"`
	DestLabels      *FlowLogLabelsJSONOutput `json:"dest_labels"`
	Proto           string                   `json:"proto"`

	Action   string `json:"action"`
	Reporter string `json:"reporter"`

	Policies *FlowLogPoliciesJSONOutput `json:"policies"`

	BytesIn               int64 `json:"bytes_in"`
	BytesOut              int64 `json:"bytes_out"`
	NumFlows              int64 `json:"num_flows"`
	NumFlowsStarted       int64 `json:"num_flows_started"`
	NumFlowsCompleted     int64 `json:"num_flows_completed"`
	PacketsIn             int64 `json:"packets_in"`
	PacketsOut            int64 `json:"packets_out"`
	HTTPRequestsAllowedIn int64 `json:"http_requests_allowed_in"`
	HTTPRequestsDeniedIn  int64 `json:"http_requests_denied_in"`
}

type FlowLogLabelsJSONOutput struct {
	Labels []string `json:"labels"`
}

type FlowLogPoliciesJSONOutput struct {
	AllPolicies []string `json:"all_policies"`
}
