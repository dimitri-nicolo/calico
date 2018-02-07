// Copyright (c) 2016-2017 Tigera, Inc. All rights reserved.

package collector

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/mipearson/rfw"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/jitter"
	"github.com/projectcalico/felix/lookup"
	"github.com/tigera/nfnetlink"
)

type Config struct {
	StatsDumpFilePath string

	NfNetlinkBufSize int
	IngressGroup     int
	EgressGroup      int

	ConntrackPollingInterval time.Duration

	AgeTimeout            time.Duration
	InitialReportingDelay time.Duration
	ExportingInterval     time.Duration
}

type epLookup interface {
	GetEndpointKey(addr [16]byte) (interface{}, error)
	GetPolicyIndex(epKey interface{}, policyName, tierName []byte) int
}

// A Collector (a StatsManager really) collects StatUpdates from data sources
// and stores them as a Data object in a map keyed by Tuple.
// All data source channels must be specified when creating the collector.
type Collector struct {
	lum            epLookup
	nfIngressC     chan *nfnetlink.NflogPacketAggregate
	nfEgressC      chan *nfnetlink.NflogPacketAggregate
	nfIngressDoneC chan struct{}
	nfEgressDoneC  chan struct{}
	ctEntriesC     chan []nfnetlink.CtEntry
	epStats        map[Tuple]*Data
	ticker         *jitter.Ticker
	sigChan        chan os.Signal
	config         *Config
	dumpLog        *log.Logger
	reporterMgr    *ReporterManager
}

func NewCollector(lm epLookup, rm *ReporterManager, config *Config) *Collector {
	return &Collector{
		lum:            lm,
		nfIngressC:     make(chan *nfnetlink.NflogPacketAggregate, 1000),
		nfEgressC:      make(chan *nfnetlink.NflogPacketAggregate, 1000),
		nfIngressDoneC: make(chan struct{}),
		nfEgressDoneC:  make(chan struct{}),
		ctEntriesC:     make(chan []nfnetlink.CtEntry, 10),
		epStats:        make(map[Tuple]*Data),
		ticker:         jitter.NewTicker(config.ExportingInterval, config.ExportingInterval/10),
		sigChan:        make(chan os.Signal, 1),
		config:         config,
		dumpLog:        log.New(),
		reporterMgr:    rm,
	}
}

func (c *Collector) Start() {
	go c.startStatsCollectionAndReporting()
	c.setupStatsDumping()
	err := subscribeToNflog(c.config.IngressGroup, c.config.NfNetlinkBufSize, c.nfIngressC, c.nfIngressDoneC)
	if err != nil {
		log.Errorf("Error when subscribing to NFLOG: %v", err)
		return
	}
	err = subscribeToNflog(c.config.EgressGroup, c.config.NfNetlinkBufSize, c.nfEgressC, c.nfEgressDoneC)
	if err != nil {
		log.Errorf("Error when subscribing to NFLOG: %v", err)
		return
	}
	go pollConntrack(c.config.ConntrackPollingInterval, c.ctEntriesC)
}

func (c *Collector) startStatsCollectionAndReporting() {
	// When a collector is started, we respond to the following events:
	// 1. StatUpdates for incoming datasources (chan c.mux).
	// 2. A signal handler that will dump logs on receiving SIGUSR2.
	// 3. A done channel for stopping and cleaning up collector (TODO).
	for {
		select {
		case ctEntries := <-c.ctEntriesC:
			c.convertCtEntryAndApplyUpdate(ctEntries)
		case nflogPacketAggr := <-c.nfIngressC:
			c.convertNflogPktAndApplyUpdate(DirIn, nflogPacketAggr)
		case nflogPacketAggr := <-c.nfEgressC:
			c.convertNflogPktAndApplyUpdate(DirOut, nflogPacketAggr)
		case <-c.ticker.C:
			c.checkEpStats()
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

func (c *Collector) applyStatUpdate(tuple Tuple, packets int, bytes int, reversePackets int, reverseBytes int, ctrType CounterType, dir Direction, tp *RuleTracePoint) {
	data, ok := c.epStats[tuple]
	if !ok {
		// The entry does not exist. Go ahead and create one.
		data = NewData(
			tuple,
			c.config.AgeTimeout)
		if tp != nil {
			data.AddRuleTracePoint(tp, dir)
		}
		if ctrType == AbsoluteCounter {
			data.SetCounters(packets, bytes)
			data.SetCountersReverse(reversePackets, reverseBytes)
		} else {
			data.IncreaseCounters(packets, bytes)
			data.IncreaseCountersReverse(reversePackets, reverseBytes)
		}
		c.epStats[tuple] = data
		return
	}
	// Entry does exists. Go ahead and update it.
	if ctrType == AbsoluteCounter {
		data.SetCounters(packets, bytes)
		data.SetCountersReverse(reversePackets, reverseBytes)
	} else {
		data.IncreaseCounters(packets, bytes)
		data.IncreaseCountersReverse(reversePackets, reverseBytes)
	}
	if tp != nil {
		err := data.AddRuleTracePoint(tp, dir)
		if err != nil {
			// When a RuleTracePoint is replaced, we have to do some housekeeping before
			// we can replace the RuleTracePoint, the first of which is to remove
			// references from the reporter, which is done by calling expireMetric,
			// followed by resetting counters.
			if data.DurationSinceCreate() > c.config.InitialReportingDelay {
				// We only need to expire metric entries that've probably been reported
				// in the first place.
				c.expireMetric(data)
			}
			data.ResetCounters()
			data.ReplaceRuleTracePoint(tp, dir)
		}
	}
	c.epStats[tuple] = data
}

func (c *Collector) expireMetric(data *Data) {
	if data.EgressRuleTrace.Action() == DenyAction {
		mu := NewMetricUpdateFromRuleTrace(data.Tuple, data.EgressRuleTrace)
		c.reporterMgr.ExpireChan <- mu
	}
	if data.IngressRuleTrace.Action() == DenyAction {
		mu := NewMetricUpdateFromRuleTrace(data.Tuple, data.IngressRuleTrace)
		c.reporterMgr.ExpireChan <- mu
	}
}

func (c *Collector) expireData(data *Data) {
	c.expireMetric(data)
	delete(c.epStats, data.Tuple)
}

func (c *Collector) checkEpStats() {
	// For each entry
	// - report metrics
	// - check age and expire the entry if needed.
	for _, data := range c.epStats {
		if data.IsDirty() && data.DurationSinceCreate() >= c.config.InitialReportingDelay {
			// We report Metrics only after an initial delay to allow any policy/rule
			// changes to show up as part of data.
			c.reportMetrics(data)
		}
		if data.DurationSinceLastUpdate() >= c.config.AgeTimeout {
			c.expireData(data)
		}
	}
}

func (c *Collector) reportMetrics(data *Data) {
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

func (c *Collector) convertCtEntryAndApplyUpdate(ctEntries []nfnetlink.CtEntry) error {
	var (
		ctTuple nfnetlink.CtTuple
		err     error
	)

	// We expect and process connections (conntrack entries) of 3 different flavors.
	//
	// - Connections that *neither* begin *nor* terminate locally.
	// - Connections that either begin or terminate locally.
	// - Connections that begin *and* terminate locally.
	//
	// When processing these, we also check if the connection is flagged as a
	// destination NAT (DNAT) connection. If it is a DNAT-ed connection, we
	// process the conntrack entry after we figure out the connection's original
	// destination IP address before DNAT modified the connections' destination
	// IP/port.
	for _, ctEntry := range ctEntries {
		ctTuple, err = ctEntry.OriginalTuple()
		if err != nil {
			log.Error("Error when getting original tuple:", err)
			continue
		}

		// A conntrack entry that has the destination NAT (DNAT) flag set
		// will have its destination ip-address set to the NAT-ed IP rather
		// than the actual workload/host endpoint. To continue processing
		// this conntrack entry, we need the actual IP address that corresponds
		// to a Workload/Host Endpoint.
		if ctEntry.IsDNAT() {
			ctTuple, err = ctEntry.OriginalTupleWithoutDNAT()
			if err != nil {
				log.Error("Error when extracting tuple without DNAT:", err)
				continue
			}
		}

		// Check if the connection begins and/or terminates on this host. This is done
		// by checking if the source and/or destination IP address from the conntrack
		// entry that we are processing belong to endpoints.
		_, errSrc := c.lum.GetEndpointKey(ctTuple.Src)
		_, errDst := c.lum.GetEndpointKey(ctTuple.Dst)

		// If we cannot find an endpoint for both the source and destination IP Addresses
		// this means that this connection neither begins nor terminates locally.
		// We can skip processing this conntrack entry.
		if errSrc == lookup.UnknownEndpointError && errDst == lookup.UnknownEndpointError {
			// Unknown conntrack entries are expected for things such as
			// management or local traffic. This log can get spammy if we log everything
			// because of which we don't return an error and don't log anything here.
			continue
		}

		// At this point either the source or destination IP address from the conntrack entry
		// belongs to an endpoint i.e., the connection either begins or terminates locally.
		tuple := extractTupleFromCtEntryTuple(ctTuple, false)
		c.applyStatUpdate(tuple,
			ctEntry.OriginalCounters.Packets, ctEntry.OriginalCounters.Bytes,
			ctEntry.ReplyCounters.Packets, ctEntry.ReplyCounters.Bytes,
			AbsoluteCounter, DirUnknown, nil)

		// We create a reversed tuple, if we know that both the source and destination IP
		// addresses from the conntrack entry belong to endpoints, i.e., the connection
		// begins *and* terminates locally.
		if errSrc == nil && errDst == nil {
			// Packets/Connections from a local endpoint to another local endpoint
			// require a reversed tuple to collect reply stats.
			tuple := extractTupleFromCtEntryTuple(ctTuple, true)
			c.applyStatUpdate(tuple,
				ctEntry.ReplyCounters.Packets, ctEntry.ReplyCounters.Bytes,
				ctEntry.OriginalCounters.Packets, ctEntry.OriginalCounters.Bytes,
				AbsoluteCounter, DirUnknown, nil)
		}
	}
	return nil
}

func (c *Collector) convertNflogPktAndApplyUpdate(dir Direction, nPktAggr *nfnetlink.NflogPacketAggregate) error {
	var (
		numPkts, numBytes int
		epKey             interface{}
		err               error
	)
	nflogTuple := nPktAggr.Tuple
	// Determine the endpoint that this packet hit a rule for. This depends on the direction
	// because local -> local packets will be NFLOGed twice.
	if dir == DirIn {
		epKey, err = c.lum.GetEndpointKey(nflogTuple.Dst)
	} else {
		epKey, err = c.lum.GetEndpointKey(nflogTuple.Src)
	}

	if err == lookup.UnknownEndpointError {
		// TODO (Matt): This branch becomes much more interesting with graceful restart.
		return errors.New("Couldn't find endpoint info for NFLOG packet")
	}
	for _, prefix := range nPktAggr.Prefixes {
		tp, err := lookupRule(c.lum, prefix.Prefix, prefix.Len, epKey)
		if err != nil {
			continue
		}
		if tp.Action == DenyAction || tp.Action == AllowAction {
			// NFLog based counters make sense only for denied packets or allowed packets
			// under NOTRACK. When NOTRACK is not enabled, the conntrack based absolute
			// counters will overwrite these values anyway.
			numPkts = prefix.Packets
			numBytes = prefix.Bytes
		} else {
			// Don't update packet counts for NextTierAction to avoid multiply counting.
			numPkts = 0
			numBytes = 0
		}
		tp.Ctr = *NewCounter(numPkts, numBytes)
		tuple := extractTupleFromNflogTuple(nPktAggr.Tuple)
		// TODO(doublek): This DeltaCounter could be removed.
		c.applyStatUpdate(tuple, 0, 0, 0, 0, DeltaCounter, dir, tp)
	}
	return nil
}

func subscribeToNflog(gn int, nlBufSiz int, nflogChan chan *nfnetlink.NflogPacketAggregate, nflogDoneChan chan struct{}) error {
	return nfnetlink.NflogSubscribe(gn, nlBufSiz, nflogChan, nflogDoneChan)
}

func extractTupleFromNflogTuple(nflogTuple *nfnetlink.NflogPacketTuple) Tuple {
	var l4Src, l4Dst int
	if nflogTuple.Proto == 1 {
		l4Src = nflogTuple.L4Src.Id
		l4Dst = int(uint16(nflogTuple.L4Dst.Type)<<8 | uint16(nflogTuple.L4Dst.Code))
	} else {
		l4Src = nflogTuple.L4Src.Port
		l4Dst = nflogTuple.L4Dst.Port
	}
	return *NewTuple(nflogTuple.Src, nflogTuple.Dst, nflogTuple.Proto, l4Src, l4Dst)
}

func pollConntrack(pollInterval time.Duration, ctEntriesChan chan []nfnetlink.CtEntry) {
	poller := jitter.NewTicker(pollInterval, pollInterval/10)
	for _ = range poller.C {
		cte, err := nfnetlink.ConntrackList()
		if err != nil {
			log.Errorf("Error: ConntrackList: %v", err)
			continue
		}
		ctEntriesChan <- cte
	}
}

func extractTupleFromCtEntryTuple(ctTuple nfnetlink.CtTuple, reverse bool) Tuple {
	var l4Src, l4Dst int
	if ctTuple.ProtoNum == 1 {
		l4Src = ctTuple.L4Src.Id
		l4Dst = int(uint16(ctTuple.L4Dst.Type)<<8 | uint16(ctTuple.L4Dst.Code))
	} else {
		l4Src = ctTuple.L4Src.Port
		l4Dst = ctTuple.L4Dst.Port
	}
	if !reverse {
		return *NewTuple(ctTuple.Src, ctTuple.Dst, ctTuple.ProtoNum, l4Src, l4Dst)
	} else {
		return *NewTuple(ctTuple.Dst, ctTuple.Src, ctTuple.ProtoNum, l4Dst, l4Src)
	}
}

func lookupRule(lum epLookup, prefix [64]byte, prefixLen int, epKey interface{}) (*RuleTracePoint, error) {
	rtp, err := NewRuleTracePoint(prefix, prefixLen, epKey)
	if err != nil {
		return rtp, err
	}
	index := lum.GetPolicyIndex(epKey, rtp.PolicyID(), rtp.TierID())
	rtp.Index = index
	return rtp, nil
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
