// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package v1

type EventException struct {
	ID                string `json:"id,omitempty"`
	Type              string `json:"type,omitempty"`
	Event             string `json:"event,omitempty"`
	Namespace         string `json:"namespace,omitempty"`
	Pod               string `json:"pod,omitempty"`
	UseNameAggr       bool   `json:"use_name_aggr,omitempty"`
	Description       string `json:"description,omitempty"`
	HasUnexpectedData bool   `json:"has_unexpected_data,omitempty"`
	Count             int    `json:"count,omitempty"`
}
