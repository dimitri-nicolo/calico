// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package templates

import (
	_ "embed"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
)

//go:embed flowlog_mappings.json
var FlowLogMappings string

//go:embed l7log_mappings.json
var L7LogMappings string

//go:embed dnslog_mappings.json
var DNSLogMappings string

//go:embed dnslog_settings.json
var DNSLogSettings string

//go:embed audit_mappings.json
var AuditMappings string

//go:embed bgp_mappings.json
var BGPMappings string

//go:embed events_mappings.json
var EventsMappings string

//go:embed waf_mappings.json
var WAFMappings string

//go:embed report_mappings.json
var ReportMappings string

//go:embed benchmarks_mappings.json
var BenchmarksMappings string

//go:embed snapshots_mappings.json
var SnapshotMappings string

//go:embed runtime_mappings.json
var RuntimeReportsMappings string

// SettingsLookup will keep track if an index requires special settings, add its settings template to the map.
var SettingsLookup = map[bapi.DataType]string{
	bapi.DNSLogs: DNSLogSettings,
}

// IndexPatternsPrefixLookup will keep track of the index patterns created
var IndexPatternsPrefixLookup = map[bapi.DataType]string{
	bapi.AuditEELogs:    "tigera_secure_ee_audit_ee",
	bapi.AuditKubeLogs:  "tigera_secure_ee_audit_kube",
	bapi.BGPLogs:        "tigera_secure_ee_bgp",
	bapi.FlowLogs:       "tigera_secure_ee_flows",
	bapi.L7Logs:         "tigera_secure_ee_l7",
	bapi.DNSLogs:        "tigera_secure_ee_dns",
	bapi.Events:         "tigera_secure_ee_events",
	bapi.WAFLogs:        "tigera_secure_ee_waf",
	bapi.RuntimeReports: "tigera_secure_ee_runtime",
	bapi.ReportData:     "tigera_secure_ee_compliance_reports",
	bapi.Benchmarks:     "tigera_secure_ee_benchmark_results",
	bapi.Snapshots:      "tigera_secure_ee_snapshots",
}

// TemplateNamePatternLookup will keep track of the template names created
var TemplateNamePatternLookup = map[bapi.DataType]string{
	bapi.AuditEELogs:    "tigera_secure_ee_audit_ee.%s.",
	bapi.AuditKubeLogs:  "tigera_secure_ee_audit_kube.%s.",
	bapi.BGPLogs:        "tigera_secure_ee_bgp.%s.",
	bapi.FlowLogs:       "tigera_secure_ee_flows.%s.",
	bapi.L7Logs:         "tigera_secure_ee_l7.%s.",
	bapi.DNSLogs:        "tigera_secure_ee_dns.%s.",
	bapi.Events:         "tigera_secure_ee_events.%s",
	bapi.WAFLogs:        "tigera_secure_ee_waf.%s.",
	bapi.RuntimeReports: "tigera_secure_ee_runtime.%s.",
	bapi.ReportData:     "tigera_secure_ee_compliance_reports.%s",
	bapi.Benchmarks:     "tigera_secure_ee_benchmark_results.%s",
	bapi.Snapshots:      "tigera_secure_ee_snapshots.%s",
}

// BootstrapIndexPatternLookup will keep track of the boostrap indices that will be created
var BootstrapIndexPatternLookup = map[bapi.DataType]string{
	bapi.AuditEELogs:    "<tigera_secure_ee_audit_ee.%s.fluentd-{now/s{yyyyMMdd}}-000001>",
	bapi.AuditKubeLogs:  "<tigera_secure_ee_audit_kube.%s.fluentd-{now/s{yyyyMMdd}}-000001>",
	bapi.BGPLogs:        "<tigera_secure_ee_bgp.%s.fluentd-{now/s{yyyyMMdd}}-000001>",
	bapi.FlowLogs:       "<tigera_secure_ee_flows.%s.fluentd-{now/s{yyyyMMdd}}-000001>",
	bapi.L7Logs:         "<tigera_secure_ee_l7.%s.fluentd-{now/s{yyyyMMdd}}-000001>",
	bapi.DNSLogs:        "<tigera_secure_ee_dns.%s.fluentd-{now/s{yyyyMMdd}}-000001>",
	bapi.Events:         "<tigera_secure_ee_events.%s.lma-{now/s{yyyyMMdd}}-000000>",
	bapi.WAFLogs:        "<tigera_secure_ee_waf.%s.fluentd-{now/s{yyyyMMdd}}-000001>",
	bapi.RuntimeReports: "<tigera_secure_ee_runtime.%s.fluentd-{now/s{yyyyMMdd}}-000001>",
	bapi.ReportData:     "<tigera_secure_ee_compliance_reports.%s.lma-{now/s{yyyyMMdd}}-000000>",
	bapi.Benchmarks:     "<tigera_secure_ee_benchmark_results.%s.lma-{now/s{yyyyMMdd}}-000000>",
	bapi.Snapshots:      "<tigera_secure_ee_snapshots.%s.lma-{now/s{yyyyMMdd}}-000000>",
}
