// Copyright (c) 2016-2017 Tigera, Inc. All rights reserved.

package collector

import (
	"errors"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/projectcalico/felix/jitter"
	"github.com/projectcalico/felix/lookup"
	"github.com/tigera/nfnetlink"
)

type epLookup interface {
	GetEndpointKey(addr [16]byte) (interface{}, error)
	GetPolicyIndex(epKey interface{}, policyName, tierName []byte) int
}

const PollingInterval = time.Duration(1) * time.Second

type NflogDataSource struct {
	sink          chan<- *StatUpdate
	groupNum      int
	direction     Direction
	nlBufSiz      int
	lum           epLookup
	nflogChan     chan *nfnetlink.NflogPacketAggregate
	nflogDoneChan chan struct{}
}

func NewNflogDataSource(lm epLookup, sink chan<- *StatUpdate, groupNum int, dir Direction, nlBufSiz int) *NflogDataSource {
	nflogChan := make(chan *nfnetlink.NflogPacketAggregate, 1000)
	done := make(chan struct{})
	return newNflogDataSource(lm, sink, groupNum, dir, nlBufSiz, nflogChan, done)
}

// Internal constructor to help mocking from UTs.
func newNflogDataSource(lm epLookup, sink chan<- *StatUpdate, gn int, dir Direction, nbs int, nc chan *nfnetlink.NflogPacketAggregate, done chan struct{}) *NflogDataSource {
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
	for nflogPacketAggr := range ds.nflogChan {
		go func(nflogPacketAggr *nfnetlink.NflogPacketAggregate) {
			statUpdates, err := ds.convertNflogPktToStat(nflogPacketAggr)
			if err != nil {
				log.Debugf("Cannot convert Nflog packet %v to StatUpdate", nflogPacketAggr)
				return
			}
			for _, statUpdate := range statUpdates {
				ds.sink <- statUpdate
			}
		}(nflogPacketAggr)
	}
}

func (ds *NflogDataSource) convertNflogPktToStat(nPktAggr *nfnetlink.NflogPacketAggregate) ([]*StatUpdate, error) {
	var (
		numPkts, numBytes int
		statUpdates       []*StatUpdate
		epKey             interface{}
		err               error
	)
	nflogTuple := nPktAggr.Tuple
	// Determine the endpoint that this packet hit a rule for. This depends on the direction
	// because local -> local packets will be NFLOGed twice.
	if ds.direction == DirIn {
		epKey, err = ds.lum.GetEndpointKey(nflogTuple.Dst)
	} else {
		epKey, err = ds.lum.GetEndpointKey(nflogTuple.Src)
	}

	if err == lookup.UnknownEndpointError {
		// TODO (Matt): This branch becomes much more interesting with graceful restart.
		log.Debugf("Failed to find endpoint for NFLOG packet %v/%v", nflogTuple, ds.direction)
		return statUpdates, errors.New("Couldn't find endpoint info for NFLOG packet")
	}
	for _, prefix := range nPktAggr.Prefixes {
		tp, err := lookupRule(ds.lum, prefix.Prefix, prefix.Len, epKey)
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
		statUpdate := NewStatUpdate(tuple, 0, 0, 0, 0, DeltaCounter, ds.direction, tp)
		statUpdates = append(statUpdates, statUpdate)
	}
	return statUpdates, nil
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

type ConntrackDataSource struct {
	poller    *jitter.Ticker
	pollerC   <-chan time.Time
	converter chan []nfnetlink.CtEntry
	sink      chan<- *StatUpdate
	lum       epLookup
}

func NewConntrackDataSource(lm epLookup, sink chan<- *StatUpdate) *ConntrackDataSource {
	poller := jitter.NewTicker(PollingInterval, PollingInterval/10)
	converter := make(chan []nfnetlink.CtEntry)
	return newConntrackDataSource(lm, sink, poller, poller.C, converter)
}

// Internal constructor to help mocking from UTs.
func newConntrackDataSource(lm epLookup, sink chan<- *StatUpdate, poller *jitter.Ticker, pollerC <-chan time.Time, converter chan []nfnetlink.CtEntry) *ConntrackDataSource {
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
			go func(ctentry nfnetlink.CtEntry) {
				statUpdates, err := ds.convertCtEntryToStat(ctentry)
				if err != nil {
					log.Errorf("Couldn't convert ctentry %v to StatUpdate", ctentry)
					return
				}
				for _, su := range statUpdates {
					ds.sink <- su
				}
			}(ctentry)
		}
	}
}

func (ds *ConntrackDataSource) convertCtEntryToStat(ctEntry nfnetlink.CtEntry) ([]*StatUpdate, error) {
	// There can be a maximum of 2 stat updates per ctentry, in the case of
	// local-to-local traffic.
	statUpdates := []*StatUpdate{}
	// The last entry is the tuple entry for endpoints
	ctTuple, err := ctEntry.OriginalTuple()
	if err != nil {
		log.Errorf("Error when extracting OriginalTuple: '%v'", err)
		return statUpdates, err
	}

	// We care about DNAT only which modifies the destination parts of the OriginalTuple.
	if ctEntry.IsDNAT() {
		ctTuple, err = ctEntry.OriginalTupleWithoutDNAT()
		if err != nil {
			log.Error("Error when extracting tuple without DNAT:", err)
		}
	}
	_, errSrc := ds.lum.GetEndpointKey(ctTuple.Src)
	_, errDst := ds.lum.GetEndpointKey(ctTuple.Dst)
	if errSrc == lookup.UnknownEndpointError && errDst == lookup.UnknownEndpointError {
		// We always expect unknown entries for conntrack for things such as
		// management or local traffic. This log can get spammy if we log everything
		// because of which we don't return an error and log at debug level.
		log.Debugf("No known endpoints found for %v", ctEntry)
		return nil, nil
	}
	tuple := extractTupleFromCtEntryTuple(ctTuple, false)
	su := NewStatUpdate(tuple,
		ctEntry.OriginalCounters.Packets, ctEntry.OriginalCounters.Bytes,
		ctEntry.ReplyCounters.Packets, ctEntry.ReplyCounters.Bytes,
		AbsoluteCounter, DirUnknown, EmptyRuleTracePoint)
	statUpdates = append(statUpdates, su)
	if errSrc == nil && errDst == nil {
		// Locally to local packet will require a reversed tuple to collect reply stats.
		tuple := extractTupleFromCtEntryTuple(ctTuple, true)
		su := NewStatUpdate(tuple,
			ctEntry.ReplyCounters.Packets, ctEntry.ReplyCounters.Bytes,
			ctEntry.OriginalCounters.Packets, ctEntry.OriginalCounters.Bytes,
			AbsoluteCounter, DirUnknown, EmptyRuleTracePoint)
		statUpdates = append(statUpdates, su)
	}
	return statUpdates, nil
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

// Stubs
// TODO (Matt): Refactor these in better.

func lookupRule(lum epLookup, prefix [64]byte, prefixLen int, epKey interface{}) (RuleTracePoint, error) {
	rtp, err := NewRuleTracePoint(prefix, prefixLen, epKey)
	if err != nil {
		return rtp, err
	}
	index := lum.GetPolicyIndex(epKey, rtp.PolicyID(), rtp.TierID())
	rtp.Index = index
	return rtp, nil
}

// End Stubs
