// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

import (
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
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
	tuple     Tuple
	action    FlowLogAction
	direction FlowLogDirection
}

// cloudWatchAggregator builds and implements the FlowLogAggregator and
// FlowLogGetter interfaces.
type cloudWatchAggregator struct {
	kind                 AggregationKind
	flowLogs             map[FlowLogKey]FlowLog
	flMutex              sync.RWMutex
	includeLabels        bool
	aggregationStartTime time.Time
}

// NewCloudWatchAggregator constructs a FlowLogAggregator
func NewCloudWatchAggregator() FlowLogAggregator {
	return &cloudWatchAggregator{
		kind:                 Default,
		flowLogs:             make(map[FlowLogKey]FlowLog),
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

// FeedUpdate will be responsible for doing aggregation.
func (c *cloudWatchAggregator) FeedUpdate(mu MetricUpdate) error {
	var err error

	fla, fld := getFlowLogActionAndDirFromRuleID(mu.ruleID)
	flKey := FlowLogKey{
		tuple:     getTupleForAggreagation(mu.tuple, c.kind),
		action:    fla,
		direction: fld,
	}

	c.flMutex.Lock()
	defer c.flMutex.Unlock()
	fl, ok := c.flowLogs[flKey]
	if !ok {
		log.Infof("New Key %+v", flKey)
		fl, err = getFlowLogFromMetricUpdate(mu)
		if err != nil {
			log.WithError(err).Errorf("Could not convert MetricUpdate %v to Flow log", mu)
			return err
		}
	} else {
		log.Infof("Existing Key %+v", flKey)
		err = fl.aggregateMetricUpdate(mu)
		if err != nil {
			log.WithError(err).Errorf("Could not aggregated MetricUpdate %v to Flow log %v", mu, fl)
			return err
		}
	}
	c.flowLogs[flKey] = fl
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

func getTupleForAggreagation(orig Tuple, kind AggregationKind) Tuple {
	var aggTuple Tuple
	switch kind {
	case Default:
		aggTuple = orig
	case SourcePort:
		// "4-tuple"
		aggTuple = Tuple{
			src:   orig.src,
			dst:   orig.dst,
			proto: orig.proto,
			l4Dst: orig.l4Dst,
		}
	case PrefixName:
		// only destination port survives the aggregation.
		aggTuple = Tuple{
			l4Dst: orig.l4Dst,
		}
	}
	return aggTuple
}
