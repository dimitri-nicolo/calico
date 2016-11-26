// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"
	"sync"
	"time"

	"github.com/projectcalico/felix/go/felix/jitter"
	"github.com/tigera/felix-private/go/felix/collector/stats"
)

// TODO(doublek): Need to hook these into configuration
const DefaultAgeTimeout = time.Duration(10) * time.Second
const InitialExportDelayTime = time.Duration(2) * time.Second
const ExportingInterval = time.Duration(1) * time.Second

type Collector struct {
	sources []<-chan stats.StatUpdate
	sinks   []chan<- *stats.Data
	epStats struct {
		sync.RWMutex
		entries map[stats.Tuple]*stats.Data
	}
	mux            chan stats.StatUpdate
	statAgeTimeout chan *stats.Data
	statTicker     *jitter.Ticker
}

func NewCollector(sources []<-chan stats.StatUpdate, sinks []chan<- *stats.Data) *Collector {
	return &Collector{
		sources: sources,
		sinks:   sinks,
		epStats: struct {
			sync.RWMutex
			entries map[stats.Tuple]*stats.Data
		}{entries: make(map[stats.Tuple]*stats.Data)},
		mux:            make(chan stats.StatUpdate),
		statAgeTimeout: make(chan *stats.Data),
		statTicker:     jitter.NewTicker(ExportingInterval, ExportingInterval/10),
	}
}

func (c *Collector) Start() {
	c.mergeDataSources()
	go c.startStatsCollectionAndReporting()
}

func (c *Collector) startStatsCollectionAndReporting() {
	// 1. c.mux for incoming datasources
	// 2. a agetimeout channel where all timeouts are sent in say from
	//    registerAgeTimer
	// 3. A periodic send thingy.
	// 4. A done channel for stopping and cleaning up stats collector.
	for {
		select {
		case update := <-c.mux:
			c.applyStatUpdate(update)
		case data := <-c.statAgeTimeout:
			c.expireEntry(data)
		case <-c.statTicker.C:
			c.exportStat()
		}
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
	data, ok := c.epStats.entries[update.Tuple]
	// If the tuple entry does not exist, create it.
	if !ok {
		data = stats.NewData(
			update.Tuple,
			update.WlEpKey,
			update.InPackets,
			update.InBytes,
			update.OutPackets,
			update.OutBytes,
			DefaultAgeTimeout)
		if update.Tp != (stats.RuleTracePoint{}) {
			data.AddRuleTracePoint(update.Tp)
		}
		c.registerAgeTimer(data)
		c.epStats.entries[update.Tuple] = data
		return
	}
	// If it does exist then update it, carefully.
	if update.Tp == (stats.TracePoint{}) {
		// We don't have to mess with the trace. Simply update the counters and be
		// done with it.
		data.SetCountersIn(update.InPackets, update.InBytes)
		data.SetCountersOut(update.OutPackets, update.OutBytes)
		c.epStats.entries[update.Tuple] = data
		return
	}
	data.UpdateCountersIn(update.InPackets, update.InBytes)
	data.UpdateCountersOut(update.OutPackets, update.OutBytes)
	err := data.AddRuleTracePoint(update.Tp)
	if err != nil {
		c.exportEntry(data)
		data.ResetCounters()
		data.ReplaceRuleTracePoint(update.Tp)
	}
	c.epStats.entries[update.Tuple] = data
}

func (c *Collector) expireEntry(data *stats.Data) {
	fmt.Println("expireEntry: Timer expired for data: ", fmtEntry(data))
	tuple := data.Tuple
	c.exportEntry(data)
	delete(c.epStats.entries, tuple)
}

func (c *Collector) registerAgeTimer(data *stats.Data) {
	// Wait for timer to fire and send the corresponding expired data to be
	// deleted.
	timer := data.AgeTimer()
	go func() {
		<-timer.C
		c.statAgeTimeout <- data
	}()
}

func (c *Collector) exportStat() {
	//fmt.Println("exportEntry: data: ", fmtEntry(&data))
	for _, data := range c.epStats.entries {
		c.exportEntry(data)
	}
}

func (c *Collector) exportEntry(data *stats.Data) {
	fmt.Println("exportEntry: data: ", fmtEntry(data))
}

func (c *Collector) PrintStats() {
	fmt.Printf("Number of Entries: %v\n", len(c.epStats.entries))
	for _, v := range c.epStats.entries {
		fmt.Println(fmtEntry(v))
	}
}

func fmtEntry(data *stats.Data) string {
	return fmt.Sprintf("%+v: %+v RuleTrace: %+v", data.Tuple, *data, *(data.RuleTrace))
}
