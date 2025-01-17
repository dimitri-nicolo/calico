// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package v1

type BGPLogIPVersion string

const (
	IPv4BGPLog BGPLogIPVersion = "IPv4"
	IPv6BGPLog BGPLogIPVersion = "IPv6"
)

// BGPLogTimeFormat is the expected format to use for LogTime on BGP logs.
// For golang, e.g., LogTime: time.Format(v1.BGPLogTimeFormat)
const BGPLogTimeFormat = "2006-01-02T15:04:05"

type BGPLog struct {
	LogTime   string          `json:"logtime"`
	Message   string          `json:"message"`
	Host      string          `json:"host"`
	IPVersion BGPLogIPVersion `json:"ip_version"`

	// Cluster is populated by linseed from the request context.
	Cluster string `json:"cluster,omitempty"`
}

// BGPLogParams define querying parameters to retrieve BGP logs
type BGPLogParams struct {
	QueryParams `json:",inline" validate:"required"`
}
