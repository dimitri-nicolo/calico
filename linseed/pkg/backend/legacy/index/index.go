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
var (
	FlowLogIndex             bapi.Index = singleIndex{name: "calico_flowlogs", dataType: bapi.FlowLogs}
	BGPLogIndex              bapi.Index = singleIndex{name: "calico_bgplogs", dataType: bapi.BGPLogs}
	ComplianceSnapshotIndex  bapi.Index = singleIndex{name: "calico_compliance_snapshots", dataType: bapi.Snapshots}
	ComplianceBenchmarkIndex bapi.Index = singleIndex{name: "calico_compliance_benchmark_results", dataType: bapi.Benchmarks}
	ComplianceReportIndex    bapi.Index = singleIndex{name: "calico_compliance_reports", dataType: bapi.ReportData}
	DNSLogIndex              bapi.Index = singleIndex{name: "calico_dnslogs", dataType: bapi.DNSLogs}
	AlertsIndex              bapi.Index = singleIndex{name: "calico_alerts", dataType: bapi.Events}
	L7LogIndex               bapi.Index = singleIndex{name: "calico_l7logs", dataType: bapi.L7Logs}
	RuntimeReportIndex       bapi.Index = singleIndex{name: "calico_runtime_reports", dataType: bapi.RuntimeReports}
	ThreatfeedsDomainIndex   bapi.Index = singleIndex{name: "calico_threatfeeds_domainnameset", dataType: bapi.DomainNameSet}
	ThreatfeedsIPSetIndex    bapi.Index = singleIndex{name: "calico_threatfeeds_ipset", dataType: bapi.IPSet}
	WAFLogIndex              bapi.Index = singleIndex{name: "calico_waf", dataType: bapi.WAFLogs}

	// The AuditLogIndex uses data type AuditEELogs, but it's actually used for both AuditEELogs and AuditKubeLogs.
	// This is OK because our code for initializing indicies treats these the same anyway.
	AuditLogIndex bapi.Index = singleIndex{name: "calico_auditlogs", dataType: bapi.AuditEELogs}
)

// singleIndex implements the Index interface for an index mode that uses a single index
// to store data for multiple clusters and tenants.
type singleIndex struct {
	name     string
	dataType bapi.DataType
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
	// TODO: This uses the old polciy name until we implement ILM policies for the new indicies.
	// This allows the new indicies to be compatible with ILM policies already being provisioned by the operator.
	return fmt.Sprintf("tigera_secure_ee_%s_policy", i.DataType())
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
