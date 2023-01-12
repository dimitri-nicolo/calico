// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package v1

// Types of stats available for querying on an L3 flow.
type StatsType string

const (
	StatsTypeTraffic StatsType = "traffic"
	StatsTypeTCP     StatsType = "tcp"
	StatsTypeFlowLog StatsType = "flow"
	StatsTypeProcess StatsType = "process"
)

// Types of endpoints present in log data.
type EndpointType string

const (
	WEP        EndpointType = "wep"
	HEP        EndpointType = "hep"
	Network    EndpointType = "net"
	NetworkSet EndpointType = "ns"
)

// L3FlowParams define querying parameters to retrieve L3 Flows
type L3FlowParams struct {
	// Source will filter L3 flows generated by the desired endpoint
	// If no Source and Destination are present, all L3 Flows will be
	// returned
	Source *Endpoint `json:"source" validate:"omitempty"`

	// Destination will filter L3 flows that target the desired endpoint
	// If no Source and Destination are present, all L3 Flows will be
	// returned
	Destination *Endpoint `json:"destination" validate:"omitempty"`

	// Statistics will include different metrics for the L3 flows that are queried
	// The following metrics can be extracted: connection, tcp, flow and process
	// If missing, only flow metrics will be generated
	Statistics []StatsType `json:"statistics" validate:"omitempty,dive,oneof=tcp connection flow process"`

	// QueryParams are general query parameters for flows, such as:
	// - filter and aggregate flows within a desired time range
	// - allow users to specify a desired time that the request should run
	// until cancelling the execution
	QueryParams *QueryParams `json:"query_params" validate:"required"`

	// Limit the maximum number of returned results.
	MaxResults int
}

type Endpoint struct {
	Type           EndpointType    `json:"type" validate:"omitempty,oneof=wep hep net ns"`
	Name           string          `json:"name" validate:"omitempty"`
	AggregatedName string          `json:"aggregated_name" validate:"omitempty"`
	Namespace      string          `json:"namespace" validate:"omitempty"`
	Port           int64           `json:"port" validate:"omitempty"`
	Labels         []LabelSelector `json:"labels" validate:"omitempty"`
}

type LabelSelector struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values"`
}
