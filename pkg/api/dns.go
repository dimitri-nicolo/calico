// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package api

import (
	"context"
	"time"
)

const (
	DNSLogStartTime       = "start_time"
	DNSLogClientNamespace = "client_namespace"
	DNSLogQtype           = "qtype"
)

// Container type to hold the alert events and/or an error.
type DNSResult struct {
	*DNSLog
	Err error
}

type DNSLog struct {
	StartTime       time.Time        `json:"start_time"`
	ClientNamespace string           `json:"client_namespace"`
	Qtype           string           `json:"qtype"`
	RRSets          []ResourceRecord `json:"rrsets"`
}

type ResourceRecord struct {
	Name  string   `json:"name"`
	Class string   `json:"class"`
	Type  string   `json:"type"`
	Rdata []string `json:"rdata"`
}

type DNSLogsSelection struct {
	// Resources lists the resources that will be included in the DNS logs retrieved.
	// Blank fields in the listed ResourceID structs are treated as wildcards.
	Resources []DNSResource `json:"resources,omitempty" validate:"omitempty"`
}

// Used to filter DNS logs.
// An empty field value indicates a wildcard.
type DNSResource struct {
	ClientNamespace string `json:"client_namespace,omitempty" validate:"omitempty"`
	Qtype           string `json:"qtype,omitempty" validate:"omitempty"`
}

type DNSLogReportHandler interface {
	SearchDNSLogs(ctx context.Context, filter *DNSLogsSelection, start, end *time.Time) <-chan *DNSResult
}
