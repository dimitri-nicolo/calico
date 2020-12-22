// Copyright (c) 2018-2020 Tigera, Inc. All rights reserved.
package collector

import (
	"strings"
	"sync"
	"time"

	"sigs.k8s.io/kind/pkg/errors"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/windows-networking/pkg/vfpctrl"

	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/jitter"
	"github.com/projectcalico/felix/rules"
)

const windowsCollectorETWSession = "tigera-calico-etw-vfp"

// VFPInfoReader implements collector.PacketInfoReader and collector.ConntrackInfoReader.
// It makes sense to have a single go routine handling VFP events/flows to avoid possible race
// on same endpoints cache of underlying structure.
type VFPInfoReader struct {
	callOnce sync.Once
	wg       sync.WaitGroup
	stopC    chan struct{}

	luc *calc.LookupsCache

	eventAggrC chan *vfpctrl.EventAggregate
	eventDoneC chan struct{}

	etw *vfpctrl.EtwOperations
	vfp *vfpctrl.VfpOperations

	packetInfoC chan PacketInfo

	ticker         jitter.JitterTicker
	conntrackInfoC chan ConntrackInfo
}

func NewVFPInfoReader(lookupsCache *calc.LookupsCache, period time.Duration) *VFPInfoReader {
	etw, err := vfpctrl.NewEtwOperations([]int{vfpctrl.VFP_EVENT_ID_ENDPOINT_ACL}, windowsCollectorETWSession)
	if err != nil {
		log.WithError(err).Fatalf("Failed to create ETW operations")
	}

	vfp := vfpctrl.NewVfpOperations()

	return &VFPInfoReader{
		stopC:          make(chan struct{}),
		luc:            lookupsCache,
		etw:            etw,
		vfp:            vfp,
		eventAggrC:     make(chan *vfpctrl.EventAggregate, 1000),
		eventDoneC:     make(chan struct{}, 1),
		packetInfoC:    make(chan PacketInfo, 1000),
		ticker:         jitter.NewTicker(period, period/10),
		conntrackInfoC: make(chan ConntrackInfo, 1000),
	}
}

func (r *VFPInfoReader) Start() error {
	var ret error
	r.callOnce.Do(func() {
		if err := r.subscribe(); err != nil {
			ret = err
			return
		}

		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			r.run()
		}()
	})

	return ret
}

func (r *VFPInfoReader) Stop() {
	r.callOnce.Do(func() {
		close(r.stopC)
	})
}

// PacketInfoChan returns the channel with converted PacketInfo.
func (r *VFPInfoReader) PacketInfoChan() <-chan PacketInfo {
	return r.packetInfoC
}

// ConntrackInfoChan returns the channel with converted ConntrackInfo.
func (r *VFPInfoReader) ConntrackInfoChan() <-chan ConntrackInfo {
	return r.conntrackInfoC
}

// VfpEventChan returns the channel to send down events consumed by VFP.
func (r *VFPInfoReader) VfpEventChan() chan<- interface{} {
	return r.vfp.EventChan()
}

// Subscribe subscribes the reader to the ETW event stream.
func (r *VFPInfoReader) subscribe() error {
	return r.etw.Subscribe(r.eventAggrC, r.eventDoneC)
}

func (r *VFPInfoReader) run() {
	for {
		select {
		case <-r.stopC:
			return
		case eventAggr := <-r.eventAggrC:
			infoPointer, err := r.convertEventAggrPkt(eventAggr)
			if err == nil {
				r.packetInfoC <- *infoPointer
			}
		case <-r.ticker.Channel():
			r.vfp.ListFlows(r.handleFlowEntry)
		case endpointEvent := <-r.vfp.EventChan():
			r.vfp.HandleEndpointEvent(endpointEvent)
		}
	}
}

func (r *VFPInfoReader) convertEventAggrPkt(ea *vfpctrl.EventAggregate) (*PacketInfo, error) {
	var dir rules.RuleDir

	tuple, err := extractTupleFromEventAggr(ea)
	if err != nil {
		log.WithError(err).Errorf("failed to get tuple from ETW event")
		return nil, err
	}

	if ea.Event.IsIngress() {
		dir = rules.RuleDirIngress
	} else {
		dir = rules.RuleDirEgress
	}

	// Event could happen on an endpoint before we get a notification from Felix endpoint manager.
	r.vfp.MayAddNewEndpoint(ea.Event.EndpointID())

	ruleName, err := r.vfp.GetRuleFriendlyNameForEvent(ea.Event)
	if err != nil {
		log.WithError(err).Warnf("failed to get rule name from ETW event")
		return nil, err
	}

	// Lookup the ruleID from the prefix.
	var arr [64]byte
	prefixStr := extractPrefixStrFromRuleName(ruleName)
	copy(arr[:], prefixStr)
	ruleID := r.luc.GetRuleIDFromNFLOGPrefix(arr)
	if ruleID == nil {
		return nil, errors.New("failed to get rule id by policy lookup")
	}

	// Etw Event has one RuleHits prefix.
	// It has no service ip information (DNAT).
	// It has no bytes information.
	info := PacketInfo{
		IsDNAT:    false,
		Direction: dir,
		RuleHits:  make([]RuleHit, 0, 1),
		Tuple:     *tuple,
	}

	info.RuleHits = append(info.RuleHits, RuleHit{
		RuleID: ruleID,
		Hits:   ea.Count,
		Bytes:  0,
	})

	return &info, nil
}

func convertFlowEntry(fe *vfpctrl.FlowEntry) (*ConntrackInfo, error) {
	tuple, err := extractTupleFromFlowEntry(fe)
	if err != nil {
		return nil, err
	}

	// In the case of TCP, check if we can expire the entry early. We try to expire
	// entries early so that we don't send any spurious MetricUpdates for an expiring
	// conntrack entry.
	entryExpired := fe.ConnectionClosed()

	ctInfo := ConntrackInfo{
		Tuple:   *tuple,
		Expired: entryExpired,
		Counters: ConntrackCounters{
			Packets: fe.PktsOut,
			Bytes:   fe.BytesOut,
		},
		ReplyCounters: ConntrackCounters{
			Packets: fe.PktsIn,
			Bytes:   fe.BytesIn,
		},
	}

	if fe.IsDNAT() {
		vTuple, err := extractPreDNATTupleFromFlowEntry(fe)
		if err != nil {
			return nil, err
		}
		ctInfo.IsDNAT = true
		ctInfo.PreDNATTuple = *vTuple
	}

	return &ctInfo, nil
}

func (r *VFPInfoReader) handleFlowEntry(fe *vfpctrl.FlowEntry) {
	ctInfoPointer, err := convertFlowEntry(fe)
	if err != nil {
		log.WithError(err).Warnf("failed to convert flow entry")
		return
	}

	select {
	case r.conntrackInfoC <- *ctInfoPointer:
	case <-r.stopC:
	}
}

func extractPrefixStrFromRuleName(name string) string {
	// Windows dataplane programs hns rules with two types of format for rule name.
	// prefix---sequence number   This is used for policy rules.
	// prefix                     This is used for default deny rules.
	strs := strings.Split(name, "---")
	if len(strs) != 2 {
		return name
	}
	return strs[0]
}

func extractTupleFromEventAggr(ea *vfpctrl.EventAggregate) (*Tuple, error) {
	tuple, err := ea.Event.Tuple()
	if err != nil {
		return nil, err
	}
	return NewTuple(tuple.Src, tuple.Dst, tuple.Proto, tuple.L4DstPort, tuple.L4DstPort), nil
}

func extractTupleFromFlowEntry(fe *vfpctrl.FlowEntry) (*Tuple, error) {
	tuple, err := fe.Tuple()
	if err != nil {
		return nil, err
	}
	return NewTuple(tuple.Src, tuple.Dst, tuple.Proto, tuple.L4DstPort, tuple.L4DstPort), nil
}

func extractPreDNATTupleFromFlowEntry(fe *vfpctrl.FlowEntry) (*Tuple, error) {
	tuple, err := fe.TuplePreDNAT()
	if err != nil {
		return nil, err
	}
	return NewTuple(tuple.Src, tuple.Dst, tuple.Proto, tuple.L4DstPort, tuple.L4DstPort), nil
}
