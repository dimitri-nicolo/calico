// Copyright 2019 Tigera Inc. All rights reserved.

package events

import (
	"fmt"
	"strings"

	"github.com/tigera/intrusion-detection/controller/pkg/util"
)

type SuspiciousIPSecurityEvent struct {
	Time             int64    `json:"time"`
	Type             string   `json:"type"`
	Description      string   `json:"description"`
	Severity         int      `json:"severity"`
	FlowLogIndex     string   `json:"flow_log_index"`
	FlowLogID        string   `json:"flow_log_id"`
	Protocol         string   `json:"protocol"`
	SourceIP         *string  `json:"source_ip"`
	SourcePort       *int64   `json:"source_port"`
	SourceNamespace  string   `json:"source_namespace"`
	SourceName       string   `json:"source_name"`
	DestIP           *string  `json:"dest_ip"`
	DestPort         *int64   `json:"dest_port"`
	DestNamespace    string   `json:"dest_namespace"`
	DestName         string   `json:"dest_name"`
	FlowAction       string   `json:"flow_action"`
	Feeds            []string `json:"feeds,omitempty"`
	SuspiciousPrefix *string  `json:"suspicious_prefix"`
}

func (s SuspiciousIPSecurityEvent) ID() string {
	feed := "unknown"
	if len(s.Feeds) > 0 {
		// Use ~ as separator because it's allowed in URLs, but not in feed names (which are K8s names)
		feed = strings.Join(s.Feeds, "~")
	}
	// Use _ as a separator because it's allowed in URLs, but not in any of the components of this ID
	return fmt.Sprintf("%s_%d_%s_%s_%s_%s_%s",
		feed,
		s.Time,
		s.Protocol,
		util.StringPtrWrapper{S: s.SourceIP},
		util.Int64PtrWrapper{I: s.SourcePort},
		util.StringPtrWrapper{S: s.DestIP},
		util.Int64PtrWrapper{I: s.DestPort},
	)
}

type SuspiciousDomainSecurityEvent struct {
	Time              int64    `json:"time"`
	Type              string   `json:"type"`
	Description       string   `json:"description"`
	Severity          int      `json:"severity"`
	DNSLogIndex       string   `json:"dns_log_index"`
	DNSLogID          string   `json:"dns_log_id"`
	SourceIP          *string  `json:"source_ip"`
	SourceNamespace   string   `json:"source_namespace"`
	SourceName        string   `json:"source_name"`
	Feeds             []string `json:"feeds,omitempty"`
	SuspiciousDomains []string `json:"suspicious_domains"`
}

func (s SuspiciousDomainSecurityEvent) ID() string {
	feed := "unknown"
	if len(s.Feeds) > 0 {
		// Use ~ as separator because it's allowed in URLs, but not in feed names (which are K8s names)
		feed = strings.Join(s.Feeds, "~")
	}
	domains := "unknown"
	if len(s.SuspiciousDomains) > 0 {
		// Use ~ as a separator because it's allowed in URLs, but not in domain names
		domains = strings.Join(s.SuspiciousDomains, "~")
	}
	// Use _ as a separator because it's allowed in URLs, but not in any of the components of this ID
	return fmt.Sprintf("%s_%d_%s_%s",
		feed,
		s.Time,
		util.StringPtrWrapper{S: s.SourceIP},
		domains,
	)
}
