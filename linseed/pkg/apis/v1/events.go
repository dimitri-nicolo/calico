// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package v1

import (
	"fmt"
	"time"

	"github.com/projectcalico/calico/libcalico-go/lib/json"
)

// EventParams define querying parameters to retrieve events
type EventParams struct {
	QueryParams        `json:",inline" validate:"required"`
	QuerySortParams    `json:",inline"`
	LogSelectionParams `json:",inline"`
}

type TimestampOrDate struct {
	intVal  *int64
	timeVal *time.Time
}

// ISO8601Format is the format Anomaly Detection
// alerts use for field "Time". Example: 2023-04-28T19:38:14+00:00
// Anomaly detection makes use of `isoformat` method available in python libraries.
// This format is similar to RFC3339, but it has some small differences.
// RFC 3339 uses HH:mm:ssZ to mark a timestamp is on GMT timezone,
// while this format will render this infomation like +00:00
const ISO8601Format = "2006-01-02T15:04:05-07:00"

type Event struct {
	ID              string          `json:"id"`
	Time            TimestampOrDate `json:"time" validate:"required"`
	Description     string          `json:"description" validate:"required"`
	Origin          string          `json:"origin" validate:"required"`
	Severity        int             `json:"severity" validate:"required"`
	Type            string          `json:"type" validate:"required"`
	DestIP          *string         `json:"dest_ip,omitempty"`
	DestName        string          `json:"dest_name,omitempty"`
	DestNameAggr    string          `json:"dest_name_aggr,omitempty"`
	DestNamespace   string          `json:"dest_namespace,omitempty"`
	DestPort        *int64          `json:"dest_port,omitempty"`
	Protocol        string          `json:"protocol,omitempty"`
	Dismissed       bool            `json:"dismissed,omitempty"`
	Host            string          `json:"host,omitempty"`
	SourceIP        *string         `json:"source_ip,omitempty"`
	SourceName      string          `json:"source_name,omitempty"`
	SourceNameAggr  string          `json:"source_name_aggr,omitempty"`
	SourceNamespace string          `json:"source_namespace,omitempty"`
	SourcePort      *int64          `json:"source_port,omitempty"`
	Name            string          `json:"name,omitempty"`
	AttackVector    string          `json:"attack_vector,omitempty"`
	AttackPhase     string          `json:"attack_phase,omitempty"`
	MitreIDs        *[]string       `json:"mitre_ids,omitempty"`
	Mitigations     *[]string       `json:"mitigations,omitempty"`
	Record          interface{}     `json:"record,omitempty"`
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

type DPIRecord struct {
	SnortSignatureID       string `json:"snort_signature_id"`
	SnortSignatureRevision string `json:"snort_signature_revision"`
	SnortAlert             string `json:"snort_alert"`
}

// NewEventTimestamp will create a new TimestampOrDate
// that has only timestamp field populated with a value
// that represents unix time in seconds
func NewEventTimestamp(val int64) TimestampOrDate {
	return TimestampOrDate{
		intVal: &val,
	}
}

func (t *TimestampOrDate) UnmarshalJSON(data []byte) error {
	if t == nil {
		return fmt.Errorf("cannot unmarshal nil value from JSON")
	}

	if len(data) == 0 {
		return nil
	}

	if data[0] == '"' {
		return json.Unmarshal(data, &t.timeVal)
	}

	return json.Unmarshal(data, &t.intVal)
}

func (t *TimestampOrDate) MarshalJSON() ([]byte, error) {
	if t == nil {
		return nil, fmt.Errorf("cannot marshal nil value into JSON")
	}

	if t.intVal != nil && t.timeVal != nil {
		return nil, fmt.Errorf("time should either be as unix timestamp or ISO8601 time format")
	}

	if t.intVal != nil {
		return json.Marshal(*t.intVal)
	}

	if t.timeVal != nil {
		return json.Marshal(t.timeVal.Format(ISO8601Format))
	}

	if t.timeVal == nil && t.intVal == nil {
		var zero = 0
		return json.Marshal(&zero)
	}

	return nil, nil
}

func (t *TimestampOrDate) GetTime() time.Time {
	if t.intVal != nil {
		return time.Unix(*t.intVal, 0)
	}

	if t.timeVal != nil {
		return *t.timeVal
	}

	return time.Time{}
}
