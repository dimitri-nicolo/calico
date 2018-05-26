// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

import (
	"net"
	"strconv"
	"sync"

	log "github.com/sirupsen/logrus"
)

const (
	flowLogBufferSize = 1000
)

// cloudWatchAggregator implements the FlowLogAggregator and
// FlowLogGetter interfaces.
type cloudWatchAggregator struct {
	flowLogs map[string]int
	sync.RWMutex
}

// NewCloudWatchAggregator constructs a FlowLogAggregator
func NewCloudWatchAggregator() FlowLogAggregator {
	return &cloudWatchAggregator{
		flowLogs: map[string]int{},
	}
}

// TODO: As the Flow Log format matures and MetricUpdate object
// transforms this will change
func constructFlowLog(mu MetricUpdate) string {
	src := net.IP(mu.tuple.src[:4]).String()
	dst := net.IP(mu.tuple.dst[:4]).String()
	dstPort := strconv.Itoa(mu.tuple.l4Dst)
	proto := strconv.Itoa(mu.tuple.proto)
	flowLogKey := src + " " + dst + " " + dstPort + " " + proto
	log.Infof("Constructing flow log: %s", flowLogKey)
	return flowLogKey
}

// FeedUpdate will be responsible for doing aggregation. TODO: As the design matures
// this will need updating.
func (c *cloudWatchAggregator) FeedUpdate(mu MetricUpdate) error {
	flString := constructFlowLog(mu)
	c.flowLogs[flString]++
	return nil
}

func (c *cloudWatchAggregator) Get() []*string {
	c.Lock()
	defer c.Unlock()

	resp := []*string{}

	for k, _ := range c.flowLogs {
		resp = append(resp, &k)
	}

	return resp
}
