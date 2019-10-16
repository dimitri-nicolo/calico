// +build !windows

// Copyright (c) 2016-2019 Tigera, Inc. All rights reserved.

package collector

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/gavv/monotime"

	"github.com/google/gopacket/layers"
	"github.com/mipearson/rfw"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/nfnetlink"
	"github.com/tigera/nfnetlink/nfnl"

	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/jitter"
	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/felix/rules"
)

const (
	expectedLocalEither byte = iota
	expectedLocalDestination
	expectedLocalSource
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
	EnableNetworkSets     bool

	MaxOriginalSourceIPsIncluded int
}

// A collector (a StatsManager really) collects StatUpdates from data sources
// and stores them as a Data object in a map keyed by Tuple.
// All data source channels must be specified when creating the
//
// Note that the dataplane statistics channel (ds) is currently just used for the
// policy syncer but will eventually also include NFLOG stats as well.
type collector struct {
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
	ds             chan *proto.DataplaneStats
	dnsLogReporter DNSLogReporterInterface
}

// newCollector instantiates a new collector. The StartDataplaneStatsCollector function is the only public
// function for collector instantiation.
func newCollector(lc *calc.LookupsCache, rm *ReporterManager, cfg *Config) Collector {
	return &collector{
		luc:            lc,
		nfIngressC:     make(chan *nfnetlink.NflogPacketAggregate, 1000),
		nfEgressC:      make(chan *nfnetlink.NflogPacketAggregate, 1000),
		nfIngressDoneC: make(chan struct{}),
		nfEgressDoneC:  make(chan struct{}),
		epStats:        make(map[Tuple]*Data),
		poller:         jitter.NewTicker(cfg.ConntrackPollingInterval, cfg.ConntrackPollingInterval/10),
		ticker:         jitter.NewTicker(cfg.ExportingInterval, cfg.ExportingInterval/10),
		sigChan:        make(chan os.Signal, 1),
		config:         cfg,
		dumpLog:        log.New(),
		reporterMgr:    rm,
		ds:             make(chan *proto.DataplaneStats, 1000),
	}
}

// ReportingChannel returns the channel used to report dataplane statistics.
func (c *collector) ReportingChannel() chan<- *proto.DataplaneStats {
	return c.ds
}

// SubscribeToNflog should only be called by the internal dataplane driver to subscribe this collector
// to NFLog stats.
// TODO: This will be removed when NFLOG and Conntrack handling is moved to the int_dataplane.
func (c *collector) SubscribeToNflog() {
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

func (c *collector) Start() {
	go c.startStatsCollectionAndReporting()
	c.setupStatsDumping()
	if c.dnsLogReporter != nil {
		c.dnsLogReporter.Start()
	}
}

func (c *collector) startStatsCollectionAndReporting() {
	// When a collector is started, we respond to the following events:
	// 1. StatUpdates for incoming datasources (chan c.mux).
	// 2. A signal handler that will dump logs on receiving SIGUSR2.
	// 3. A done channel for stopping and cleaning up collector (TODO).
	for {
		select {
		case <-c.poller.C:
			_ = nfnetlink.ConntrackList(c.handleCtEntry)
		case nflogPacketAggr := <-c.nfIngressC:
			c.convertNflogPktAndApplyUpdate(rules.RuleDirIngress, nflogPacketAggr)
		case nflogPacketAggr := <-c.nfEgressC:
			c.convertNflogPktAndApplyUpdate(rules.RuleDirEgress, nflogPacketAggr)
		case <-c.ticker.C:
			c.checkEpStats()
		case <-c.sigChan:
			c.dumpStats()
		case ds := <-c.ds:
			c.convertDataplaneStatsAndApplyUpdate(ds)
		}
	}
}

func (c *collector) setupStatsDumping() {
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

// getDataAndUpdateEndpoints returns a pointer to the data structure keyed off the supplied tuple.  If there
// is no entry and the tuple is for an active flow then an entry is created.
//
// This may return nil if the endpoint data does not match up with the requested data type.
//
// This method also updates the endpoint data from the cache, so beware - it is not as lightweight as a
// simple map lookup.
func (c *collector) getDataAndUpdateEndpoints(tuple Tuple, expired bool, expectedLocal byte) *Data {
	var dstEp, srcEp *calc.EndpointData
	var ok bool
	var dstLocal, srcLocal bool

	data, okData := c.epStats[tuple]

	if expired {
		// If the connection has expired then return the data as is. If there is no entry, that's fine too.
		return data
	}

	// Get the endpoint data for this entry, preferentially using a real endpoint over a NetworkSet. If a Calico
	// endpoint determine if local to this node.
	if dstEp, ok = c.luc.GetEndpoint(tuple.dst); ok {
		dstLocal = dstEp.IsLocal
	} else if c.config.EnableNetworkSets {
		dstEp, _ = c.luc.GetNetworkset(tuple.dst)
	}
	if srcEp, ok = c.luc.GetEndpoint(tuple.src); ok {
		srcLocal = srcEp.IsLocal
	} else if c.config.EnableNetworkSets {
		srcEp, _ = c.luc.GetNetworkset(tuple.src)
	}

	switch expectedLocal {
	case expectedLocalSource:
		if !srcLocal {
			return nil
		}
	case expectedLocalDestination:
		if !dstLocal {
			return nil
		}
	default:
		if !srcLocal && !dstLocal {
			return nil
		}
	}

	if !okData {
		// The entry does not exist. Go ahead and create a new one and add it to the map.
		data = NewData(tuple, srcEp, dstEp, c.config.MaxOriginalSourceIPsIncluded)
		c.epStats[tuple] = data

		// Return the new entry.
		return data
	}

	// Update endpoint info in data.
	// TODO(rlb): If key has changed, should expire/report the metrics first.
	data.dstEp = dstEp
	data.srcEp = srcEp

	return data
}

// applyConntrackStatUpdate applies a stats update from a conn track poll.
// If entryExpired is set then, this means that the update is for a recently
// expired entry. One of the following will be done:
// - If we already track the tuple, then the stats will be updated and will
//   then be expired from epStats.
// - If we don't track the tuple, this call will be a no-op as this update
//   is just waiting for the conntrack entry to timeout.
func (c *collector) applyConntrackStatUpdate(
	data *Data, packets int, bytes int, reversePackets int, reverseBytes int, entryExpired bool,
) {
	if data != nil {
		data.SetConntrackCounters(packets, bytes)
		data.SetConntrackCountersReverse(reversePackets, reverseBytes)

		if entryExpired {
			// Try to report metrics first before trying to expire the data.
			// This needs to be done in this order because if this is the
			// first time we are seeing this connection then we'll have
			// to report an update followed by expiring it signaling
			// the completion.
			c.reportMetrics(data)
			c.expireMetrics(data)
			c.deleteData(data)
		}
	}
}

// applyNflogStatUpdate applies a stats update from an NFLOG.
func (c *collector) applyNflogStatUpdate(data *Data, ruleID *calc.RuleID, matchIdx, numPkts, numBytes int) {
	//TODO: RLB: What happens if we get an NFLOG metric update while we *think* we have a connection up?
	if ru := data.AddRuleID(ruleID, matchIdx, numPkts, numBytes); ru == RuleMatchIsDifferent {
		// When a RuleTrace RuleID is replaced, we have to do some housekeeping
		// before we can replace it, the first of which is to remove references
		// from the reporter, which is done by calling expireMetrics, followed
		// by resetting counters.
		if data.DurationSinceLastRuleUpdate() > c.config.InitialReportingDelay {
			// We only need to expire metric entries that've probably been reported
			// in the first place.
			c.expireMetrics(data)
		}

		// Reset counters and replace the rule.
		data.ResetConntrackCounters()
		data.ResetApplicationCounters()
		data.ReplaceRuleID(ruleID, matchIdx, numPkts, numBytes)

		// The rule has just been set, update the last rule update time. This provides a window during which we can
		// gather any remaining rule hits.
		data.ruleUpdatedAt = monotime.Now()
	} else if ru == RuleMatchSet {
		// The rule has just been set, update the last rule update time. This provides a window during which we can
		// gather any remaining rule hits.
		data.ruleUpdatedAt = monotime.Now()
	}
}

func (c *collector) checkEpStats() {
	// We report stats at initial reporting delay after the last rule update. This aims to ensure we have the full set
	// of rules before we report the stats. As a minor finesse, calculate the latest update time to consider reporting.
	minLastRuleUpdatedAt := monotime.Now() - c.config.InitialReportingDelay
	minExpirationAt := monotime.Now() - c.config.AgeTimeout

	// For each entry
	// - report metrics
	// - check age and expire the entry if needed.
	for _, data := range c.epStats {
		if data.IsDirty() && data.RuleUpdatedAt() < minLastRuleUpdatedAt {
			// We report Metrics only after an initial delay to allow any Policy/rule
			// changes to show up as part of data.
			c.reportMetrics(data)
		}
		if data.UpdatedAt() < minExpirationAt {
			c.expireMetrics(data)
			c.deleteData(data)
		}
	}
}

func (c *collector) reportMetrics(data *Data) {
	data.Report(c.reporterMgr.ReportChan, false)
}

func (c *collector) expireMetrics(data *Data) {
	data.Report(c.reporterMgr.ReportChan, true)
}

func (c *collector) deleteData(data *Data) {
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
func (c *collector) handleCtEntry(ctEntry nfnetlink.CtEntry) {
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

	// At this point either the source or destination IP address from the conntrack entry
	// belongs to an endpoint i.e., the connection either begins or terminates locally.
	tuple := extractTupleFromCtEntryTuple(ctTuple)

	// In the case of TCP, check if we can expire the entry early. We try to expire
	// entries early so that we don't send any spurious MetricUpdates for an expiring
	// conntrack entry.
	entryExpired := (ctTuple.ProtoNum == nfnl.TCP_PROTO && ctEntry.ProtoInfo.State >= nfnl.TCP_CONNTRACK_TIME_WAIT)

	// Get or create a data entry and update the counters. If no entry is returned then neither source nor dest are
	// calico managed endpoints. A relevant conntrack entry requires at least one of the endpoints to be a local
	// Calico managed endpoint.
	if data := c.getDataAndUpdateEndpoints(tuple, entryExpired, expectedLocalEither); data != nil {
		c.applyConntrackStatUpdate(data,
			ctEntry.OriginalCounters.Packets, ctEntry.OriginalCounters.Bytes,
			ctEntry.ReplyCounters.Packets, ctEntry.ReplyCounters.Bytes,
			entryExpired)
	}
}

func (c *collector) convertNflogPktAndApplyUpdate(dir rules.RuleDir, nPktAggr *nfnetlink.NflogPacketAggregate) {
	var (
		localEp        *calc.EndpointData
		localMatchData *calc.MatchData
		data           *Data
	)

	// Extract the tuple used for lookup for this set of packet prefixes.
	tuple := extractTupleFromNflogTuple(nPktAggr.Tuple)

	// Determine the local endpoint for this update.
	switch dir {
	case rules.RuleDirIngress:
		if data = c.getDataAndUpdateEndpoints(tuple, false, expectedLocalDestination); data == nil {
			// Data is nil, so the destination endpoint cannot be managed by local Calico.
			return
		}
		// The cache will return nil for ingress if the destination endpoint is not local.
		localEp = data.dstEp
		localMatchData = localEp.Ingress
	case rules.RuleDirEgress:
		if data = c.getDataAndUpdateEndpoints(tuple, false, expectedLocalSource); data == nil {
			// Data is nil, so the source endpoint cannot be managed by local Calico.
			return
		}
		// The cache will return nil for egress if the source endpoint is not local.
		localEp = data.srcEp
		localMatchData = localEp.Egress
	default:
		return
	}

	for _, prefix := range nPktAggr.Prefixes {
		// Lookup the ruleID from the prefix.
		ruleID := c.luc.GetRuleIDFromNFLOGPrefix(prefix.Prefix)
		if ruleID == nil {
			continue
		}

		if ruleID.IsProfile() {
			// This is a profile verdict. Apply the rule unchanged, but at the profile match index (which is at the
			// very end of the match slice).
			c.applyNflogStatUpdate(data, ruleID, localMatchData.ProfileMatchIndex, prefix.Packets, prefix.Bytes)
			continue
		}

		if ruleID.IsEndOfTier() {
			// This is an end-of-tier action.
			// -  For deny convert the ruleID to the implicit drop rule
			// -  For pass leave the rule unchanged. We never return this to the user, but instead use it to determine
			//    whether we add staged policy end-of-tier denies.
			// For both deny and pass, add the rule at the end of tier match index.
			tier, ok := localMatchData.TierData[ruleID.Tier]
			if !ok {
				continue
			}

			switch ruleID.Action {
			case rules.RuleActionDeny:
				c.applyNflogStatUpdate(
					data, tier.ImplicitDropRuleID, tier.EndOfTierMatchIndex,
					prefix.Packets, prefix.Bytes,
				)
			case rules.RuleActionPass:
				c.applyNflogStatUpdate(
					data, ruleID, tier.EndOfTierMatchIndex,
					prefix.Packets, prefix.Bytes,
				)
			}
			continue
		}

		// This is one of:
		// -  An enforced rule match
		// -  A staged policy match
		// -  A staged policy miss
		// -  An end-of-tier pass (from tiers only containing staged policies)
		//
		// For all these cases simply add the unchanged ruleID using the match index reserved for that policy.
		// Extract the policy data from the ruleID.
		policyIdx, ok := localMatchData.PolicyMatches[ruleID.PolicyID]
		if !ok {
			continue
		}

		c.applyNflogStatUpdate(data, ruleID, policyIdx, prefix.Packets, prefix.Bytes)
	}
}

// convertDataplaneStatsAndApplyUpdate merges the proto.DataplaneStatistics into the current
// data stored for the specific connection tuple.
func (c *collector) convertDataplaneStatsAndApplyUpdate(d *proto.DataplaneStats) {
	log.Debugf("Received dataplane stats update %+v", d)
	// Create a Tuple representing the DataplaneStats.
	t, err := extractTupleFromDataplaneStats(d)
	if err != nil {
		log.Errorf("unable to extract 5-tuple from DataplaneStats: %v", err)
		return
	}

	// Locate the data for this connection, creating if not yet available (it's possible to get an update
	// from the dataplane before nflogs or conntrack).
	data := c.getDataAndUpdateEndpoints(t, false, expectedLocalDestination)

	var httpDataCount int
	for _, s := range d.Stats {
		if s.Relativity != proto.Statistic_DELTA {
			// Currently we only expect delta HTTP requests from the dataplane statistics API.
			log.WithField("relativity", s.Relativity.String()).Warning("Received a statistic from the dataplane that Felix cannot process")
			continue
		}
		switch s.Kind {
		case proto.Statistic_HTTP_REQUESTS:
			switch s.Action {
			case proto.Action_ALLOWED:
				data.IncreaseHTTPRequestAllowedCounter(int(s.Value))
			case proto.Action_DENIED:
				data.IncreaseHTTPRequestDeniedCounter(int(s.Value))
			}
		case proto.Statistic_HTTP_DATA:
			httpDataCount = int(s.Value)
		default:
			log.WithField("kind", s.Kind.String()).Warnf("Received a statistic from the dataplane that Felix cannot process")
			continue
		}

	}
	ips := make([]net.IP, 0, len(d.HttpData))
	for _, hd := range d.HttpData {
		var origSrcIP string
		if len(hd.XRealIp) != 0 {
			origSrcIP = hd.XRealIp
		} else if len(hd.XForwardedFor) != 0 {
			origSrcIP = hd.XForwardedFor
		} else {
			continue
		}
		sip := net.ParseIP(origSrcIP)
		if sip == nil {
			log.WithField("IP", origSrcIP).Warn("bad source IP")
			continue
		}
		ips = append(ips, sip)
	}
	if len(ips) != 0 {
		if httpDataCount == 0 {
			httpDataCount = len(ips)
		}
		bs := NewBoundedSetFromSliceWithTotalCount(c.config.MaxOriginalSourceIPsIncluded, ips, httpDataCount)
		data.AddOriginalSourceIPs(bs)
	} else if httpDataCount != 0 {
		data.IncreaseNumUniqueOriginalSourceIPs(httpDataCount)
	}
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

func extractTupleFromDataplaneStats(d *proto.DataplaneStats) (Tuple, error) {
	var protocol int32
	switch n := d.Protocol.GetNumberOrName().(type) {
	case *proto.Protocol_Number:
		protocol = n.Number
	case *proto.Protocol_Name:
		switch strings.ToLower(n.Name) {
		case "tcp":
			protocol = 6
		case "udp":
			protocol = 17
		default:
			return Tuple{}, fmt.Errorf("unhandled protocol: %s", n)
		}
	}

	// Use the standard go net library to parse the IP since this always returns IPs as 16 bytes.
	srcIP := net.ParseIP(d.SrcIp)
	if srcIP == nil {
		return Tuple{}, fmt.Errorf("bad source IP: %s", d.SrcIp)
	}
	dstIP := net.ParseIP(d.DstIp)
	if dstIP == nil {
		return Tuple{}, fmt.Errorf("bad destination IP: %s", d.DstIp)
	}

	// But invoke the To16() just to be sure.
	var srcArray, dstArray [16]byte
	copy(srcArray[:], srcIP.To16())
	copy(dstArray[:], dstIP.To16())

	// Locate the data for this connection, creating if not yet available (it's possible to get an update
	// before nflogs or conntrack).
	return *NewTuple(srcArray, dstArray, int(protocol), int(d.SrcPort), int(d.DstPort)), nil
}

// Write stats to file pointed by Config.StatsDumpFilePath.
// When called, clear the contents of the file Config.StatsDumpFilePath before
// writing the stats to it.
func (c *collector) dumpStats() {
	log.Debugf("Dumping Stats to %v", c.config.StatsDumpFilePath)

	_ = os.Truncate(c.config.StatsDumpFilePath, 0)
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

// DNS activity logging.
func (c *collector) SetDNSLogReporter(reporter DNSLogReporterInterface) {
	c.dnsLogReporter = reporter
}

func (c *collector) LogDNS(src, dst net.IP, dns *layers.DNS) {
	if c.dnsLogReporter == nil {
		return
	}
	// DNS responses come through here, so the source IP is the DNS server and the dest IP is
	// the client.
	serverEP, _ := c.luc.GetEndpoint(ipTo16Byte(src))
	clientEP, _ := c.luc.GetEndpoint(ipTo16Byte(dst))
	if serverEP == nil {
		serverEP, _ = c.luc.GetNetworkset(ipTo16Byte(src))
	}
	log.Debugf("Src %v -> Server %v", src, serverEP)
	log.Debugf("Dst %v -> Client %v", dst, clientEP)
	update := DNSUpdate{
		ClientIP: dst,
		ClientEP: clientEP,
		ServerIP: src,
		ServerEP: serverEP,
		DNS:      dns,
	}
	if err := c.dnsLogReporter.Log(update); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"src": src,
			"dst": dst,
			"dns": dns,
		}).Error("Failed to log DNS packet")
	}
}
