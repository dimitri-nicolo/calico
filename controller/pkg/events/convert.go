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

func ConvertFlowLog(flowLog FlowLogJSONOutput, key string, hit *elastic.SearchHit, feeds ...string) SecurityEvent {
	var description string
	switch key {
	case "source_ip":
		description = fmt.Sprintf("Suspicious IP %s from list %s connected to pod %s/%s", util.StringPtrWrapper{S: flowLog.SourceIP}, strings.Join(feeds, ", "), flowLog.SourceNamespace, flowLog.SourceName)
	case "dest_ip":
		description = fmt.Sprintf("Pod %s/%s connected to suspicious IP %s from list %s", flowLog.DestNamespace, flowLog.DestName, util.StringPtrWrapper{S: flowLog.DestIP}, strings.Join(feeds, ", "))
	default:
		description = fmt.Sprintf("%s connected to %s", util.StringPtrWrapper{S: flowLog.SourceIP}, util.StringPtrWrapper{S: flowLog.DestIP})
	}

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
