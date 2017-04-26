// Copyright (c) 2016-2017 Tigera, Inc. All rights reserved.

package collector

import (
	"bytes"
	"errors"
	"net"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/projectcalico/felix/jitter"
	"github.com/projectcalico/felix/lookup"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/tigera/nfnetlink"
)

type epLookup interface {
	GetEndpointKey(addr net.IP) (interface{}, error)
	GetPolicyIndex(epKey interface{}, policyKey *model.PolicyKey) int
}

const PollingInterval = time.Duration(1) * time.Second

type NflogDataSource struct {
	sink          chan<- StatUpdate
	groupNum      int
	direction     Direction
	nlBufSiz      int
	lum           epLookup
	nflogChan     chan nfnetlink.NflogPacket
	nflogDoneChan chan struct{}
}

func NewNflogDataSource(lm epLookup, sink chan<- StatUpdate, groupNum int, dir Direction, nlBufSiz int) *NflogDataSource {
	nflogChan := make(chan nfnetlink.NflogPacket)
	done := make(chan struct{})
	return newNflogDataSource(lm, sink, groupNum, dir, nlBufSiz, nflogChan, done)
}

// Internal constructor to help mocking from UTs.
func newNflogDataSource(lm epLookup, sink chan<- StatUpdate, gn int, dir Direction, nbs int, nc chan nfnetlink.NflogPacket, done chan struct{}) *NflogDataSource {
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

func (ds *NflogDataSource) convertNflogPktToStat(nPkt nfnetlink.NflogPacket) (*StatUpdate, error) {
	nflogTuple := nPkt.Tuple
	var numPkts, numBytes int
	var statUpdate *StatUpdate
	var epKey interface{}
	var err error
	var prefixAction RuleAction
	_, _, _, prefixAction = parsePrefix(nPkt.Prefix)
	if prefixAction == DenyAction || prefixAction == AllowAction {
		// NFLog based counters make sense only for denied packets or allowed packets
		// under NOTRACK. When NOTRACK is not enabled, the conntrack based absolute
		// counters will overwrite these values anyway.
		numPkts = 1
		numBytes = nPkt.Bytes
	} else {
		// Allowed packet counters are updated via conntrack datasource.
		// Don't update packet counts for NextTierAction to avoid multiply counting.
		numPkts = 0
		numBytes = 0
	}

	// Determine the endpoint that this packet hit a rule for. This depends on the direction
	// because local -> local packets will be NFLOGed twice.
	if ds.direction == DirIn {
		epKey, err = ds.lum.GetEndpointKey(nflogTuple.Dst)
	} else {
		epKey, err = ds.lum.GetEndpointKey(nflogTuple.Src)
	}

	if err != lookup.UnknownEndpointError {
		tp := lookupRule(ds.lum, nPkt.Prefix, epKey)
		tuple := extractTupleFromNflogTuple(nPkt.Tuple)
		statUpdate = NewStatUpdate(tuple, numPkts, numBytes, 0, 0, DeltaCounter, ds.direction, tp)
	} else {
		// TODO (Matt): This branch becomes much more interesting with graceful restart.
		log.Warn("Failed to find endpoint for NFLOG packet ", nflogTuple, "/", ds.direction)
		return nil, errors.New("Couldn't find endpoint info for NFLOG packet")
	}

	log.Debug("Built NFLOG stat update: ", statUpdate)
	return statUpdate, nil
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

type ConntrackDataSource struct {
	poller    *jitter.Ticker
	pollerC   <-chan time.Time
	converter chan []nfnetlink.CtEntry
	sink      chan<- StatUpdate
	lum       epLookup
}

func NewConntrackDataSource(lm epLookup, sink chan<- StatUpdate) *ConntrackDataSource {
	poller := jitter.NewTicker(PollingInterval, PollingInterval/10)
	converter := make(chan []nfnetlink.CtEntry)
	return newConntrackDataSource(lm, sink, poller, poller.C, converter)
}

// Internal constructor to help mocking from UTs.
func newConntrackDataSource(lm epLookup, sink chan<- StatUpdate, poller *jitter.Ticker, pollerC <-chan time.Time, converter chan []nfnetlink.CtEntry) *ConntrackDataSource {
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

func (ds *ConntrackDataSource) convertCtEntryToStat(ctEntry nfnetlink.CtEntry) ([]StatUpdate, error) {
	// There can be a maximum of 2 stat updates per ctentry, in the case of
	// local-to-local traffic.
	statUpdates := []StatUpdate{}
	// The last entry is the tuple entry for endpoints
	ctTuple, err := ctEntry.OriginalTuple()
	if err != nil {
		log.Error("Error when extracting OriginalTuple:", err)
		return statUpdates, err
	}

	// We care about DNAT only which modifies the destination parts of the OriginalTuple.
	if ctEntry.IsDNAT() {
		log.Debugf("Entry is DNAT %+v", ctEntry)
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
	statUpdates = append(statUpdates, *su)
	if errSrc == nil && errDst == nil {
		// Locally to local packet will require a reversed tuple to collect reply stats.
		tuple := extractTupleFromCtEntryTuple(ctTuple, true)
		su := NewStatUpdate(tuple,
			ctEntry.ReplyCounters.Packets, ctEntry.ReplyCounters.Bytes,
			ctEntry.OriginalCounters.Packets, ctEntry.OriginalCounters.Bytes,
			AbsoluteCounter, DirUnknown, EmptyRuleTracePoint)
		statUpdates = append(statUpdates, *su)
	}
	for _, update := range statUpdates {
		log.Debug("Built conntrack stat update: ", update)
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

func lookupAction(action string) RuleAction {
	switch action {
	case "A":
		return AllowAction
	case "D":
		return DenyAction
	case "N":
		return NextTierAction
	default:
		log.Error("Unknown action ", action)
		return NextTierAction
	}
}

func parsePrefix(prefix string) (tier, policy, rule string, action RuleAction) {
	// Prefix formats are:
	// - A/rule index/profile name
	// - A/rule index/policy name/tier name
	// TODO (Matt): Add sensible rule UUIDs
	prefixChunks := strings.Split(prefix, "/")
	if len(prefixChunks) == 3 {
		// Profile
		// TODO (Matt): Need something better than profile;
		//              it won't work if that's the name of a tier.
		tier = "profile"
	} else if len(prefixChunks) == 4 {
		// Tiered Policy
		// TODO(doublek): Should fix where the null character was introduced,
		// which could be nfnetlink.
		tier = string(bytes.Trim([]byte(prefixChunks[3]), "\x00"))
	} else {
		log.Error("Unable to parse NFLOG prefix ", prefix)
	}

	action = lookupAction(prefixChunks[0])
	rule = prefixChunks[1]
	policy = prefixChunks[2]
	return
}

func lookupRule(lum epLookup, prefix string, epKey interface{}) RuleTracePoint {
	log.Infof("Looking up rule prefix %s", prefix)
	tier, policy, rule, action := parsePrefix(prefix)
	return RuleTracePoint{
		TierID:   tier,
		PolicyID: policy,
		Rule:     rule,
		Action:   action,
		Index:    lum.GetPolicyIndex(epKey, &model.PolicyKey{Name: policy, Tier: tier}),
		EpKey:    epKey,
	}
}

// End Stubs
