// Copyright (c) 2017 Tigera, Inc. All rights reserved.

package collector

import (
	"time"
	"testing"

	"github.com/projectcalico/felix/jitter"
	"github.com/projectcalico/felix/lookup"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/tigera/nfnetlink"
	"github.com/tigera/nfnetlink/nfnl"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	ipv4       = 0x800
	proto_icmp = 1
	proto_tcp  = 6
	proto_udp  = 17
)

var (
	localIp1     = [16]byte{10, 0, 0, 1}
	localIp2     = [16]byte{10, 0, 0, 2}
	remoteIp1    = [16]byte{20, 0, 0, 1}
	remoteIp2    = [16]byte{20, 0, 0, 2}
	localIp1DNAT = [16]byte{192, 168, 0, 1}
	localIp2DNAT = [16]byte{192, 168, 0, 2}
)

var (
	srcPort     = 54123
	dstPort     = 80
	dstPortDNAT = 8080
)

var localWlEPKey1 = &model.WorkloadEndpointKey{
	Hostname:       "localhost",
	OrchestratorID: "orchestrator",
	WorkloadID:     "localworkloadid1",
	EndpointID:     "localepid1",
}

var localWlEPKey2 = &model.WorkloadEndpointKey{
	Hostname:       "localhost",
	OrchestratorID: "orchestrator",
	WorkloadID:     "localworkloadid2",
	EndpointID:     "localepid2",
}

var remoteWlEPKey1 = &model.WorkloadEndpointKey{
	Hostname:       "localhost",
	OrchestratorID: "orchestrator",
	WorkloadID:     "localworkloadid1",
	EndpointID:     "remoteepid1",
}

// Entry remoteIp1:srcPort -> localIp1:dstPort
var inCtEntry = nfnetlink.CtEntry{
	OriginalTuples: []nfnetlink.CtTuple{
		nfnetlink.CtTuple{
			Src:        remoteIp1,
			Dst:        localIp1,
			L3ProtoNum: ipv4,
			ProtoNum:   proto_tcp,
			L4Src:      nfnetlink.CtL4Src{Port: srcPort},
			L4Dst:      nfnetlink.CtL4Dst{Port: dstPort},
		},
	},
	ReplyTuples: []nfnetlink.CtTuple{
		nfnetlink.CtTuple{
			Src:        localIp1,
			Dst:        remoteIp1,
			L3ProtoNum: ipv4,
			ProtoNum:   proto_tcp,
			L4Src:      nfnetlink.CtL4Src{Port: dstPort},
			L4Dst:      nfnetlink.CtL4Dst{Port: srcPort},
		},
	},
	OriginalCounters: nfnetlink.CtCounters{Packets: 1, Bytes: 100},
	ReplyCounters:    nfnetlink.CtCounters{Packets: 2, Bytes: 250},
}

var outCtEntry = nfnetlink.CtEntry{
	OriginalTuples: []nfnetlink.CtTuple{
		nfnetlink.CtTuple{
			Src:        localIp1,
			Dst:        remoteIp1,
			L3ProtoNum: ipv4,
			ProtoNum:   proto_tcp,
			L4Src:      nfnetlink.CtL4Src{Port: srcPort},
			L4Dst:      nfnetlink.CtL4Dst{Port: dstPort},
		},
	},
	ReplyTuples: []nfnetlink.CtTuple{
		nfnetlink.CtTuple{
			Src:        remoteIp1,
			Dst:        localIp1,
			L3ProtoNum: ipv4,
			ProtoNum:   proto_tcp,
			L4Src:      nfnetlink.CtL4Src{Port: dstPort},
			L4Dst:      nfnetlink.CtL4Dst{Port: srcPort},
		},
	},
	OriginalCounters: nfnetlink.CtCounters{Packets: 1, Bytes: 100},
	ReplyCounters:    nfnetlink.CtCounters{Packets: 2, Bytes: 250},
}

var localCtEntry = nfnetlink.CtEntry{
	OriginalTuples: []nfnetlink.CtTuple{
		nfnetlink.CtTuple{
			Src:        localIp1,
			Dst:        localIp2,
			L3ProtoNum: ipv4,
			ProtoNum:   proto_tcp,
			L4Src:      nfnetlink.CtL4Src{Port: srcPort},
			L4Dst:      nfnetlink.CtL4Dst{Port: dstPort},
		},
	},
	ReplyTuples: []nfnetlink.CtTuple{
		nfnetlink.CtTuple{
			Src:        localIp2,
			Dst:        localIp1,
			L3ProtoNum: ipv4,
			ProtoNum:   proto_tcp,
			L4Src:      nfnetlink.CtL4Src{Port: dstPort},
			L4Dst:      nfnetlink.CtL4Dst{Port: srcPort},
		},
	},
	OriginalCounters: nfnetlink.CtCounters{Packets: 1, Bytes: 100},
	ReplyCounters:    nfnetlink.CtCounters{Packets: 2, Bytes: 250},
}

// DNAT Conntrack Entries
// DNAT from localIp1DNAT:dstPortDNAT --> localIp1:dstPort
var inCtEntryWithDNAT = nfnetlink.CtEntry{
	OriginalTuples: []nfnetlink.CtTuple{
		nfnetlink.CtTuple{
			Src:        remoteIp1,
			Dst:        localIp1DNAT,
			L3ProtoNum: ipv4,
			ProtoNum:   proto_tcp,
			L4Src:      nfnetlink.CtL4Src{Port: srcPort},
			L4Dst:      nfnetlink.CtL4Dst{Port: dstPortDNAT},
		},
	},
	ReplyTuples: []nfnetlink.CtTuple{
		nfnetlink.CtTuple{
			Src:        localIp1,
			Dst:        remoteIp1,
			L3ProtoNum: ipv4,
			ProtoNum:   proto_tcp,
			L4Src:      nfnetlink.CtL4Src{Port: dstPort},
			L4Dst:      nfnetlink.CtL4Dst{Port: srcPort},
		},
	},
	Status:           nfnl.IPS_DST_NAT,
	OriginalCounters: nfnetlink.CtCounters{Packets: 1, Bytes: 100},
	ReplyCounters:    nfnetlink.CtCounters{Packets: 2, Bytes: 250},
}

// DNAT from localIp2DNAT:dstPortDNAT --> localIp2:dstPort
var localCtEntryWithDNAT = nfnetlink.CtEntry{
	OriginalTuples: []nfnetlink.CtTuple{
		nfnetlink.CtTuple{
			Src:        localIp1,
			Dst:        localIp2DNAT,
			L3ProtoNum: ipv4,
			ProtoNum:   proto_tcp,
			L4Src:      nfnetlink.CtL4Src{Port: srcPort},
			L4Dst:      nfnetlink.CtL4Dst{Port: dstPortDNAT},
		},
	},
	ReplyTuples: []nfnetlink.CtTuple{
		nfnetlink.CtTuple{
			Src:        localIp2,
			Dst:        localIp1,
			L3ProtoNum: ipv4,
			ProtoNum:   proto_tcp,
			L4Src:      nfnetlink.CtL4Src{Port: dstPort},
			L4Dst:      nfnetlink.CtL4Dst{Port: srcPort},
		},
	},
	Status:           nfnl.IPS_DST_NAT,
	OriginalCounters: nfnetlink.CtCounters{Packets: 1, Bytes: 100},
	ReplyCounters:    nfnetlink.CtCounters{Packets: 2, Bytes: 250},
}

var _ = Describe("Conntrack Datasource", func() {
	var ctSource *ConntrackDataSource
	var sink chan *StatUpdate
	var dataFeeder chan []nfnetlink.CtEntry
	BeforeEach(func() {
		epMap := map[[16]byte]*model.WorkloadEndpointKey{
			localIp1: localWlEPKey1,
			localIp2: localWlEPKey2,
		}
		lm := newMockLookupManager(epMap)
		sink = make(chan *StatUpdate)
		poller := jitter.NewTicker(time.Second, time.Second/10)
		mockTickerChan := make(chan time.Time)
		dataFeeder = make(chan []nfnetlink.CtEntry)
		ctSource = newConntrackDataSource(lm, sink, poller, mockTickerChan, dataFeeder)
		ctSource.Start()
	})
	Describe("Test local destination", func() {
		It("should receive a single stat update", func() {
			t := NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort)
			su := NewStatUpdate(*t,
				inCtEntry.OriginalCounters.Packets, inCtEntry.OriginalCounters.Bytes,
				inCtEntry.ReplyCounters.Packets, inCtEntry.ReplyCounters.Bytes,
				AbsoluteCounter, DirUnknown, EmptyRuleTracePoint)
			dataFeeder <- []nfnetlink.CtEntry{inCtEntry}
			Eventually(sink).Should(Receive(Equal(su)))
		})
	})
	Describe("Test local source", func() {
		It("should receive a single stat update", func() {
			t := NewTuple(localIp1, remoteIp1, proto_tcp, srcPort, dstPort)
			su := NewStatUpdate(*t,
				outCtEntry.OriginalCounters.Packets, outCtEntry.OriginalCounters.Bytes,
				outCtEntry.ReplyCounters.Packets, outCtEntry.ReplyCounters.Bytes,
				AbsoluteCounter, DirUnknown, EmptyRuleTracePoint)
			dataFeeder <- []nfnetlink.CtEntry{outCtEntry}
			Eventually(sink).Should(Receive(Equal(su)))
		})
	})
	Describe("Test local source to local destination", func() {
		It("should receive two stat updates - one for each endpoint", func() {
			t1 := NewTuple(localIp1, localIp2, proto_tcp, srcPort, dstPort)
			su1 := NewStatUpdate(*t1,
				localCtEntry.OriginalCounters.Packets, localCtEntry.OriginalCounters.Bytes,
				localCtEntry.ReplyCounters.Packets, localCtEntry.ReplyCounters.Bytes,
				AbsoluteCounter, DirUnknown, EmptyRuleTracePoint)
			t2 := NewTuple(localIp2, localIp1, proto_tcp, dstPort, srcPort)
			su2 := NewStatUpdate(*t2,
				localCtEntry.ReplyCounters.Packets, localCtEntry.ReplyCounters.Bytes,
				localCtEntry.OriginalCounters.Packets, localCtEntry.OriginalCounters.Bytes,
				AbsoluteCounter, DirUnknown, EmptyRuleTracePoint)
			dataFeeder <- []nfnetlink.CtEntry{localCtEntry}
			Eventually(sink).Should(Receive(Equal(su1)))
			Eventually(sink).Should(Receive(Equal(su2)))
		})
	})
	Describe("Test local destination with DNAT", func() {
		It("should receive a single stat update with correct tuple extracted", func() {
			t := NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort)
			su := NewStatUpdate(*t,
				inCtEntryWithDNAT.OriginalCounters.Packets, inCtEntryWithDNAT.OriginalCounters.Bytes,
				inCtEntryWithDNAT.ReplyCounters.Packets, inCtEntryWithDNAT.ReplyCounters.Bytes,
				AbsoluteCounter, DirUnknown, EmptyRuleTracePoint)
			dataFeeder <- []nfnetlink.CtEntry{inCtEntry}
			Eventually(sink).Should(Receive(Equal(su)))
		})
	})
	Describe("Test local source to local destination with DNAT", func() {
		It("should receive two stat updates - one for each endpoint - with correct tuple extracted", func() {
			t1 := NewTuple(localIp1, localIp2, proto_tcp, srcPort, dstPort)
			su1 := NewStatUpdate(*t1,
				localCtEntryWithDNAT.OriginalCounters.Packets,
				localCtEntryWithDNAT.OriginalCounters.Bytes,
				localCtEntryWithDNAT.ReplyCounters.Packets,
				localCtEntryWithDNAT.ReplyCounters.Bytes,
				AbsoluteCounter, DirUnknown, EmptyRuleTracePoint)
			t2 := NewTuple(localIp2, localIp1, proto_tcp, dstPort, srcPort)
			su2 := NewStatUpdate(*t2,
				localCtEntryWithDNAT.ReplyCounters.Packets,
				localCtEntryWithDNAT.ReplyCounters.Bytes,
				localCtEntryWithDNAT.OriginalCounters.Packets,
				localCtEntryWithDNAT.OriginalCounters.Bytes,
				AbsoluteCounter, DirUnknown, EmptyRuleTracePoint)
			dataFeeder <- []nfnetlink.CtEntry{localCtEntryWithDNAT}
			Eventually(sink).Should(Receive(Equal(su1)))
			Eventually(sink).Should(Receive(Equal(su2)))
		})
	})

})

// NFLOG datasource test parameters

var (
	defTierAllow = "A/0/policy1/default"
	defTierDeny  = "D/0/policy2/default"
	tier1Allow   = "A/0/policy3/tier1"
	tier1Deny    = "D/0/polic4/tier1"
)

var defTierAllowTp = RuleTracePoint{
	TierID:   "default",
	PolicyID: "policy1",
	Rule:     "0",
	Action:   AllowAction,
	Index:    0,
	EpKey:    localWlEPKey1,
	Ctr:      *NewCounter(1, 100),
}
var defTierDenyTp = RuleTracePoint{
	TierID:   "default",
	PolicyID: "policy2",
	Rule:     "0",
	Action:   DenyAction,
	Index:    0,
	EpKey:    localWlEPKey2,
	Ctr:      *NewCounter(1, 100),
}
var tier1AllowTp = RuleTracePoint{
	TierID:   "tier1",
	PolicyID: "policy3",
	Rule:     "0",
	Action:   AllowAction,
	Index:    1,
}
var tier1DenyTp = RuleTracePoint{
	TierID:   "tier1",
	PolicyID: "policy4",
	Rule:     "0",
	Action:   DenyAction,
	Index:    1,
}

var inPkt = nfnetlink.NflogPacket{
	Prefix: defTierAllow,
	Tuple: nfnetlink.NflogPacketTuple{
		Src:   remoteIp1,
		Dst:   localIp1,
		Proto: proto_tcp,
		L4Src: nfnetlink.NflogL4Info{Port: srcPort},
		L4Dst: nfnetlink.NflogL4Info{Port: dstPort},
	},
	Bytes: 100,
}

var localPkt = nfnetlink.NflogPacket{
	Prefix: defTierDeny,
	Tuple: nfnetlink.NflogPacketTuple{
		Src:   localIp1,
		Dst:   localIp2,
		Proto: proto_tcp,
		L4Src: nfnetlink.NflogL4Info{Port: srcPort},
		L4Dst: nfnetlink.NflogL4Info{Port: dstPort},
	},
	Bytes: 100,
}

var _ = Describe("NFLOG Datasource", func() {
	Describe("NFLOG Incoming Packets", func() {
		// Inject info nflogChan
		// expect a single packet in sink
		var nflogSource *NflogDataSource
		var sink chan *StatUpdate
		var dataFeeder chan nfnetlink.NflogPacket
		dir := DirIn
		BeforeEach(func() {
			epMap := map[[16]byte]*model.WorkloadEndpointKey{
				localIp1: localWlEPKey1,
				localIp2: localWlEPKey2,
			}
			lm := newMockLookupManager(epMap)
			sink = make(chan *StatUpdate)
			done := make(chan struct{})
			dataFeeder = make(chan nfnetlink.NflogPacket)
			gn := 1200
			nflogSource = newNflogDataSource(lm, sink, gn, dir, 65535, dataFeeder, done)
			nflogSource.Start()
		})
		Describe("Test local destination", func() {
			It("should receive a single stat update with allow rule tracepoint", func() {
				t := NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort)
				su := NewStatUpdate(*t, 0, 0, 0, 0,
					DeltaCounter, DirIn, defTierAllowTp)
				dataFeeder <- inPkt
				Eventually(sink).Should(Receive(Equal(su)))
			})
		})
		Describe("Test local to local", func() {
			It("should receive a single stat update with deny rule tracepoint", func() {
				t := NewTuple(localIp1, localIp2, proto_tcp, srcPort, dstPort)
				su := NewStatUpdate(*t, 0, 0, 0, 0,
					DeltaCounter, DirIn, defTierDenyTp)
				dataFeeder <- localPkt
				Eventually(sink).Should(Receive(Equal(su)))
			})
		})
	})
})

type mockLookupManager struct {
	epMap map[[16]byte]*model.WorkloadEndpointKey
}

func newMockLookupManager(em map[[16]byte]*model.WorkloadEndpointKey) *mockLookupManager {
	return &mockLookupManager{
		epMap: em,
	}
}

func (lm *mockLookupManager) GetEndpointKey(addr [16]byte) (interface{}, error) {
	data, _ := lm.epMap[addr]
	if data != nil {
		return data, nil
	} else {
		return nil, lookup.UnknownEndpointError
	}
}

func (lm *mockLookupManager) GetPolicyIndex(epKey interface{}, policyKey *model.PolicyKey) int {
	return 0
}


func BenchmarkNflogPktToStat(b *testing.B) {
       var nflogSource *NflogDataSource
       var sink chan *StatUpdate
       var dataFeeder chan nfnetlink.NflogPacket
       dir := DirIn
       epMap := map[[16]byte]*model.WorkloadEndpointKey{
               localIp1: localWlEPKey1,
               localIp2: localWlEPKey2,
       }
       lm := newMockLookupManager(epMap)
       sink = make(chan *StatUpdate)
       done := make(chan struct{})
       dataFeeder = make(chan nfnetlink.NflogPacket)
       gn := 1200
       nflogSource = newNflogDataSource(lm, sink, gn, dir, 65535, dataFeeder, done)
       b.ResetTimer()
       b.ReportAllocs()
       for n := 0; n < b.N; n++ {
               nflogSource.convertNflogPktToStat(inPkt)
       }
}
