// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package v1

// EventParams define querying parameters to retrieve events
type EventParams struct {
	QueryParams `json:",inline" validate:"required"`
}

type Event struct {
	ID              string       `json:"id"`
	Time            int64        `json:"time" validate:"required"`
	Description     string       `json:"description" validate:"required"`
	Origin          string       `json:"origin" validate:"required"`
	Severity        int          `json:"severity" validate:"required"`
	Type            string       `json:"type" validate:"required"`
	Alert           string       `json:"alert,omitempty"`
	DestIP          *string      `json:"dest_ip,omitempty"`
	DestName        string       `json:"dest_name,omitempty"`
	DestNameAggr    string       `json:"dest_name_aggr,omitempty"`
	DestNamespace   string       `json:"dest_namespace,omitempty"`
	DestPort        *int64       `json:"dest_port,omitempty"`
	Protocol        string       `json:"protocol,omitempty"`
	Dismissed       bool         `json:"dismissed,omitempty"`
	Host            string       `json:"host,omitempty"`
	SourceIP        *string      `json:"source_ip,omitempty"`
	SourceName      string       `json:"source_name,omitempty"`
	SourceNameAggr  string       `json:"source_name_aggr,omitempty"`
	SourceNamespace string       `json:"source_namespace,omitempty"`
	SourcePort      *int64       `json:"source_port,omitempty"`
	Record          *EventRecord `json:"record,omitempty"`
}

// TODO: This object can take a lot of forms. Right now, we're just making one object that is the
// superset of all the possible kinds. e.g.,
// RawEventRecord
// SuspiciousIPEventRecord
// SuspiciousDomainEventRecord
// HoneyPodAlertRecord
// etc.
type EventRecord struct {
	ResponseObjectKind string `json:"responseObject.kind,omitempty"` // TODO - what's with the dots? find some real data to compare.
	ObjectRefResource  string `json:"objectRef.resource,omitempty"`
	ObjectRefNamespace string `json:"objectRef.namespace,omitempty"`
	ObjectRefName      string `json:"objectRef.name,omitempty"`
	ClientNamespace    string `json:"client_namespace,omitempty"`
	ClientName         string `json:"client_name,omitempty"`
	ClientNameAggr     string `json:"client_name_aggr,omitempty"`
	SourceType         string `json:"source_type,omitempty"`
	SourceNamespace    string `json:"source_namespace,omitempty"`
	SourceNameAggr     string `json:"source_name_aggr,omitempty"`
	SourceName         string `json:"source_name,omitempty"`
	DestType           string `json:"dest_type,omitempty"`
	DestNamespace      string `json:"dest_namespace,omitempty"`
	DestNameAggr       string `json:"dest_name_aggr,omitempty"`
	DestName           string `json:"dest_name,omitempty"`
	DestPort           int    `json:"dest_port,omitempty"`
	Protocol           string `json:"proto,omitempty"`
}
