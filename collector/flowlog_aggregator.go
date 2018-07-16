// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

import (
	"sync"
	"time"

	"github.com/projectcalico/felix/rules"
	log "github.com/sirupsen/logrus"
)

// AggregationKind determines the flow log key
type AggregationKind int

const (
	// Default is based on purely duration.
	Default AggregationKind = iota
	// SourcePort accumulates tuples with everything same but the source port
	SourcePort
	// PrefixName accumulates tuples with everything same but the prefix name
	PrefixName
)

const noRuleActionDefined = 0

// cloudWatchAggregator builds and implements the FlowLogAggregator and
// FlowLogGetter interfaces.
// The cloudWatchAggregator is responsible for creating, aggregating, and storing
// aggregated flow logs until the flow logs are exported.
type cloudWatchAggregator struct {
	kind                 AggregationKind
	flowStore            map[FlowMeta]FlowStats
	flMutex              sync.RWMutex
	includeLabels        bool
	aggregationStartTime time.Time
	handledAction        rules.RuleAction
}

// NewCloudWatchAggregator constructs a FlowLogAggregator
func NewCloudWatchAggregator() FlowLogAggregator {
	return &cloudWatchAggregator{
		kind:                 Default,
		flowStore:            make(map[FlowMeta]FlowStats),
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

func (c *cloudWatchAggregator) ForAction(ra rules.RuleAction) FlowLogAggregator {
	c.handledAction = ra
	return c
}

// FeedUpdate constructs and aggregates flow logs from MetricUpdates.
func (c *cloudWatchAggregator) FeedUpdate(mu MetricUpdate) error {
	// Filter out any action that we aren't configured to handle.
	if c.handledAction != noRuleActionDefined && c.handledAction != mu.ruleID.Action {
		log.Debugf("Update %v not handled", mu)
		return nil
	}

	flowMeta, err := NewFlowMeta(mu, c.kind)
	if err != nil {
		return err
	}
	c.flMutex.Lock()
	defer c.flMutex.Unlock()
	fl, ok := c.flowStore[flowMeta]
	if !ok {
		fl = NewFlowStats(mu)
	} else {
		fl.aggregateMetricUpdate(mu)
	}
	c.flowStore[flowMeta] = fl

	return nil
}

// Get returns all aggregated flow logs, as a list of string pointers, since the last time a Get
// was called. Calling Get will also clear the stored flow logs once the flow logs are returned.
func (c *cloudWatchAggregator) Get() []*string {
	resp := make([]*string, 0, len(c.flowStore))
	aggregationEndTime := time.Now()
	c.flMutex.Lock()
	defer c.flMutex.Unlock()
	for flowMeta, flowStats := range c.flowStore {
		flowLog := FlowLog{flowMeta, flowStats}.Serialize(c.aggregationStartTime, aggregationEndTime, c.includeLabels)
		resp = append(resp, &flowLog)
		c.purge(flowMeta)
	}
	c.aggregationStartTime = aggregationEndTime
	return resp
}

func (c *cloudWatchAggregator) purge(flowMeta FlowMeta) {
	// reset flow stats for the next interval
	resetFlowStats := c.flowStore[flowMeta].reset()
	c.flowStore[flowMeta] = resetFlowStats

	// discontinue tracking the stats associated with the
	// flow meta if no more associated 5-tuples exist.
	if resetFlowStats.getFlowsCount() == 0 {
		delete(c.flowStore, flowMeta)
	}
}
