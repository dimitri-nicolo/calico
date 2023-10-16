// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package index

import (
	"fmt"
	"strings"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
)

// Legacy indices - these all use multiple indices per-cluster and per-tenant.
var (
	ThreatfeedsDomainMultiIndex   bapi.Index = multiIndex{baseName: "tigera_secure_ee_threatfeeds_domainnameset", dataType: bapi.DomainNameSet}
	ThreatfeedsIPSetMultiIndex    bapi.Index = multiIndex{baseName: "tigera_secure_ee_threatfeeds_ipset", dataType: bapi.IPSet}
	EventsMultiIndex              bapi.Index = multiIndex{baseName: "tigera_secure_ee_events", dataType: bapi.Events}
	ComplianceSnapshotMultiIndex  bapi.Index = multiIndex{baseName: "tigera_secure_ee_snapshots", dataType: bapi.Snapshots}
	ComplianceBenchmarkMultiIndex bapi.Index = multiIndex{baseName: "tigera_secure_ee_benchmark_results", dataType: bapi.Benchmarks}
	ComplianceReportMultiIndex    bapi.Index = multiIndex{baseName: "tigera_secure_ee_compliance_reports", dataType: bapi.ReportData}
	WAFLogMultiIndex              bapi.Index = multiIndex{baseName: "tigera_secure_ee_waf", dataType: bapi.WAFLogs}
	L7LogMultiIndex               bapi.Index = multiIndex{baseName: "tigera_secure_ee_l7", dataType: bapi.L7Logs}
	BGPLogMultiIndex              bapi.Index = multiIndex{baseName: "tigera_secure_ee_bgp", dataType: bapi.BGPLogs}
	AuditLogEEMultiIndex          bapi.Index = multiIndex{baseName: "tigera_secure_ee_audit_ee", dataType: bapi.AuditEELogs}
	AuditLogKubeMultiIndex        bapi.Index = multiIndex{baseName: "tigera_secure_ee_audit_kube", dataType: bapi.AuditKubeLogs}
	DNSLogMultiIndex              bapi.Index = multiIndex{baseName: "tigera_secure_ee_dns", dataType: bapi.DNSLogs}
	FlowLogMultiIndex             bapi.Index = multiIndex{baseName: "tigera_secure_ee_flows", dataType: bapi.FlowLogs}
	RuntimeReportMultiIndex       bapi.Index = multiIndex{baseName: "tigera_secure_ee_runtime", dataType: bapi.RuntimeReports}
)

// Single index - these all use a single index for all clusters and tenants.

type Option func(index bapi.Index)

func WithIndexName(name string) Option {
	return func(index bapi.Index) {
		//index.SetName(name)
	}
}

func WithPolicyName(name string) Option {
	return func(index bapi.Index) {
		//index.SetPolicyName(name)
	}
}

func AlertsIndex(options ...Option) bapi.Index {
	index := singleIndex{name: "calico_alerts", policyName: "tigera_secure_ee_events_policy", dataType: bapi.Events}

	for _, opt := range options {
		opt(index)
	}

	return &index
}

func AuditLogIndex(options ...Option) bapi.Index {
	// The AuditLogIndex uses data type AuditEELogs, but it's actually used for both AuditEELogs and AuditKubeLogs.
	// This is OK because our code for initializing indicies treats these the same anyway.
	index := singleIndex{name: "calico_auditlogs", policyName: "tigera_secure_ee_audit_ee_policy", dataType: bapi.AuditEELogs}

	for _, opt := range options {
		opt(index)
	}

	return &index
}

func BGPLogIndex(options ...Option) bapi.Index {
	index := singleIndex{name: "calico_bgplogs", policyName: "tigera_secure_ee_bgp_policy", dataType: bapi.BGPLogs}

	for _, opt := range options {
		opt(index)
	}

	return &index
}

func ComplianceBenchmarksIndex(options ...Option) bapi.Index {
	index := singleIndex{name: "calico_compliance_benchmark_results", policyName: "tigera_secure_ee_benchmark_results_policy", dataType: bapi.Benchmarks}

	for _, opt := range options {
		opt(index)
	}

	return &index
}

func ComplianceReportsIndex(options ...Option) bapi.Index {
	index := singleIndex{name: "calico_compliance_reports", policyName: "tigera_secure_ee_compliance_reports_policy", dataType: bapi.ReportData}

	for _, opt := range options {
		opt(index)
	}

	return &index
}

func ComplianceSnapshotsIndex(options ...Option) bapi.Index {
	index := singleIndex{name: "calico_compliance_snapshots", policyName: "tigera_secure_ee_snapshots_policy", dataType: bapi.Snapshots}

	for _, opt := range options {
		opt(index)
	}

	return &index
}
func DNSLogIndex(options ...Option) bapi.Index {
	index := singleIndex{name: "calico_dnslogs", policyName: "tigera_secure_ee_dns_policy", dataType: bapi.DNSLogs}

	for _, opt := range options {
		opt(index)
	}

	return &index
}

func FlowLogIndex(options ...Option) bapi.Index {
	index := singleIndex{name: "calico_flowlogs", policyName: "tigera_secure_ee_flows_policy", dataType: bapi.FlowLogs}

	for _, opt := range options {
		opt(index)
	}

	return &index
}

func L7LogIndex(options ...Option) bapi.Index {
	index := singleIndex{name: "calico_l7logs", policyName: "tigera_secure_ee_l7_policy", dataType: bapi.L7Logs}

	for _, opt := range options {
		opt(index)
	}

	return &index
}

func RuntimeReportsIndex(options ...Option) bapi.Index {
	index := singleIndex{name: "calico_runtime_reports", policyName: "tigera_secure_ee_runtime_policy", dataType: bapi.RuntimeReports}

	for _, opt := range options {
		opt(index)
	}

	return &index
}

func ThreatFeedsIPSetIndex(options ...Option) bapi.Index {
	index := singleIndex{name: "calico_threatfeeds_ipset", dataType: bapi.IPSet}

	for _, opt := range options {
		opt(index)
	}

	return &index
}

func ThreatFeedsDomainSetIndex(options ...Option) bapi.Index {
	index := singleIndex{name: "calico_threatfeeds_domainnameset", dataType: bapi.DomainNameSet}

	for _, opt := range options {
		opt(index)
	}

	return &index
}
func WAFLogIndex(options ...Option) bapi.Index {
	index := singleIndex{name: "calico_waf", policyName: "tigera_secure_ee_waf_policy", dataType: bapi.WAFLogs}

	for _, opt := range options {
		opt(index)
	}

	return &index
}

// singleIndex implements the Index interface for an index mode that uses a single index
// to store data for multiple clusters and tenants.
type singleIndex struct {
	name       string
	policyName string
	dataType   bapi.DataType
}

func (i singleIndex) Name(info bapi.ClusterInfo) string {
	return i.name
}

func (i singleIndex) BootstrapIndexName(info bapi.ClusterInfo) string {
	pattern := "<%s.linseed-{now/s{yyyyMMdd}}-000001>"
	return fmt.Sprintf(pattern, i.name)
}

func (i singleIndex) Index(info bapi.ClusterInfo) string {
	return fmt.Sprintf("%s.*", i.name)
}

func (i singleIndex) Alias(info bapi.ClusterInfo) string {
	return fmt.Sprintf("%s.", i.name)
}

func (i singleIndex) IndexTemplateName(info bapi.ClusterInfo) string {
	return fmt.Sprintf("%s.", i.name)
}

func (i singleIndex) IsSingleIndex() bool {
	return true
}

func (i singleIndex) DataType() bapi.DataType {
	return i.dataType
}

func (i singleIndex) ILMPolicyName() string {
	return i.policyName
}

func NewMultiIndex(baseName string, dataType bapi.DataType) bapi.Index {
	return multiIndex{baseName: baseName, dataType: dataType}
}

// multiIndex implements the Index interface for an index mode that uses multiple
// indicies to store data for multiple clusters and tenants.
type multiIndex struct {
	baseName string
	dataType bapi.DataType
}

func (i multiIndex) DataType() bapi.DataType {
	return i.dataType
}

func (i multiIndex) Name(info bapi.ClusterInfo) string {
	if info.Tenant == "" {
		return fmt.Sprintf("%s-%s", strings.ToLower(string(i.dataType)), info.Cluster)
	}

	return fmt.Sprintf("%s-%s-%s", strings.ToLower(string(i.dataType)), info.Cluster, info.Tenant)
}

func (i multiIndex) BootstrapIndexName(info bapi.ClusterInfo) string {
	template, ok := BootstrapIndexPatternLookup[i.DataType()]
	if !ok {
		panic("bootstrap index name for log type not implemented")
	}
	if info.Tenant == "" {
		return fmt.Sprintf(template, info.Cluster)
	}

	return fmt.Sprintf(template, fmt.Sprintf("%s.%s", info.Tenant, info.Cluster))
}

func (i multiIndex) Index(info bapi.ClusterInfo) string {
	if info.Tenant != "" {
		// If a tenant is provided, then we must include it in the index.
		return fmt.Sprintf("%s.%s.%s.*", i.baseName, info.Tenant, info.Cluster)
	}
	// Otherwise, this is a single-tenant cluster and we only need the cluster.
	return fmt.Sprintf("%s.%s.*", i.baseName, info.Cluster)
}

func (i multiIndex) Alias(info bapi.ClusterInfo) string {
	if info.Tenant == "" {
		return fmt.Sprintf("%s.%s.", i.baseName, info.Cluster)
	}
	return fmt.Sprintf("%s.%s.%s.", i.baseName, info.Tenant, info.Cluster)
}

func (i multiIndex) IndexTemplateName(info bapi.ClusterInfo) string {
	template, ok := TemplateNamePatternLookup[i.DataType()]
	if !ok {
		panic("template name for log type not implemented")
	}
	if info.Tenant == "" {
		return fmt.Sprintf(template, info.Cluster)
	}

	return fmt.Sprintf(template, fmt.Sprintf("%s.%s", info.Tenant, info.Cluster))
}

func (i multiIndex) IsSingleIndex() bool {
	return false
}

func (i multiIndex) ILMPolicyName() string {
	return fmt.Sprintf("tigera_secure_ee_%s_policy", i.DataType())
}
