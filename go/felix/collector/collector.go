// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"
	"sync"

	"github.com/tigera/felix-private/go/felix/collector/stats"
)

type Collector struct {
	sources []<-chan stats.StatUpdate
	sinks   []chan<- *stats.Data
	epStats struct {
		sync.RWMutex
		entries map[stats.Tuple]*stats.Data
	}
	mux chan stats.StatUpdate
}

func NewCollector(sources []<-chan stats.StatUpdate, sinks []chan<- *stats.Data) *Collector {
	return &Collector{
		sources: sources,
		sinks:   sinks,
		epStats: struct {
			sync.RWMutex
			entries map[stats.Tuple]*stats.Data
		}{entries: make(map[stats.Tuple]*stats.Data)},
		mux: make(chan stats.StatUpdate),
	}
}

func (c *Collector) Start() {
	c.mergeDataSources()
	go c.startStatsCollectionAndReporting()
	// TODO(doublek): We need to add a timer implementation to look at:
	// 1. The age of a flow
	// 2. The send timer (for the first send - when filling in rule trace)
}

func (c *Collector) startStatsCollectionAndReporting() {
	for update := range c.mux {
		c.applyStatUpdate(update)
	}
}

func (c *Collector) mergeDataSources() {
	// Can't use a select here as we don't really know the number of sources that
	// we have.
	for _, source := range c.sources {
		go func(input <-chan stats.StatUpdate) {
			for {
				c.mux <- <-input
			}
		}(source)
	}
}

func (c *Collector) applyStatUpdate(update stats.StatUpdate) {
	c.epStats.RLock()
	data, ok := c.epStats.entries[update.Tuple]
	c.epStats.RUnlock()
	// If the tuple entry does not exist, create it.
	if !ok {
		data = stats.NewData(
			update.Tuple,
			update.WlEpKey,
			update.InPackets,
			update.InBytes,
			update.OutPackets,
			update.OutBytes)
		if update.Tp != (stats.TracePoint{}) {
			data.AddTrace(update.Tp)
		}
		c.epStats.Lock()
		c.epStats.entries[update.Tuple] = data
		c.epStats.Unlock()
		return
	}
	// If it does exist then update it, carefully.
	if update.Tp == (stats.TracePoint{}) {
		// We don't have to mess with the trace. Simply update the counters and be
		// done with it.
		data.SetCountersIn(update.InPackets, update.InBytes)
		data.SetCountersOut(update.OutPackets, update.OutBytes)
		c.epStats.Lock()
		c.epStats.entries[update.Tuple] = data
		c.epStats.Unlock()
		return
	}
	data.UpdateCountersIn(update.InPackets, update.InBytes)
	data.UpdateCountersOut(update.OutPackets, update.OutBytes)
	err := data.AddTrace(update.Tp)
	if err != nil {
		// TODO(doublek): Force send stats out at this point.
		data.ResetCounters()
		data.ReplaceTrace(update.Tp)
	}
	c.epStats.Lock()
	c.epStats.entries[update.Tuple] = data
	c.epStats.Unlock()
}

func (c *Collector) PrintStats() {
	c.epStats.RLock()
	defer c.epStats.RUnlock()
	fmt.Printf("Number of Entries: %v\n", len(c.epStats.entries))
	for k, v := range c.epStats.entries {
		fmt.Printf("%+v: %+v Trace: %+v\n", k, *v, *(v.Trace()))
	}
}
