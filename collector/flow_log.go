// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

import (
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"strings"
	"time"

	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/rules"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	log "github.com/sirupsen/logrus"
)

type FlowLogAction string
type FlowLogDirection string
type FlowLogEndpointType string

const (
	flowLogBufferSize = 1000

	flowLogNamespaceGlobal  = "_G_"
	flowLogFieldNotIncluded = "-"

	FlowLogActionAllow FlowLogAction = "A"
	FlowLogActionDeny  FlowLogAction = "D"

	FlowLogDirectionIn  FlowLogDirection = "I"
	FlowLogDirectionOut FlowLogDirection = "O"

	FlowLogEndpointTypeWep FlowLogEndpointType = "wep"
	FlowLogEndpointTypeHep FlowLogEndpointType = "hep"
	FlowLogEndpointTypeNs  FlowLogEndpointType = "ns"
	FlowLogEndpointTypePvt FlowLogEndpointType = "pvt"
	FlowLogEndpointTypePub FlowLogEndpointType = "pub"
)

type EndpointMetadata struct {
	Type      FlowLogEndpointType `json:"type,"`
	Namespace string              `json:"namespace,"`
	Name      string              `json:"name,"`
	Labels    map[string]string   `json:"labels,"`
}

// FlowLog captures the context and information associated with
// 5-tuple update.
type FlowLog struct {
	Tuple             Tuple            `json:"srcType"`
	SrcMeta           EndpointMetadata `json:"srcMeta"`
	DstMeta           EndpointMetadata `json:"dstMeta"`
	NumFlows          int              `json:"numFlows"`
	NumFlowsStarted   int              `json:"numFlowsStarted"`
	NumFlowsCompleted int              `json:"numFlowsCompleted"`
	PacketsIn         int              `json:"packetsIn"`
	PacketsOut        int              `json:"packetsOut"`
	BytesIn           int              `json:"bytesIn"`
	BytesOut          int              `json:"bytesOut"`
	Action            FlowLogAction    `json:"action"`
	FlowDirection     FlowLogDirection `json:"flowDirection"`
}

func (f FlowLog) ToString(startTime, endTime time.Time, includeLabels bool) string {
	var (
		srcLabels, dstLabels string
		l4Src, l4Dst         string
	)

	if includeLabels {
		sl, err := json.Marshal(f.SrcMeta.Labels)
		if err == nil {
			srcLabels = string(sl)
		}
		dl, err := json.Marshal(f.DstMeta.Labels)
		if err == nil {
			dstLabels = string(dl)
		}
	} else {
		srcLabels = flowLogFieldNotIncluded
		dstLabels = flowLogFieldNotIncluded
	}

	srcIP := net.IP(f.Tuple.src[:16]).String()
	dstIP := net.IP(f.Tuple.dst[:16]).String()

	if f.Tuple.proto != 1 {
		l4Src = fmt.Sprintf("%d", f.Tuple.l4Src)
		l4Dst = fmt.Sprintf("%d", f.Tuple.l4Dst)
	} else {
		l4Src = flowLogFieldNotIncluded
		l4Dst = flowLogFieldNotIncluded
	}

	// Format is
	// startTime endTime srcType srcNamespace srcName srcLabels dstType dstNamespace dstName dstLabels srcIP dstIP proto srcPort dstPort numFlows numFlowsStarted numFlowsCompleted flowDirection packetsIn packetsOut bytesIn bytesOut action
	return fmt.Sprintf("%v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v",
		startTime.Unix(), endTime.Unix(),
		f.SrcMeta.Type, f.SrcMeta.Namespace, f.SrcMeta.Name, srcLabels,
		f.DstMeta.Type, f.DstMeta.Namespace, f.DstMeta.Name, dstLabels,
		srcIP, dstIP, f.Tuple.proto, l4Src, l4Dst,
		f.NumFlows, f.NumFlowsStarted, f.NumFlowsCompleted, f.FlowDirection,
		f.PacketsIn, f.PacketsOut, f.BytesIn, f.BytesOut,
		f.Action)
}

func (f *FlowLog) aggregateMetricUpdate(mu MetricUpdate) error {
	// TODO(doublek): Handle metadata updates.
	if mu.updateType == UpdateTypeExpire {
		f.NumFlowsCompleted++
	}
	if mu.isInitialReport {
		f.NumFlows++
		f.NumFlowsStarted++
	}
	f.PacketsIn += mu.inMetric.deltaPackets
	f.BytesIn += mu.inMetric.deltaBytes
	f.PacketsOut += mu.outMetric.deltaPackets
	f.BytesOut += mu.outMetric.deltaBytes
	return nil
}

func deconstructNameAndNamespaceFromWepName(wepName string) (string, string, error) {
	parts := strings.Split(wepName, "/")
	if len(parts) == 2 {
		return parts[0], parts[1], nil
	}
	return "", "", fmt.Errorf("Could not parse name %v", wepName)
}

func getFlowLogEndpointMetadata(ed *calc.EndpointData) (EndpointMetadata, error) {
	var (
		em  EndpointMetadata
		err error
	)
	switch k := ed.Key.(type) {
	case model.WorkloadEndpointKey:
		name, ns, err := deconstructNameAndNamespaceFromWepName(k.WorkloadID)
		if err != nil {
			return EndpointMetadata{}, err
		}
		v := ed.Endpoint.(*model.WorkloadEndpoint)
		em = EndpointMetadata{
			Type:      FlowLogEndpointTypeWep,
			Name:      name,
			Namespace: ns,
			Labels:    v.Labels,
		}
	case model.HostEndpointKey:
		v := ed.Endpoint.(*model.HostEndpoint)
		em = EndpointMetadata{
			Type:      FlowLogEndpointTypeHep,
			Name:      k.EndpointID,
			Namespace: flowLogNamespaceGlobal,
			Labels:    v.Labels,
		}
	default:
		err = fmt.Errorf("Unknown key %#v of type %v", ed.Key, reflect.TypeOf(ed.Key))
	}
	return em, err
}

func getFlowLogFromMetricUpdate(mu MetricUpdate) (FlowLog, error) {
	var (
		srcMeta, dstMeta EndpointMetadata
		err              error
	)
	if mu.srcEp != nil {
		srcMeta, err = getFlowLogEndpointMetadata(mu.srcEp)
		if err != nil {
			log.WithError(err).Errorf("Could not extract metadata for source %v", mu.srcEp)
			return FlowLog{}, err
		}
	}
	if mu.dstEp != nil {
		dstMeta, err = getFlowLogEndpointMetadata(mu.dstEp)
		if err != nil {
			log.WithError(err).Errorf("Could not extract metadata for destination %v", mu.dstEp)
			return FlowLog{}, err
		}
	}

	var nf, nfs, nfc int
	switch mu.updateType {
	case UpdateTypeReport:
		nfs = 1
	case UpdateTypeExpire:
		nfc = 1
	}
	// 1 always when we create the flow
	nf = 1

	action, flowDir := getFlowLogActionAndDirFromRuleID(mu.ruleID)

	return FlowLog{
		Tuple:             mu.tuple,
		SrcMeta:           srcMeta,
		DstMeta:           dstMeta,
		NumFlows:          nf,
		NumFlowsStarted:   nfs,
		NumFlowsCompleted: nfc,
		PacketsIn:         mu.inMetric.deltaPackets,
		BytesIn:           mu.inMetric.deltaBytes,
		PacketsOut:        mu.outMetric.deltaPackets,
		BytesOut:          mu.outMetric.deltaBytes,
		Action:            action,
		FlowDirection:     flowDir,
	}, nil
}

// getFlowLogActionAndDirFromRuleID converts the action to a string value.
func getFlowLogActionAndDirFromRuleID(r *calc.RuleID) (fla FlowLogAction, fld FlowLogDirection) {
	switch r.Action {
	case rules.RuleActionDeny:
		fla = FlowLogActionDeny
	case rules.RuleActionAllow:
		fla = FlowLogActionAllow
	}
	switch r.Direction {
	case rules.RuleDirIngress:
		fld = FlowLogDirectionIn
	case rules.RuleDirEgress:
		fld = FlowLogDirectionOut
	}
	return
}
