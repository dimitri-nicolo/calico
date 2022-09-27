// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package v1

import (
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
)

type ApplicationRequest struct {
	// The cluster name. Defaults to "cluster".
	ClusterName string `json:"cluster" validate:"omitempty"`

	// Selector defines a query string for raw logs. [Default: empty]
	Selector string `json:"selector" validate:"omitempty"`

	// Time range.
	TimeRange *lmav1.TimeRange `json:"time_range" validate:"required"`
}

type ServiceResponse struct {
	// Services contains a list of service info.
	Services []Service `json:"services" validate:"required"`
}

type Service struct {
	// Name of the service.
	Name string `json:"name" validate:"required"`

	// ErrorRate calculates the percentage of 400-599 HTTP response code of the service.
	ErrorRate float64 `json:"errorRate" validate:"required"`

	// Latency of the service in microseconds.
	Latency float64 `json:"latency" validate:"required"`

	// InboundThroughput of the service in bytes per second.
	InboundThroughput float64 `json:"inboundThroughput" validate:"required"`

	// OutboundThroughput of the service in bytes per second.
	OutboundThroughput float64 `json:"outboundThroughput" validate:"required"`

	// RequestThroughput of the service per second.
	RequestThroughput float64 `json:"requestThroughput" validate:"required"`
}

type URLResponse struct {
	// URLs contains a list of service info.
	URLs []URL `json:"urls" validate:"required"`
}

type URL struct {
	// URL of the request.
	URL string `json:"url" validate:"required"`

	// Service initiates the request.
	Service string `json:"service" validate:"required"`

	// RequestCount counts the requests.
	RequestCount int `json:"requestCount" validate:"required"`
}
