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

	"github.com/projectcalico/felix/jitter"
	"github.com/projectcalico/felix/lookup"
	"github.com/projectcalico/felix/rules"

	"github.com/tigera/nfnetlink"
)

var (
	ruleSep      = byte('|')
	namespaceSep = byte('/')
	tierSep      = byte('.')
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
	lum            lookup.QueryInterface
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

func NewCollector(lm lookup.QueryInterface, rm *ReporterManager, config *Config) *Collector {
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
func (c *Collector) applyNflogStatUpdate(tuple Tuple, tp *RuleTracePoint) {
	//TODO: RLB: What happens if we get an NFLOG metric update while we *think* we have a connection up?
	data := c.getData(tuple)
	if err := data.AddRuleTracePoint(tp); err == RuleTracePointConflict {
		// When a RuleTracePoint is replaced, we have to do some housekeeping before
		// we can replace the RuleTracePoint, the first of which is to remove
		// references from the reporter, which is done by calling expireMetrics,
		// followed by resetting counters.
		if data.DurationSinceCreate() > c.config.InitialReportingDelay {
			// We only need to expire metric entries that've probably been reported
			// in the first place.
			c.expireMetrics(data)
		}

		data.ResetConntrackCounters()
		if err = data.ReplaceRuleTracePoint(tp); err != nil {
			log.WithError(err).Warning("Unable to update rule trace point in metrics")
		}
	} else if err != nil {
		log.WithError(err).Warning("Unable to add rule trace point to metrics")
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
		if errSrc != nil && errDst != nil {
			// Unknown conntrack entries are expected for things such as
			// management or local traffic. This log can get spammy if we log everything
			// because of which we don't return an error and don't log anything here.
			continue
		}

		// At this point either the source or destination IP address from the conntrack entry
		// belongs to an endpoint i.e., the connection either begins or terminates locally.
		tuple := extractTupleFromCtEntryTuple(ctTuple)
		c.applyConnTrackStatUpdate(tuple,
			ctEntry.OriginalCounters.Packets, ctEntry.OriginalCounters.Bytes,
			ctEntry.ReplyCounters.Packets, ctEntry.ReplyCounters.Bytes)
	}
	return nil
}

func (c *Collector) convertNflogPktAndApplyUpdate(dir rules.RuleDirection, nPktAggr *nfnetlink.NflogPacketAggregate) error {
	var (
		numPkts, numBytes int
		epKey             interface{}
		err               error
	)
	nflogTuple := nPktAggr.Tuple

	// Determine the endpoint that this packet hit a rule for. This depends on the Direction
	// because local -> local packets will be NFLOGed twice.
	if dir == rules.RuleDirIngress {
		log.WithField("Dst", nflogTuple.Dst).Debug("Searching for endpoint")
		epKey, err = c.lum.GetEndpointKey(nflogTuple.Dst)
	} else {
		log.WithField("Src", nflogTuple.Src).Debug("Searching for endpoint")
		epKey, err = c.lum.GetEndpointKey(nflogTuple.Src)
	}

	if err != nil {
		// TODO (Matt): This branch becomes much more interesting with graceful restart.
		return errors.New("couldn't find endpoint info for NFLOG packet")
	}
	for _, prefix := range nPktAggr.Prefixes {
		// Lookup the ruleIDs from the prefix.
		ruleIDs, err := c.lookupRuleIDsFromPrefix(dir, prefix.Prefix, prefix.Len)
		if err != nil {
			continue
		}

		// Determine the index of this trace point.  This is the index of the effective
		// tiers for the endpoint (since there is only one allow/deny rule hit per Tier).
		tierIdx := c.lum.GetTierIndex(epKey, ruleIDs.Tier)

		// Determine the starting number of packets and bytes.
		if ruleIDs.Action == rules.ActionDeny || ruleIDs.Action == rules.ActionAllow {
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

		tp := NewRuleTracePoint(ruleIDs, epKey, tierIdx, numPkts, numBytes)
		tuple := extractTupleFromNflogTuple(nPktAggr.Tuple)
		c.applyNflogStatUpdate(tuple, tp)
	}
	return nil
}

func subscribeToNflog(gn int, nlBufSiz int, nflogChan chan *nfnetlink.NflogPacketAggregate, nflogDoneChan chan struct{}) error {
	return nfnetlink.NflogSubscribe(gn, nlBufSiz, nflogChan, nflogDoneChan)
}

// lookupRuleIDsFromPrefix determines the RuleIDs from a given rule direction and NFLOG prefix.
func (c *Collector) lookupRuleIDsFromPrefix(dir rules.RuleDirection, prefix [64]byte, prefixLen int) (*rules.RuleIDs, error) {
	// Extract the RuleIDs from the prefix.
	//TODO: RLB: We should keep a map[[64]byte]*RuleIDs to perform a fast lookup of the prefix
	// to the rules IDs (using pointers to avoid additional allocation).  It is a little naughty
	// passing pointers around since these structs are theoretically mutable. If we defined these
	// structs in a separate types package then we could make them immutable by requiring accessor
	// methods to access the private member data.
	//TODO: RLB: I think the prefix should be able to give us the rule direction too.

	// Should have at least 2 separators, a action character and a rule (assuming
	// we allow empty Policy names).
	if prefixLen < 4 {
		log.Errorf("Prefix is too short: %s (%d chars)", string(prefix[:prefixLen]), prefixLen)
		return nil, RuleTracePointParseError
	}

	// Initialise the RuleIDs struct.
	ruleIDs := &rules.RuleIDs{
		Direction: dir,
	}

	// Extract and convert the action.
	switch prefix[0] {
	case 'A':
		ruleIDs.Action = rules.ActionAllow
	case 'D':
		ruleIDs.Action = rules.ActionDeny
	case 'N':
		ruleIDs.Action = rules.ActionNextTier
	default:
		log.Errorf("Unknown action %v: %v", prefix[0], string(prefix[:prefixLen]))
		return nil, RuleTracePointParseError
	}

	// Determine the indices of the rule/policy/tier separators.
	ruleIdx := 2
	policySep := bytes.IndexByte(prefix[ruleIdx:], ruleSep)
	if policySep == -1 {
		log.Errorf("No separator char: %v", string(prefix[:prefixLen]))
		return nil, RuleTracePointParseError
	}
	policyIdx := ruleIdx + policySep + 1

	// Set the rule index.
	ruleIDs.Index = string(prefix[ruleIdx : policyIdx-1])

	var policyIDByte [64]byte
	copy(policyIDByte[:], prefix[policyIdx:prefixLen])

	policyIDUnhashed, err := c.lum.GetNFLOGHashToPolicyID(policyIDByte)
	if err != nil {
		return nil, fmt.Errorf("error getting NFLOG policy/profile name identifier from the hash: %s", err)
	}

	if string(policyIDUnhashed[len(policyIDUnhashed)-3:]) == "|pr" {
		ruleIDs.Tier = "profile"
		ruleIDs.Policy = string(policyIDUnhashed[:len(policyIDUnhashed)-3])
	} else {
		nsIdx := bytes.IndexByte(policyIDUnhashed[:], namespaceSep)
		tierIdx := bytes.IndexByte(policyIDUnhashed[:], tierSep)
		if nsIdx == -1 {
			// No namespace in the policy ID.
			if tierIdx == -1 {
				// It's a profile. Should already be handled.
			} else {
				// Policy without a namespace (default)

				// Check if it's a knp.default policy.
				if bytes.HasPrefix(policyIDUnhashed[:12], []byte("knp.default.")) {
					ruleIDs.Tier = "default"
					ruleIDs.Policy = string(policyIDUnhashed[12 : len(policyIDUnhashed)-3])
				} else {
					// It's a non-k8s policy.
					ruleIDs.Tier = string(policyIDUnhashed[:tierIdx])
					ruleIDs.Policy = string(policyIDUnhashed[tierIdx+1 : len(policyIDUnhashed)-3])
				}
			}
		} else {
			if tierIdx == -1 {
				// No tier means it's a profile, but profiles don't have namespace, so it should never get here.
			} else {
				ruleIDs.Tier = string(policyIDUnhashed[nsIdx+1 : tierIdx])
				ruleIDs.Policy = string(policyIDUnhashed[tierIdx+1 : len(policyIDUnhashed)-3])
			}
		}
	}

	return ruleIDs, nil
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
