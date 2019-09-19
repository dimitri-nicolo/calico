// Copyright 2019 Tigera Inc. All rights reserved.

package events

import (
	"fmt"
	"strings"

	"github.com/olivere/elastic"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/util"
)

const (
	SuspiciousFlow     = "suspicious_flow"
	SuspiciousDNSQuery = "suspicious_dns_query"
	Severity           = 100
)

func ConvertFlowLog(flowLog FlowLogJSONOutput, key db.QueryKey, hit *elastic.SearchHit, feeds ...string) SuspiciousIPSecurityEvent {
	var description string
	switch key {
	case db.QueryKeyFlowLogSourceIP:
		description = fmt.Sprintf("suspicious IP %s from list %s connected to %s %s/%s", util.StringPtrWrapper{S: flowLog.SourceIP}, strings.Join(feeds, ", "), flowLog.DestType, flowLog.DestNamespace, flowLog.DestName)
	case db.QueryKeyFlowLogDestIP:
		description = fmt.Sprintf("%s %s/%s connected to suspicious IP %s from list %s", flowLog.SourceType, flowLog.SourceNamespace, flowLog.SourceName, util.StringPtrWrapper{S: flowLog.DestIP}, strings.Join(feeds, ", "))
	default:
		description = fmt.Sprintf("%s %s connected to %s %s", flowLog.SourceType, util.StringPtrWrapper{S: flowLog.SourceIP}, flowLog.DestType, util.StringPtrWrapper{S: flowLog.DestIP})
	}

	return SuspiciousIPSecurityEvent{
		Time:             flowLog.StartTime,
		Type:             SuspiciousFlow,
		Description:      description,
		Severity:         Severity,
		FlowLogIndex:     hit.Index,
		FlowLogID:        hit.Id,
		Protocol:         flowLog.Proto,
		SourceIP:         flowLog.SourceIP,
		SourcePort:       flowLog.SourcePort,
		SourceNamespace:  flowLog.SourceNamespace,
		SourceName:       flowLog.SourceName,
		DestIP:           flowLog.DestIP,
		DestPort:         flowLog.DestPort,
		DestNamespace:    flowLog.DestNamespace,
		DestName:         flowLog.DestName,
		FlowAction:       flowLog.Action,
		Feeds:            append(feeds),
		SuspiciousPrefix: nil,
	}
}

func ConvertDNSLog(l DNSLog, key db.QueryKey, hit *elastic.SearchHit, domains map[string]struct{}, feeds ...string) SuspiciousDomainSecurityEvent {
	var sname string
	if l.ClientName != "-" && l.ClientName != "" {
		sname = l.ClientName
	} else {
		sname = l.ClientNameAggr
	}

	var desc string
	var sDomains []string
	switch key {
	case db.QueryKeyDNSLogQName:
		sDomains = []string{l.QName}
		desc = fmt.Sprintf("%s/%s queried the domain name %s from global threat feed(s) %s",
			l.ClientNamespace,
			sname,
			l.QName,
			strings.Join(feeds, ", "))
	case db.QueryKeyDNSLogRRSetsName:
		// In this case, there might be more than one rrset, so we don't know which one(s) triggered
		// the search hit a priori. So, look up the names one at time.
		for _, rrset := range l.RRSets {
			if _, ok := domains[rrset.Name]; ok {
				sDomains = append(sDomains, rrset.Name)
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
		desc = fmt.Sprintf("%s/%s got DNS query results including suspicious domain(s) %s from global threat feed(s) %s",
			l.ClientNamespace,
			sname,
			strings.Join(sDomains, ", "),
			strings.Join(feeds, ", "))
	case db.QueryKeyDNSLogRRSetsRData:
		// In this case, there might be more than one rrset, so we don't know which one(s) triggered
		// the search hit a priori. So, look up the rdatas one at time.
		for _, rrset := range l.RRSets {
			for _, rdata := range rrset.RData {
				if _, ok := domains[rdata]; ok {
					sDomains = append(sDomains, rdata)
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
		desc = fmt.Sprintf("%s/%s got DNS query results including suspicious domain(s) %s from global threat feed(s) %s",
			l.ClientNamespace,
			sname,
			strings.Join(sDomains, ", "),
			strings.Join(feeds, ", "))
	}

	return SuspiciousDomainSecurityEvent{
		Time:              l.StartTime.Unix(),
		Type:              SuspiciousDNSQuery,
		Description:       desc,
		Severity:          Severity,
		DNSLogIndex:       hit.Index,
		DNSLogID:          hit.Id,
		SourceIP:          l.ClientIP,
		SourceNamespace:   l.ClientNamespace,
		SourceName:        sname,
		Feeds:             feeds,
		SuspiciousDomains: sDomains,
	}
}
