package events

import (
	"fmt"
	"strings"

	"github.com/olivere/elastic"
)

const (
	SuspiciousFlow = "suspicious_flow"
	Severity       = 100
)

func ConvertFlowLog(flowLog FlowLogJSONOutput, hit *elastic.SearchHit, feeds ...string) SecurityEvent {
	description := fmt.Sprintf("Pod %s/%s connected to suspicious IP from list %s", flowLog.SourceNamespace, flowLog.SourceName, strings.Join(feeds, ", "))

	return SecurityEvent{
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
