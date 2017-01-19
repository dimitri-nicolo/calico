// Copyright (c) 2016-2017 Tigera, Inc. All rights reserved.

package collector

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/projectcalico/felix/go/felix/collector/stats"
	"github.com/projectcalico/felix/go/felix/lookup"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/tigera/nfnetlink"
)

type NflogDataSource struct {
	sink      chan<- stats.StatUpdate
	groupNum  int
	direction stats.Direction
	nlBufSiz  int
	lum       *lookup.LookupManager
}

func NewNflogDataSource(lm *lookup.LookupManager, sink chan<- stats.StatUpdate, groupNum int, dir stats.Direction, nlBufSiz int) *NflogDataSource {
	return &NflogDataSource{
		sink:      sink,
		groupNum:  groupNum,
		direction: dir,
		nlBufSiz:  nlBufSiz,
		lum:       lm,
	}
}

func (ds *NflogDataSource) Start() {
	log.Infof("Starting NFLOG Data Source for direction %v group %v", ds.direction, ds.groupNum)
	go ds.subscribeToNflog()
}

func (ds *NflogDataSource) subscribeToNflog() {
	ch := make(chan nfnetlink.NflogPacket)
	done := make(chan struct{})
	defer close(done)
	err := nfnetlink.NflogSubscribe(ds.groupNum, ds.nlBufSiz, ch, done)
	if err != nil {
		log.Errorf("Error when subscribing to NFLOG: %v", err)
		return
	}
	for nflogPacket := range ch {
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

	// Determine the endpoint that this packet hit a rule for.  This depends on the direction
	// because local -> local packets we be NFLOGed twice.
	if ds.direction == stats.DirIn {
		inPkts = numPkts
		inBytes = numBytes
		outPkts = 0
		outBytes = 0
		wlEpKey = ds.lum.GetEndpointKey(nflogTuple.Dst)
	} else {
		inPkts = 0
		inBytes = 0
		outPkts = numPkts
		outBytes = numBytes
		wlEpKey = ds.lum.GetEndpointKey(nflogTuple.Src)
	}

	if wlEpKey != nil {
		tp := lookupRule(ds.lum, nPkt.Prefix, wlEpKey)
		tuple := extractTupleFromNflogTuple(nPkt.Tuple)
		statUpdate = stats.NewStatUpdate(tuple, *wlEpKey, inPkts, inBytes, outPkts, outBytes, stats.DeltaCounter, ds.direction, tp)
	} else {
		// TODO (Matt): This branch becomes much more interesting with graceful restart.
		log.Warn("Failed to find endpoint for NFLOG packet ", nflogTuple, "/", ds.direction)
		return nil, errors.New("Couldn't find endpoint info for NFLOG packet")
	}

	log.Debug("Built NFLOG stat update: ", statUpdate)
	return statUpdate, nil
}

func extractTupleFromNflogTuple(nflogTuple nfnetlink.NflogPacketTuple) stats.Tuple {
	var l4Src, l4Dst int
	if nflogTuple.Proto == 1 {
		l4Src = nflogTuple.L4Src.Id
		l4Dst = int(uint16(nflogTuple.L4Dst.Type)<<8 | uint16(nflogTuple.L4Dst.Code))
	} else {
		l4Src = nflogTuple.L4Src.Port
		l4Dst = nflogTuple.L4Dst.Port
	}
	return *stats.NewTuple(nflogTuple.Src, nflogTuple.Dst, nflogTuple.Proto, l4Src, l4Dst)
}

type ConntrackDataSource struct {
	sink chan<- stats.StatUpdate
	lum  *lookup.LookupManager
}

func NewConntrackDataSource(lm *lookup.LookupManager, sink chan<- stats.StatUpdate) *ConntrackDataSource {
	return &ConntrackDataSource{
		sink: sink,
		lum:  lm,
	}
}

func (ds *ConntrackDataSource) Start() {
	log.Info("Starting Conntrack Data Source")
	go ds.startPolling()
}

func (ds *ConntrackDataSource) startPolling() {
	c := time.Tick(time.Second)
	for _ = range c {
		ctentries, err := nfnetlink.ConntrackList()
		if err != nil {
			log.Errorf("Error: ConntrackList: %v", err)
			return
		}
		// TODO(doublek): Possibly do this in a separate goroutine?
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
	ctTuple := ctEntry.OrigTuples[len(ctEntry.OrigTuples)-1]
	wlEpKeySrc := ds.lum.GetEndpointKey(ctTuple.Src)
	wlEpKeyDst := ds.lum.GetEndpointKey(ctTuple.Dst)
	tuple := extractTupleFromCtEntryTuple(ctTuple)
	// Force conntrack to have empty tracep
	if wlEpKeySrc != nil {
		// Locally originating packet
		su := stats.NewStatUpdate(tuple, *wlEpKeySrc,
			ctEntry.OrigCounters.Packets, ctEntry.OrigCounters.Bytes,
			ctEntry.ReplCounters.Packets, ctEntry.ReplCounters.Bytes,
			stats.AbsoluteCounter, stats.DirUnknown, stats.EmptyRuleTracePoint)
		statUpdates = append(statUpdates, *su)
	}
	if wlEpKeyDst != nil {
		// Locally terminating packet
		su := stats.NewStatUpdate(tuple, *wlEpKeyDst,
			ctEntry.OrigCounters.Packets, ctEntry.OrigCounters.Bytes,
			ctEntry.ReplCounters.Packets, ctEntry.ReplCounters.Bytes,
			stats.AbsoluteCounter, stats.DirUnknown, stats.EmptyRuleTracePoint)
		statUpdates = append(statUpdates, *su)
	}
	for _, update := range statUpdates {
		log.Debug("Built conntrack stat update: ", update)
	}
	return statUpdates, nil
}

func extractTupleFromCtEntryTuple(ctTuple nfnetlink.CtTuple) stats.Tuple {
	var l4Src, l4Dst int
	if ctTuple.ProtoNum == 1 {
		l4Src = ctTuple.L4Src.Id
		l4Dst = int(uint16(ctTuple.L4Dst.Type)<<8 | uint16(ctTuple.L4Dst.Code))
	} else {
		l4Src = ctTuple.L4Src.Port
		l4Dst = ctTuple.L4Dst.Port
	}
	return *stats.NewTuple(ctTuple.Src, ctTuple.Dst, ctTuple.ProtoNum, l4Src, l4Dst)
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

func lookupRule(lum *lookup.LookupManager, prefix string, epKey *model.WorkloadEndpointKey) stats.RuleTracePoint {
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
