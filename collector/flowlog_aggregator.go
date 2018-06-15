// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"
	"net"
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
		c.flowLogs[flKey] = fl
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
