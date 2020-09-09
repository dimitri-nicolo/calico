// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package api

import (
	"context"
	"time"
)

const (
	ADLogResultType         = "result_type"
	ADLogJobId              = "job_id"
	ADLogTimestamp          = "timestamp"
	ADLogRecordScore        = "record_score"
	ADLogBucketSpan         = "bucket_span"
	ADLogInitialRecordScore = "initial_record_score"
	ADLogEventCount         = "event_count"
	ADLogIsInterim          = "is_interim"
	ADLogDestNamespace      = "dest_namespace"
)

const (
	// The anomaly detection indices contain several different document types.
	// This represents result_type="record".
	ADRecordResultType = "record"
)

// Container type to hold the AD log and/or an error.
type ADResult struct {
	*ADRecordLog
	Err error
}

// Anomaly detection "record" type logs (result_type=record).
type ADRecordLog struct {
	JobId              string   `json:"job_id"`
	Timestamp          int64    `json:"timestamp"`
	RecordScore        float64  `json:"record_score"`
	BucketSpan         int64    `json:"bucket_span"`
	InitialRecordScore float64  `json:"initial_record_score"`
	EventCount         int64    `json:"event_count"`
	IsInterim          bool     `json:"is_interim"`
	DestNamespaces     []string `json:"dest_namespace"`
}

type ADLogsSelection struct {
	// Resources lists the resources that will be included in the AD logs retrieved.
	// Blank fields in the listed ResourceID structs are treated as wildcards.
	Resources []ADResource `json:"resources,omitempty" validate:"omitempty"`
}

// Used to filter AD logs.
// An empty field value indicates a wildcard.
type ADResource struct {
	ResultType     string   `json:"resulttype,omitempty" validate:"omitempty"`
	MinRecordScore *float64 `json:"minrecordscore,omitempty" validate:"omitempty"`
	MaxRecordScore *float64 `json:"maxrecordscore,omitempty" validate:"omitempty"`
	MinEventCount  *int64   `json:"mineventcount,omitempty" validate:"omitempty"`
}

type ADLogReportHandler interface {
	SearchADLogs(ctx context.Context, filter *ADLogsSelection, start, end *time.Time) <-chan *ADResult
}
