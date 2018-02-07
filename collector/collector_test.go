package collector

import (
	"bytes"
	"testing"
	"time"

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

// NFLOG datasource test parameters

var (
	defTierAllow = [64]byte{'A', '|', '0', '|', 'p', 'o', 'l', 'i', 'c', 'y', '1', '|', 'd', 'e', 'f', 'a', 'u', 'l', 't'}
	defTierDeny  = [64]byte{'D', '|', '0', '|', 'p', 'o', 'l', 'i', 'c', 'y', '2', '|', 'd', 'e', 'f', 'a', 'u', 'l', 't'}
	tier1Allow   = [64]byte{'A', '|', '0', '|', 'p', 'o', 'l', 'i', 'c', 'y', '3', '|', 't', 'i', 'e', 'r', '1'}
	tier1Deny    = [64]byte{'D', '|', '0', '|', 'p', 'o', 'l', 'i', 'c', 'y', '4', '|', 't', 'i', 'e', 'r', '1'}
)

var defTierAllowTp = &RuleTracePoint{
	prefix:    defTierAllow,
	pfxlen:    19,
	tierIdx:   12,
	policyIdx: 4,
	ruleIdx:   2,
	Action:    AllowAction,
	Index:     0,
	EpKey:     localWlEPKey1,
	Ctr:       *NewCounter(1, 100),
}

var defTierDenyTp = &RuleTracePoint{
	prefix:    defTierDeny,
	pfxlen:    19,
	tierIdx:   12,
	policyIdx: 4,
	ruleIdx:   2,
	Action:    DenyAction,
	Index:     0,
	EpKey:     localWlEPKey2,
	Ctr:       *NewCounter(1, 100),
}

var tier1AllowTp = &RuleTracePoint{
	prefix:    tier1Allow,
	pfxlen:    17,
	tierIdx:   12,
	policyIdx: 4,
	ruleIdx:   2,
	Action:    AllowAction,
	Index:     1,
}

var tier1DenyTp = &RuleTracePoint{
	prefix:    tier1Deny,
	pfxlen:    17,
	tierIdx:   12,
	policyIdx: 4,
	ruleIdx:   2,
	Action:    DenyAction,
	Index:     1,
}

var inPkt = &nfnetlink.NflogPacketAggregate{
	Prefixes: []nfnetlink.NflogPrefix{
		{
			Prefix:  defTierAllow,
			Len:     19,
			Bytes:   100,
			Packets: 1,
		},
	},
	Tuple: &nfnetlink.NflogPacketTuple{
		Src:   remoteIp1,
		Dst:   localIp1,
		Proto: proto_tcp,
		L4Src: nfnetlink.NflogL4Info{Port: srcPort},
		L4Dst: nfnetlink.NflogL4Info{Port: dstPort},
	},
}

var inPktDeny = &nfnetlink.NflogPacketAggregate{
	Prefixes: []nfnetlink.NflogPrefix{
		{
			Prefix:  defTierDeny,
			Len:     19,
			Bytes:   100,
			Packets: 1,
		},
	},
	Tuple: &nfnetlink.NflogPacketTuple{
		Src:   remoteIp1,
		Dst:   localIp1,
		Proto: proto_tcp,
		L4Src: nfnetlink.NflogL4Info{Port: srcPort},
		L4Dst: nfnetlink.NflogL4Info{Port: dstPort},
	},
}

var localPkt = &nfnetlink.NflogPacketAggregate{
	Prefixes: []nfnetlink.NflogPrefix{
		{
			Prefix:  defTierDeny,
			Len:     19,
			Bytes:   100,
			Packets: 1,
		},
	},
	Tuple: &nfnetlink.NflogPacketTuple{
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
			StatsDumpFilePath:     "/tmp/qwerty",
			NfNetlinkBufSize:      65535,
			IngressGroup:          1200,
			EgressGroup:           2200,
			AgeTimeout:            time.Duration(10) * time.Second,
			InitialReportingDelay: time.Duration(5) * time.Second,
			ExportingInterval:     time.Duration(1) * time.Second,
		}
		rm := NewReporterManager()
		BeforeEach(func() {
			epMap := map[[16]byte]*model.WorkloadEndpointKey{
				localIp1: localWlEPKey1,
				localIp2: localWlEPKey2,
			}
			lm := newMockLookupManager(epMap)
			c = NewCollector(lm, rm, conf)
			go c.startStatsCollectionAndReporting()
		})
		Describe("Test local destination", func() {
			It("should receive a single stat update with allow rule tracepoint", func() {
				t := NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort)
				c.nfIngressC <- inPkt
				Eventually(c.epStats).Should(HaveKey(*t))
			})
		})
		Describe("Test local to local", func() {
			It("should receive a single stat update with deny rule tracepoint", func() {
				t := NewTuple(localIp1, localIp2, proto_tcp, srcPort, dstPort)
				c.nfIngressC <- localPkt
				Eventually(c.epStats).Should(HaveKey(*t))
			})
		})
	})
})

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
		lm := newMockLookupManager(epMap)
		c = NewCollector(lm, rm, conf)
		go c.startStatsCollectionAndReporting()
	})
	Describe("Test local destination", func() {
		It("should create a single entry", func() {
			t := NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort)
			c.ctEntriesC <- []nfnetlink.CtEntry{inCtEntry}
			Eventually(c.epStats).Should(HaveKey(*t))
			data := c.epStats[*t]
			Expect(data.Counters()).Should(Equal(*NewCounter(inCtEntry.OriginalCounters.Packets, inCtEntry.OriginalCounters.Bytes)))
			Expect(data.CountersReverse()).Should(Equal(*NewCounter(inCtEntry.ReplyCounters.Packets, inCtEntry.ReplyCounters.Bytes)))
		})
	})
	Describe("Test local source", func() {
		It("should receive a single stat update", func() {
			t := NewTuple(localIp1, remoteIp1, proto_tcp, srcPort, dstPort)
			c.ctEntriesC <- []nfnetlink.CtEntry{outCtEntry}
			Eventually(c.epStats).Should(HaveKey(*t))
			data := c.epStats[*t]
			Expect(data.Counters()).Should(Equal(*NewCounter(outCtEntry.OriginalCounters.Packets, outCtEntry.OriginalCounters.Bytes)))
			Expect(data.CountersReverse()).Should(Equal(*NewCounter(outCtEntry.ReplyCounters.Packets, outCtEntry.ReplyCounters.Bytes)))
		})
	})
	Describe("Test local source to local destination", func() {
		It("should receive two stat updates - one for each endpoint", func() {
			t1 := NewTuple(localIp1, localIp2, proto_tcp, srcPort, dstPort)
			t2 := NewTuple(localIp2, localIp1, proto_tcp, dstPort, srcPort)
			c.ctEntriesC <- []nfnetlink.CtEntry{localCtEntry}
			Eventually(c.epStats).Should(HaveKey(Equal(*t1)))
			Eventually(c.epStats).Should(HaveKey(Equal(*t2)))
			data1 := c.epStats[*t1]
			data2 := c.epStats[*t2]
			Expect(data1.Counters()).Should(Equal(*NewCounter(localCtEntry.OriginalCounters.Packets, localCtEntry.OriginalCounters.Bytes)))
			Expect(data1.CountersReverse()).Should(Equal(*NewCounter(localCtEntry.ReplyCounters.Packets, localCtEntry.ReplyCounters.Bytes)))
			// Counters are reversed.
			Expect(data2.Counters()).Should(Equal(*NewCounter(localCtEntry.ReplyCounters.Packets, localCtEntry.ReplyCounters.Bytes)))
			Expect(data2.CountersReverse()).Should(Equal(*NewCounter(localCtEntry.OriginalCounters.Packets, localCtEntry.OriginalCounters.Bytes)))
		})
	})
	Describe("Test local destination with DNAT", func() {
		It("should receive a single stat update with correct tuple extracted", func() {
			t := NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort)
			c.ctEntriesC <- []nfnetlink.CtEntry{inCtEntryWithDNAT}
			Eventually(c.epStats).Should(HaveKey(Equal(*t)))
			data := c.epStats[*t]
			Expect(data.Counters()).Should(Equal(*NewCounter(inCtEntryWithDNAT.OriginalCounters.Packets, inCtEntryWithDNAT.OriginalCounters.Bytes)))
			Expect(data.CountersReverse()).Should(Equal(*NewCounter(inCtEntryWithDNAT.ReplyCounters.Packets, inCtEntryWithDNAT.ReplyCounters.Bytes)))
			//su := NewStatUpdate(*t,
			//	inCtEntryWithDNAT.OriginalCounters.Packets, inCtEntryWithDNAT.OriginalCounters.Bytes,
			//	inCtEntryWithDNAT.ReplyCounters.Packets, inCtEntryWithDNAT.ReplyCounters.Bytes,
			//	AbsoluteCounter, DirUnknown, EmptyRuleTracePoint)
		})
	})
	Describe("Test local source to local destination with DNAT", func() {
		It("should receive two stat updates - one for each endpoint - with correct tuple extracted", func() {
			t1 := NewTuple(localIp1, localIp2, proto_tcp, srcPort, dstPort)
			t2 := NewTuple(localIp2, localIp1, proto_tcp, dstPort, srcPort)
			c.ctEntriesC <- []nfnetlink.CtEntry{localCtEntryWithDNAT}
			Eventually(c.epStats).Should(HaveKey(Equal(*t1)))
			Eventually(c.epStats).Should(HaveKey(Equal(*t2)))
			data1 := c.epStats[*t1]
			data2 := c.epStats[*t2]
			Expect(data1.Counters()).Should(Equal(*NewCounter(localCtEntryWithDNAT.OriginalCounters.Packets, localCtEntryWithDNAT.OriginalCounters.Bytes)))
			Expect(data1.CountersReverse()).Should(Equal(*NewCounter(localCtEntryWithDNAT.ReplyCounters.Packets, localCtEntryWithDNAT.ReplyCounters.Bytes)))
			// Counters are reversed.
			Expect(data2.Counters()).Should(Equal(*NewCounter(localCtEntryWithDNAT.ReplyCounters.Packets, localCtEntryWithDNAT.ReplyCounters.Bytes)))
			Expect(data2.CountersReverse()).Should(Equal(*NewCounter(localCtEntryWithDNAT.OriginalCounters.Packets, localCtEntryWithDNAT.OriginalCounters.Bytes)))
		})
	})

})

var _ = Describe("Rtp", func() {
	Describe("Rtp lookup", func() {
		Describe("Test lookupRule", func() {
			epMap := map[[16]byte]*model.WorkloadEndpointKey{
				localIp1: localWlEPKey1,
				localIp2: localWlEPKey2,
			}
			lm := newMockLookupManager(epMap)
			It("should parse correctly", func() {
				prefix := defTierAllow
				prefixLen := 19
				rtp, _ := lookupRule(lm, prefix, prefixLen, localWlEPKey1)
				rtp.Ctr = *NewCounter(1, 100)
				Expect(rtp).To(Equal(defTierAllowTp))
			})
		})
	})
})

var _ = Describe("Reporting Metrics", func() {
	var c *Collector
	const (
		ageTimeout        = time.Duration(3) * time.Second
		reportingDelay    = time.Duration(2) * time.Second
		exportingInterval = time.Duration(1) * time.Second
	)
	conf := &Config{
		StatsDumpFilePath:     "/tmp/qwerty",
		NfNetlinkBufSize:      65535,
		IngressGroup:          1200,
		EgressGroup:           2200,
		AgeTimeout:            ageTimeout,
		InitialReportingDelay: reportingDelay,
		ExportingInterval:     exportingInterval,
	}
	rm := NewReporterManager()
	mockReporter := newMockReporter()
	rm.RegisterMetricsReporter(mockReporter)
	BeforeEach(func() {
		epMap := map[[16]byte]*model.WorkloadEndpointKey{
			localIp1: localWlEPKey1,
			localIp2: localWlEPKey2,
		}
		lm := newMockLookupManager(epMap)
		rm.Start()
		c = NewCollector(lm, rm, conf)
		go c.startStatsCollectionAndReporting()
	})
	Describe("Report Denied Packets", func() {
		var t *Tuple
		BeforeEach(func() {
			t = NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort)
			c.nfIngressC <- inPktDeny
		})
		Context("reporting tick", func() {
			It("should receive metric", func() {
				tmu := &testMetricUpdate{*t, RtpToBytes(defTierDenyTp)}
				Eventually(func() *testMetricUpdate {
					return <-mockReporter.reportChan
				}, reportingDelay, exportingInterval).Should(Equal(tmu))
			})
		})
	})
	Describe("Don't Report Allowed Packets", func() {
		var t *Tuple
		BeforeEach(func() {
			t = NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort)
			c.nfIngressC <- inPkt
		})
		Context("reporting tick", func() {
			It("should not receive metric", func() {
				Consistently(func() *testMetricUpdate {
					var a *testMetricUpdate
					select {
					case a = <-mockReporter.reportChan:
					default:
					}
					return a
				}, ageTimeout, exportingInterval).Should(BeNil())
			})
		})
	})
	Describe("Don't Report Packets that switch from deny to allow", func() {
		var t *Tuple
		BeforeEach(func() {
			t = NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort)
			c.nfIngressC <- inPktDeny
			time.Sleep(time.Duration(500) * time.Millisecond)
			c.nfIngressC <- inPkt
		})
		Context("reporting tick", func() {
			It("should not receive metric", func() {
				Consistently(func() *testMetricUpdate {
					var a *testMetricUpdate
					select {
					case a = <-mockReporter.reportChan:
					default:
					}
					return a
				}, ageTimeout, exportingInterval).Should(BeNil())
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

func (lm *mockLookupManager) GetPolicyIndex(epKey interface{}, policyName, tierName []byte) int {
	return 0
}

func RtpToBytes(tp *RuleTracePoint) []byte {
	buf := &bytes.Buffer{}
	buf.Write(tp.TierID())
	buf.Write([]byte("|"))
	buf.Write(tp.PolicyID())
	buf.Write([]byte("|"))
	buf.Write(tp.Rule())
	buf.Write([]byte("|"))
	buf.Write(RuleActionToBytes[tp.Action])
	return buf.Bytes()
}

type testMetricUpdate struct {
	tuple  Tuple
	policy []byte
}

type mockReporter struct {
	reportChan chan *testMetricUpdate
	expireChan chan *testMetricUpdate
}

func newMockReporter() *mockReporter {
	return &mockReporter{
		reportChan: make(chan *testMetricUpdate),
		expireChan: make(chan *testMetricUpdate),
	}
}

func (mr *mockReporter) Start() {
	// Do nothing. We are a mock anyway.
}

func (mr *mockReporter) Report(mu *MetricUpdate) error {
	mr.reportChan <- &testMetricUpdate{mu.tuple, mu.policy}
	return nil
}

func (mr *mockReporter) Expire(mu *MetricUpdate) error {
	mr.expireChan <- &testMetricUpdate{mu.tuple, mu.policy}
	return nil
}

func BenchmarkNflogPktToStat(b *testing.B) {
	epMap := map[[16]byte]*model.WorkloadEndpointKey{
		localIp1: localWlEPKey1,
		localIp2: localWlEPKey2,
	}
	conf := &Config{
		StatsDumpFilePath:     "/tmp/qwerty",
		NfNetlinkBufSize:      65535,
		IngressGroup:          1200,
		EgressGroup:           2200,
		AgeTimeout:            time.Duration(10) * time.Second,
		InitialReportingDelay: time.Duration(5) * time.Second,
		ExportingInterval:     time.Duration(1) * time.Second,
	}
	rm := NewReporterManager()
	lm := newMockLookupManager(epMap)
	c := NewCollector(lm, rm, conf)
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		c.convertNflogPktAndApplyUpdate(DirIn, inPkt)
	}
}

func BenchmarkApplyStatUpdate(b *testing.B) {
	epMap := map[[16]byte]*model.WorkloadEndpointKey{
		localIp1: localWlEPKey1,
		localIp2: localWlEPKey2,
	}
	conf := &Config{
		StatsDumpFilePath:     "/tmp/qwerty",
		NfNetlinkBufSize:      65535,
		IngressGroup:          1200,
		EgressGroup:           2200,
		AgeTimeout:            time.Duration(10) * time.Second,
		InitialReportingDelay: time.Duration(5) * time.Second,
		ExportingInterval:     time.Duration(1) * time.Second,
	}
	rm := NewReporterManager()
	lm := newMockLookupManager(epMap)
	c := NewCollector(lm, rm, conf)
	tuples := []Tuple{}
	MaxSrcPort := 1000
	MaxDstPort := 1000
	for sp := 1; sp < MaxSrcPort; sp++ {
		for dp := 1; dp < MaxDstPort; dp++ {
			t := NewTuple(localIp1, localIp2, proto_tcp, sp, dp)
			tuples = append(tuples, *t)
		}
	}
	tps := []*RuleTracePoint{}
	MaxEntries := 10000
	for i := 0; i < MaxEntries; i++ {
		tp := defTierDenyTp
		tps = append(tps, tp)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		for i := 0; i < MaxEntries; i++ {
			c.applyStatUpdate(tuples[i], 0, 0, 0, 0, DeltaCounter, DirIn, tps[i])
		}
	}
}
