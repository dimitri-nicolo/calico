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
	// PrefixName accumulates tuples with exeverything same but the prefix name
	PrefixName
)

type MetricUpdateFilter func(mu MetricUpdate) bool

func AllowActionMetricUpdateFilter(mu MetricUpdate) bool {
	return mu.ruleID.Action != rules.RuleActionAllow
}

func DenyActionMetricUpdateFilter(mu MetricUpdate) bool {
	return mu.ruleID.Action != rules.RuleActionDeny
}

// cloudWatchAggregator builds and implements the FlowLogAggregator and
// FlowLogGetter interfaces.
type cloudWatchAggregator struct {
	kind                 AggregationKind
	flowStore            map[FlowMeta]FlowStats // TODO(SS): Abstract the storage.
	flMutex              sync.RWMutex
	includeLabels        bool
	aggregationStartTime time.Time
	filter               MetricUpdateFilter
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

func (c *cloudWatchAggregator) WithFilter(f MetricUpdateFilter) FlowLogAggregator {
	c.filter = f
	return c
}

// FeedUpdate will be responsible for doing aggregation.
func (c *cloudWatchAggregator) FeedUpdate(mu MetricUpdate) error {
	if c.filter != nil && c.filter(mu) {
		log.Debugf("Update %v filtered out", mu)
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
