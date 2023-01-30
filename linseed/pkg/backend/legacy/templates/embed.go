// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package templates

import _ "embed"

//go:embed flowlog.json
var FlowLogTemplate string

//go:embed l7log.json
var L7LogTemplate string

//go:embed dnslog.json
var DNSLogTemplate string

//go:embed audit.json
var AuditTemplate string
