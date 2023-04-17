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
