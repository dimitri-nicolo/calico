// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const (
	AlertLogType            = "type"
	AlertLogSourceNamespace = "source_namespace"
	AlertLogDestNamespace   = "dest_namespace"
	AlertLogTime            = "time"
	AlertLogAlert           = "alert"
)

// Container type to hold the alert events and/or an error.
type AlertResult struct {
	*Alert
	Err error
}

type Alert struct {
	Type            string    `json:"type"`
	SourceNamespace string    `json:"source_namespace"`
	DestNamespace   string    `json:"dest_namespace"`
	Description     string    `json:"description"`
	Severity        int64     `json:"severity"`
	Time            time.Time `json:"time"`
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

	// The dest namespace.
	DestNamespace string `json:"dest_namespace,omitempty" validate:"omitempty"`

	// Intrusion detection global alert sometimes define the alert type using an 'alert' field,
	// instead of the 'type' field (confusing, I know...)
	Alert string `json:"alert,omitempty" validate:"omitempty"`
}

type AlertLogReportHandler interface {
	SearchAlertLogs(ctx context.Context, filter *AlertLogsSelection, start, end *time.Time) <-chan *AlertResult
}

func (a *Alert) UnmarshalJSON(data []byte) error {
	s := &struct {
		Type            string      `json:"type"`
		SourceNamespace string      `json:"source_namespace"`
		DestNamespace   string      `json:"dest_namespace"`
		Description     string      `json:"description"`
		Severity        int64       `json:"severity"`
		Time            interface{} `json:"time"`
	}{}
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	a.Type = s.Type
	a.SourceNamespace = s.SourceNamespace
	a.DestNamespace = s.DestNamespace
	a.Description = s.Description
	a.Severity = s.Severity
	if t, err := parseTime(s.Time); err == nil {
		a.Time = t
	} else {
		return fmt.Errorf("Error parsing time %v, error=%v", s.Time, err)
	}
	return nil
}

// In elastic, the 'date' type can be either in string or long format.
// The output that elastic returns depends on how the data was formatted
// when it was posted. Here we check the format and convert appropriately.
func parseTime(obj interface{}) (time.Time, error) {
	// First: try converting time from long.
	if tInt, ok := obj.(int64); ok {
		return time.Unix(0, tInt*int64(time.Millisecond)), nil
	}

	// Try converting from float.
	if tFloat, ok := obj.(float64); ok {
		tInt := int64(tFloat)
		return time.Unix(0, tInt*int64(time.Millisecond)), nil
	}

	// If 'time' is not a long, try parsing it as a string.
	if tStr, ok := obj.(string); ok {
		t, err := parseTimeString(tStr)
		if err != nil {
			return time.Now(), fmt.Errorf("Error parsing time string %s, error=%v", tStr, err)
		}
		return t, nil
	} else {
		return time.Now(), fmt.Errorf("Error parsing time %v", obj)
	}
}

// Try to parse a string into a time object.
// NOTE: this function only tries the time formats currently used in our systems.
// If a time is defined in a different format, an error is raised.
func parseTimeString(s string) (time.Time, error) {
	// Check if 'time' is in 'yyyy-MM-ddThh:mm:ss:SSSSSSZ' format
	layout := "2006-01-02T15:04:05.000000Z"
	t, err := time.Parse(layout, s)
	if err == nil {
		return t, nil
	}

	// Check if 'time' is in 'yyyy-MM-ddThh:mm:ss:SSSZ' format
	layout = "2006-01-02T15:04:05.000Z"
	t, err = time.Parse(layout, s)
	if err == nil {
		return t, nil
	}

	// Check if 'time' is in 'yyyy-MM-ddThh:mm:ssZ' format
	layout = "2006-01-02T15:04:05Z"
	t, err = time.Parse(layout, s)
	if err == nil {
		return t, nil
	}

	return time.Now(), fmt.Errorf("Unrecognized time format: %s", s)
}
