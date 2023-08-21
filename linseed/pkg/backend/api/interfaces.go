// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package api

import (
	"context"

	"github.com/olivere/elastic/v7"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

// FlowBackend defines the interface for interacting with L3 flows
type FlowBackend interface {
	List(context.Context, ClusterInfo, *v1.L3FlowParams) (*v1.List[v1.L3Flow], error)
}

// FlowLogBackend defines the interface for interacting with L3 flow logs
type FlowLogBackend interface {
	// Create creates the given L3 logs.
	Create(context.Context, ClusterInfo, []v1.FlowLog) (*v1.BulkResponse, error)

	// List lists logs that match the given parameters.
	List(context.Context, ClusterInfo, *v1.FlowLogParams) (*v1.List[v1.FlowLog], error)

	// Gets flow log aggregations
	Aggregations(context.Context, ClusterInfo, *v1.FlowLogAggregationParams) (*elastic.Aggregations, error)
}

// ProcessBackend defines the interface for interacting with process information.
type ProcessBackend interface {
	// List lists processes that match the given parameters.
	List(context.Context, ClusterInfo, *v1.ProcessParams) (*v1.List[v1.ProcessInfo], error)
}

// L7FlowBackend defines the interface for interacting with L7 flows.
type L7FlowBackend interface {
	List(context.Context, ClusterInfo, *v1.L7FlowParams) (*v1.List[v1.L7Flow], error)
}

// L7LogBackend defines the interface for interacting with L7 flow logs.
type L7LogBackend interface {
	// Create creates the given L7 logs.
	Create(context.Context, ClusterInfo, []v1.L7Log) (*v1.BulkResponse, error)

	// List lists logs that match the given parameters.
	List(context.Context, ClusterInfo, *v1.L7LogParams) (*v1.List[v1.L7Log], error)

	// Gets L7 log aggregations
	Aggregations(context.Context, ClusterInfo, *v1.L7AggregationParams) (*elastic.Aggregations, error)
}

// DNSFlowBackend defines the interface for interacting with DNS flows
type DNSFlowBackend interface {
	List(context.Context, ClusterInfo, *v1.DNSFlowParams) (*v1.List[v1.DNSFlow], error)
}

// ReportsBackend defines the interface for interacting with compliance reports
type ReportsBackend interface {
	List(context.Context, ClusterInfo, *v1.ReportDataParams) (*v1.List[v1.ReportData], error)
	Create(context.Context, ClusterInfo, []v1.ReportData) (*v1.BulkResponse, error)
}

// SnapshotsBackend defines the interface for interacting with compliance snapshots
type SnapshotsBackend interface {
	List(context.Context, ClusterInfo, *v1.SnapshotParams) (*v1.List[v1.Snapshot], error)
	Create(context.Context, ClusterInfo, []v1.Snapshot) (*v1.BulkResponse, error)
}

// BenchmarksBackend defines the interface for interacting with compliance benchmarks
type BenchmarksBackend interface {
	List(context.Context, ClusterInfo, *v1.BenchmarksParams) (*v1.List[v1.Benchmarks], error)
	Create(context.Context, ClusterInfo, []v1.Benchmarks) (*v1.BulkResponse, error)
}

// DNSLogBackend defines the interface for interacting with DNS logs
type DNSLogBackend interface {
	// Create creates the given logs.
	Create(context.Context, ClusterInfo, []v1.DNSLog) (*v1.BulkResponse, error)

	// List lists logs that match the given parameters.
	List(context.Context, ClusterInfo, *v1.DNSLogParams) (*v1.List[v1.DNSLog], error)

	// Gets DNS log aggregations
	Aggregations(context.Context, ClusterInfo, *v1.DNSAggregationParams) (*elastic.Aggregations, error)
}

// AuditBackend defines the interface for interacting with audit logs.
type AuditBackend interface {
	// Create creates the given logs.
	Create(context.Context, v1.AuditLogType, ClusterInfo, []v1.AuditLog) (*v1.BulkResponse, error)

	// List lists logs that match the given parameters.
	List(context.Context, ClusterInfo, *v1.AuditLogParams) (*v1.List[v1.AuditLog], error)

	// Gets Audit log aggregations
	Aggregations(context.Context, ClusterInfo, *v1.AuditLogAggregationParams) (*elastic.Aggregations, error)
}

// BGPBackend defines the interface for interacting with bgp logs.
type BGPBackend interface {
	// Create creates the given logs.
	Create(context.Context, ClusterInfo, []v1.BGPLog) (*v1.BulkResponse, error)

	// List lists logs that match the given parameters.
	List(context.Context, ClusterInfo, *v1.BGPLogParams) (*v1.List[v1.BGPLog], error)
}

// WAFBackend defines the interface for interacting with bgp logs.
type WAFBackend interface {
	// Create creates the given logs.
	Create(context.Context, ClusterInfo, []v1.WAFLog) (*v1.BulkResponse, error)

	// List lists logs that match the given parameters.
	List(context.Context, ClusterInfo, *v1.WAFLogParams) (*v1.List[v1.WAFLog], error)

	// Gets WAF log aggregations
	Aggregations(context.Context, ClusterInfo, *v1.WAFLogAggregationParams) (*elastic.Aggregations, error)
}

// EventsBackend defines the interface for interacting with events.
type EventsBackend interface {
	// Create creates the given logs.
	Create(context.Context, ClusterInfo, []v1.Event) (*v1.BulkResponse, error)

	// List lists logs that match the given parameters.
	List(context.Context, ClusterInfo, *v1.EventParams) (*v1.List[v1.Event], error)

	// Dismiss or Restore the given events.
	UpdateDismissFlag(context.Context, ClusterInfo, []v1.Event) (*v1.BulkResponse, error)

	// Delete the given events.
	Delete(context.Context, ClusterInfo, []v1.Event) (*v1.BulkResponse, error)
}

// RuntimeBackend defines the interface for interacting with runtime reports.
type RuntimeBackend interface {
	// Create creates the given logs.
	Create(context.Context, ClusterInfo, []v1.Report) (*v1.BulkResponse, error)

	// List lists reports that match the given parameters.
	List(context.Context, ClusterInfo, *v1.RuntimeReportParams) (*v1.List[v1.RuntimeReport], error)
}

// IPSetBackend defines the interface for interacting with ip set threat feeds.
type IPSetBackend interface {
	// Create creates the given threat feed.
	Create(context.Context, ClusterInfo, []v1.IPSetThreatFeed) (*v1.BulkResponse, error)

	// List lists threat feeds that match the given parameters.
	List(context.Context, ClusterInfo, *v1.IPSetThreatFeedParams) (*v1.List[v1.IPSetThreatFeed], error)

	// Delete the given threat feeds.
	Delete(context.Context, ClusterInfo, []v1.IPSetThreatFeed) (*v1.BulkResponse, error)
}

// DomainNameSetBackend defines the interface for interacting with domain name threat feeds.
type DomainNameSetBackend interface {
	// Create creates the given threat feed.
	Create(context.Context, ClusterInfo, []v1.DomainNameSetThreatFeed) (*v1.BulkResponse, error)

	// List lists threat feeds that match the given parameters.
	List(context.Context, ClusterInfo, *v1.DomainNameSetThreatFeedParams) (*v1.List[v1.DomainNameSetThreatFeed], error)

	// Delete the given threat feeds.
	Delete(context.Context, ClusterInfo, []v1.DomainNameSetThreatFeed) (*v1.BulkResponse, error)
}

// DataType determines the type of logs supported
// to be ingested via bulk APIs
type DataType string

const (
	FlowLogs       DataType = "flows"
	DNSLogs        DataType = "dns"
	L7Logs         DataType = "l7"
	AuditEELogs    DataType = "audit_ee"
	AuditKubeLogs  DataType = "audit_kube"
	BGPLogs        DataType = "bgp"
	Events         DataType = "events"
	WAFLogs        DataType = "waf"
	ReportData     DataType = "compliance_reports"
	Snapshots      DataType = "snapshots"
	Benchmarks     DataType = "benchmark_results"
	RuntimeReports DataType = "runtime"
	IPSet          DataType = "threatfeeds_ipset"
	DomainNameSet  DataType = "threatfeeds_domainnameset"
)

// Cache is a cache for the templates in order
// to create mappings, write aliases and rollover
// indices only once. It will store as key-value pair
// a definition of the template. The key used
// is composed of types of logs and cluster info
type Cache interface {
	// InitializeIfNeeded will retrieve the template from the cache. If not found,
	// tt will proceed to store a template against the pairs of
	// DataType and ClusterInfo. An error will be returned if template creation
	// or index boostrap fails.
	InitializeIfNeeded(context.Context, DataType, ClusterInfo) error
}
