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

// SettingsLookup will keep track if an index requires special settings, add its settings template to the map.
var SettingsLookup = map[bapi.LogsType]string{
	bapi.DNSLogs: DNSLogSettings,
}

// IndexPatternsLookup will keep track of the index patterns created
var IndexPatternsLookup = map[bapi.LogsType]string{
	bapi.AuditEELogs:   "tigera_secure_ee_audit_*",
	bapi.AuditKubeLogs: "tigera_secure_ee_audit_*",
	bapi.BGPLogs:       "tigera_secure_ee_bgp*",
	bapi.FlowLogs:      "tigera_secure_ee_flows*",
	bapi.L7Logs:        "tigera_secure_ee_l7*",
	bapi.DNSLogs:       "tigera_secure_ee_dns*",
	bapi.Events:        "tigera_secure_ee_events*",
}
