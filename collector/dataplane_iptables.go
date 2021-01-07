// Copyright (c) 2018-2020 Tigera, Inc. All rights reserved.

// +build linux

package collector

import (
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/nfnetlink"
	"github.com/tigera/nfnetlink/nfnl"

	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/jitter"
	"github.com/projectcalico/felix/rules"
)

// NFLogReader consumes NFLog data and converts them to a format used by collector.
type NFLogReader struct {
	stopOnce sync.Once
	wg       sync.WaitGroup
	stopC    chan struct{}

	luc            *calc.LookupsCache
	nfIngressC     chan *nfnetlink.NflogPacketAggregate
	nfEgressC      chan *nfnetlink.NflogPacketAggregate
	nfIngressDoneC chan struct{}
	nfEgressDoneC  chan struct{}

	packetInfoC chan PacketInfo

	netlinkIngressGroup int
	netlinkEgressGroup  int
	bufSize             int
	servicesEnabled     bool
}

func NewNFLogReader(lookupsCache *calc.LookupsCache, inGrp, eGrp, bufSize int, services bool) *NFLogReader {
	return &NFLogReader{
		stopC:          make(chan struct{}),
		luc:            lookupsCache,
		nfIngressC:     make(chan *nfnetlink.NflogPacketAggregate, 1000),
		nfEgressC:      make(chan *nfnetlink.NflogPacketAggregate, 1000),
		nfIngressDoneC: make(chan struct{}),
		nfEgressDoneC:  make(chan struct{}),

		packetInfoC: make(chan PacketInfo, 1000),

		netlinkIngressGroup: inGrp,
		netlinkEgressGroup:  eGrp,
		bufSize:             bufSize,
		servicesEnabled:     services,
	}
}

func (r *NFLogReader) Start() error {
	if err := r.subscribe(); err != nil {
		return nil
	}

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.run()
	}()

	return nil
}

func (r *NFLogReader) Stop() {
	r.stopOnce.Do(func() {
		close(r.stopC)
	})
}

// Chan returns the channel with converted data structures
func (r *NFLogReader) PacketInfoChan() <-chan PacketInfo {
	return r.packetInfoC
}

func subscribeToNflog(gn int, nlBufSiz int, nflogChan chan *nfnetlink.NflogPacketAggregate, nflogDoneChan chan struct{}, enableServices bool) error {
	return nfnetlink.NflogSubscribe(gn, nlBufSiz, nflogChan, nflogDoneChan, enableServices)
}

func (r *NFLogReader) subscribe() error {
	err := subscribeToNflog(r.netlinkIngressGroup, r.bufSize, r.nfIngressC, r.nfIngressDoneC, r.servicesEnabled)
	if err != nil {
		return fmt.Errorf("Error when subscribing to NFLOG (ingress): %w", err)
	}

	err = subscribeToNflog(r.netlinkEgressGroup, r.bufSize, r.nfEgressC, r.nfEgressDoneC, r.servicesEnabled)
	if err != nil {
		return fmt.Errorf("Error when subscribing to NFLOG (egress): %w", err)
	}

	return nil
}

func (r *NFLogReader) run() {
	for {
		select {
		case <-r.stopC:
			return
		case nflogPacketAggr := <-r.nfIngressC:
			info := r.convertNflogPkt(rules.RuleDirIngress, nflogPacketAggr)
			r.packetInfoC <- info
		case nflogPacketAggr := <-r.nfEgressC:
			info := r.convertNflogPkt(rules.RuleDirEgress, nflogPacketAggr)
			r.packetInfoC <- info
		}
	}
}

func (r *NFLogReader) convertNflogPkt(dir rules.RuleDir, nPktAggr *nfnetlink.NflogPacketAggregate) PacketInfo {
	info := PacketInfo{
		Direction: dir,
		RuleHits:  make([]RuleHit, 0, len(nPktAggr.Prefixes)),
	}

	info.Tuple = extractTupleFromNflogTuple(nPktAggr.Tuple)
	if nPktAggr.IsDNAT {
		info.IsDNAT = true
		info.PreDNATTuple = extractTupleFromCtEntryTuple(nPktAggr.OriginalTuple)
	}

	for _, prefix := range nPktAggr.Prefixes {
		ruleID := r.luc.GetRuleIDFromNFLOGPrefix(prefix.Prefix)
		if ruleID == nil {
			continue
		}

		info.RuleHits = append(info.RuleHits, RuleHit{
			RuleID: ruleID,
			Hits:   prefix.Packets,
			Bytes:  prefix.Bytes,
		})
	}

	return info
}

func convertCtEntryToConntrackInfo(ctEntry nfnetlink.CtEntry) (ConntrackInfo, error) {
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
		ctTuple, err = ctEntry.OriginalTuplePostDNAT()
		if err != nil {
			return ConntrackInfo{}, fmt.Errorf("Error when extracting tuple without DNAT: %w", err)
		}
	}

	// At this point either the source or destination IP address from the conntrack entry
	// belongs to an endpoint i.e., the connection either begins or terminates locally.
	tuple := extractTupleFromCtEntryTuple(ctTuple)

	// In the case of TCP, check if we can expire the entry early. We try to expire
	// entries early so that we don't send any spurious MetricUpdates for an expiring
	// conntrack entry.
	entryExpired := (ctTuple.ProtoNum == nfnl.TCP_PROTO && ctEntry.ProtoInfo.State >= nfnl.TCP_CONNTRACK_TIME_WAIT)

	ctInfo := ConntrackInfo{
		Tuple:   tuple,
		Expired: entryExpired,
		Counters: ConntrackCounters{
			Packets: ctEntry.OriginalCounters.Packets,
			Bytes:   ctEntry.OriginalCounters.Bytes,
		},
		ReplyCounters: ConntrackCounters{
			Packets: ctEntry.ReplyCounters.Packets,
			Bytes:   ctEntry.ReplyCounters.Bytes,
		},
	}

	if ctEntry.IsDNAT() {
		ctInfo.IsDNAT = true
		ctInfo.PreDNATTuple = extractTupleFromCtEntryTuple(ctEntry.OriginalTuple)
	}

	return ctInfo, nil
}

// NetLinkConntrackReader reads connrack information from Linux via netlink.
type NetLinkConntrackReader struct {
	stopOnce sync.Once
	wg       sync.WaitGroup
	stopC    chan struct{}

	ticker jitter.JitterTicker
	outC   chan ConntrackInfo
}

// NewNetLinkConntrackReader returns a new NetLinkConntrackReader
func NewNetLinkConntrackReader(period time.Duration) *NetLinkConntrackReader {
	return &NetLinkConntrackReader{
		stopC:  make(chan struct{}),
		ticker: jitter.NewTicker(period, period/10),
		outC:   make(chan ConntrackInfo, 1000),
	}
}

func (r *NetLinkConntrackReader) Start() error {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.run()
	}()

	return nil
}

func (r *NetLinkConntrackReader) Stop() {
	r.stopOnce.Do(func() {
		close(r.stopC)
		r.ticker.Stop()
	})
}

func (r *NetLinkConntrackReader) run() {
	for {
		select {
		case <-r.stopC:
			return
		case <-r.ticker.Channel():
			_ = nfnetlink.ConntrackList(r.processCtEntry)
		}
	}
}

func (r *NetLinkConntrackReader) processCtEntry(ctEntry nfnetlink.CtEntry) {
	ctInfo, err := convertCtEntryToConntrackInfo(ctEntry)
	if err != nil {
		log.Error(err.Error())
		return
	}

	select {
	case r.outC <- ctInfo:
	case <-r.stopC:
	}
}

func (r *NetLinkConntrackReader) ConntrackInfoChan() <-chan ConntrackInfo {
	return r.outC
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
	return MakeTuple(nflogTuple.Src, nflogTuple.Dst, nflogTuple.Proto, l4Src, l4Dst)
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
	return MakeTuple(ctTuple.Src, ctTuple.Dst, ctTuple.ProtoNum, l4Src, l4Dst)
}
