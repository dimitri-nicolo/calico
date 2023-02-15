// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package v1

import "time"

// WAFLogParams define querying parameters to retrieve BGP logs
type WAFLogParams struct {
	QueryParams *QueryParams `json:"query_params" validate:"required"`
}

type WAFLog struct {
	Timestamp   time.Time    `json:"@timestamp"`
	Source      *WAFEndpoint `json:"source"`
	Destination *WAFEndpoint `json:"destination"`
	Path        string       `json:"path"`
	Method      string       `json:"method"`
	Protocol    string       `json:"protocol"`
	Msg         string       `json:"msg"`
	RuleInfo    string       `json:"rule_info"`
	Node        string       `json:"node"`
}

type WAFEndpoint struct {
	IP       string `json:"ip"`
	PortNum  int32  `json:"port_num"`
	Hostname string `json:"hostname"`
}
