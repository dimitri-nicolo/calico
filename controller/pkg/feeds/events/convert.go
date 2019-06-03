// Copyright 2019 Tigera Inc. All rights reserved.

package events

import (
	"fmt"
	"strings"

	"github.com/olivere/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/util"
)

const (
	SuspiciousFlow = "suspicious_flow"
	Severity       = 100
)

func ConvertFlowLog(flowLog FlowLogJSONOutput, key string, hit *elastic.SearchHit, feeds ...string) SuspiciousIPSecurityEvent {
	var description string
	switch key {
	case "source_ip":
		description = fmt.Sprintf("suspicious IP %s from list %s connected to %s %s/%s", util.StringPtrWrapper{S: flowLog.SourceIP}, strings.Join(feeds, ", "), flowLog.DestType, flowLog.DestNamespace, flowLog.DestName)
	case "dest_ip":
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
