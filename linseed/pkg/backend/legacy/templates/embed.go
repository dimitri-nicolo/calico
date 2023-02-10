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

// SettingsLookup will keep track if an index requires special settings, add its settings template to the map.
var SettingsLookup = map[bapi.LogsType]string{
	bapi.DNSLogs: DNSLogSettings,
}
