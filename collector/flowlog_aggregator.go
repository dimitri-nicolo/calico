// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/rules"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	log "github.com/sirupsen/logrus"
)

// AggregationKind determines the flow log key
type AggregationKind int

const (
	// Default is based on purely duration.
	Default AggregationKind = iota
	// SourcePort accumulates tuples with everything same but the source port
	SourcePort
	// PrefixName accumulates tuples with exeverything same but the prefix name
	PrefixName
)

type FlowLogKey struct {
	tuple Tuple
	kind  AggregationKind
}

func (f FlowLogKey) String() string {
	switch f.kind {
	case Default:
		return f.tuple.String()
	case SourcePort:
		return fmt.Sprintf("src=%v dst=%v proto=%v dport=%v", net.IP(f.tuple.src[:16]).String(), net.IP(f.tuple.dst[:16]).String(), f.tuple.proto, f.tuple.l4Dst)
	case PrefixName:
		return "todo" //TODO
	}
	return ""
}

// cloudWatchAggregator builds and implements the FlowLogAggregator and
// FlowLogGetter interfaces.
type cloudWatchAggregator struct {
	kind                 AggregationKind
	flowLogs             map[string]FlowLog
	flMutex              sync.RWMutex
	includeLabels        bool
	aggregationStartTime time.Time
}

// NewCloudWatchAggregator constructs a FlowLogAggregator
func NewCloudWatchAggregator() FlowLogAggregator {
	return &cloudWatchAggregator{
		kind:                 Default,
		flowLogs:             make(map[string]FlowLog),
		flMutex:              sync.RWMutex{},
		aggregationStartTime: time.Now(),
	}
}

func (c *cloudWatchAggregator) AggregateOver(kind AggregationKind) FlowLogAggregator {
	c.kind = kind
	return c
}

func (c *cloudWatchAggregator) IncludeLabels(b bool) FlowLogAggregator {
	c.includeLabels = b
	return c
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

// FeedUpdate will be responsible for doing aggregation.
func (c *cloudWatchAggregator) FeedUpdate(mu MetricUpdate) error {
	c.flMutex.Lock()
	defer c.flMutex.Unlock()
	var err error
	// TODO: Key construction isn't the most optimal. Revisit.
	flKey := FlowLogKey{mu.tuple, c.kind}.String()
	fl, ok := c.flowLogs[flKey]
	if !ok {
		fl, err = getFlowLogFromMetricUpdate(mu)
		if err != nil {
			log.WithError(err).Errorf("Could not convert MetricUpdate %v to Flow log", mu)
			return err
		}
		c.flowLogs[flKey] = fl
	} else {
		err = fl.aggregateMetricUpdate(mu)
		if err != nil {
			log.WithError(err).Errorf("Could not aggregated MetricUpdate %v to Flow log %v", mu, fl)
			return err
		}
	}
	return nil
}

func (c *cloudWatchAggregator) Get() []*string {
	resp := make([]*string, 0, len(c.flowLogs))
	aggregationEndTime := time.Now()
	c.flMutex.Lock()
	for k, flowLog := range c.flowLogs {
		resp = append(resp, aws.String(flowLog.ToString(c.aggregationStartTime, aggregationEndTime, c.includeLabels)))
		delete(c.flowLogs, k)
	}
	c.flMutex.Unlock()
	c.aggregationStartTime = aggregationEndTime
	return resp
}
