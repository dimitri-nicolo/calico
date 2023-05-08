// Copyright 2019 Tigera Inc. All rights reserved.

package events

import (
	"fmt"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/storage"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/util"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

const (
	SuspiciousFlow     = "gtf_suspicious_flow"
	SuspiciousDNSQuery = "gtf_suspicious_dns_query"
	Severity           = 100
)

func ConvertFlowLog(flowLog v1.FlowLog, key storage.QueryKey, feeds ...string) v1.Event {
	var description string
	switch key {
	case storage.QueryKeyFlowLogSourceIP:
		description = fmt.Sprintf("suspicious IP %s from list %s connected to %s %s/%s", util.StringPtrWrapper{S: flowLog.SourceIP}, strings.Join(feeds, ", "), flowLog.DestType, flowLog.DestNamespace, flowLog.DestName)
	case storage.QueryKeyFlowLogDestIP:
		description = fmt.Sprintf("%s %s/%s connected to suspicious IP %s from list %s", flowLog.SourceType, flowLog.SourceNamespace, flowLog.SourceName, util.StringPtrWrapper{S: flowLog.DestIP}, strings.Join(feeds, ", "))
	default:
		description = fmt.Sprintf("%s %s connected to %s %s", flowLog.SourceType, util.StringPtrWrapper{S: flowLog.SourceIP}, flowLog.DestType, util.StringPtrWrapper{S: flowLog.DestIP})
	}

	record := v1.SuspiciousIPEventRecord{
		FlowAction:       flowLog.Action,
		FlowLogID:        flowLog.ID,
		Protocol:         flowLog.Protocol,
		Feeds:            feeds,
		SuspiciousPrefix: nil,
	}

	return v1.Event{
		ID:              generateSuspicousIPSetID(flowLog.StartTime, flowLog.SourceIP, flowLog.SourcePort, flowLog.DestIP, flowLog.DestPort, record),
		Time:            v1.NewEventTimestamp(flowLog.StartTime),
		Type:            SuspiciousFlow,
		Description:     description,
		Severity:        Severity,
		Origin:          feeds[0],
		SourceIP:        flowLog.SourceIP,
		SourcePort:      flowLog.SourcePort,
		SourceNamespace: flowLog.SourceNamespace,
		SourceName:      flowLog.SourceName,
		SourceNameAggr:  flowLog.SourceNameAggr,
		DestIP:          flowLog.DestIP,
		DestPort:        flowLog.DestPort,
		DestNamespace:   flowLog.DestNamespace,
		DestName:        flowLog.DestName,
		DestNameAggr:    flowLog.DestNameAggr,
		Record:          record,
	}
}

func generateSuspicousIPSetID(startTime int64, sourceIP *string, sourcePort *int64, destinationIP *string, destinationPort *int64, record v1.SuspiciousIPEventRecord) string {
	feed := "unknown"
	if len(record.Feeds) > 0 {
		// Use ~ as separator because it's allowed in URLs, but not in feed names (which are K8s names)
		feed = strings.Join(record.Feeds, "~")
	}
	// Use _ as a separator because it's allowed in URLs, but not in any of the components of this ID
	return fmt.Sprintf("%s_%d_%s_%s_%s_%s_%s",
		feed,
		startTime,
		record.Protocol,
		util.StringPtrWrapper{S: sourceIP},
		util.Int64PtrWrapper{I: sourcePort},
		util.StringPtrWrapper{S: destinationIP},
		util.Int64PtrWrapper{I: destinationPort},
	)
}

func ConvertDNSLog(l v1.DNSLog, key storage.QueryKey, domains map[string]struct{}, feeds ...string) v1.Event {
	var sname string
	if l.ClientName != "-" && l.ClientName != "" {
		sname = l.ClientName
	} else {
		sname = l.ClientNameAggr
	}

	var desc string
	var sDomains []string
	switch key {
	case storage.QueryKeyDNSLogQName:
		sDomains = []string{string(l.QName)}
		desc = fmt.Sprintf("%s/%s queried the domain name %s from global threat feed(s) %s",
			l.ClientNamespace,
			sname,
			l.QName,
			strings.Join(feeds, ", "))
	case storage.QueryKeyDNSLogRRSetsName:
		// In this case, there might be more than one rrset, so we don't know which one(s) triggered
		// the search hit a priori. So, look up the names one at time.
		for dnsName := range l.RRSets {
			if _, ok := domains[dnsName.Name]; ok {
				sDomains = append(sDomains, dnsName.Name)
			}
		}
		if len(sDomains) == 0 {
			// This shouldn't happen, and means that none of the rrset names was in our feed list. This indicates
			// an upstream problem with our query, or elasticsearch itself.
			log.WithFields(log.Fields{
				"feeds":  feeds,
				"rrsets": l.RRSets,
			}).Warn("couldn't determine which rrset.name triggered suspicious domains search hit")

			// But, press on anyway.
		}
		sort.Strings(sDomains)
		desc = fmt.Sprintf("%s/%s got DNS query results including suspicious domain(s) %s from global threat feed(s) %s",
			l.ClientNamespace,
			sname,
			strings.Join(sDomains, ", "),
			strings.Join(feeds, ", "))
	case storage.QueryKeyDNSLogRRSetsRData:
		// In this case, there might be more than one rrset, so we don't know which one(s) triggered
		// the search hit a priori. So, look up the rdatas one at time.
		for _, rdatas := range l.RRSets {
			for _, rdata := range rdatas {
				if _, ok := domains[string(rdata.Raw)]; ok {
					sDomains = append(sDomains, string(rdata.Raw))
				}
			}
		}
		if len(sDomains) == 0 {
			// This shouldn't happen, and means that none of the rdatas was in our feed list. This indicates
			// an upstream problem with our query, or elasticsearch itself.
			log.WithFields(log.Fields{
				"feeds":  feeds,
				"rrsets": l.RRSets,
			}).Warn("couldn't determine which rrset.rdata triggered suspicious domains search hit")

			// But, press on anyway.
		}
		sort.Strings(sDomains)
		desc = fmt.Sprintf("%s/%s got DNS query results including suspicious domain(s) %s from global threat feed(s) %s",
			l.ClientNamespace,
			sname,
			strings.Join(sDomains, ", "),
			strings.Join(feeds, ", "))
	}

	record := v1.SuspiciousDomainEventRecord{
		DNSLogID:          l.ID,
		Feeds:             feeds,
		SuspiciousDomains: sDomains,
	}
	startTime := l.StartTime.Unix()
	return v1.Event{
		ID:              generateSuspiciousDNSDomainID(startTime, util.StrPtr(l.ClientIP), record),
		Time:            v1.NewEventTimestamp(startTime),
		Type:            SuspiciousDNSQuery,
		Description:     desc,
		Severity:        Severity,
		Origin:          feeds[0],
		SourceIP:        util.StrPtr(l.ClientIP),
		SourceNamespace: l.ClientNamespace,
		SourceName:      l.ClientName,
		SourceNameAggr:  l.ClientNameAggr,
		Record:          record,
	}
}

func generateSuspiciousDNSDomainID(startTime int64, sourceIP *string, record v1.SuspiciousDomainEventRecord) string {
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
		startTime,
		util.StringPtrWrapper{S: sourceIP},
		domains,
	)
}
