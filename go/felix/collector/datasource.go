// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package collector

import (
	"errors"
	"net"
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
	lum       *lookup.LookupManager
}

func NewNflogDataSource(lm *lookup.LookupManager, sink chan<- stats.StatUpdate, groupNum int, dir stats.Direction) *NflogDataSource {
	return &NflogDataSource{
		sink:      sink,
		groupNum:  groupNum,
		direction: dir,
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
	err := nfnetlink.NflogSubscribe(ds.groupNum, ch, done)
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
	var reverse bool
	var wlEpKey *model.WorkloadEndpointKey
	if lookupAction(nPkt.Prefix) == stats.DenyAction {
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
		wlEpKey = lookupEndpoint(ds.lum, nflogTuple.Dst)
		reverse = true
	} else {
		inPkts = 0
		inBytes = 0
		outPkts = numPkts
		outBytes = numBytes
		wlEpKey = lookupEndpoint(ds.lum, nflogTuple.Src)
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
	wlEpKeySrc := lookupEndpoint(ds.lum, ctTuple.Src)
	wlEpKeyDst := lookupEndpoint(ds.lum, ctTuple.Dst)
	// Force conntrack to have empty tracep
	if wlEpKeySrc != nil {
		// Locally originating packet
		tuple := extractTupleFromCtEntryTuple(ctTuple, false)
		su := stats.NewStatUpdate(tuple, *wlEpKeySrc,
			ctEntry.ReplCounters.Packets, ctEntry.ReplCounters.Bytes,
			ctEntry.OrigCounters.Packets, ctEntry.OrigCounters.Bytes,
			stats.AbsoluteCounter, stats.EmptyRuleTracePoint)
		statUpdates = append(statUpdates, *su)
	}
	if wlEpKeyDst != nil {
		// Locally terminating packet
		tuple := extractTupleFromCtEntryTuple(ctTuple, true)
		su := stats.NewStatUpdate(tuple, *wlEpKeyDst,
			ctEntry.OrigCounters.Packets, ctEntry.OrigCounters.Bytes,
			ctEntry.ReplCounters.Packets, ctEntry.ReplCounters.Bytes,
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

func lookupEndpoint(lum *lookup.LookupManager, ipAddr net.IP) *model.WorkloadEndpointKey {
	epid := lum.GetEndpointID(ipAddr)
	if epid == nil {
		return nil
	} else {
		// TODO (Matt): Need to lookup hostname.
		return &model.WorkloadEndpointKey{
			Hostname:       "matt-k8s",
			OrchestratorID: epid.OrchestratorId,
			WorkloadID:     epid.WorkloadId,
			EndpointID:     epid.EndpointId,
		}
	}
}

func lookupAction(prefix string) stats.RuleAction {
	switch prefix[0] {
	case 'A':
		return stats.AllowAction
	case 'D':
		return stats.DenyAction
	case 'N':
		return stats.NextTierAction
	default:
		log.Error("Unknown action in ", prefix)
		return stats.NextTierAction
	}
}

func lookupRule(lum *lookup.LookupManager, prefix string, epKey *model.WorkloadEndpointKey) stats.RuleTracePoint {
	log.Infof("Looking up rule prefix %s", prefix)
	var tier, policy string
	// Prefix formats are:
	// - A/rule index/profile name
	// - D/rule index/policy name / tier name
	// TODO (Matt): Add sensible rule UUIDs
	prefixChunks := strings.Split(prefix, "/")
	if len(prefixChunks) == 3 {
		// Profile
		tier = "profile"
		policy = prefixChunks[2]
	} else if len(prefixChunks) == 4 {
		// Tiered Policy
		tier = prefixChunks[3]
		policy = prefixChunks[2]
	} else {
		log.Error("Unable to parse NFLOG prefix ", prefix)
	}

	return stats.RuleTracePoint{
		TierID:   tier,
		PolicyID: policy,
		Rule:     prefixChunks[1],
		Action:   lookupAction(prefix),
		Index:    lum.GetPolicyIndex(epKey, &model.PolicyKey{Name: policy, Tier: tier}),
	}
}

// End Stubs
