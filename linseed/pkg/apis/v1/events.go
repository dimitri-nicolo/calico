// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package v1

import "github.com/projectcalico/calico/libcalico-go/lib/json"

// EventParams define querying parameters to retrieve events
type EventParams struct {
	QueryParams        `json:",inline" validate:"required"`
	LogSelectionParams `json:",inline"`
}

type Event struct {
	ID              string      `json:"id"`
	Time            int64       `json:"time" validate:"required"`
	Description     string      `json:"description" validate:"required"`
	Origin          string      `json:"origin" validate:"required"`
	Severity        int         `json:"severity" validate:"required"`
	Type            string      `json:"type" validate:"required"`
	Alert           string      `json:"alert,omitempty"`
	DestIP          *string     `json:"dest_ip,omitempty"`
	DestName        string      `json:"dest_name,omitempty"`
	DestNameAggr    string      `json:"dest_name_aggr,omitempty"`
	DestNamespace   string      `json:"dest_namespace,omitempty"`
	DestPort        *int64      `json:"dest_port,omitempty"`
	Protocol        string      `json:"protocol,omitempty"`
	Dismissed       bool        `json:"dismissed,omitempty"`
	Host            string      `json:"host,omitempty"`
	SourceIP        *string     `json:"source_ip,omitempty"`
	SourceName      string      `json:"source_name,omitempty"`
	SourceNameAggr  string      `json:"source_name_aggr,omitempty"`
	SourceNamespace string      `json:"source_namespace,omitempty"`
	SourcePort      *int64      `json:"source_port,omitempty"`
	Record          interface{} `json:"record,omitempty"`
}

// Events can take records of numerous forms. GetRecord extracts the record
// on the event into the given object.
func (e *Event) GetRecord(into interface{}) error {
	bs, err := json.Marshal(e.Record)
	if err != nil {
		return err
	}
	return json.Unmarshal(bs, into)
}

// RawRecordData is a generic record with arbitrary fields.
type RawRecordData map[string]interface{}

type EventRecord struct {
	// Structured fields
	ResponseObjectKind string `json:"responseObject.kind,omitempty"`
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

type SuspiciousDomainEventRecord struct {
	DNSLogID          string   `json:"dns_log_id"`
	Feeds             []string `json:"feeds,omitempty"`
	SuspiciousDomains []string `json:"suspicious_domains"`
}

type SuspiciousIPEventRecord struct {
	FlowAction       string   `json:"flow_action"`
	FlowLogID        string   `json:"flow_log_id"`
	Protocol         string   `json:"protocol"`
	Feeds            []string `json:"feeds,omitempty"`
	SuspiciousPrefix *string  `json:"suspicious_prefix"`
}

type HoneypodAlertRecord struct {
	Count       *int64  `json:"count,omitempty"`
	HostKeyword *string `json:"host.keyword,omitempty"`
}

type HoneypodSnortEventRecord struct {
	Snort *Snort `json:"snort,omitempty"`
}

type Snort struct {
	Category    string `json:"category,omitempty"`
	Description string `json:"description,omitempty"`
	Flags       string `json:"flags,omitempty"`
	Occurrence  string `json:"occurrence,omitempty"`
	Other       string `json:"other,omitempty"`
}
