// Copyright (c) 2016-2017 Tigera, Inc. All rights reserved.

package collector

import (
	"bytes"
	"errors"
	"net"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/projectcalico/felix/go/felix/collector/stats"
	"github.com/projectcalico/felix/go/felix/jitter"
	"github.com/projectcalico/felix/go/felix/lookup"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/tigera/nfnetlink"
)

type epLookup interface {
	GetEndpointKey(addr net.IP) *model.WorkloadEndpointKey
	GetPolicyIndex(epKey *model.WorkloadEndpointKey, policyKey *model.PolicyKey) int
}

const PollingInterval = time.Duration(1) * time.Second

type NflogDataSource struct {
	sink          chan<- stats.StatUpdate
	groupNum      int
	direction     stats.Direction
	nlBufSiz      int
	lum           epLookup
	nflogChan     chan nfnetlink.NflogPacket
	nflogDoneChan chan struct{}
}

func NewNflogDataSource(lm *lookup.LookupManager, sink chan<- stats.StatUpdate, groupNum int, dir stats.Direction, nlBufSiz int) *NflogDataSource {
	nflogChan := make(chan nfnetlink.NflogPacket)
	done := make(chan struct{})
	return newNflogDataSource(lm, sink, groupNum, dir, nlBufSiz, nflogChan, done)
}

// Internal constructor to help mocking from UTs.
func newNflogDataSource(lm epLookup, sink chan<- stats.StatUpdate, gn int, dir stats.Direction, nbs int, nc chan nfnetlink.NflogPacket, done chan struct{}) *NflogDataSource {
	return &NflogDataSource{
		sink:          sink,
		groupNum:      gn,
		direction:     dir,
		nlBufSiz:      nbs,
		lum:           lm,
		nflogChan:     nc,
		nflogDoneChan: done,
	}
}

func (ds *NflogDataSource) Start() {
	log.Infof("Starting NFLOG Data Source for direction %v group %v", ds.direction, ds.groupNum)
	err := ds.subscribeToNflog()
	if err != nil {
		log.Errorf("Error when subscribing to NFLOG: %v", err)
		return
	}
	go ds.startProcessingPackets()
}

func (ds *NflogDataSource) subscribeToNflog() error {
	return nfnetlink.NflogSubscribe(ds.groupNum, ds.nlBufSiz, ds.nflogChan, ds.nflogDoneChan)
}

func (ds *NflogDataSource) startProcessingPackets() {
	defer close(ds.nflogDoneChan)
	for nflogPacket := range ds.nflogChan {
		statUpdate, err := ds.convertNflogPktToStat(nflogPacket)
		if err != nil {
			log.Errorf("Cannot convert Nflog packet %v to StatUpdate", nflogPacket)
			continue
		}
		ds.sink <- *statUpdate
	}
}

func (ds *NflogDataSource) convertNflogPktToStat(nPkt nfnetlink.NflogPacket) (*stats.StatUpdate, error) {
	nflogTuple := nPkt.Tuple
	var numPkts, numBytes, inPkts, inBytes, outPkts, outBytes int
	var statUpdate *stats.StatUpdate
	var reverse bool
	var wlEpKey *model.WorkloadEndpointKey
	var prefixAction stats.RuleAction
	_, _, _, prefixAction, _ = parsePrefix(nPkt.Prefix)
	if prefixAction == stats.DenyAction {
		// NFLog based counters make sense only for denied packets.
		numPkts = 1
		numBytes = nPkt.Bytes
	} else {
		// Allowed packet counters are updated via conntrack datasource.
		// Don't update packet counts for NextTierAction to avoid multiply counting.
		// FIXME(doublek): This assumption is not true in the case of NOTRACK.
		numPkts = 0
		numBytes = 0
	}

	// Determine the endpoint that this packet hit a rule for. This depends on the direction
	// because local -> local packets will be NFLOGed twice.
	if ds.direction == stats.DirIn {
		inPkts = numPkts
		inBytes = numBytes
		outPkts = 0
		outBytes = 0
		wlEpKey = ds.lum.GetEndpointKey(nflogTuple.Dst)
		reverse = true
	} else {
		inPkts = 0
		inBytes = 0
		outPkts = numPkts
		outBytes = numBytes
		wlEpKey = ds.lum.GetEndpointKey(nflogTuple.Src)
		reverse = false
	}

	if wlEpKey != nil {
		tp := lookupRule(ds.lum, nPkt.Prefix, wlEpKey)
		tuple := extractTupleFromNflogTuple(nPkt.Tuple, reverse)
		statUpdate = stats.NewStatUpdate(tuple, *wlEpKey, inPkts, inBytes, outPkts, outBytes, stats.DeltaCounter, tp)
	} else {
		// TODO (Matt): This branch becomes much more interesting with graceful restart.
		log.Warn("Failed to find endpoint for NFLOG packet ", nflogTuple, "/", ds.direction)
		return nil, errors.New("Couldn't find endpoint info for NFLOG packet")
	}

	log.Debug("Built NFLOG stat update: ", statUpdate)
	return statUpdate, nil
}

func extractTupleFromNflogTuple(nflogTuple nfnetlink.NflogPacketTuple, reverse bool) stats.Tuple {
	var l4Src, l4Dst int
	if nflogTuple.Proto == 1 {
		l4Src = nflogTuple.L4Src.Id
		l4Dst = int(uint16(nflogTuple.L4Dst.Type)<<8 | uint16(nflogTuple.L4Dst.Code))
	} else {
		l4Src = nflogTuple.L4Src.Port
		l4Dst = nflogTuple.L4Dst.Port
	}
	if !reverse {
		return *stats.NewTuple(nflogTuple.Src, nflogTuple.Dst, nflogTuple.Proto, l4Src, l4Dst)
	} else {
		return *stats.NewTuple(nflogTuple.Dst, nflogTuple.Src, nflogTuple.Proto, l4Dst, l4Src)
	}
}

type ConntrackDataSource struct {
	poller    *jitter.Ticker
	pollerC   <-chan time.Time
	converter chan []nfnetlink.CtEntry
	sink      chan<- stats.StatUpdate
	lum       epLookup
}

func NewConntrackDataSource(lm *lookup.LookupManager, sink chan<- stats.StatUpdate) *ConntrackDataSource {
	poller := jitter.NewTicker(PollingInterval, PollingInterval/10)
	converter := make(chan []nfnetlink.CtEntry)
	return newConntrackDataSource(lm, sink, poller, poller.C, converter)
}

// Internal constructor to help mocking from UTs.
func newConntrackDataSource(lm epLookup, sink chan<- stats.StatUpdate, poller *jitter.Ticker, pollerC <-chan time.Time, converter chan []nfnetlink.CtEntry) *ConntrackDataSource {
	return &ConntrackDataSource{
		poller:    poller,
		pollerC:   pollerC,
		converter: converter,
		sink:      sink,
		lum:       lm,
	}
}

func (ds *ConntrackDataSource) Start() {
	log.Info("Starting Conntrack Data Source")
	go ds.startPolling()
	go ds.startProcessor()
}

func (ds *ConntrackDataSource) startPolling() {
	for _ = range ds.pollerC {
		cte, err := nfnetlink.ConntrackList()
		if err != nil {
			log.Errorf("Error: ConntrackList: %v", err)
			return
		}
		ds.converter <- cte
	}
}
func (ds *ConntrackDataSource) startProcessor() {
	for ctentries := range ds.converter {
		for _, ctentry := range ctentries {
			statUpdates, err := ds.convertCtEntryToStat(ctentry)
			if err != nil {
				log.Errorf("Couldn't convert ctentry %v to StatUpdate", ctentry)
				continue
			}
			for _, su := range statUpdates {
				ds.sink <- su
			}
		}
	}
}

func (ds *ConntrackDataSource) convertCtEntryToStat(ctEntry nfnetlink.CtEntry) ([]stats.StatUpdate, error) {
	// There can be a maximum of 2 stat updates per ctentry, in the case of
	// local-to-local traffic.
	statUpdates := []stats.StatUpdate{}
	// The last entry is the tuple entry for endpoints
	ctTuple, err := ctEntry.OrigTuple()
	if err != nil {
		log.Error("Error when extracting OrigTuple:", err)
		return statUpdates, err
	}

	// We care about DNAT only which modifies the destination parts of the OrigTuple.
	if ctEntry.IsDNAT() {
		log.Debugf("Entry is DNAT %+v", ctEntry)
		ctTuple, err = ctEntry.OrigTupleWithoutDNAT()
		if err != nil {
			log.Error("Error when extracting tuple without DNAT:", err)
		}
	}
	wlEpKeySrc := ds.lum.GetEndpointKey(ctTuple.Src)
	wlEpKeyDst := ds.lum.GetEndpointKey(ctTuple.Dst)

	// Force conntrack to have empty tracepoint
	if wlEpKeySrc != nil {
		// Locally originating packet
		tuple := extractTupleFromCtEntryTuple(ctTuple, false)
		su := stats.NewStatUpdate(tuple, *wlEpKeySrc,
			ctEntry.OrigCounters.Packets, ctEntry.OrigCounters.Bytes,
			ctEntry.ReplCounters.Packets, ctEntry.ReplCounters.Bytes,
			stats.AbsoluteCounter, stats.EmptyRuleTracePoint)
		statUpdates = append(statUpdates, *su)
	}
	if wlEpKeyDst != nil {
		// Locally terminating packet
		tuple := extractTupleFromCtEntryTuple(ctTuple, true)
		su := stats.NewStatUpdate(tuple, *wlEpKeyDst,
			ctEntry.ReplCounters.Packets, ctEntry.ReplCounters.Bytes,
			ctEntry.OrigCounters.Packets, ctEntry.OrigCounters.Bytes,
			stats.AbsoluteCounter, stats.EmptyRuleTracePoint)
		statUpdates = append(statUpdates, *su)
	}
	for _, update := range statUpdates {
		log.Debug("Built conntrack stat update: ", update)
	}
	return statUpdates, nil
}

func extractTupleFromCtEntryTuple(ctTuple nfnetlink.CtTuple, reverse bool) stats.Tuple {
	var l4Src, l4Dst int
	if ctTuple.ProtoNum == 1 {
		l4Src = ctTuple.L4Src.Id
		l4Dst = int(uint16(ctTuple.L4Dst.Type)<<8 | uint16(ctTuple.L4Dst.Code))
	} else {
		l4Src = ctTuple.L4Src.Port
		l4Dst = ctTuple.L4Dst.Port
	}
	if !reverse {
		return *stats.NewTuple(ctTuple.Src, ctTuple.Dst, ctTuple.ProtoNum, l4Src, l4Dst)
	} else {
		return *stats.NewTuple(ctTuple.Dst, ctTuple.Src, ctTuple.ProtoNum, l4Dst, l4Src)
	}
}

// Stubs
// TODO (Matt): Refactor these in better.

func lookupAction(action string) stats.RuleAction {
	switch action {
	case "A":
		return stats.AllowAction
	case "D":
		return stats.DenyAction
	case "N":
		return stats.NextTierAction
	default:
		log.Error("Unknown action ", action)
		return stats.NextTierAction
	}
}

func parsePrefix(prefix string) (tier, policy, rule string, action stats.RuleAction, export bool) {
	// Prefix formats are:
	// - T/A/rule index/profile name
	// - T/A/rule index/policy name/tier name
	// TODO (Matt): Add sensible rule UUIDs
	prefixChunks := strings.Split(prefix, "/")
	if len(prefixChunks) == 4 {
		// Profile
		// TODO (Matt): Need something better than profile;
		//              it won't work if that's the name of a tier.
		tier = "profile"
	} else if len(prefixChunks) == 5 {
		// Tiered Policy
		// TODO(doublek): Should fix where the null character was introduced,
		// which could be nfnetlink.
		tier = string(bytes.Trim([]byte(prefixChunks[4]), "\x00"))
	} else {
		log.Error("Unable to parse NFLOG prefix ", prefix)
	}

	action = lookupAction(prefixChunks[1])
	rule = prefixChunks[2]
	policy = prefixChunks[3]
	export, err := strconv.ParseBool(prefixChunks[0])
	if err != nil {
		log.Error("Unable to parse export flag ", prefixChunks[0])
		export = false
	}
	return
}

func lookupRule(lum epLookup, prefix string, epKey *model.WorkloadEndpointKey) stats.RuleTracePoint {
	log.Infof("Looking up rule prefix %s", prefix)
	tier, policy, rule, action, export := parsePrefix(prefix)
	return stats.RuleTracePoint{
		TierID:   tier,
		PolicyID: policy,
		Rule:     rule,
		Action:   action,
		Export:   export,
		Index:    lum.GetPolicyIndex(epKey, &model.PolicyKey{Name: policy, Tier: tier}),
	}
}

// End Stubs
