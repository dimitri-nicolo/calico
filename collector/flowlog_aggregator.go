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

type FlowLogMeta struct {
	tuple     Tuple
	action    FlowLogAction
	direction FlowLogDirection
}

// cloudWatchAggregator builds and implements the FlowLogAggregator and
// FlowLogGetter interfaces.
type cloudWatchAggregator struct {
	kind                 AggregationKind
	flowLogs             map[FlowLogMeta]FlowLog
	flMutex              sync.RWMutex
	includeLabels        bool
	aggregationStartTime time.Time
}

// NewCloudWatchAggregator constructs a FlowLogAggregator
func NewCloudWatchAggregator() FlowLogAggregator {
	return &cloudWatchAggregator{
		kind:                 Default,
		flowLogs:             make(map[FlowLogMeta]FlowLog),
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
	flKey := FlowLogMeta{
		tuple:     getTupleForAggreagation(mu.tuple, c.kind),
		action:    fla,
		direction: fld,
	}

	c.flMutex.Lock()
	defer c.flMutex.Unlock()
	fl, ok := c.flowLogs[flKey]
	if !ok {
		fl, err = getFlowLogFromMetricUpdate(mu, c.kind)
		if err != nil {
			log.WithError(err).Errorf("Could not convert MetricUpdate %v to Flow log", mu)
			return err
		}
	} else {
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
	defer c.flMutex.Unlock()
	for k, flowLog := range c.flowLogs {
		resp = append(resp, aws.String(flowLog.ToString(c.aggregationStartTime, aggregationEndTime, c.includeLabels)))
		delete(c.flowLogs, k)
	}
	c.aggregationStartTime = aggregationEndTime
	return resp
}
