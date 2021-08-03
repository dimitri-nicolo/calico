// Copyright 2019 Tigera Inc. All rights reserved.

package db

import (
	"context"
	"encoding/json"
	apiV3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"time"
)

type Meta struct {
	Name        string
	SeqNo       *int64
	PrimaryTerm *int64
}

type IPSet interface {
	PutIPSet(ctx context.Context, name string, set IPSetSpec) error
	GetIPSet(ctx context.Context, name string) (IPSetSpec, error)
	GetIPSetModified(ctx context.Context, name string) (time.Time, error)
	ListIPSets(ctx context.Context) ([]Meta, error)
	DeleteIPSet(ctx context.Context, m Meta) error
}

type DomainNameSet interface {
	PutDomainNameSet(ctx context.Context, name string, set DomainNameSetSpec) error
	GetDomainNameSetModified(ctx context.Context, name string) (time.Time, error)
	ListDomainNameSets(ctx context.Context) ([]Meta, error)
	DeleteDomainNameSet(ctx context.Context, m Meta) error
}

type SuspiciousSet interface {
	QuerySet(ctx context.Context, feed *apiV3.GlobalThreatFeed) ([]SecurityEventInterface, time.Time, string, error)
}

type SecurityEventInterface interface {
	ID() string
}

type Events interface {
	PutSecurityEvent(context.Context, SecurityEventInterface) error
	GetSecurityEvents(ctx context.Context, start, end time.Time, allClusters bool) ([]SecurityEvent, error)
	PutForwarderConfig(ctx context.Context, id string, f *ForwarderConfig) error
	GetForwarderConfig(ctx context.Context, id string) (*ForwarderConfig, error)
}

type AuditLog interface {
	ObjectCreatedBetween(ctx context.Context, kind, namespace, name string, before, after time.Time) (bool, error)
	ObjectDeletedBetween(ctx context.Context, kind, namespace, name string, before, after time.Time) (bool, error)
}

// IPs are sent as strings to avoid overhead of decoding and encoding net.IP, since they are strings on the
// wire to elastic.
type IPSetSpec []string

type DomainNameSetSpec []string

type QueryKey int

// SecurityEvent represents an event generated by the controller or one of the watchers created by the controller.
// This type is used solely for easy access to event data when querying the ES index.
type SecurityEvent struct {
	Data json.RawMessage
	ID   string
}

// ForwarderConfig represents configuration state used for the event forwarder. This is saved to the datastore for
// recovery purposes. Note that LastSuccessfulRunEndTime and LastSuccessfulEventTime are different.
// LastSuccessfulRunEndTime represents the end point of the time ranged used by the last successful run. This is the
// time we use as the starting point for the next run. LastSuccessfulEventTime is the time value from the last
// successfully forwarded event, which may or may not match the LastSuccessfulRunEndTime. It's useful for record
// keeping and debugging.
type ForwarderConfig struct {
	LastSuccessfulEventTime  *time.Time `json:"last_successful_event_time,omitempty"` // Time field from the last successfully forwarded event
	LastSuccessfulEventID    *string    `json:"last_successful_event_id,omitempty"`   // ID of the successfully forwarded last event
	LastSuccessfulRunEndTime *time.Time `json:"last_successful_run_endtime"`          // End time of the last run
}

const (
	QueryKeyUnknown QueryKey = iota
	QueryKeyFlowLogSourceIP
	QueryKeyFlowLogDestIP
	QueryKeyDNSLogQName
	QueryKeyDNSLogRRSetsName
	QueryKeyDNSLogRRSetsRData
)

const (
	IpSetHashKey         = "hash.intrusion-detection.tigera.io/ip-set"
	DomainNameSetHashKey = "hash.intrusion-detection.tigera.io/domain-name-set"
)
