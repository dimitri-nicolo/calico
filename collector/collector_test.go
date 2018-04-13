// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package collector

import (
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/felix/lookup"
	"github.com/projectcalico/felix/rules"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/tigera/nfnetlink"
	"github.com/tigera/nfnetlink/nfnl"
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

// Nflog prefix test parameters
var (
	defTierAllowIngressNFLOGPrefix   = [64]byte{'A', 'P', 'I', '0', '|', 'd', 'e', 'f', 'a', 'u', 'l', 't', '.', 'p', 'o', 'l', 'i', 'c', 'y', '1'}
	defTierAllowEgressNFLOGPrefix    = [64]byte{'A', 'P', 'E', '0', '|', 'd', 'e', 'f', 'a', 'u', 'l', 't', '.', 'p', 'o', 'l', 'i', 'c', 'y', '1'}
	defTierDenyIngressNFLOGPrefix    = [64]byte{'D', 'P', 'I', '0', '|', 'd', 'e', 'f', 'a', 'u', 'l', 't', '.', 'p', 'o', 'l', 'i', 'c', 'y', '2'}
	defTierPolicy1AllowIngressRuleID = &lookup.RuleID{
		Tier:      "default",
		Name:      "policy1",
		Namespace: "",
		Index:     0,
		IndexStr:  "0",
		Action:    rules.RuleActionAllow,
		Direction: rules.RuleDirIngress,
	}
	defTierPolicy1AllowEgressRuleID = &lookup.RuleID{
		Tier:      "default",
		Name:      "policy1",
		Namespace: "",
		Index:     0,
		IndexStr:  "0",
		Action:    rules.RuleActionAllow,
		Direction: rules.RuleDirEgress,
	}
	defTierPolicy2DenyIngressRuleID = &lookup.RuleID{
		Tier:      "default",
		Name:      "policy2",
		Namespace: "",
		Index:     0,
		IndexStr:  "0",
		Action:    rules.RuleActionDeny,
		Direction: rules.RuleDirIngress,
	}
)

var (
	wl1Ep1 = "WEP(orchestrator/localworkloadid1/localepid1)"
	wl2Ep2 = "WEP(orchestrator/localworkloadid2/localepid2)"
)

var ingressPktAllow = &nfnetlink.NflogPacketAggregate{
	Prefixes: []nfnetlink.NflogPrefix{
		{
			Prefix:  defTierAllowIngressNFLOGPrefix,
			Len:     20,
			Bytes:   100,
			Packets: 1,
		},
	},
	Tuple: nfnetlink.NflogPacketTuple{
		Src:   remoteIp1,
		Dst:   localIp1,
		Proto: proto_tcp,
		L4Src: nfnetlink.NflogL4Info{Port: srcPort},
		L4Dst: nfnetlink.NflogL4Info{Port: dstPort},
	},
}

var ingressPktAllowTuple = NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort)

var egressPktAllow = &nfnetlink.NflogPacketAggregate{
	Prefixes: []nfnetlink.NflogPrefix{
		{
			Prefix:  defTierAllowEgressNFLOGPrefix,
			Len:     20,
			Bytes:   100,
			Packets: 1,
		},
	},
	Tuple: nfnetlink.NflogPacketTuple{
		Src:   localIp1,
		Dst:   remoteIp1,
		Proto: proto_udp,
		L4Src: nfnetlink.NflogL4Info{Port: srcPort},
		L4Dst: nfnetlink.NflogL4Info{Port: dstPort},
	},
}
var egressPktAllowTuple = NewTuple(localIp1, remoteIp1, proto_udp, srcPort, dstPort)

var ingressPktDeny = &nfnetlink.NflogPacketAggregate{
	Prefixes: []nfnetlink.NflogPrefix{
		{
			Prefix:  defTierDenyIngressNFLOGPrefix,
			Len:     20,
			Bytes:   100,
			Packets: 1,
		},
	},
	Tuple: nfnetlink.NflogPacketTuple{
		Src:   remoteIp1,
		Dst:   localIp1,
		Proto: proto_tcp,
		L4Src: nfnetlink.NflogL4Info{Port: srcPort},
		L4Dst: nfnetlink.NflogL4Info{Port: dstPort},
	},
}
var ingressPktDenyTuple = NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort)

var localPkt = &nfnetlink.NflogPacketAggregate{
	Prefixes: []nfnetlink.NflogPrefix{
		{
			Prefix:  defTierDenyIngressNFLOGPrefix,
			Len:     22,
			Bytes:   100,
			Packets: 1,
		},
	},
	Tuple: nfnetlink.NflogPacketTuple{
		Src:   localIp1,
		Dst:   localIp2,
		Proto: proto_tcp,
		L4Src: nfnetlink.NflogL4Info{Port: srcPort},
		L4Dst: nfnetlink.NflogL4Info{Port: dstPort},
	},
}

var _ = Describe("NFLOG Datasource", func() {
	Describe("NFLOG Incoming Packets", func() {
		// Inject info nflogChan
		var c *Collector
		conf := &Config{
			StatsDumpFilePath:        "/tmp/qwerty",
			NfNetlinkBufSize:         65535,
			IngressGroup:             1200,
			EgressGroup:              2200,
			AgeTimeout:               time.Duration(10) * time.Second,
			ConntrackPollingInterval: time.Duration(1) * time.Second,
			InitialReportingDelay:    time.Duration(5) * time.Second,
			ExportingInterval:        time.Duration(1) * time.Second,
		}
		rm := NewReporterManager()
		BeforeEach(func() {
			epMap := map[[16]byte]*model.WorkloadEndpointKey{
				localIp1: localWlEPKey1,
				localIp2: localWlEPKey2,
			}
			nflogMap := map[[64]byte]*lookup.RuleID{}

			for _, rid := range []*lookup.RuleID{defTierPolicy1AllowEgressRuleID, defTierPolicy1AllowIngressRuleID, defTierPolicy2DenyIngressRuleID} {
				nflogMap[policyIDStrToRuleIDParts(rid)] = rid
			}

			lm := newMockLookupManager(epMap, nflogMap)
			c = NewCollector(lm, rm, conf)
			go c.startStatsCollectionAndReporting()
		})
		Describe("Test local destination", func() {
			It("should receive a single stat update with allow ruleid trace", func() {
				t := NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort)
				c.nfIngressC <- ingressPktAllow
				Eventually(c.epStats).Should(HaveKey(*t))
			})
		})
		Describe("Test local to local", func() {
			It("should receive a single stat update with deny ruleid trace", func() {
				t := NewTuple(localIp1, localIp2, proto_tcp, srcPort, dstPort)
				c.nfIngressC <- localPkt
				Eventually(c.epStats).Should(HaveKey(*t))
			})
		})
	})
})

// Entry remoteIp1:srcPort -> localIp1:dstPort
var inCtEntry = nfnetlink.CtEntry{
	OriginalTuple: nfnetlink.CtTuple{
		Src:        remoteIp1,
		Dst:        localIp1,
		L3ProtoNum: ipv4,
		ProtoNum:   proto_tcp,
		L4Src:      nfnetlink.CtL4Src{Port: srcPort},
		L4Dst:      nfnetlink.CtL4Dst{Port: dstPort},
	},
	ReplyTuple: nfnetlink.CtTuple{
		Src:        localIp1,
		Dst:        remoteIp1,
		L3ProtoNum: ipv4,
		ProtoNum:   proto_tcp,
		L4Src:      nfnetlink.CtL4Src{Port: dstPort},
		L4Dst:      nfnetlink.CtL4Dst{Port: srcPort},
	},
	OriginalCounters: nfnetlink.CtCounters{Packets: 1, Bytes: 100},
	ReplyCounters:    nfnetlink.CtCounters{Packets: 2, Bytes: 250},
}

var outCtEntry = nfnetlink.CtEntry{
	OriginalTuple: nfnetlink.CtTuple{
		Src:        localIp1,
		Dst:        remoteIp1,
		L3ProtoNum: ipv4,
		ProtoNum:   proto_tcp,
		L4Src:      nfnetlink.CtL4Src{Port: srcPort},
		L4Dst:      nfnetlink.CtL4Dst{Port: dstPort},
	},
	ReplyTuple: nfnetlink.CtTuple{
		Src:        remoteIp1,
		Dst:        localIp1,
		L3ProtoNum: ipv4,
		ProtoNum:   proto_tcp,
		L4Src:      nfnetlink.CtL4Src{Port: dstPort},
		L4Dst:      nfnetlink.CtL4Dst{Port: srcPort},
	},
	OriginalCounters: nfnetlink.CtCounters{Packets: 1, Bytes: 100},
	ReplyCounters:    nfnetlink.CtCounters{Packets: 2, Bytes: 250},
}

var localCtEntry = nfnetlink.CtEntry{
	OriginalTuple: nfnetlink.CtTuple{
		Src:        localIp1,
		Dst:        localIp2,
		L3ProtoNum: ipv4,
		ProtoNum:   proto_tcp,
		L4Src:      nfnetlink.CtL4Src{Port: srcPort},
		L4Dst:      nfnetlink.CtL4Dst{Port: dstPort},
	},
	ReplyTuple: nfnetlink.CtTuple{
		Src:        localIp2,
		Dst:        localIp1,
		L3ProtoNum: ipv4,
		ProtoNum:   proto_tcp,
		L4Src:      nfnetlink.CtL4Src{Port: dstPort},
		L4Dst:      nfnetlink.CtL4Dst{Port: srcPort},
	},
	OriginalCounters: nfnetlink.CtCounters{Packets: 1, Bytes: 100},
	ReplyCounters:    nfnetlink.CtCounters{Packets: 2, Bytes: 250},
}

// DNAT Conntrack Entries
// DNAT from localIp1DNAT:dstPortDNAT --> localIp1:dstPort
var inCtEntryWithDNAT = nfnetlink.CtEntry{
	OriginalTuple: nfnetlink.CtTuple{
		Src:        remoteIp1,
		Dst:        localIp1DNAT,
		L3ProtoNum: ipv4,
		ProtoNum:   proto_tcp,
		L4Src:      nfnetlink.CtL4Src{Port: srcPort},
		L4Dst:      nfnetlink.CtL4Dst{Port: dstPortDNAT},
	},
	ReplyTuple: nfnetlink.CtTuple{
		Src:        localIp1,
		Dst:        remoteIp1,
		L3ProtoNum: ipv4,
		ProtoNum:   proto_tcp,
		L4Src:      nfnetlink.CtL4Src{Port: dstPort},
		L4Dst:      nfnetlink.CtL4Dst{Port: srcPort},
	},
	Status:           nfnl.IPS_DST_NAT,
	OriginalCounters: nfnetlink.CtCounters{Packets: 1, Bytes: 100},
	ReplyCounters:    nfnetlink.CtCounters{Packets: 2, Bytes: 250},
}

// DNAT from localIp2DNAT:dstPortDNAT --> localIp2:dstPort
var localCtEntryWithDNAT = nfnetlink.CtEntry{
	OriginalTuple: nfnetlink.CtTuple{
		Src:        localIp1,
		Dst:        localIp2DNAT,
		L3ProtoNum: ipv4,
		ProtoNum:   proto_tcp,
		L4Src:      nfnetlink.CtL4Src{Port: srcPort},
		L4Dst:      nfnetlink.CtL4Dst{Port: dstPortDNAT},
	},
	ReplyTuple: nfnetlink.CtTuple{
		Src:        localIp2,
		Dst:        localIp1,
		L3ProtoNum: ipv4,
		ProtoNum:   proto_tcp,
		L4Src:      nfnetlink.CtL4Src{Port: dstPort},
		L4Dst:      nfnetlink.CtL4Dst{Port: srcPort},
	},
	Status:           nfnl.IPS_DST_NAT,
	OriginalCounters: nfnetlink.CtCounters{Packets: 1, Bytes: 100},
	ReplyCounters:    nfnetlink.CtCounters{Packets: 2, Bytes: 250},
}

var _ = Describe("Conntrack Datasource", func() {
	var c *Collector
	conf := &Config{
		StatsDumpFilePath:        "/tmp/qwerty",
		NfNetlinkBufSize:         65535,
		IngressGroup:             1200,
		EgressGroup:              2200,
		AgeTimeout:               time.Duration(10) * time.Second,
		ConntrackPollingInterval: time.Duration(1) * time.Second,
		InitialReportingDelay:    time.Duration(5) * time.Second,
		ExportingInterval:        time.Duration(1) * time.Second,
	}
	rm := NewReporterManager()
	BeforeEach(func() {
		epMap := map[[16]byte]*model.WorkloadEndpointKey{
			localIp1: localWlEPKey1,
			localIp2: localWlEPKey2,
		}

		nflogMap := map[[64]byte]*lookup.RuleID{}

		for _, rid := range []*lookup.RuleID{defTierPolicy1AllowEgressRuleID, defTierPolicy1AllowIngressRuleID, defTierPolicy2DenyIngressRuleID} {
			nflogMap[policyIDStrToRuleIDParts(rid)] = rid
		}

		lm := newMockLookupManager(epMap, nflogMap)
		c = NewCollector(lm, rm, conf)
		go c.startStatsCollectionAndReporting()
	})
	Describe("Test local destination", func() {
		It("should create a single entry in inbound direction", func() {
			t := NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort)
			c.handleCtEntry(inCtEntry)
			Eventually(c.epStats).Should(HaveKey(*t))
			data := c.epStats[*t]
			Expect(data.Counters()).Should(Equal(*NewCounter(inCtEntry.OriginalCounters.Packets, inCtEntry.OriginalCounters.Bytes)))
			Expect(data.CountersReverse()).Should(Equal(*NewCounter(inCtEntry.ReplyCounters.Packets, inCtEntry.ReplyCounters.Bytes)))
		})
	})
	Describe("Test local source", func() {
		It("should create a single entry with outbound direction", func() {
			t := NewTuple(localIp1, remoteIp1, proto_tcp, srcPort, dstPort)
			c.handleCtEntry(outCtEntry)
			Eventually(c.epStats).Should(HaveKey(*t))
			data := c.epStats[*t]
			Expect(data.Counters()).Should(Equal(*NewCounter(outCtEntry.OriginalCounters.Packets, outCtEntry.OriginalCounters.Bytes)))
			Expect(data.CountersReverse()).Should(Equal(*NewCounter(outCtEntry.ReplyCounters.Packets, outCtEntry.ReplyCounters.Bytes)))
		})
	})
	Describe("Test local source to local destination", func() {
		It("should create a single entry with 'local' direction", func() {
			t1 := NewTuple(localIp1, localIp2, proto_tcp, srcPort, dstPort)
			c.handleCtEntry(localCtEntry)
			Eventually(c.epStats).Should(HaveKey(Equal(*t1)))
			data := c.epStats[*t1]
			Expect(data.Counters()).Should(Equal(*NewCounter(localCtEntry.OriginalCounters.Packets, localCtEntry.OriginalCounters.Bytes)))
			Expect(data.CountersReverse()).Should(Equal(*NewCounter(localCtEntry.ReplyCounters.Packets, localCtEntry.ReplyCounters.Bytes)))
		})
	})
	Describe("Test local destination with DNAT", func() {
		It("should create a single entry with inbound connection direction and with correct tuple extracted", func() {
			t := NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort)
			c.handleCtEntry(inCtEntryWithDNAT)
			Eventually(c.epStats).Should(HaveKey(Equal(*t)))
			data := c.epStats[*t]
			Expect(data.Counters()).Should(Equal(*NewCounter(inCtEntryWithDNAT.OriginalCounters.Packets, inCtEntryWithDNAT.OriginalCounters.Bytes)))
			Expect(data.CountersReverse()).Should(Equal(*NewCounter(inCtEntryWithDNAT.ReplyCounters.Packets, inCtEntryWithDNAT.ReplyCounters.Bytes)))
		})
	})
	Describe("Test local source to local destination with DNAT", func() {
		It("should create a single entry with 'local' connection direction and with correct tuple extracted", func() {
			t1 := NewTuple(localIp1, localIp2, proto_tcp, srcPort, dstPort)
			c.handleCtEntry(localCtEntryWithDNAT)
			Eventually(c.epStats).Should(HaveKey(Equal(*t1)))
			data := c.epStats[*t1]
			Expect(data.Counters()).Should(Equal(*NewCounter(localCtEntryWithDNAT.OriginalCounters.Packets, localCtEntryWithDNAT.OriginalCounters.Bytes)))
			Expect(data.CountersReverse()).Should(Equal(*NewCounter(localCtEntryWithDNAT.ReplyCounters.Packets, localCtEntryWithDNAT.ReplyCounters.Bytes)))
		})
	})

})

func policyIDStrToRuleIDParts(r *lookup.RuleID) [64]byte {
	var (
		name  string
		byt64 [64]byte
	)

	if r.Namespace != "" {
		if strings.HasPrefix(r.Name, "knp.default.") {
			name = fmt.Sprintf("%s/%s", r.Namespace, r.Name)
		} else {
			name = fmt.Sprintf("%s/%s.%s", r.Namespace, r.Tier, r.Name)
		}
	} else {
		name = fmt.Sprintf("%s.%s", r.Tier, r.Name)
	}

	prefix := rules.CalculateNFLOGPrefixStr(r.Action, rules.RuleOwnerTypePolicy, r.Direction, r.Index, name)
	copy(byt64[:], []byte(prefix))
	return byt64
}

var _ = Describe("Reporting Metrics", func() {
	var c *Collector
	const (
		ageTimeout        = time.Duration(3) * time.Second
		reportingDelay    = time.Duration(2) * time.Second
		exportingInterval = time.Duration(1) * time.Second
		pollingInterval   = time.Duration(1) * time.Second
	)
	conf := &Config{
		StatsDumpFilePath:        "/tmp/qwerty",
		NfNetlinkBufSize:         65535,
		IngressGroup:             1200,
		EgressGroup:              2200,
		AgeTimeout:               ageTimeout,
		ConntrackPollingInterval: pollingInterval,
		InitialReportingDelay:    reportingDelay,
		ExportingInterval:        exportingInterval,
	}
	rm := NewReporterManager()
	mockReporter := newMockReporter()
	rm.RegisterMetricsReporter(mockReporter)
	BeforeEach(func() {
		epMap := map[[16]byte]*model.WorkloadEndpointKey{
			localIp1: localWlEPKey1,
			localIp2: localWlEPKey2,
		}

		nflogMap := map[[64]byte]*lookup.RuleID{}

		for _, rid := range []*lookup.RuleID{defTierPolicy1AllowEgressRuleID, defTierPolicy1AllowIngressRuleID, defTierPolicy2DenyIngressRuleID} {
			nflogMap[policyIDStrToRuleIDParts(rid)] = rid
		}

		lm := newMockLookupManager(epMap, nflogMap)
		rm.Start()
		c = NewCollector(lm, rm, conf)
		go c.startStatsCollectionAndReporting()
	})
	Describe("Report Denied Packets", func() {
		BeforeEach(func() {
			c.nfIngressC <- ingressPktDeny
		})
		Context("reporting tick", func() {
			It("should receive metric", func() {
				tmu := testMetricUpdate{
					updateType:   UpdateTypeReport,
					tuple:        *ingressPktDenyTuple,
					ruleID:       defTierPolicy2DenyIngressRuleID,
					isConnection: false,
				}
				Eventually(mockReporter.reportChan, reportingDelay*2).Should(Receive(Equal(tmu)))
			})
		})
	})
	Describe("Report Allowed Packets (ingress)", func() {
		BeforeEach(func() {
			c.nfIngressC <- ingressPktAllow
		})
		Context("reporting tick", func() {
			It("should receive metric", func() {
				tmu := testMetricUpdate{
					updateType:   UpdateTypeReport,
					tuple:        *ingressPktAllowTuple,
					ruleID:       defTierPolicy1AllowIngressRuleID,
					isConnection: false,
				}
				Eventually(mockReporter.reportChan, reportingDelay*2).Should(Receive(Equal(tmu)))
			})
		})
	})
	Describe("Report Packets that switch from deny to allow", func() {
		BeforeEach(func() {
			c.nfIngressC <- ingressPktDeny
			time.Sleep(time.Duration(500) * time.Millisecond)
			c.nfIngressC <- ingressPktAllow
		})
		Context("reporting tick", func() {
			It("should receive metric", func() {
				tmu := testMetricUpdate{
					updateType:   UpdateTypeReport,
					tuple:        *ingressPktAllowTuple,
					ruleID:       defTierPolicy1AllowIngressRuleID,
					isConnection: false,
				}
				Eventually(mockReporter.reportChan, reportingDelay*2).Should(Receive(Equal(tmu)))
			})
		})
	})
	Describe("Report Allowed Packets (egress)", func() {
		BeforeEach(func() {
			c.nfEgressC <- egressPktAllow
		})
		Context("reporting tick", func() {
			It("should receive metric", func() {
				tmu := testMetricUpdate{
					updateType:   UpdateTypeReport,
					tuple:        *egressPktAllowTuple,
					ruleID:       defTierPolicy1AllowEgressRuleID,
					isConnection: false,
				}
				Eventually(mockReporter.reportChan, reportingDelay*2).Should(Receive(Equal(tmu)))
			})
		})
	})
})

func newMockLookupManager(
	em map[[16]byte]*model.WorkloadEndpointKey, nm map[[64]byte]*lookup.RuleID,
) *lookup.LookupManager {
	l := lookup.NewLookupManager()
	l.SetMockData(em, nm)
	return l
}

// Define a separate metric type that doesn't include the actual stats.  We use this
// for simpler comparisons.
type testMetricUpdate struct {
	updateType UpdateType

	// Tuple key
	tuple Tuple

	// Rule identification
	ruleID *lookup.RuleID

	// isConnection is true if this update is from an active connection (i.e. a conntrack
	// update compared to an NFLOG update).
	isConnection bool
}

// Create a mockReporter that acts as a pass-thru of the updates.
type mockReporter struct {
	reportChan chan testMetricUpdate
}

func newMockReporter() *mockReporter {
	return &mockReporter{
		reportChan: make(chan testMetricUpdate),
	}
}

func (mr *mockReporter) Start() {
	// Do nothing. We are a mock anyway.
}

func (mr *mockReporter) Report(mu MetricUpdate) error {
	mr.reportChan <- testMetricUpdate{
		updateType:   UpdateTypeReport,
		tuple:        mu.tuple,
		ruleID:       mu.ruleID,
		isConnection: mu.isConnection,
	}
	return nil
}

func BenchmarkNflogPktToStat(b *testing.B) {
	epMap := map[[16]byte]*model.WorkloadEndpointKey{
		localIp1: localWlEPKey1,
		localIp2: localWlEPKey2,
	}

	nflogMap := map[[64]byte]*lookup.RuleID{}

	for _, rid := range []*lookup.RuleID{defTierPolicy1AllowEgressRuleID, defTierPolicy1AllowIngressRuleID, defTierPolicy2DenyIngressRuleID} {
		nflogMap[policyIDStrToRuleIDParts(rid)] = rid
	}

	conf := &Config{
		StatsDumpFilePath:        "/tmp/qwerty",
		NfNetlinkBufSize:         65535,
		IngressGroup:             1200,
		EgressGroup:              2200,
		AgeTimeout:               time.Duration(10) * time.Second,
		ConntrackPollingInterval: time.Duration(1) * time.Second,
		InitialReportingDelay:    time.Duration(5) * time.Second,
		ExportingInterval:        time.Duration(1) * time.Second,
	}
	rm := NewReporterManager()
	lm := newMockLookupManager(epMap, nflogMap)
	c := NewCollector(lm, rm, conf)
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		c.convertNflogPktAndApplyUpdate(rules.RuleDirIngress, ingressPktAllow)
	}
}

func BenchmarkApplyStatUpdate(b *testing.B) {
	epMap := map[[16]byte]*model.WorkloadEndpointKey{
		localIp1: localWlEPKey1,
		localIp2: localWlEPKey2,
	}

	nflogMap := map[[64]byte]*lookup.RuleID{}
	for _, rid := range []*lookup.RuleID{defTierPolicy1AllowEgressRuleID, defTierPolicy1AllowIngressRuleID, defTierPolicy2DenyIngressRuleID} {
		nflogMap[policyIDStrToRuleIDParts(rid)] = rid
	}

	conf := &Config{
		StatsDumpFilePath:        "/tmp/qwerty",
		NfNetlinkBufSize:         65535,
		IngressGroup:             1200,
		EgressGroup:              2200,
		AgeTimeout:               time.Duration(10) * time.Second,
		ConntrackPollingInterval: time.Duration(1) * time.Second,
		InitialReportingDelay:    time.Duration(5) * time.Second,
		ExportingInterval:        time.Duration(1) * time.Second,
	}
	rm := NewReporterManager()
	lm := newMockLookupManager(epMap, nflogMap)
	c := NewCollector(lm, rm, conf)
	var tuples []Tuple
	MaxSrcPort := 1000
	MaxDstPort := 1000
	for sp := 1; sp < MaxSrcPort; sp++ {
		for dp := 1; dp < MaxDstPort; dp++ {
			t := NewTuple(localIp1, localIp2, proto_tcp, sp, dp)
			tuples = append(tuples, *t)
		}
	}
	var rids []*lookup.RuleID
	MaxEntries := 10000
	for i := 0; i < MaxEntries; i++ {
		rid := defTierPolicy1AllowIngressRuleID
		rids = append(rids, rid)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		for i := 0; i < MaxEntries; i++ {
			c.applyNflogStatUpdate(tuples[i], rids[i], wl1Ep1, 0, 1, 2)
		}
	}
}
