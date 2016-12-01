// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/projectcalico/felix/go/felix/collector/stats"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/tigera/nfnetlink"
)

type NflogDataSource struct {
	sink      chan<- stats.StatUpdate
	groupNum  int
	direction stats.Direction
}

func NewNflogDataSource(sink chan<- stats.StatUpdate, groupNum int, dir stats.Direction) *NflogDataSource {
	return &NflogDataSource{
		sink:      sink,
		groupNum:  groupNum,
		direction: dir,
	}
}

func (ds *NflogDataSource) Start() {
	log.Infof("Starting NFLOG Data Source for direction %v group %v", ds.direction, ds.groupNum)
	seedRand()
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
		statUpdates, err := ds.convertNflogPktToStat(nflogPacket)
		if err != nil {
			log.Errorf("Cannot convert Nflog packet %v to StatUpdate", nflogPacket)
			continue
		}
		for _, su := range statUpdates {
			ds.sink <- su
		}
	}
}

func (ds *NflogDataSource) convertNflogPktToStat(nPkt nfnetlink.NflogPacket) ([]stats.StatUpdate, error) {
	statUpdates := []stats.StatUpdate{}
	nflogTuple := nPkt.Tuple
	wlEpKeySrc := lookupEndpoint(nflogTuple.Src)
	wlEpKeyDst := lookupEndpoint(nflogTuple.Dst)
	tp := lookupRule(nPkt.Prefix)
	var numPkts, numBytes, inPkts, inBytes, outPkts, outBytes int
	if tp.Action == stats.DenyAction {
		// NFLog based counters make sense only for denied packets.
		// Allowed packet counters are updated via conntrack datasource.
		// FIXME(doublek): This assumption is not true in the case of NOTRACK.
		numPkts = 1
		numBytes = nPkt.Bytes
	} else {
		numPkts = 0
		numBytes = 0
	}

	if ds.direction == stats.DirIn {
		inPkts = numPkts
		inBytes = numBytes
		outPkts = 0
		outBytes = 0
	} else {
		inPkts = 0
		inBytes = 0
		outPkts = numPkts
		outBytes = numBytes
	}

	// FIXME(doublek): We should not increase packet counters for the same packet
	// hitting multiple rules more than once. Right now it is.
	// One way to do this could be, only increment counters when the actual "allow/deny"
	// rule is hit.
	if wlEpKeySrc != nil {
		// Locally originating packet
		tuple := extractTupleFromNflogTuple(nPkt.Tuple, false)
		su := stats.NewStatUpdate(tuple, *wlEpKeySrc, inPkts, inBytes, outPkts, outBytes, stats.DeltaCounter, tp)
		statUpdates = append(statUpdates, *su)
	}
	if wlEpKeyDst != nil {
		// Locally terminating packet
		tuple := extractTupleFromNflogTuple(nPkt.Tuple, false)
		su := stats.NewStatUpdate(tuple, *wlEpKeyDst, inPkts, inBytes, outPkts, outBytes, stats.DeltaCounter, tp)
		statUpdates = append(statUpdates, *su)
	}
	return statUpdates, nil
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
}

func NewConntrackDataSource(sink chan<- stats.StatUpdate) *ConntrackDataSource {
	return &ConntrackDataSource{
		sink: sink,
	}
}

func (ds *ConntrackDataSource) Start() {
	log.Info("Starting Conntrack Data Source")
	seedRand()
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
			statUpdates, err := convertCtEntryToStat(ctentry)
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

func convertCtEntryToStat(ctEntry nfnetlink.CtEntry) ([]stats.StatUpdate, error) {
	// There can be a maximum of 2 stat updates per ctentry, in the case of
	// local-to-local traffic.
	statUpdates := []stats.StatUpdate{}
	// The last entry is the tuple entry for endpoints
	ctTuple := ctEntry.OrigTuples[len(ctEntry.OrigTuples)-1]
	wlEpKeySrc := lookupEndpoint(ctTuple.Src)
	wlEpKeyDst := lookupEndpoint(ctTuple.Dst)
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
const Seed = 72

var r *rand.Rand

func seedRand() {
	r = rand.New(rand.NewSource(Seed))
}

func randomName(prefix string) string {
	return fmt.Sprintf("%v-%v", prefix, r.Int())
}

func lookupEndpoint(ipAddr net.IP) *model.WorkloadEndpointKey {
	// TODO(doublek): Look at IP and return appropriately.
	workloadId := randomName("workload")
	endpointId := randomName("endpoint")
	return &model.WorkloadEndpointKey{
		Hostname:       "MyHost",
		OrchestratorID: "ASDF",
		WorkloadID:     workloadId,
		EndpointID:     endpointId,
	}
}

func lookupRule(prefix string) stats.RuleTracePoint {
	var action stats.RuleAction
	log.Infof("Looking up rule prefix %s", prefix)
	switch prefix[0] {
	case 'A':
		action = stats.AllowAction
	case 'D':
		action = stats.DenyAction
	case 'N':
		action = stats.NextTierAction
	}
	// TODO (Matt): This doesn't really work
	idx, _ := strconv.Atoi(prefix[8 : len(prefix)-1])
	return stats.RuleTracePoint{
		TierID:   prefix[1:2],
		PolicyID: prefix[3:5],
		Action:   action,
		Index:    idx,
	}
}

// End Stubs
