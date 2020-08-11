// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package api

import (
	"context"
	"time"
)

const (
	AlertLogType            = "type"
	AlertLogSourceNamespace = "source_namespace"
	AlertLogTime            = "time"
)

// Container type to hold the alert events and/or an error.
type AlertResult struct {
	*Alert
	Err error
}

type Alert struct {
	Type            string `json:"type"`
	SourceNamespace string `json:"source_namespace"`
}

type AlertLogsSelection struct {
	// Resources lists the resources that will be included in the alert logs retrieved.
	// Blank fields in the listed ResourceID structs are treated as wildcards.
	Resources []AlertResource `json:"resources,omitempty" validate:"omitempty"`
}

// Used to filter alert logs.
// An empty field value indicates a wildcard.
type AlertResource struct {
	// The alert type.
	Type string `json:"type,omitempty" validate:"omitempty"`

	// The source namespace.
	SourceNamespace string `json:"source_namespace,omitempty" validate:"omitempty"`
}

type AlertLogReportHandler interface {
	SearchAlertLogs(ctx context.Context, filter *AlertLogsSelection, start, end *time.Time) <-chan *AlertResult
}
