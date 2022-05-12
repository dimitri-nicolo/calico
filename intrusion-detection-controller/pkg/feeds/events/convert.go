// Copyright 2019 Tigera Inc. All rights reserved.

package events

import (
	"fmt"
	"strings"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/db"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/util"
	lmaAPI "github.com/projectcalico/calico/lma/pkg/api"
)

const (
	SuspiciousFlow     = "gtf_suspicious_flow"
	SuspiciousDNSQuery = "gtf_suspicious_dns_query"
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
		EventsData: lmaAPI.EventsData{
			Time:            flowLog.StartTime,
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
			Record: SuspiciousIPEventRecord{
				FlowAction:       flowLog.Action,
				FlowLogID:        hit.Id,
				Protocol:         flowLog.Proto,
				Feeds:            feeds,
				SuspiciousPrefix: nil,
			},
		},
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
		EventsData: lmaAPI.EventsData{
			Time:            l.StartTime.Unix(),
			Type:            SuspiciousDNSQuery,
			Description:     desc,
			Severity:        Severity,
			Origin:          feeds[0],
			SourceIP:        l.ClientIP,
			SourceNamespace: l.ClientNamespace,
			SourceName:      sname,
			SourceNameAggr:  l.ClientNameAggr,
			Record: SuspiciousDomainEventRecord{
				DNSLogID:          hit.Id,
				Feeds:             feeds,
				SuspiciousDomains: sDomains,
			},
		},
	}
}
