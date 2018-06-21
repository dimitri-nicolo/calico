// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

import (
	"sync"
	"time"
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

// cloudWatchAggregator builds and implements the FlowLogAggregator and
// FlowLogGetter interfaces.
type cloudWatchAggregator struct {
	kind                 AggregationKind
	flowStore            map[FlowMeta]FlowStats // TODO(SS): Abstract the storage.
	flMutex              sync.RWMutex
	includeLabels        bool
	aggregationStartTime time.Time
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

// FeedUpdate will be responsible for doing aggregation.
func (c *cloudWatchAggregator) FeedUpdate(mu MetricUpdate) error {
	var err error

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

func (c *cloudWatchAggregator) Get() []*string {
	resp := make([]*string, 0, len(c.flowStore))
	aggregationEndTime := time.Now()
	c.flMutex.Lock()
	defer c.flMutex.Unlock()
	for flowMeta, flowStats := range c.flowStore {
		flowLog := FlowLog{flowMeta, flowStats}.Serialize(c.aggregationStartTime, aggregationEndTime, c.includeLabels, c.kind)
		resp = append(resp, &flowLog)
		delete(c.flowStore, flowMeta)
	}
	c.aggregationStartTime = aggregationEndTime
	return resp
}
