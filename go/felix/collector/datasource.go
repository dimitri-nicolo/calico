// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"time"

	"github.com/tigera/felix-private/go/felix/collector/stats"
	"github.com/tigera/libcalico-go-private/lib/backend/model"
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
	fmt.Println("Starting NFLOG Data Source for direction", ds.direction)
	seedRand()
	go ds.subscribeToNflog()
}

func (ds *NflogDataSource) subscribeToNflog() {
	ch := make(chan nfnetlink.NflogPacket)
	done := make(chan struct{})
	defer close(done)
	err := nfnetlink.NflogSubscribe(ds.groupNum, ch, done)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	for nflogPacket := range ch {
		statUpdates, err := ds.convertNflogPktToStat(nflogPacket)
		if err != nil {
			fmt.Println("Error: ", err)
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
	var numPkts, numBytes int
	if tp.Action == stats.DenyAction {
		// NFLog based counters make sense only for denied packets.
		// Allowed packet counters are updated via conntrack datasource.
		// FIXME(doublek): The above assumption is not true in the case of NOTRACK.
		numPkts = 1
		numBytes = nPkt.Bytes
	} else {
		numPkts = 0
		numBytes = 0
	}

	// FIXME(doublek): We should not increase packet counters for the same packet
	// hitting multiple rules more than once. Right now it is.
	// One way to do this could be, only increment counters when the actual "allow/deny"
	// rule is hit.
	switch {
	case wlEpKeySrc != nil && wlEpKeyDst != nil:
		// Local to Local Traffic, the direction is determined by the datasource from
		// which we received the packet.
		var su *stats.StatUpdate
		tuple := extractTupleFromNflogTuple(nPkt.Tuple)
		if ds.direction == stats.DirIn {
			su = stats.NewStatUpdate(tuple, *wlEpKeyDst, numPkts, numBytes, 0, 0, stats.DeltaCounter, tp)
		} else {
			su = stats.NewStatUpdate(tuple, *wlEpKeySrc, 0, 0, numPkts, numBytes, stats.DeltaCounter, tp)
		}
		statUpdates = append(statUpdates, *su)
	case wlEpKeySrc != nil:
		// Locally originating packet
		tuple := extractTupleFromNflogTuple(nPkt.Tuple)
		su := stats.NewStatUpdate(tuple, *wlEpKeySrc, 0, 0, numPkts, numBytes, stats.DeltaCounter, tp)
		statUpdates = append(statUpdates, *su)
	case wlEpKeyDst != nil:
		// Locally terminating packet
		tuple := extractTupleFromNflogTuple(nPkt.Tuple)
		su := stats.NewStatUpdate(tuple, *wlEpKeyDst, numPkts, numBytes, 0, 0, stats.DeltaCounter, tp)
		statUpdates = append(statUpdates, *su)
	}
	return statUpdates, nil
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

func extractTupleFromNflogTupleReverse(nflogTuple nfnetlink.NflogPacketTuple) stats.Tuple {
	var l4Src, l4Dst int
	if nflogTuple.Proto == 1 {
		l4Src = nflogTuple.L4Src.Id
		l4Dst = int(uint16(nflogTuple.L4Dst.Type)<<8 | uint16(nflogTuple.L4Dst.Code))
	} else {
		l4Src = nflogTuple.L4Src.Port
		l4Dst = nflogTuple.L4Dst.Port
	}
	return *stats.NewTuple(nflogTuple.Dst, nflogTuple.Src, nflogTuple.Proto, l4Dst, l4Src)
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
	fmt.Println("Starting Conntrack Data Source")
	seedRand()
	go ds.startPolling()
}

func (ds *ConntrackDataSource) startPolling() {
	c := time.Tick(time.Second)
	for _ = range c {
		ctentries, err := nfnetlink.ConntrackList()
		if err != nil {
			fmt.Println("Error: ", err)
			return
		}
		// TODO(doublek): Possibly do this in a separate goroutine?
		for _, ctentry := range ctentries {
			statUpdates, err := convertCtEntryToStat(ctentry)
			if err != nil {
				fmt.Println("Error: ", err)
				continue
			}
			for _, su := range statUpdates {
				ds.sink <- su
			}
		}
	}
}

func convertCtEntryToStat(ctEntry nfnetlink.CtEntry) ([]stats.StatUpdate, error) {
	// There can be a maximum of 2 stat updates per Nflog Packet, in the case of
	// local-to-local traffic.
	statUpdates := []stats.StatUpdate{}
	// The last entry is the tuple entry for endpoints
	ctTuple := ctEntry.OrigTuples[len(ctEntry.OrigTuples)-1]
	wlEpKeySrc := lookupEndpoint(ctTuple.Src)
	wlEpKeyDst := lookupEndpoint(ctTuple.Dst)
	// Force conntrack to have empty tracep
	tp := stats.RuleTracePoint{}
	switch {
	case wlEpKeySrc != nil && wlEpKeyDst != nil:
		// Local to Local Traffic
		var tuple stats.Tuple
		var su *stats.StatUpdate
		tuple = extractTupleFromCtEntryTuple(ctTuple)
		su = stats.NewStatUpdate(tuple, *wlEpKeySrc,
			ctEntry.ReplCounters.Packets, ctEntry.ReplCounters.Bytes,
			ctEntry.OrigCounters.Packets, ctEntry.OrigCounters.Bytes, stats.AbsoluteCounter, tp)
		statUpdates = append(statUpdates, *su)
		tuple = extractTupleFromCtEntryTupleReverse(ctTuple)
		su = stats.NewStatUpdate(tuple, *wlEpKeyDst,
			ctEntry.OrigCounters.Packets, ctEntry.OrigCounters.Bytes,
			ctEntry.ReplCounters.Packets, ctEntry.ReplCounters.Bytes, stats.AbsoluteCounter, tp)
		statUpdates = append(statUpdates, *su)
	case wlEpKeySrc != nil:
		// Locally originating packet
		tuple := extractTupleFromCtEntryTuple(ctTuple)
		su := stats.NewStatUpdate(tuple, *wlEpKeySrc,
			ctEntry.ReplCounters.Packets, ctEntry.ReplCounters.Bytes,
			ctEntry.OrigCounters.Packets, ctEntry.OrigCounters.Bytes, stats.AbsoluteCounter, tp)
		statUpdates = append(statUpdates, *su)
	case wlEpKeyDst != nil:
		// Locally terminating packet
		tuple := extractTupleFromCtEntryTupleReverse(ctTuple)
		su := stats.NewStatUpdate(tuple, *wlEpKeyDst,
			ctEntry.OrigCounters.Packets, ctEntry.OrigCounters.Bytes,
			ctEntry.ReplCounters.Packets, ctEntry.ReplCounters.Bytes, stats.AbsoluteCounter, tp)
		statUpdates = append(statUpdates, *su)
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

func extractTupleFromCtEntryTupleReverse(ctTuple nfnetlink.CtTuple) stats.Tuple {
	var l4Src, l4Dst int
	if ctTuple.ProtoNum == 1 {
		l4Src = ctTuple.L4Src.Id
		l4Dst = int(uint16(ctTuple.L4Dst.Type)<<8 | uint16(ctTuple.L4Dst.Code))
	} else {
		l4Src = ctTuple.L4Src.Port
		l4Dst = ctTuple.L4Dst.Port
	}
	return *stats.NewTuple(ctTuple.Dst, ctTuple.Src, ctTuple.ProtoNum, l4Dst, l4Src)
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
	switch prefix[4] {
	case 'A':
		action = stats.AllowAction
	case 'D':
		action = stats.DenyAction
	case 'N':
		action = stats.NextTierAction
	}
	idx, _ := strconv.Atoi(prefix[5 : len(prefix)-1])
	return stats.RuleTracePoint{
		TierID:   prefix[0:2],
		PolicyID: prefix[2:4],
		Action:   action,
		Index:    idx,
	}
}

// End Stubs
