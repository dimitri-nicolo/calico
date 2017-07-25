package collector

import (
	"testing"

	"github.com/projectcalico/felix/lookup"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/tigera/nfnetlink"

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
	defTierAllow = [64]byte{'A', '/', '0', '/', 'p', 'o', 'l', 'i', 'c', 'y', '1', '/', 'd', 'e', 'f', 'a', 'u', 'l', 't'}
	defTierDeny  = [64]byte{'D', '/', '0', '/', 'p', 'o', 'l', 'i', 'c', 'y', '2', '/', 'd', 'e', 'f', 'a', 'u', 'l', 't'}
	tier1Allow   = [64]byte{'A', '/', '0', '/', 'p', 'o', 'l', 'i', 'c', 'y', '3', '/', 't', 'i', 'e', 'r', '1'}
	tier1Deny    = [64]byte{'D', '/', '0', '/', 'p', 'o', 'l', 'i', 'c', 'y', '4', '/', 't', 'i', 'e', 'r', '1'}
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
			c.Start()
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
	buf.Write([]byte("/"))
	buf.Write(tp.PolicyID())
	buf.Write([]byte("/"))
	buf.Write(tp.Rule())
	buf.Write([]byte("/"))
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
