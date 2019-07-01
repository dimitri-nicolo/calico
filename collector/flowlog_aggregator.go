// Copyright (c) 2018-2019 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/rules"
)

// FlowAggregationKind determines the flow log key
type FlowAggregationKind int

const (
	// FlowDefault is based on purely duration.
	FlowDefault FlowAggregationKind = iota
	// FlowSourcePort accumulates tuples with everything same but the source port
	FlowSourcePort
	// FlowPrefixName accumulates tuples with everything same but the prefix name
	FlowPrefixName
)

const (
	noRuleActionDefined  = 0
	defaultMaxOrigIPSize = 50
)

// flowLogAggregator builds and implements the FlowLogAggregator and
// FlowLogGetter interfaces.
// The flowLogAggregator is responsible for creating, aggregating, and storing
// aggregated flow logs until the flow logs are exported.
type flowLogAggregator struct {
	kind                 FlowAggregationKind
	flowStore            map[FlowMeta]FlowSpec
	flMutex              sync.RWMutex
	includeLabels        bool
	includePolicies      bool
	maxOriginalIPsSize   int
	aggregationStartTime time.Time
	handledAction        rules.RuleAction
}

// NewFlowLogAggregator constructs a FlowLogAggregator
func NewFlowLogAggregator() FlowLogAggregator {
	return &flowLogAggregator{
		kind:                 FlowDefault,
		flowStore:            make(map[FlowMeta]FlowSpec),
		flMutex:              sync.RWMutex{},
		maxOriginalIPsSize:   defaultMaxOrigIPSize,
		aggregationStartTime: time.Now(),
	}
}

func (c *flowLogAggregator) AggregateOver(kind FlowAggregationKind) FlowLogAggregator {
	c.kind = kind
	return c
}

func (c *flowLogAggregator) IncludeLabels(b bool) FlowLogAggregator {
	c.includeLabels = b
	return c
}

func (c *flowLogAggregator) IncludePolicies(b bool) FlowLogAggregator {
	c.includePolicies = b
	return c
}

func (c *flowLogAggregator) MaxOriginalIPsSize(s int) FlowLogAggregator {
	c.maxOriginalIPsSize = s
	return c
}

func (c *flowLogAggregator) ForAction(ra rules.RuleAction) FlowLogAggregator {
	c.handledAction = ra
	return c
}

// FeedUpdate constructs and aggregates flow logs from MetricUpdates.
func (c *flowLogAggregator) FeedUpdate(mu MetricUpdate) error {
	lastRuleID := mu.GetLastRuleID()
	if lastRuleID == nil {
		log.WithField("metric update", mu).Error("no last rule id present")
		return fmt.Errorf("Invalid metric update")
	}
	// Filter out any action that we aren't configured to handle.
	if c.handledAction != noRuleActionDefined && c.handledAction != lastRuleID.Action {
		log.Debugf("Update %v not handled", mu)
		return nil
	}

	log.WithField("update", mu).Debug("Flow Log Aggregator got Metric Update")
	flowMeta, err := NewFlowMeta(mu, c.kind)
	if err != nil {
		return err
	}
	c.flMutex.Lock()
	defer c.flMutex.Unlock()
	fl, ok := c.flowStore[flowMeta]
	if !ok {
		fl = NewFlowSpec(mu, c.maxOriginalIPsSize)
	} else {
		fl.aggregateMetricUpdate(mu)
	}
	c.flowStore[flowMeta] = fl

	return nil
}

// Get returns all aggregated flow logs, as a list of string pointers, since the last time a Get
// was called. Calling Get will also clear the stored flow logs once the flow logs are returned.
func (c *flowLogAggregator) Get() []*FlowLog {
	log.Debug("Get from flow log aggregator")
	resp := make([]*FlowLog, 0, len(c.flowStore))
	aggregationEndTime := time.Now()
	c.flMutex.Lock()
	defer c.flMutex.Unlock()
	for flowMeta, flowSpecs := range c.flowStore {
		flowLog := FlowData{flowMeta, flowSpecs}.ToFlowLog(c.aggregationStartTime, aggregationEndTime, c.includeLabels, c.includePolicies)
		resp = append(resp, &flowLog)
		c.calibrateFlowStore(flowMeta)
	}
	c.aggregationStartTime = aggregationEndTime
	return resp
}

func (c *flowLogAggregator) calibrateFlowStore(flowMeta FlowMeta) {
	// discontinue tracking the stats associated with the
	// flow meta if no more associated 5-tuples exist.
	if c.flowStore[flowMeta].getActiveFlowsCount() == 0 {
		delete(c.flowStore, flowMeta)
		return
	}

	// reset flow stats for the next interval
	c.flowStore[flowMeta] = c.flowStore[flowMeta].reset()
}
