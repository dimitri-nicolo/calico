// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.

package events

import (
	"net"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/collector"
)

var (
	eventsCollectorBlocksCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "felix_bpf_events_collector_blocks",
		Help: "CollectorPolicyListener blocks",
	})
)

func init() {
	prometheus.MustRegister(eventsCollectorBlocksCounter)
}

// CollectorPolicyListener is a backend plugin for the Collector to consume
// events from BPF policy programs and turn them into the common format.
type CollectorPolicyListener struct {
	lc   *calc.LookupsCache
	inC  chan PolicyVerdict
	outC chan collector.PacketInfo
}

// NewCollectorPolicyListener return a new instance of a CollectorPolicyListener.
func NewCollectorPolicyListener(lc *calc.LookupsCache) *CollectorPolicyListener {
	return &CollectorPolicyListener{
		lc:   lc,
		inC:  make(chan PolicyVerdict, 100),
		outC: make(chan collector.PacketInfo),
	}
}

// EventHandler can be registered as a sink/callback to consume the event.
func (c *CollectorPolicyListener) EventHandler(e Event) {
	pv := ParsePolicyVerdict(e.Data())
	c.inC <- pv
}

// Start starts consuming events, converting and passong them to collector
func (c *CollectorPolicyListener) Start() error {
	go c.run()
	return nil
}

func makeTuple(src, dst net.IP, proto uint8, srcPort, dstPort uint16) collector.Tuple {
	var src16, dst16 [16]byte

	copy(src16[:], src.To16())
	copy(dst16[:], dst.To16())

	return collector.MakeTuple(src16, dst16, int(proto), int(srcPort), int(dstPort))
}

func (c *CollectorPolicyListener) run() {
	for {
		e, ok := <-c.inC

		if !ok {
			return
		}

		if e.RulesHit == 0 {
			// This should never happen, so just to be sure. We cannot determine
			// direction, skip it.
			continue
		}

		pktInfo := collector.PacketInfo{
			IsDNAT:   !e.DstAddr.Equal(e.PostNATDstAddr) || e.DstPort != e.PostNATDstPort,
			Tuple:    makeTuple(e.SrcAddr, e.PostNATDstAddr, e.IPProto, e.SrcPort, e.PostNATDstPort),
			RuleHits: make([]collector.RuleHit, e.RulesHit),
		}

		if pktInfo.IsDNAT {
			pktInfo.PreDNATTuple = makeTuple(e.SrcAddr, e.DstAddr, e.IPProto, e.SrcPort, e.DstPort)
		}

		for i := 0; i < int(e.RulesHit); i++ {
			id := e.RuleIDs[i]
			rid := c.lc.GetRuleIDFromID64(id)
			pktInfo.RuleHits[i] = collector.RuleHit{
				RuleID: rid,
				Hits:   1,
				Bytes:  int(e.IPSize),
			}

			// All directions should be the same
			if rid != nil {
				pktInfo.Direction = rid.Direction
			}
		}

		select {
		case c.outC <- pktInfo:
			// nothing, all good
		default:
			eventsCollectorBlocksCounter.Inc()
			c.outC <- pktInfo
		}
	}
}

// Stop stops the listener, mainly for testing purposes.
func (c *CollectorPolicyListener) Stop() {
	close(c.inC)
}

// PacketInfoChan provides the output channel with converted information.
func (c *CollectorPolicyListener) PacketInfoChan() <-chan collector.PacketInfo {
	return c.outC
}
