// Copyright (c) 2016-2017 Tigera, Inc. All rights reserved.

package collector

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/mipearson/rfw"
	"github.com/projectcalico/felix/go/felix/collector/stats"
	"github.com/projectcalico/felix/go/felix/ipfix"
	"github.com/projectcalico/felix/go/felix/jitter"
)

// TODO(doublek): Need to hook these into configuration
const DefaultAgeTimeout = time.Duration(10) * time.Second
const InitialExportDelayTime = time.Duration(2) * time.Second
const ExportingInterval = time.Duration(1) * time.Second

type Config struct {
	StatsDumpFilePath string

	IpfixExportTierDropRules bool
}

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
	reporterTicker *jitter.Ticker
	exportSink     chan<- *ipfix.ExportRecord
	sigChan        chan os.Signal
	config         *Config
	dumpLog        *log.Logger
	aggStats       map[string]map[string]Metrics
}

func NewCollector(sources []<-chan stats.StatUpdate, sinks []chan<- *stats.Data, exportSink chan<- *ipfix.ExportRecord, config *Config) *Collector {
	return &Collector{
		sources:        sources,
		sinks:          sinks,
		epStats:        make(map[stats.Tuple]*stats.Data),
		mux:            make(chan stats.StatUpdate),
		statAgeTimeout: make(chan *stats.Data),
		statTicker:     jitter.NewTicker(ExportingInterval, ExportingInterval/10),
		reporterTicker: jitter.NewTicker(ExportingInterval, ExportingInterval/10),
		exportSink:     exportSink,
		sigChan:        make(chan os.Signal, 1),
		config:         config,
		dumpLog:        log.New(),
		aggStats:       make(map[string]map[string]Metrics),
	}
}

func (c *Collector) Start() {
	c.mergeDataSources()
	c.setupStatsDumping()
	go c.startStatsCollectionAndReporting()
}

func (c *Collector) startStatsCollectionAndReporting() {
	// When a collector is started, we respond to the following events:
	// 1. StatUpdates for incoming datasources (chan c.mux).
	// 2. stats.Data age timeouts via the c.statAgeTimeout channel.
	// 3. A periodic exporter via the c.statTicker channel.
	// 4. A signal handler that will dump logs on receiving SIGUSR2.
	// 5. A done channel for stopping and cleaning up stats collector (TODO).
	for {
		select {
		case update := <-c.mux:
			log.Info("Stats collector received update: ", update)
			c.applyStatUpdate(update)
		case data := <-c.statAgeTimeout:
			log.Info("Stats entry timed out: ", data)
			c.expireEntry(data)
		case <-c.statTicker.C:
			log.Info("Stats export timer ticked")
			c.exportStat()
		case <-c.reporterTicker.C:
			log.Info("Metrics reporter timer ticked")
			c.reportMetrics()
		case <-c.sigChan:
			c.dumpStats()
		}
	}
}

func (c *Collector) setupStatsDumping() {
	// TODO (doublek): This may not be the best place to put this. Consider
	// moving the signal handler and logging to file logic out of the collector
	// and simply out to appropriate sink on different messages.
	signal.Notify(c.sigChan, syscall.SIGUSR2)

	err := os.MkdirAll(path.Dir(c.config.StatsDumpFilePath), 0755)
	if err != nil {
		log.WithError(err).Fatal("Failed to create log dir")
	}

	rotAwareFile, err := rfw.Open(c.config.StatsDumpFilePath, 0644)
	if err != nil {
		log.WithError(err).Fatal("Failed to open log file")
	}

	// Attributes have to be directly set for instantiated logger as opposed
	// to the module level log object.
	c.dumpLog.Formatter = &MessageOnlyFormatter{}
	c.dumpLog.Level = log.InfoLevel
	c.dumpLog.Out = rotAwareFile
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
	log.Debug("Stats update: ", update)
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
			if data.IsExportEnabled() {
				c.exportEntry(data.ToExportRecord(ipfix.ForcedEnd))
			}
			data.ResetCounters()
			data.ReplaceRuleTracePoint(update.Tp)
		}
	}
	c.epStats[update.Tuple] = data
}

func (c *Collector) expireEntry(data *stats.Data) {
	log.Infof("Timer expired for entry: %v", data)
	tuple := data.Tuple
	if data.IsExportEnabled() {
		c.exportEntry(data.ToExportRecord(ipfix.IdleTimeout))
	}
	c.removeSourceIP(data.RuleTrace.ToString(), data.SourceIp())
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
	log.Debug("Exporting Stats")
	for _, data := range c.epStats {
		// TODO(doublek): If we haven't send an update in a while, we may be required
		// to send one out. Check RFC and implement if required.
		if !data.IsDirty() || !data.IsExportEnabled() {
			log.Debug("Skipping exporting ", fmtEntry(data))
			continue
		}
		c.exportEntry(data.ToExportRecord(ipfix.ActiveTimeout))
	}
}

func (c *Collector) exportEntry(record *ipfix.ExportRecord) {
	log.Debugf("Exporting entry %v", record)
	c.exportSink <- record
}

func (c *Collector) reportMetrics() {
	log.Debug("Aggregating and reporting metrics")
	var (
		ok       bool
		sipStats map[string]Metrics
		bytes    int
		packets  int
		rt       string
		srcIP    string
	)
	for _, data := range c.epStats {
		if data.Action() != stats.DenyAction {
			continue
		}
		srcIP = data.SourceIp()
		rt = data.RuleTrace.ToString()
		bytes = data.CountersIn().Bytes()
		packets = data.CountersIn().Packets()
		// TODO(doublek): This is a temporary workaround until direction awareness
		// of tuples/data via NFLOG makes its way in.
		if packets == 0 {
			bytes = data.CountersOut().Bytes()
			packets = data.CountersOut().Packets()
		}
		sipStats, ok = c.aggStats[rt]
		if !ok {
			sipStats = make(map[string]Metrics)
			entry := Metrics{
				Bytes:   bytes,
				Packets: packets,
			}
			sipStats[srcIP] = entry
		} else {
			entry, ok := sipStats[srcIP]
			if !ok {
				entry = Metrics{
					Bytes:   bytes,
					Packets: packets,
				}
			} else {
				entry.Bytes = bytes
				entry.Packets = packets
			}
			sipStats[srcIP] = entry
		}
		c.aggStats[rt] = sipStats
	}
	log.Debugf("Aggregated stats %+v", c.aggStats)
	for policy, sipStats := range c.aggStats {
		UpdateMetrics(policy, sipStats)
	}
}

func (c *Collector) removeSourceIP(ruleTrace string, srcIP string) {
	log.Debugf("removeSourceIP: RuleTrace %v Source IP: %v", ruleTrace, srcIP)
	sipStats, ok := c.aggStats[ruleTrace]
	if !ok {
		log.Infof("removeSourceIP: RuleTrace %v not present in aggregate stats", ruleTrace)
		return
	}
	_, ok = sipStats[srcIP]
	if !ok {
		log.Infof("removeSourceIP: Source IP %v not present in aggregate stats", srcIP)
		return
	}
	DeleteMetric(ruleTrace, srcIP)
	delete(sipStats, srcIP)
	if len(sipStats) == 0 {
		delete(c.aggStats, ruleTrace)
	} else {
		c.aggStats[ruleTrace] = sipStats
	}
	return
}

// Write stats to file pointed by Config.StatsDumpFilePath.
// When called, clear the contents of the file Config.StatsDumpFilePath before
// writing the stats to it.
func (c *Collector) dumpStats() {
	log.Debugf("Dumping Stats to %v", c.config.StatsDumpFilePath)

	os.Truncate(c.config.StatsDumpFilePath, 0)
	c.dumpLog.Infof("Stats Dump Started: %v", time.Now().Format("2006-01-02 15:04:05.000"))
	c.dumpLog.Infof("Number of Entries: %v", len(c.epStats))
	for _, v := range c.epStats {
		c.dumpLog.Info(fmtEntry(v))
	}
	c.dumpLog.Infof("Stats Dump Completed: %v", time.Now().Format("2006-01-02 15:04:05.000"))
}

func fmtEntry(data *stats.Data) string {
	return fmt.Sprintf("%v", data)
}

// Logrus Formatter that strips the log entry of formatting such as time, log
// level and simply outputs *only* the message.
type MessageOnlyFormatter struct{}

func (f *MessageOnlyFormatter) Format(entry *log.Entry) ([]byte, error) {
	b := &bytes.Buffer{}
	b.WriteString(entry.Message)
	b.WriteByte('\n')
	return b.Bytes(), nil
}
