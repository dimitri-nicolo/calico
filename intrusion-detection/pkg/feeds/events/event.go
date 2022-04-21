// Copyright 2019 Tigera Inc. All rights reserved.

package events

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/intrusion-detection/controller/pkg/util"
	lmaAPI "github.com/projectcalico/calico/lma/pkg/api"
)

type SuspiciousIPSecurityEvent struct {
	lmaAPI.EventsData
}

type SuspiciousIPEventRecord struct {
	FlowAction       string   `json:"flow_action"`
	FlowLogID        string   `json:"flow_log_id"`
	Protocol         string   `json:"protocol"`
	Feeds            []string `json:"feeds,omitempty"`
	SuspiciousPrefix *string  `json:"suspicious_prefix"`
}

func (s SuspiciousIPSecurityEvent) GetEventsData() lmaAPI.EventsData {
	return s.EventsData
}

func (s SuspiciousIPSecurityEvent) GetID() string {
	record, ok := s.Record.(SuspiciousIPEventRecord)
	if !ok {
		log.Error("Type caset failed to get ID")
	}

	feed := "unknown"
	if len(record.Feeds) > 0 {
		// Use ~ as separator because it's allowed in URLs, but not in feed names (which are K8s names)
		feed = strings.Join(record.Feeds, "~")
	}
	// Use _ as a separator because it's allowed in URLs, but not in any of the components of this ID
	return fmt.Sprintf("%s_%d_%s_%s_%s_%s_%s",
		feed,
		s.Time,
		record.Protocol,
		util.StringPtrWrapper{S: s.SourceIP},
		util.Int64PtrWrapper{I: s.SourcePort},
		util.StringPtrWrapper{S: s.DestIP},
		util.Int64PtrWrapper{I: s.DestPort},
	)
}

type SuspiciousDomainSecurityEvent struct {
	lmaAPI.EventsData
}

type SuspiciousDomainEventRecord struct {
	DNSLogID          string   `json:"dns_log_id"`
	Feeds             []string `json:"feeds,omitempty"`
	SuspiciousDomains []string `json:"suspicious_domains"`
}

func (s SuspiciousDomainSecurityEvent) GetEventsData() lmaAPI.EventsData {
	return s.EventsData
}

func (s SuspiciousDomainSecurityEvent) GetID() string {
	record, ok := s.Record.(SuspiciousDomainEventRecord)
	if !ok {
		log.Error("Type caset failed to get ID")
	}

	feed := "unknown"
	if len(record.Feeds) > 0 {
		// Use ~ as separator because it's allowed in URLs, but not in feed names (which are K8s names)
		feed = strings.Join(record.Feeds, "~")
	}
	domains := "unknown"
	if len(record.SuspiciousDomains) > 0 {
		// Use ~ as a separator because it's allowed in URLs, but not in domain names
		domains = strings.Join(record.SuspiciousDomains, "~")
	}
	// Use _ as a separator because it's allowed in URLs, but not in any of the components of this ID
	return fmt.Sprintf("%s_%d_%s_%s",
		feed,
		s.Time,
		util.StringPtrWrapper{S: s.SourceIP},
		domains,
	)
}
