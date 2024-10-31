// Copyright 2019 Tigera Inc. All rights reserved.

package events

import (
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	geodb "github.com/projectcalico/calico/intrusion-detection-controller/pkg/feeds/geodb"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/storage"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/util"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

const (
	SuspiciousFlow         = "gtf_suspicious_flow"
	SuspiciousDNSQuery     = "gtf_suspicious_dns_query"
	SuspiciousFlowName     = "Suspicious Flow"
	SuspiciousDnsQueryName = "Suspicious DNS Query"
	Severity               = 100
)

func ConvertFlowLog(flowLog v1.FlowLog, key storage.QueryKey, geoDB geodb.GeoDatabase, feeds ...string) v1.Event {
	var description string
	switch key {
	case storage.QueryKeyFlowLogSourceIP:
		description = fmt.Sprintf("suspicious IP %s, listed in Global Threat Feed %s, connected to %s/%s", util.StringPtrWrapper{S: flowLog.SourceIP}, strings.Join(feeds, ", "), flowLog.DestNamespace, flowLog.DestName)
	case storage.QueryKeyFlowLogDestIP:
		description = fmt.Sprintf("pod %s/%s connected to suspicious IP %s which is listed in Global Threat Feed %s", flowLog.SourceNamespace, flowLog.SourceName, util.StringPtrWrapper{S: flowLog.DestIP}, strings.Join(feeds, ", "))
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

	var mitreID []string
	var mitreTactic string
	if flowLog.Reporter == "dst" {
		mitreID = []string{"T1090"}
		mitreTactic = "Command and Control"
	} else {
		mitreID = []string{"T1041"}
		mitreTactic = "Exfiltration"
	}

	var mitigations []string
	if record.FlowAction == "deny" {
		mitigations = []string{"No mitigation needed. This network traffic was blocked by Calico"}
	} else {
		if flowLog.Reporter == "dst" {
			mitigations = []string{"Create a global network policy to prevent traffic to this IP address"}
		} else {
			mitigations = []string{"Create a global network policy to prevent traffic from this IP address"}
		}
	}

	parsedIP := net.ParseIP(*flowLog.DestIP)
	geoInfo, err := geoDB.City(parsedIP)
	if err != nil {
		log.WithFields(log.Fields{
			"feeds": feeds,
		}).Error("[Global Threat Feeds] Could not find City/Country of origin for destination IP")
	}

	geoInfo.ASN, err = geoDB.ASN(parsedIP)
	if err != nil {
		log.WithFields(log.Fields{
			"feeds": feeds,
		}).Warn("[Global Threat Feeds] Could not find ASN for destination IP")
	}

	return v1.Event{
		ID:              generateSuspicousIPSetID(flowLog.StartTime, flowLog.SourceIP, flowLog.SourcePort, flowLog.DestIP, flowLog.DestPort, record),
		Time:            v1.NewEventTimestamp(time.Now().Unix()),
		Type:            SuspiciousFlow,
		Description:     description,
		Severity:        Severity,
		Origin:          SuspiciousFlowName,
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
		GeoInfo:         geoInfo,

		Name:         SuspiciousFlowName,
		AttackVector: "Network",
		MitreIDs:     &mitreID,
		Mitigations:  &mitigations,
		MitreTactic:  mitreTactic,
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
	var sourceName string
	if l.ClientName != "-" && l.ClientName != "" {
		sourceName = l.ClientName
	} else {
		sourceName = l.ClientNameAggr

	}

	var desc string
	var sDomains []string
	switch key {
	case storage.QueryKeyDNSLogQName:
		sDomains = []string{string(l.QName)}
		desc = fmt.Sprintf("%s/%s queried the domain name %s from global threat feed(s) %s",
			l.ClientNamespace,
			sourceName,
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
			// an upstream problem with our query, or linseed itself.
			log.WithFields(log.Fields{
				"feeds":  feeds,
				"rrsets": l.RRSets,
			}).Warn("[Global Threat Feeds] couldn't determine which rrset.name triggered suspicious domains search hit")

			// But, press on anyway.
		}
		sort.Strings(sDomains)
		desc = fmt.Sprintf("A request originating from %v/%v queried the domain name %v, which is listed in the threat feed %v",
			l.ClientNamespace,
			sourceName,
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
			// an upstream problem with our query, or linseed itself.
			log.WithFields(log.Fields{
				"feeds":  feeds,
				"rrsets": l.RRSets,
			}).Warn("[Global Threat Feeds] couldn't determine which rrset.rdata triggered suspicious domains search hit")

			// But, press on anyway.
		}
		sort.Strings(sDomains)
		desc = fmt.Sprintf("%s/%s got DNS query results including suspicious domain(s) %s from global threat feed(s) %s",
			l.ClientNamespace,
			sourceName,
			strings.Join(sDomains, ", "),
			strings.Join(feeds, ", "))
	}

	record := v1.SuspiciousDomainEventRecord{
		DNSLogID:          l.ID,
		Feeds:             feeds,
		SuspiciousDomains: sDomains,
	}

	if strings.HasSuffix(sourceName, "*") {
		// change the source name back to '-' if it's part of a deployment
		sourceName = "-"
	}

	startTime := l.StartTime.Unix()
	return v1.Event{
		ID:              generateSuspiciousDNSDomainID(startTime, util.StrPtr(l.ClientIP), record),
		Time:            v1.NewEventTimestamp(time.Now().Unix()),
		Type:            SuspiciousDNSQuery,
		Description:     desc,
		Severity:        Severity,
		Origin:          SuspiciousDnsQueryName,
		SourceIP:        util.StrPtr(l.ClientIP),
		SourceNamespace: l.ClientNamespace,
		SourceName:      sourceName,
		SourceNameAggr:  l.ClientNameAggr,
		Record:          record,

		Name:         SuspiciousDnsQueryName,
		AttackVector: "Network",
		MitreIDs:     &[]string{"T1041"},
		Mitigations:  &[]string{"Create a global network policy to prevent traffic from this IP address"},
		MitreTactic:  "Exfiltration",
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
