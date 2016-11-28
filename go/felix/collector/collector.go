// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"
	"time"

	"github.com/projectcalico/felix/go/felix/jitter"
	"github.com/tigera/felix-private/go/felix/collector/stats"
	"github.com/tigera/felix-private/go/felix/ipfix"
)

// TODO(doublek): Need to hook these into configuration
const DefaultAgeTimeout = time.Duration(10) * time.Second
const InitialExportDelayTime = time.Duration(2) * time.Second
const ExportingInterval = time.Duration(1) * time.Second

// A Collector (a StatsManager really) collects StatUpdates from data sources
// and stores them as a stats.Data object in a map keyed by stats.Tuple.
// It also periodically exports all entries of this map to a IPFIX exporter.
// All data source channels and IPFIX exporter channel must be specified when
// creating the collector.
type Collector struct {
	sources        []<-chan stats.StatUpdate
	sinks          []chan<- *stats.Data
	epStats        map[stats.Tuple]*stats.Data
	mux            chan stats.StatUpdate
	statAgeTimeout chan *stats.Data
	statTicker     *jitter.Ticker
	exportSink     chan<- *ipfix.ExportRecord
}

func NewCollector(sources []<-chan stats.StatUpdate, sinks []chan<- *stats.Data, exportSink chan<- *ipfix.ExportRecord) *Collector {
	return &Collector{
		sources:        sources,
		sinks:          sinks,
		epStats:        make(map[stats.Tuple]*stats.Data),
		mux:            make(chan stats.StatUpdate),
		statAgeTimeout: make(chan *stats.Data),
		statTicker:     jitter.NewTicker(ExportingInterval, ExportingInterval/10),
		exportSink:     exportSink,
	}
}

func (c *Collector) Start() {
	c.mergeDataSources()
	go c.startStatsCollectionAndReporting()
}

func (c *Collector) startStatsCollectionAndReporting() {
	// When a collector is started, we respond to the following events:
	// 1. StatUpdates for incoming datasources (chan c.mux).
	// 2. stats.Data age timeouts via the c.statAgeTimeout channel.
	// 3. A periodic exporter via the c.statTicker channel.
	// 4. A done channel for stopping and cleaning up stats collector (TODO).
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
	data, ok := c.epStats[update.Tuple]
	if !ok {
		// The entry does not exist. Go ahead and create one.
		data = stats.NewData(
			update.Tuple,
			update.WlEpKey,
			update.InPackets,
			update.InBytes,
			update.OutPackets,
			update.OutBytes,
			DefaultAgeTimeout)
		if update.Tp != stats.EmptyRuleTracePoint {
			data.AddRuleTracePoint(update.Tp)
		}
		c.registerAgeTimer(data)
		c.epStats[update.Tuple] = data
		return
	}
	// Entry does exists. Go agead and update it.
	if update.CtrType == stats.AbsoluteCounter {
		data.SetCountersIn(update.InPackets, update.InBytes)
		data.SetCountersOut(update.OutPackets, update.OutBytes)
	} else {
		data.IncreaseCountersIn(update.InPackets, update.InBytes)
		data.IncreaseCountersOut(update.OutPackets, update.OutBytes)
	}
	if update.Tp != stats.EmptyRuleTracePoint {
		err := data.AddRuleTracePoint(update.Tp)
		if err != nil {
			c.exportEntry(data.ToExportRecord(ipfix.ForcedEnd))
			data.ResetCounters()
			data.ReplaceRuleTracePoint(update.Tp)
		}
	}
	c.epStats[update.Tuple] = data
}

func (c *Collector) expireEntry(data *stats.Data) {
	fmt.Println("expireEntry: Timer expired for data: ", fmtEntry(data))
	tuple := data.Tuple
	c.exportEntry(data.ToExportRecord(ipfix.IdleTimeout))
	delete(c.epStats, tuple)
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
	for _, data := range c.epStats {
		c.exportEntry(data.ToExportRecord(ipfix.ActiveTimeout))
	}
}

func (c *Collector) exportEntry(record *ipfix.ExportRecord) {
	c.exportSink <- record
}

func (c *Collector) PrintStats() {
	fmt.Printf("Number of Entries: %v\n", len(c.epStats))
	for _, v := range c.epStats {
		fmt.Println(fmtEntry(v))
	}
}

func fmtEntry(data *stats.Data) string {
	return fmt.Sprintf("%+v: %+v RuleTrace: %+v", data.Tuple, *data, *(data.RuleTrace))
}
