// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package collector

import (
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"fmt"

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
	defTierAllowNFLOGPrefix          = [64]byte{'A', '|', '0', '|', 'd', 'e', 'f', 'a', 'u', 'l', 't', '.', 'p', 'o', 'l', 'i', 'c', 'y', '1', '|', 'p', 'o'}
	defTierDenyNFLOGPrefix           = [64]byte{'D', '|', '0', '|', 'd', 'e', 'f', 'a', 'u', 'l', 't', '.', 'p', 'o', 'l', 'i', 'c', 'y', '2', '|', 'p', 'o'}
	defTierPolicy1AllowIngressRuleID = rules.RuleIDs{
		Tier:      "default",
		Policy:    "policy1",
		Namespace: "__GLOBAL__",
		Index:     "0",
		Action:    rules.ActionAllow,
		Direction: rules.RuleDirIngress,
	}
	defTierPolicy1AllowEgressRuleID = rules.RuleIDs{
		Tier:      "default",
		Policy:    "policy1",
		Namespace: "__GLOBAL__",
		Index:     "0",
		Action:    rules.ActionAllow,
		Direction: rules.RuleDirEgress,
	}
	defTierPolicy2DenyIngressRuleID = rules.RuleIDs{
		Tier:      "default",
		Policy:    "policy2",
		Namespace: "__GLOBAL__",
		Index:     "0",
		Action:    rules.ActionDeny,
		Direction: rules.RuleDirIngress,
	}
)

var _ = Describe("Lookup Rule ID", func() {
	// Test setup
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

	// Test data
	tier1 := "tier1"
	namespace1 := "namespace1"
	np1 := "np1"   // Should include namespace and tier to make this a valid policy name
	gnp1 := "gnp1" // Should include tier to make this a valid policy name.
	k8snp1 := "knp.default.knp1"
	profile1 := "profile1"

	type testTierPolicy struct {
		namespace string
		tier      string
		policy    string
	}
	data := []testTierPolicy{
		{namespace1, tier1, np1},
		{"__GLOBAL__", tier1, gnp1},
		{namespace1, "default", k8snp1},
		{namespace1, "profile", profile1},
	}

	epMap := map[[16]byte]*model.WorkloadEndpointKey{
		localIp1: localWlEPKey1,
		localIp2: localWlEPKey2,
	}

	nflogMap := map[[64]byte]lookup.RuleIDParts{}

	for _, d := range data {
		k, v := policyIDStrToRuleIDParts(d.namespace, d.tier, d.policy)
		nflogMap[k] = v
	}

	lm := newMockLookupManager(epMap, nflogMap)
	rm.Start()
	c = NewCollector(lm, rm, conf)

	It("should process policies", func() {
		By("processing ingress allow policies")
		pfx, pfxLen, expectedRuleIDs := getNflogPrefix(rules.RuleDirIngress, rules.ActionAllow, 0, tier1, np1, namespace1)
		actualRuleIds, err := c.lookupRuleIDsFromPrefix(rules.RuleDirIngress, pfx, pfxLen)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(actualRuleIds).Should(Equal(expectedRuleIDs))

		By("processing egress allow policies")
		pfx, pfxLen, expectedRuleIDs = getNflogPrefix(rules.RuleDirEgress, rules.ActionAllow, 0, tier1, np1, namespace1)
		actualRuleIds, err = c.lookupRuleIDsFromPrefix(rules.RuleDirEgress, pfx, pfxLen)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(actualRuleIds).Should(Equal(expectedRuleIDs))

		By("processing ingress deny policies")
		pfx, pfxLen, expectedRuleIDs = getNflogPrefix(rules.RuleDirIngress, rules.ActionDeny, 0, tier1, np1, namespace1)
		actualRuleIds, err = c.lookupRuleIDsFromPrefix(rules.RuleDirIngress, pfx, pfxLen)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(actualRuleIds).Should(Equal(expectedRuleIDs))

		By("processing ingress allow gnp")
		pfx, pfxLen, expectedRuleIDs = getNflogPrefix(rules.RuleDirIngress, rules.ActionAllow, 0, tier1, gnp1, "")
		actualRuleIds, err = c.lookupRuleIDsFromPrefix(rules.RuleDirIngress, pfx, pfxLen)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(actualRuleIds).Should(Equal(expectedRuleIDs))
	})

	It("should process k8s network policies", func() {
		By("processing ingress allow k8s policy")
		pfx, pfxLen, expectedRuleIDs := getNflogPrefix(rules.RuleDirIngress, rules.ActionAllow, 0, "", k8snp1, namespace1)
		actualRuleIds, err := c.lookupRuleIDsFromPrefix(rules.RuleDirIngress, pfx, pfxLen)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(actualRuleIds).Should(Equal(expectedRuleIDs))
	})

	It("should process profiles", func() {
		By("processing ingress allow profile")
		pfx, pfxLen, expectedRuleIDs := getNflogPrefix(rules.RuleDirIngress, rules.ActionAllow, 0, "profile", profile1, namespace1)
		actualRuleIds, err := c.lookupRuleIDsFromPrefix(rules.RuleDirIngress, pfx, pfxLen)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(actualRuleIds).Should(Equal(expectedRuleIDs))
	})
})

var defTierAllowIngressTp = &RuleTracePoint{
	RuleIDs: defTierPolicy1AllowIngressRuleID,
	Index:   0,
	EpKey:   localWlEPKey1,
	Ctr:     *NewCounter(1, 100),
}

var defTierAllowEgressTp = &RuleTracePoint{
	RuleIDs: defTierPolicy1AllowEgressRuleID,
	Index:   0,
	EpKey:   localWlEPKey1,
	Ctr:     *NewCounter(1, 100),
}

var defTierDenyIngressTp = &RuleTracePoint{
	RuleIDs: defTierPolicy2DenyIngressRuleID,
	Index:   0,
	EpKey:   localWlEPKey2,
	Ctr:     *NewCounter(1, 100),
}

var ingressPktAllow = &nfnetlink.NflogPacketAggregate{
	Prefixes: []nfnetlink.NflogPrefix{
		{
			Prefix:  defTierAllowNFLOGPrefix,
			Len:     22,
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
var ingressPktAllowTuple = NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort)

var egressPktAllow = &nfnetlink.NflogPacketAggregate{
	Prefixes: []nfnetlink.NflogPrefix{
		{
			Prefix:  defTierAllowNFLOGPrefix,
			Len:     22,
			Bytes:   100,
			Packets: 1,
		},
	},
	Tuple: &nfnetlink.NflogPacketTuple{
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
			Prefix:  defTierDenyNFLOGPrefix,
			Len:     22,
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
var ingressPktDenyTuple = NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort)

var localPkt = &nfnetlink.NflogPacketAggregate{
	Prefixes: []nfnetlink.NflogPrefix{
		{
			Prefix:  defTierDenyNFLOGPrefix,
			Len:     22,
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
			nflogMap := map[[64]byte]lookup.RuleIDParts{}

			for _, rtp := range []*RuleTracePoint{defTierAllowEgressTp, defTierAllowIngressTp, defTierDenyIngressTp} {
				k, v := policyIDStrToRuleIDParts("__GLOBAL__", rtp.RuleIDs.Tier, rtp.RuleIDs.Policy)
				nflogMap[k] = v
			}

			lm := newMockLookupManager(epMap, nflogMap)
			c = NewCollector(lm, rm, conf)
			go c.startStatsCollectionAndReporting()
		})
		Describe("Test local destination", func() {
			It("should receive a single stat update with allow rule tracepoint", func() {
				t := NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort)
				c.nfIngressC <- ingressPktAllow
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

		nflogMap := map[[64]byte]lookup.RuleIDParts{}

		for _, rtp := range []*RuleTracePoint{defTierAllowEgressTp, defTierAllowIngressTp, defTierDenyIngressTp} {
			k, v := policyIDStrToRuleIDParts("__GLOBAL__", rtp.RuleIDs.Tier, rtp.RuleIDs.Policy)
			nflogMap[k] = v
		}

		lm := newMockLookupManager(epMap, nflogMap)
		c = NewCollector(lm, rm, conf)
		go c.startStatsCollectionAndReporting()
	})
	Describe("Test local destination", func() {
		It("should create a single entry in inbound direction", func() {
			t := NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort)
			c.ctEntriesC <- []nfnetlink.CtEntry{inCtEntry}
			Eventually(c.epStats).Should(HaveKey(*t))
			data := c.epStats[*t]
			Expect(data.Counters()).Should(Equal(*NewCounter(inCtEntry.OriginalCounters.Packets, inCtEntry.OriginalCounters.Bytes)))
			Expect(data.CountersReverse()).Should(Equal(*NewCounter(inCtEntry.ReplyCounters.Packets, inCtEntry.ReplyCounters.Bytes)))
		})
	})
	Describe("Test local source", func() {
		It("should create a single entry with outbound direction", func() {
			t := NewTuple(localIp1, remoteIp1, proto_tcp, srcPort, dstPort)
			c.ctEntriesC <- []nfnetlink.CtEntry{outCtEntry}
			Eventually(c.epStats).Should(HaveKey(*t))
			data := c.epStats[*t]
			Expect(data.Counters()).Should(Equal(*NewCounter(outCtEntry.OriginalCounters.Packets, outCtEntry.OriginalCounters.Bytes)))
			Expect(data.CountersReverse()).Should(Equal(*NewCounter(outCtEntry.ReplyCounters.Packets, outCtEntry.ReplyCounters.Bytes)))
		})
	})
	Describe("Test local source to local destination", func() {
		It("should create a single entry with 'local' direction", func() {
			t1 := NewTuple(localIp1, localIp2, proto_tcp, srcPort, dstPort)
			c.ctEntriesC <- []nfnetlink.CtEntry{localCtEntry}
			Eventually(c.epStats).Should(HaveKey(Equal(*t1)))
			data := c.epStats[*t1]
			Expect(data.Counters()).Should(Equal(*NewCounter(localCtEntry.OriginalCounters.Packets, localCtEntry.OriginalCounters.Bytes)))
			Expect(data.CountersReverse()).Should(Equal(*NewCounter(localCtEntry.ReplyCounters.Packets, localCtEntry.ReplyCounters.Bytes)))
		})
	})
	Describe("Test local destination with DNAT", func() {
		It("should create a single entry with inbound connection direction and with correct tuple extracted", func() {
			t := NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort)
			c.ctEntriesC <- []nfnetlink.CtEntry{inCtEntryWithDNAT}
			Eventually(c.epStats).Should(HaveKey(Equal(*t)))
			data := c.epStats[*t]
			Expect(data.Counters()).Should(Equal(*NewCounter(inCtEntryWithDNAT.OriginalCounters.Packets, inCtEntryWithDNAT.OriginalCounters.Bytes)))
			Expect(data.CountersReverse()).Should(Equal(*NewCounter(inCtEntryWithDNAT.ReplyCounters.Packets, inCtEntryWithDNAT.ReplyCounters.Bytes)))
		})
	})
	Describe("Test local source to local destination with DNAT", func() {
		It("should create a single entry with 'local' connection direction and with correct tuple extracted", func() {
			t1 := NewTuple(localIp1, localIp2, proto_tcp, srcPort, dstPort)
			c.ctEntriesC <- []nfnetlink.CtEntry{localCtEntryWithDNAT}
			Eventually(c.epStats).Should(HaveKey(Equal(*t1)))
			data := c.epStats[*t1]
			Expect(data.Counters()).Should(Equal(*NewCounter(localCtEntryWithDNAT.OriginalCounters.Packets, localCtEntryWithDNAT.OriginalCounters.Bytes)))
			Expect(data.CountersReverse()).Should(Equal(*NewCounter(localCtEntryWithDNAT.ReplyCounters.Packets, localCtEntryWithDNAT.ReplyCounters.Bytes)))
		})
	})

})

func policyIDStrToRuleIDParts(namespace, tier, policy string) ([64]byte, lookup.RuleIDParts) {
	var (
		s     string
		byt64 [64]byte
	)
	if namespace != "__GLOBAL__" {
		if strings.HasPrefix(policy, "knp.default.") {
			s = fmt.Sprintf("%s/%s|po", namespace, policy)
		} else {
			s = fmt.Sprintf("%s/%s.%s|po", namespace, tier, policy)
		}
	} else {
		s = fmt.Sprintf("%s.%s|po", tier, policy)
	}

	byt := []byte(s)
	copy(byt64[:], byt)

	return byt64, lookup.RuleIDParts{
		Namespace: namespace,
		Tier:      tier,
		Policy:    policy,
	}
}

func getNflogPrefix(ruleDir rules.RuleDirection, action rules.RuleAction, ruleIndex int, tier, policy, namespace string) ([64]byte, int, rules.RuleIDs) {
	var (
		actPfx, ns string
		pfxArr     [64]byte
		prefix     []byte
	)
	switch action {
	case rules.ActionAllow:
		actPfx = "A"
	case rules.ActionDeny:
		actPfx = "D"
	case rules.ActionNextTier:
		actPfx = "N"
	}

	if namespace != "" {
		if strings.HasPrefix(policy, "knp.default.") {
			prefix = []byte(fmt.Sprintf("%s|%d|%s/%s|po", actPfx, ruleIndex, namespace, policy))
			tier = "default"
		} else {
			prefix = []byte(fmt.Sprintf("%s|%d|%s/%s.%s|po", actPfx, ruleIndex, namespace, tier, policy))
		}
		ns = namespace
	} else {
		prefix = []byte(fmt.Sprintf("%s|%d|%s.%s|po", actPfx, ruleIndex, tier, policy))
		ns = "__GLOBAL__"
	}
	copy(pfxArr[:], prefix)

	rid := rules.RuleIDs{
		Tier:      tier,
		Policy:    policy,
		Namespace: ns,
		Index:     fmt.Sprintf("%d", ruleIndex),
		Action:    action,
		Direction: ruleDir,
	}
	return pfxArr, len(prefix), rid
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

		nflogMap := map[[64]byte]lookup.RuleIDParts{}

		for _, rtp := range []*RuleTracePoint{defTierAllowEgressTp, defTierAllowIngressTp, defTierDenyIngressTp} {
			k, v := policyIDStrToRuleIDParts("__GLOBAL__", rtp.RuleIDs.Tier, rtp.RuleIDs.Policy)
			nflogMap[k] = v
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
					ruleIDs:      defTierDenyIngressTp.RuleIDs,
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
					ruleIDs:      defTierAllowIngressTp.RuleIDs,
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
					ruleIDs:      defTierAllowIngressTp.RuleIDs,
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
					ruleIDs:      defTierAllowEgressTp.RuleIDs,
					isConnection: false,
				}
				Eventually(mockReporter.reportChan, reportingDelay*2).Should(Receive(Equal(tmu)))
			})
		})
	})
})

type mockLookupManager struct {
	epMap    map[[16]byte]*model.WorkloadEndpointKey
	nflogMap map[[64]byte]lookup.RuleIDParts
}

func newMockLookupManager(em map[[16]byte]*model.WorkloadEndpointKey, nm map[[64]byte]lookup.RuleIDParts) *mockLookupManager {
	return &mockLookupManager{
		epMap:    em,
		nflogMap: nm,
	}
}

func (lm *mockLookupManager) GetEndpointKey(addr [16]byte) (interface{}, error) {
	data, _ := lm.epMap[addr]
	if data != nil {
		return data, nil
	}
	return nil, lookup.UnknownEndpointError

}

func (lm *mockLookupManager) GetTierIndex(epKey interface{}, tierName string) int {
	return 0
}

func (m *mockLookupManager) GetNFLOGHashToPolicyID(prefixHash [64]byte) (lookup.RuleIDParts, error) {
	policyID, ok := m.nflogMap[prefixHash]
	if !ok {
		return lookup.RuleIDParts{}, fmt.Errorf("cannot find the specified NFLOG prefix string or hash: %s in the lookup manager", prefixHash)
	}

	return policyID, nil
}

// Define a separate metric type that doesn't include the actual stats.  We use this
// for simpler comparisons.
type testMetricUpdate struct {
	updateType UpdateType

	// Tuple key
	tuple Tuple

	// Rule identification
	ruleIDs rules.RuleIDs

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

func (mr *mockReporter) Report(mu *MetricUpdate) error {
	mr.reportChan <- testMetricUpdate{
		updateType:   UpdateTypeReport,
		tuple:        mu.tuple,
		ruleIDs:      mu.ruleIDs,
		isConnection: mu.isConnection,
	}
	return nil
}

func BenchmarkNflogPktToStat(b *testing.B) {
	epMap := map[[16]byte]*model.WorkloadEndpointKey{
		localIp1: localWlEPKey1,
		localIp2: localWlEPKey2,
	}

	nflogMap := map[[64]byte]lookup.RuleIDParts{}

	for _, rtp := range []*RuleTracePoint{defTierAllowEgressTp, defTierAllowIngressTp, defTierDenyIngressTp} {
		k, v := policyIDStrToRuleIDParts("__GLOBAL__", rtp.RuleIDs.Tier, rtp.RuleIDs.Policy)
		nflogMap[k] = v
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

	nflogMap := map[[64]byte]lookup.RuleIDParts{}

	for _, rtp := range []*RuleTracePoint{defTierAllowEgressTp, defTierAllowIngressTp, defTierDenyIngressTp} {
		k, v := policyIDStrToRuleIDParts("__GLOBAL__", rtp.RuleIDs.Tier, rtp.RuleIDs.Policy)
		nflogMap[k] = v
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
	var tps []*RuleTracePoint
	MaxEntries := 10000
	for i := 0; i < MaxEntries; i++ {
		tp := defTierDenyIngressTp
		tps = append(tps, tp)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		for i := 0; i < MaxEntries; i++ {
			c.applyNflogStatUpdate(tuples[i], tps[i])
		}
	}
}

var r rules.RuleIDs

func benchmarkLookupRuleIDs(b *testing.B, nflogMap map[[64]byte]lookup.RuleIDParts, ruleDir rules.RuleDirection, pfx [64]byte, pfxLen int) {
	epMap := map[[16]byte]*model.WorkloadEndpointKey{
		localIp1: localWlEPKey1,
		localIp2: localWlEPKey2,
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

	var actualRuleIds rules.RuleIDs
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		actualRuleIds, _ = c.lookupRuleIDsFromPrefix(ruleDir, pfx, pfxLen)
	}
	r = actualRuleIds
}

func BenchmarkLookupRuleIDs(b *testing.B) {
	nflogMap := map[[64]byte]lookup.RuleIDParts{}

	// Test data
	tier1 := "tier1"
	namespace1 := "namespace1"
	np1 := "np1"   // Should include namespace and tier to make this a valid policy name
	gnp1 := "gnp1" // Should include tier to make this a valid policy name.
	k8snp1 := "knp.default.knp1"

	type testTierPolicy struct {
		namespace string
		tier      string
		policy    string
	}
	data := []testTierPolicy{
		{namespace1, tier1, np1},
		{"__GLOBAL__", tier1, gnp1},
		{namespace1, "default", k8snp1},
	}

	for _, d := range data {
		k, v := policyIDStrToRuleIDParts(d.namespace, d.tier, d.policy)
		nflogMap[k] = v
	}

	pfx, pfxLen, _ := getNflogPrefix(rules.RuleDirIngress, rules.ActionAllow, 0, tier1, np1, namespace1)
	benchmarkLookupRuleIDs(b, nflogMap, rules.RuleDirIngress, pfx, pfxLen)
}

func BenchmarkLookupRuleIDsNoPolicyMatchWithMap(b *testing.B) {
	nflogMap := map[[64]byte]lookup.RuleIDParts{}

	// Test data
	tier1 := "tier1"
	namespace1 := "namespace1"
	np1 := "np1"   // Should include namespace and tier to make this a valid policy name
	gnp1 := "gnp1" // Should include tier to make this a valid policy name.
	k8snp1 := "knp.default.knp1"
	noPolicyIn := "no-policy-match-inbound"
	noPolicyOut := "no-policy-match-outbound"

	type testTierPolicy struct {
		namespace string
		tier      string
		policy    string
	}
	data := []testTierPolicy{
		{namespace1, tier1, np1},
		{"__GLOBAL__", tier1, gnp1},
		{namespace1, "default", k8snp1},
		{"__GLOBAL__", "default", noPolicyIn},
		{"__GLOBAL__", "default", noPolicyOut},
	}

	for _, d := range data {
		k, v := policyIDStrToRuleIDParts(d.namespace, d.tier, d.policy)
		nflogMap[k] = v
	}

	pfx, pfxLen, _ := getNflogPrefix(rules.RuleDirIngress, rules.ActionDeny, 0, "default", noPolicyIn, "")
	benchmarkLookupRuleIDs(b, nflogMap, rules.RuleDirIngress, pfx, pfxLen)
}
