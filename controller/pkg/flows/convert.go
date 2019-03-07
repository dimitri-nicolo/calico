package flows

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

func ConvertFlowLog(flowLog FlowLogJSONOutput, hit *elastic.SearchHit, feeds ...string) FlowLog {
	description := fmt.Sprintf("Pod %s/%s connected to suspicious IP from list %s", flowLog.SourceNamespace, flowLog.SourceName, strings.Join(feeds, ", "))

	id := fmt.Sprintf("%d-%s-%s-%s-%s-%s",
		flowLog.StartTime,
		flowLog.Proto,
		util.StringPtrWrapper{flowLog.SourceIP},
		util.Int64PtrWrapper{flowLog.SourcePort},
		util.StringPtrWrapper{flowLog.DestIP},
		util.Int64PtrWrapper{flowLog.DestPort},
	)

	return FlowLog{
		Time:             flowLog.StartTime,
		Type:             SuspiciousFlow,
		Description:      description,
		Severity:         Severity,
		ID:               id,
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
