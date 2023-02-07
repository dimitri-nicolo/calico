// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package v1

type BGPLogIPVersion string

const (
	IPv4BGPLog BGPLogIPVersion = "IPv4"
	IPv6BGPLog BGPLogIPVersion = "IPv6"
)

// BGPLogTimeFormat is the expected format to use for LogTime on BGP logs.
// For golang, e.g., LogTime: time.Format(v1.BGPLogTimeFormat)
const BGPLogTimeFormat = "2006-01-02T03:04:05"

type BGPLog struct {
	LogTime   string          `json:"logtime"`
	Message   string          `json:"message"`
	IPVersion BGPLogIPVersion `json:"ip_version"`
	Host      string          `json:"host"`
}

// BGPLogParams define querying parameters to retrieve BGP logs
type BGPLogParams struct {
	QueryParams *QueryParams `json:"query_params" validate:"required"`
}
