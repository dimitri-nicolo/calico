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

	"github.com/projectcalico/felix/jitter"
)

// TODO(doublek): Need to hook these into configuration
const DefaultAgeTimeout = time.Duration(10) * time.Second
const ExportingInterval = time.Duration(1) * time.Second

type Config struct {
	StatsDumpFilePath string
}

// A Collector (a StatsManager really) collects StatUpdates from data sources
// and stores them as a Data object in a map keyed by Tuple.
// All data source channels must be specified when creating the collector.
type Collector struct {
	sources        []<-chan *StatUpdate
	epStats        map[Tuple]*Data
	mux            chan *StatUpdate
	statAgeTimeout chan *Data
	reporterTicker *jitter.Ticker
	sigChan        chan os.Signal
	config         *Config
	dumpLog        *log.Logger
	reporterMgr    *ReporterManager
}

func NewCollector(sources []<-chan *StatUpdate, rm *ReporterManager, config *Config) *Collector {
	return &Collector{
		sources:        sources,
		epStats:        make(map[Tuple]*Data),
		mux:            make(chan *StatUpdate, 5000),
		statAgeTimeout: make(chan *Data),
		reporterTicker: jitter.NewTicker(ExportingInterval, ExportingInterval/10),
		sigChan:        make(chan os.Signal, 1),
		config:         config,
		dumpLog:        log.New(),
		reporterMgr:    rm,
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
	// 2. Data age timeouts via the c.statAgeTimeout channel.
	// 3. A signal handler that will dump logs on receiving SIGUSR2.
	// 4. A done channel for stopping and cleaning up collector (TODO).
	for {
		select {
		case update := <-c.mux:
			c.applyStatUpdate(update)
		case data := <-c.statAgeTimeout:
			c.expireEntry(data)
		case <-c.reporterTicker.C:
			log.Debug("Metrics reporter timer ticked")
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
		go func(input <-chan *StatUpdate) {
			for {
				c.mux <- <-input
			}
		}(source)
	}
}

func (c *Collector) applyStatUpdate(update *StatUpdate) {
	data, ok := c.epStats[update.Tuple]
	if !ok {
		// The entry does not exist. Go ahead and create one.
		data = NewData(
			update.Tuple,
			DefaultAgeTimeout)
		if update.Tp != EmptyRuleTracePoint {
			data.AddRuleTracePoint(update.Tp, update.Dir)
		}
		if update.CtrType == AbsoluteCounter {
			data.SetCounters(update.Packets, update.Bytes)
			data.SetCountersReverse(update.ReversePackets, update.ReverseBytes)
		} else {
			data.IncreaseCounters(update.Packets, update.Bytes)
			data.IncreaseCountersReverse(update.ReversePackets, update.ReverseBytes)
		}
		c.registerAgeTimer(data)
		c.epStats[update.Tuple] = data
		return
	}
	// Entry does exists. Go ahead and update it.
	if update.CtrType == AbsoluteCounter {
		data.SetCounters(update.Packets, update.Bytes)
		data.SetCountersReverse(update.ReversePackets, update.ReverseBytes)
	} else {
		data.IncreaseCounters(update.Packets, update.Bytes)
		data.IncreaseCountersReverse(update.ReversePackets, update.ReverseBytes)
	}
	if update.Tp != EmptyRuleTracePoint {
		err := data.AddRuleTracePoint(update.Tp, update.Dir)
		if err != nil {
			c.reportData(data)
			data.ResetCounters()
			data.ReplaceRuleTracePoint(update.Tp, update.Dir)
		}
	}
	c.epStats[update.Tuple] = data
}

func (c *Collector) expireEntry(data *Data) {
	tuple := data.Tuple
	c.reportData(data)
	if data.EgressRuleTrace.Action() == DenyAction {
		mu := NewMetricUpdateFromRuleTrace(data.Tuple, data.EgressRuleTrace)
		c.reporterMgr.ExpireChan <- mu
	}
	if data.IngressRuleTrace.Action() == DenyAction {
		mu := NewMetricUpdateFromRuleTrace(data.Tuple, data.IngressRuleTrace)
		c.reporterMgr.ExpireChan <- mu
	}
	delete(c.epStats, tuple)
}

func (c *Collector) registerAgeTimer(data *Data) {
	// Wait for timer to fire and send the corresponding expired data to be
	// deleted.
	timer := data.AgeTimer()
	go func() {
		<-timer.C
		c.statAgeTimeout <- data
	}()
}

func (c *Collector) reportMetrics() {
	for _, data := range c.epStats {
		c.reportData(data)
	}
}

func (c *Collector) reportData(data *Data) {
	if !data.IsDirty() {
		return
	}
	if data.EgressRuleTrace.Action() == DenyAction && data.EgressRuleTrace.IsDirty() {
		mu := NewMetricUpdateFromRuleTrace(data.Tuple, data.EgressRuleTrace)
		c.reporterMgr.ReportChan <- mu
	}
	if data.IngressRuleTrace.Action() == DenyAction && data.IngressRuleTrace.IsDirty() {
		mu := NewMetricUpdateFromRuleTrace(data.Tuple, data.IngressRuleTrace)
		c.reporterMgr.ReportChan <- mu
	}
	data.clearDirtyFlag()
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

func fmtEntry(data *Data) string {
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
