// Copyright (c) 2016-2018 Tigera, Inc. All rights reserved.

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

	"github.com/tigera/nfnetlink"

	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/jitter"
	"github.com/projectcalico/felix/rules"
)

var (
	noEndpointErr = errors.New("couldn't find endpoint info for NFLOG packet")
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

// A Collector (a StatsManager really) collects StatUpdates from data sources
// and stores them as a Data object in a map keyed by Tuple.
// All data source channels must be specified when creating the collector.
type Collector struct {
	luc            *calc.LookupsCache
	nfIngressC     chan *nfnetlink.NflogPacketAggregate
	nfEgressC      chan *nfnetlink.NflogPacketAggregate
	nfIngressDoneC chan struct{}
	nfEgressDoneC  chan struct{}
	epStats        map[Tuple]*Data
	poller         *jitter.Ticker
	ticker         *jitter.Ticker
	sigChan        chan os.Signal
	config         *Config
	dumpLog        *log.Logger
	reporterMgr    *ReporterManager
}

func NewCollector(lc *calc.LookupsCache, rm *ReporterManager, config *Config) *Collector {
	return &Collector{
		luc:            lc,
		nfIngressC:     make(chan *nfnetlink.NflogPacketAggregate, 1000),
		nfEgressC:      make(chan *nfnetlink.NflogPacketAggregate, 1000),
		nfIngressDoneC: make(chan struct{}),
		nfEgressDoneC:  make(chan struct{}),
		epStats:        make(map[Tuple]*Data),
		poller:         jitter.NewTicker(config.ConntrackPollingInterval, config.ConntrackPollingInterval/10),
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
}

func (c *Collector) startStatsCollectionAndReporting() {
	// When a collector is started, we respond to the following events:
	// 1. StatUpdates for incoming datasources (chan c.mux).
	// 2. A signal handler that will dump logs on receiving SIGUSR2.
	// 3. A done channel for stopping and cleaning up collector (TODO).
	for {
		select {
		case <-c.poller.C:
			nfnetlink.ConntrackList(c.handleCtEntry)
		case nflogPacketAggr := <-c.nfIngressC:
			c.convertNflogPktAndApplyUpdate(rules.RuleDirIngress, nflogPacketAggr)
		case nflogPacketAggr := <-c.nfEgressC:
			c.convertNflogPktAndApplyUpdate(rules.RuleDirEgress, nflogPacketAggr)
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

// getData returns a pointer to the data structure keyed off the supplied tuple.  If there
// is no entry one is created.
func (c *Collector) getData(tuple Tuple) *Data {
	data, ok := c.epStats[tuple]
	if !ok {
		// The entry does not exist. Go ahead and create a new one and add it to the map.
		data = NewData(tuple, c.config.AgeTimeout)
		c.epStats[tuple] = data
	}
	return data
}

// applyConnTrackStatUpdate applies a stats update from a conn track poll.
func (c *Collector) applyConnTrackStatUpdate(
	tuple Tuple, packets int, bytes int, reversePackets int, reverseBytes int,
) {
	// Update the counters for the entry.  Since data is a pointer, we are updating the map
	// entry in situ.
	data := c.getData(tuple)
	data.SetCounters(packets, bytes)
	data.SetCountersReverse(reversePackets, reverseBytes)
}

// applyNflogStatUpdate applies a stats update from an NFLOG.
func (c *Collector) applyNflogStatUpdate(tuple Tuple, ruleID *calc.RuleID, srcEp, dstEp *calc.EndpointData, tierIdx, numPkts, numBytes int) {
	//TODO: RLB: What happens if we get an NFLOG metric update while we *think* we have a connection up?
	data := c.getData(tuple)
	if srcEp != nil {
		data.SetSourceEndpointData(srcEp)
	}
	if dstEp != nil {
		data.SetDestinationEndpointData(dstEp)
	}
	if ok := data.AddRuleID(ruleID, tierIdx, numPkts, numBytes); !ok {
		// When a RuleTrace RuleID is replaced, we have to do some housekeeping
		// before we can replace it, the first of which is to remove references
		// references from the reporter, which is done by calling expireMetrics,
		// followed by resetting counters.
		if data.DurationSinceCreate() > c.config.InitialReportingDelay {
			// We only need to expire metric entries that've probably been reported
			// in the first place.
			c.expireMetrics(data)
		}

		data.ResetConntrackCounters()
		if ok = data.ReplaceRuleID(ruleID, tierIdx, numPkts, numBytes); !ok {
			log.Warning("Unable to update rule trace point in metrics")
		}
	}
}

func (c *Collector) checkEpStats() {
	// For each entry
	// - report metrics
	// - check age and expire the entry if needed.
	for _, data := range c.epStats {
		if data.IsDirty() && data.DurationSinceCreate() >= c.config.InitialReportingDelay {
			// We report Metrics only after an initial delay to allow any Policy/rule
			// changes to show up as part of data.
			c.reportMetrics(data)
		}
		if data.DurationSinceLastUpdate() >= c.config.AgeTimeout {
			c.expireData(data)
		}
	}
}

func (c *Collector) reportMetrics(data *Data) {
	data.Report(c.reporterMgr.ReportChan, false)
}

func (c *Collector) expireMetrics(data *Data) {
	data.Report(c.reporterMgr.ReportChan, true)
}

func (c *Collector) expireData(data *Data) {
	c.expireMetrics(data)
	delete(c.epStats, data.Tuple)
}

// handleCtEntry handles a conntrack entry
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
func (c *Collector) handleCtEntry(ctEntry nfnetlink.CtEntry) {
	var (
		ctTuple nfnetlink.CtTuple
		err     error
	)

	ctTuple = ctEntry.OriginalTuple

	// A conntrack entry that has the destination NAT (DNAT) flag set
	// will have its destination ip-address set to the NAT-ed IP rather
	// than the actual workload/host endpoint. To continue processing
	// this conntrack entry, we need the actual IP address that corresponds
	// to a Workload/Host Endpoint.
	if ctEntry.IsDNAT() {
		ctTuple, err = ctEntry.OriginalTupleWithoutDNAT()
		if err != nil {
			log.Error("Error when extracting tuple without DNAT:", err)
			return
		}
	}

	// Check if the connection begins and/or terminates on this host. This is done
	// by checking if the source and/or destination IP address from the conntrack
	// entry that we are processing belong to endpoints.
	// If we cannot find an endpoint for both the source and destination IP Addresses
	// this means that this connection neither begins nor terminates locally.
	// We can skip processing this conntrack entry.
	if !c.luc.IsEndpoint(ctTuple.Src) && !c.luc.IsEndpoint(ctTuple.Dst) {
		return
	}

	// At this point either the source or destination IP address from the conntrack entry
	// belongs to an endpoint i.e., the connection either begins or terminates locally.
	tuple := extractTupleFromCtEntryTuple(ctTuple)
	c.applyConnTrackStatUpdate(tuple,
		ctEntry.OriginalCounters.Packets, ctEntry.OriginalCounters.Bytes,
		ctEntry.ReplyCounters.Packets, ctEntry.ReplyCounters.Bytes)
	return
}

func (c *Collector) convertNflogPktAndApplyUpdate(dir rules.RuleDir, nPktAggr *nfnetlink.NflogPacketAggregate) error {
	var (
		numPkts, numBytes     int
		localEp, srcEp, dstEp *calc.EndpointData
		ok                    bool
	)
	nflogTuple := nPktAggr.Tuple

	// Determine the endpoint that this packet hit a rule for. This depends on the Direction
	// because local -> local packets will be NFLOGed twice.
	if dir == rules.RuleDirIngress {
		dstEp, ok = c.luc.GetEndpoint(nflogTuple.Dst)
		srcEp, _ = c.luc.GetEndpoint(nflogTuple.Src)
		localEp = dstEp
	} else {
		srcEp, ok = c.luc.GetEndpoint(nflogTuple.Src)
		dstEp, _ = c.luc.GetEndpoint(nflogTuple.Dst)
		localEp = srcEp
	}

	if !ok {
		// TODO (Matt): This branch becomes much more interesting with graceful restart.
		return noEndpointErr
	}
	for _, prefix := range nPktAggr.Prefixes {
		// Lookup the ruleID from the prefix.
		ruleID := c.luc.GetRuleIDFromNFLOGPrefix(prefix.Prefix)
		if ruleID == nil {
			continue
		}

		apply := func(tierIdx int) {
			// Determine the starting number of packets and bytes.
			if ruleID.Action == rules.RuleActionDeny || ruleID.Action == rules.RuleActionAllow {
				// NFLog based counters make sense only for denied packets or allowed packets
				// under NOTRACK. When NOTRACK is not enabled, the conntrack based absolute
				// counters will overwrite these values anyway.
				numPkts = prefix.Packets
				numBytes = prefix.Bytes
			} else {
				// Don't update packet counts for ActionNextTier to avoid multiply counting.
				numPkts = 0
				numBytes = 0
			}

			tuple := extractTupleFromNflogTuple(nPktAggr.Tuple)
			c.applyNflogStatUpdate(tuple, ruleID, srcEp, dstEp, tierIdx, numPkts, numBytes)
		}

		// A blank tier indicates a profile match. This should be directly after the last tier.
		if len(ruleID.Tier) == 0 {
			apply(len(localEp.OrderedTiers))
			continue
		}

		for tierIdx, tier := range localEp.OrderedTiers {
			if tier == ruleID.Tier {
				apply(tierIdx)
				break
			}
		}
	}
	return nil
}

func subscribeToNflog(gn int, nlBufSiz int, nflogChan chan *nfnetlink.NflogPacketAggregate, nflogDoneChan chan struct{}) error {
	return nfnetlink.NflogSubscribe(gn, nlBufSiz, nflogChan, nflogDoneChan)
}

func extractTupleFromNflogTuple(nflogTuple nfnetlink.NflogPacketTuple) Tuple {
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

func extractTupleFromCtEntryTuple(ctTuple nfnetlink.CtTuple) Tuple {
	var l4Src, l4Dst int
	if ctTuple.ProtoNum == 1 {
		l4Src = ctTuple.L4Src.Id
		l4Dst = int(uint16(ctTuple.L4Dst.Type)<<8 | uint16(ctTuple.L4Dst.Code))
	} else {
		l4Src = ctTuple.L4Src.Port
		l4Dst = ctTuple.L4Dst.Port
	}
	return *NewTuple(ctTuple.Src, ctTuple.Dst, ctTuple.ProtoNum, l4Src, l4Dst)
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
