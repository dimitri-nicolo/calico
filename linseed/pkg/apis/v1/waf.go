// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package v1

import "time"

// WAFLogParams define querying parameters to retrieve BGP logs
type WAFLogParams struct {
	QueryParams `json:",inline" validate:"required"`
}

type WAFLog struct {
	Timestamp   time.Time    `json:"@timestamp"`
	Destination *WAFEndpoint `json:"destination"`
	Level       string       `json:"level"`
	Method      string       `json:"method"`
	Msg         string       `json:"msg"`
	Path        string       `json:"path"`
	Protocol    string       `json:"protocol"`
	RuleInfo    string       `json:"rule_info"`
	Source      *WAFEndpoint `json:"source"`
	Host        string       `json:"host"`
}

type WAFEndpoint struct {
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
	PortNum  int32  `json:"port_num"`
}
