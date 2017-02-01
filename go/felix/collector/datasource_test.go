// Copyright (c) 2017 Tigera, Inc. All rights reserved.

package collector

import (
	"net"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/projectcalico/felix/go/felix/collector/stats"
	"github.com/projectcalico/felix/go/felix/jitter"
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
	localIp1     = net.ParseIP("10.0.0.1")
	localIp2     = net.ParseIP("10.0.0.2")
	remoteIp1    = net.ParseIP("20.0.0.1")
	localIp1DNAT = net.ParseIP("192.168.0.1")
	localIp2DNAT = net.ParseIP("192.168.0.2")
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
	OrigTuples: []nfnetlink.CtTuple{
		nfnetlink.CtTuple{
			Src:        remoteIp1,
			Dst:        localIp1,
			L3ProtoNum: ipv4,
			ProtoNum:   proto_tcp,
			L4Src:      nfnetlink.CtL4Src{Port: srcPort},
			L4Dst:      nfnetlink.CtL4Dst{Port: dstPort},
		},
	},
	ReplTuples: []nfnetlink.CtTuple{
		nfnetlink.CtTuple{
			Src:        localIp1,
			Dst:        remoteIp1,
			L3ProtoNum: ipv4,
			ProtoNum:   proto_tcp,
			L4Src:      nfnetlink.CtL4Src{Port: dstPort},
			L4Dst:      nfnetlink.CtL4Dst{Port: srcPort},
		},
	},
	OrigCounters: nfnetlink.CtCounters{Packets: 1, Bytes: 100},
	ReplCounters: nfnetlink.CtCounters{Packets: 2, Bytes: 250},
}

var outCtEntry = nfnetlink.CtEntry{
	OrigTuples: []nfnetlink.CtTuple{
		nfnetlink.CtTuple{
			Src:        localIp1,
			Dst:        remoteIp1,
			L3ProtoNum: ipv4,
			ProtoNum:   proto_tcp,
			L4Src:      nfnetlink.CtL4Src{Port: srcPort},
			L4Dst:      nfnetlink.CtL4Dst{Port: dstPort},
		},
	},
	ReplTuples: []nfnetlink.CtTuple{
		nfnetlink.CtTuple{
			Src:        remoteIp1,
			Dst:        localIp1,
			L3ProtoNum: ipv4,
			ProtoNum:   proto_tcp,
			L4Src:      nfnetlink.CtL4Src{Port: dstPort},
			L4Dst:      nfnetlink.CtL4Dst{Port: srcPort},
		},
	},
	OrigCounters: nfnetlink.CtCounters{Packets: 1, Bytes: 100},
	ReplCounters: nfnetlink.CtCounters{Packets: 2, Bytes: 250},
}

var localCtEntry = nfnetlink.CtEntry{
	OrigTuples: []nfnetlink.CtTuple{
		nfnetlink.CtTuple{
			Src:        localIp1,
			Dst:        localIp2,
			L3ProtoNum: ipv4,
			ProtoNum:   proto_tcp,
			L4Src:      nfnetlink.CtL4Src{Port: srcPort},
			L4Dst:      nfnetlink.CtL4Dst{Port: dstPort},
		},
	},
	ReplTuples: []nfnetlink.CtTuple{
		nfnetlink.CtTuple{
			Src:        localIp2,
			Dst:        localIp1,
			L3ProtoNum: ipv4,
			ProtoNum:   proto_tcp,
			L4Src:      nfnetlink.CtL4Src{Port: dstPort},
			L4Dst:      nfnetlink.CtL4Dst{Port: srcPort},
		},
	},
	OrigCounters: nfnetlink.CtCounters{Packets: 1, Bytes: 100},
	ReplCounters: nfnetlink.CtCounters{Packets: 2, Bytes: 250},
}

// DNAT Conntrack Entries
// DNAT from localIp1DNAT:dstPortDNAT --> localIp1:dstPort
var inCtEntryWithDNAT = nfnetlink.CtEntry{
	OrigTuples: []nfnetlink.CtTuple{
		nfnetlink.CtTuple{
			Src:        remoteIp1,
			Dst:        localIp1DNAT,
			L3ProtoNum: ipv4,
			ProtoNum:   proto_tcp,
			L4Src:      nfnetlink.CtL4Src{Port: srcPort},
			L4Dst:      nfnetlink.CtL4Dst{Port: dstPortDNAT},
		},
	},
	ReplTuples: []nfnetlink.CtTuple{
		nfnetlink.CtTuple{
			Src:        localIp1,
			Dst:        remoteIp1,
			L3ProtoNum: ipv4,
			ProtoNum:   proto_tcp,
			L4Src:      nfnetlink.CtL4Src{Port: dstPort},
			L4Dst:      nfnetlink.CtL4Dst{Port: srcPort},
		},
	},
	Status:       nfnl.IPS_DST_NAT,
	OrigCounters: nfnetlink.CtCounters{Packets: 1, Bytes: 100},
	ReplCounters: nfnetlink.CtCounters{Packets: 2, Bytes: 250},
}

// DNAT from localIp2DNAT:dstPortDNAT --> localIp2:dstPort
var localCtEntryWithDNAT = nfnetlink.CtEntry{
	OrigTuples: []nfnetlink.CtTuple{
		nfnetlink.CtTuple{
			Src:        localIp1,
			Dst:        localIp2DNAT,
			L3ProtoNum: ipv4,
			ProtoNum:   proto_tcp,
			L4Src:      nfnetlink.CtL4Src{Port: srcPort},
			L4Dst:      nfnetlink.CtL4Dst{Port: dstPortDNAT},
		},
	},
	ReplTuples: []nfnetlink.CtTuple{
		nfnetlink.CtTuple{
			Src:        localIp2,
			Dst:        localIp1,
			L3ProtoNum: ipv4,
			ProtoNum:   proto_tcp,
			L4Src:      nfnetlink.CtL4Src{Port: dstPort},
			L4Dst:      nfnetlink.CtL4Dst{Port: srcPort},
		},
	},
	Status:       nfnl.IPS_DST_NAT,
	OrigCounters: nfnetlink.CtCounters{Packets: 1, Bytes: 100},
	ReplCounters: nfnetlink.CtCounters{Packets: 2, Bytes: 250},
}

var _ = Describe("Conntrack Datasource", func() {
	var ctSource *ConntrackDataSource
	var sink chan stats.StatUpdate
	var dataFeeder chan []nfnetlink.CtEntry
	BeforeEach(func() {
		epMap := map[string]*model.WorkloadEndpointKey{
			string(localIp1): localWlEPKey1,
			string(localIp2): localWlEPKey2,
		}
		lm := newMockLookupManager(epMap)
		sink = make(chan stats.StatUpdate)
		poller := jitter.NewTicker(time.Second, time.Second/10)
		mockTickerChan := make(chan time.Time)
		dataFeeder = make(chan []nfnetlink.CtEntry)
		ctSource = newConntrackDataSource(lm, sink, poller, mockTickerChan, dataFeeder)
		ctSource.Start()
	})
	Describe("Test local destination", func() {
		It("should receive a single stat update", func() {
			t := stats.NewTuple(localIp1, remoteIp1, proto_tcp, dstPort, srcPort)
			su := stats.NewStatUpdate(*t, *localWlEPKey1,
				inCtEntry.ReplCounters.Packets, inCtEntry.ReplCounters.Bytes,
				inCtEntry.OrigCounters.Packets, inCtEntry.OrigCounters.Bytes,
				stats.AbsoluteCounter, stats.EmptyRuleTracePoint)
			dataFeeder <- []nfnetlink.CtEntry{inCtEntry}
			Eventually(sink).Should(Receive(Equal(*su)))
		})
	})
	Describe("Test local source", func() {
		It("should receive a single stat update", func() {
			t := stats.NewTuple(localIp1, remoteIp1, proto_tcp, srcPort, dstPort)
			su := stats.NewStatUpdate(*t, *localWlEPKey1,
				outCtEntry.OrigCounters.Packets, outCtEntry.OrigCounters.Bytes,
				outCtEntry.ReplCounters.Packets, outCtEntry.ReplCounters.Bytes,
				stats.AbsoluteCounter, stats.EmptyRuleTracePoint)
			dataFeeder <- []nfnetlink.CtEntry{outCtEntry}
			Eventually(sink).Should(Receive(Equal(*su)))
		})
	})
	Describe("Test local source to local destination", func() {
		It("should receive two stat updates - one for each endpoint", func() {
			t1 := stats.NewTuple(localIp1, localIp2, proto_tcp, srcPort, dstPort)
			su1 := stats.NewStatUpdate(*t1, *localWlEPKey1,
				localCtEntry.OrigCounters.Packets, localCtEntry.OrigCounters.Bytes,
				localCtEntry.ReplCounters.Packets, localCtEntry.ReplCounters.Bytes,
				stats.AbsoluteCounter, stats.EmptyRuleTracePoint)
			t2 := stats.NewTuple(localIp2, localIp1, proto_tcp, dstPort, srcPort)
			su2 := stats.NewStatUpdate(*t2, *localWlEPKey2,
				localCtEntry.ReplCounters.Packets, localCtEntry.ReplCounters.Bytes,
				localCtEntry.OrigCounters.Packets, localCtEntry.OrigCounters.Bytes,
				stats.AbsoluteCounter, stats.EmptyRuleTracePoint)
			dataFeeder <- []nfnetlink.CtEntry{localCtEntry}
			Eventually(sink).Should(Receive(Equal(*su1)))
			Eventually(sink).Should(Receive(Equal(*su2)))
		})
	})
	Describe("Test local destination with DNAT", func() {
		It("should receive a single stat update with correct tuple extracted", func() {
			t := stats.NewTuple(localIp1, remoteIp1, proto_tcp, dstPort, srcPort)
			su := stats.NewStatUpdate(*t, *localWlEPKey1,
				inCtEntryWithDNAT.ReplCounters.Packets, inCtEntryWithDNAT.ReplCounters.Bytes,
				inCtEntryWithDNAT.OrigCounters.Packets, inCtEntryWithDNAT.OrigCounters.Bytes,
				stats.AbsoluteCounter, stats.EmptyRuleTracePoint)
			dataFeeder <- []nfnetlink.CtEntry{inCtEntry}
			Eventually(sink).Should(Receive(Equal(*su)))
		})
	})
	Describe("Test local source to local destination with DNAT", func() {
		It("should receive two stat updates - one for each endpoint - with correct tuple extracted", func() {
			t1 := stats.NewTuple(localIp1, localIp2, proto_tcp, srcPort, dstPort)
			su1 := stats.NewStatUpdate(*t1, *localWlEPKey1,
				localCtEntryWithDNAT.OrigCounters.Packets,
				localCtEntryWithDNAT.OrigCounters.Bytes,
				localCtEntryWithDNAT.ReplCounters.Packets,
				localCtEntryWithDNAT.ReplCounters.Bytes,
				stats.AbsoluteCounter, stats.EmptyRuleTracePoint)
			t2 := stats.NewTuple(localIp2, localIp1, proto_tcp, dstPort, srcPort)
			su2 := stats.NewStatUpdate(*t2, *localWlEPKey2,
				localCtEntryWithDNAT.ReplCounters.Packets,
				localCtEntryWithDNAT.ReplCounters.Bytes,
				localCtEntryWithDNAT.OrigCounters.Packets,
				localCtEntryWithDNAT.OrigCounters.Bytes,
				stats.AbsoluteCounter, stats.EmptyRuleTracePoint)
			dataFeeder <- []nfnetlink.CtEntry{localCtEntryWithDNAT}
			Eventually(sink).Should(Receive(Equal(*su1)))
			Eventually(sink).Should(Receive(Equal(*su2)))
		})
	})

})

// NFLOG datasource test parameters

var (
	defTierAllow = "F/A/0/policy1/default"
	defTierDeny  = "F/D/0/policy2/default"
	tier1Allow   = "F/A/0/policy3/tier1"
	tier1Deny    = "F/D/0/polic4/tier1"
)

var defTierAllowTp = stats.RuleTracePoint{
	TierID:   "default",
	PolicyID: "policy1",
	Rule:     "0",
	Action:   stats.AllowAction,
	Export:   false,
	Index:    0,
}
var defTierDenyTp = stats.RuleTracePoint{
	TierID:   "default",
	PolicyID: "policy2",
	Rule:     "0",
	Action:   stats.DenyAction,
	Export:   false,
	Index:    0,
}
var tier1AllowTp = stats.RuleTracePoint{
	TierID:   "tier1",
	PolicyID: "policy3",
	Rule:     "0",
	Action:   stats.AllowAction,
	Export:   false,
	Index:    1,
}
var tier1DenyTp = stats.RuleTracePoint{
	TierID:   "tier1",
	PolicyID: "policy4",
	Rule:     "0",
	Action:   stats.DenyAction,
	Export:   false,
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
		var sink chan stats.StatUpdate
		var dataFeeder chan nfnetlink.NflogPacket
		dir := stats.DirIn
		BeforeEach(func() {
			epMap := map[string]*model.WorkloadEndpointKey{
				string(localIp1): localWlEPKey1,
				string(localIp2): localWlEPKey2,
			}
			lm := newMockLookupManager(epMap)
			sink = make(chan stats.StatUpdate)
			done := make(chan struct{})
			dataFeeder = make(chan nfnetlink.NflogPacket)
			gn := 1200
			nflogSource = newNflogDataSource(lm, sink, gn, dir, 65535, dataFeeder, done)
			nflogSource.Start()
		})
		Describe("Test local destination", func() {
			It("should receive a single stat update with allow rule tracepoint", func() {
				t := stats.NewTuple(localIp1, remoteIp1, proto_tcp, dstPort, srcPort)
				su := stats.NewStatUpdate(*t, *localWlEPKey1, 0, 0, 0, 0,
					stats.DeltaCounter, defTierAllowTp)
				dataFeeder <- inPkt
				Eventually(sink).Should(Receive(Equal(*su)))
			})
		})
		Describe("Test local to local", func() {
			It("should receive a single stat update with deny rule tracepoint", func() {
				t := stats.NewTuple(localIp2, localIp1, proto_tcp, dstPort, srcPort)
				su := stats.NewStatUpdate(*t, *localWlEPKey2, 1, 100, 0, 0,
					stats.DeltaCounter, defTierDenyTp)
				dataFeeder <- localPkt
				Eventually(sink).Should(Receive(Equal(*su)))
			})
		})
	})
})

type mockLookupManager struct {
	epMap map[string]*model.WorkloadEndpointKey
}

func newMockLookupManager(em map[string]*model.WorkloadEndpointKey) *mockLookupManager {
	return &mockLookupManager{
		epMap: em,
	}
}

func (lm *mockLookupManager) GetEndpointKey(addr net.IP) *model.WorkloadEndpointKey {
	data, _ := lm.epMap[string(addr)]
	log.Infof("Looking up %v returned %v", addr, data)
	return data
}

func (lm *mockLookupManager) GetPolicyIndex(epKey *model.WorkloadEndpointKey, policyKey *model.PolicyKey) int {
	return 0
}
