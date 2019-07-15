// +build !windows

// Copyright (c) 2018-2019 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"
	net2 "net"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/tigera/nfnetlink"
	"github.com/tigera/nfnetlink/nfnl"

	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/felix/rules"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/net"
)

const (
	ipv4       = 0x800
	proto_icmp = 1
	proto_tcp  = 6
	proto_udp  = 17
)

var (
	localIp1Str  = "10.0.0.1"
	localIp1     = ipStrTo16Byte(localIp1Str)
	localIp2Str  = "10.0.0.2"
	localIp2     = ipStrTo16Byte(localIp2Str)
	remoteIp1Str = "20.0.0.1"
	remoteIp1    = ipStrTo16Byte(remoteIp1Str)
	remoteIp2Str = "20.0.0.2"
	remoteIp2    = ipStrTo16Byte(remoteIp2Str)
	localIp1DNAT = ipStrTo16Byte("192.168.0.1")
	localIp2DNAT = ipStrTo16Byte("192.168.0.2")
	publicIP1Str = "1.0.0.1"
	publicIP2Str = "2.0.0.2"
	netSetIp1Str = "8.8.8.8"
	netSetIp1    = ipStrTo16Byte(netSetIp1Str)
)

var (
	srcPort     = 54123
	dstPort     = 80
	dstPortDNAT = 8080
)

var (
	localWlEPKey1 = model.WorkloadEndpointKey{
		Hostname:       "localhost",
		OrchestratorID: "orchestrator",
		WorkloadID:     "localworkloadid1",
		EndpointID:     "localepid1",
	}

	localWlEPKey2 = model.WorkloadEndpointKey{
		Hostname:       "localhost",
		OrchestratorID: "orchestrator",
		WorkloadID:     "localworkloadid2",
		EndpointID:     "localepid2",
	}

	localHostEpKey1 = model.HostEndpointKey{
		Hostname:   "localhost",
		EndpointID: "eth1",
	}

	remoteWlEpKey1 = model.WorkloadEndpointKey{
		OrchestratorID: "orchestrator",
		WorkloadID:     "remoteworkloadid1",
		EndpointID:     "remoteepid1",
	}
	remoteWlEpKey2 = model.WorkloadEndpointKey{
		OrchestratorID: "orchestrator",
		WorkloadID:     "remoteworkloadid2",
		EndpointID:     "remoteepid2",
	}

	remoteHostEpKey1 = model.HostEndpointKey{
		Hostname:   "remotehost",
		EndpointID: "eth1",
	}

	localWlEp1 = &model.WorkloadEndpoint{
		State:    "active",
		Name:     "cali1",
		Mac:      mustParseMac("01:02:03:04:05:06"),
		IPv4Nets: []net.IPNet{mustParseNet("10.0.0.1/32")},
		Labels: map[string]string{
			"id": "local-ep-1",
		},
	}
	localWlEp2 = &model.WorkloadEndpoint{
		State:    "active",
		Name:     "cali2",
		Mac:      mustParseMac("01:02:03:04:05:07"),
		IPv4Nets: []net.IPNet{mustParseNet("10.0.0.2/32")},
		Labels: map[string]string{
			"id": "local-ep-2",
		},
	}
	localHostEp1 = &model.HostEndpoint{
		Name:              "eth1",
		ExpectedIPv4Addrs: []net.IP{mustParseIP("10.0.0.1")},
		Labels: map[string]string{
			"id": "loc-ep-1",
		},
	}
	remoteWlEp1 = &model.WorkloadEndpoint{
		State:    "active",
		Name:     "cali3",
		Mac:      mustParseMac("02:02:03:04:05:06"),
		IPv4Nets: []net.IPNet{mustParseNet("20.0.0.1/32")},
		Labels: map[string]string{
			"id": "remote-ep-1",
		},
	}
	remoteWlEp2 = &model.WorkloadEndpoint{
		State:    "active",
		Name:     "cali4",
		Mac:      mustParseMac("02:03:03:04:05:06"),
		IPv4Nets: []net.IPNet{mustParseNet("20.0.0.2/32")},
		Labels: map[string]string{
			"id": "remote-ep-2",
		},
	}
	remoteHostEp1 = &model.HostEndpoint{
		Name:              "eth1",
		ExpectedIPv4Addrs: []net.IP{mustParseIP("20.0.0.1")},
		Labels: map[string]string{
			"id": "rem-ep-1",
		},
	}
	localEd1 = &calc.EndpointData{
		Key:          localWlEPKey1,
		Endpoint:     localWlEp1,
		OrderedTiers: []string{"default"},
	}
	localEd2 = &calc.EndpointData{
		Key:          localWlEPKey2,
		Endpoint:     localWlEp2,
		OrderedTiers: []string{"default"},
	}
	remoteEd1 = &calc.EndpointData{
		Key:      remoteWlEpKey1,
		Endpoint: remoteWlEp1,
	}
	remoteEd2 = &calc.EndpointData{
		Key:      remoteWlEpKey2,
		Endpoint: remoteWlEp2,
	}
	localHostEd1 = &calc.EndpointData{
		Key:          localHostEpKey1,
		Endpoint:     localHostEp1,
		OrderedTiers: []string{"default"},
	}
	remoteHostEd1 = &calc.EndpointData{
		Key:      remoteHostEpKey1,
		Endpoint: remoteHostEp1,
	}

	netSetKey1 = model.NetworkSetKey{
		Name: "dns-servers",
	}
	netSet1 = model.NetworkSet{
		Nets:   []net.IPNet{mustParseNet(netSetIp1Str + "/32")},
		Labels: map[string]string{"public": "true"},
	}
)

// Nflog prefix test parameters
var (
	defTierAllowIngressNFLOGPrefix   = [64]byte{'A', 'P', 'I', '0', '|', 'd', 'e', 'f', 'a', 'u', 'l', 't', '.', 'p', 'o', 'l', 'i', 'c', 'y', '1'}
	defTierAllowEgressNFLOGPrefix    = [64]byte{'A', 'P', 'E', '0', '|', 'd', 'e', 'f', 'a', 'u', 'l', 't', '.', 'p', 'o', 'l', 'i', 'c', 'y', '1'}
	defTierDenyIngressNFLOGPrefix    = [64]byte{'D', 'P', 'I', '0', '|', 'd', 'e', 'f', 'a', 'u', 'l', 't', '.', 'p', 'o', 'l', 'i', 'c', 'y', '2'}
	defTierPolicy1AllowIngressRuleID = &calc.RuleID{
		Tier:      "default",
		Name:      "policy1",
		Namespace: "",
		Index:     0,
		IndexStr:  "0",
		Action:    rules.RuleActionAllow,
		Direction: rules.RuleDirIngress,
	}
	defTierPolicy1AllowEgressRuleID = &calc.RuleID{
		Tier:      "default",
		Name:      "policy1",
		Namespace: "",
		Index:     0,
		IndexStr:  "0",
		Action:    rules.RuleActionAllow,
		Direction: rules.RuleDirEgress,
	}
	defTierPolicy2DenyIngressRuleID = &calc.RuleID{
		Tier:      "default",
		Name:      "policy2",
		Namespace: "",
		Index:     0,
		IndexStr:  "0",
		Action:    rules.RuleActionDeny,
		Direction: rules.RuleDirIngress,
	}
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
		var c *collector
		conf := &Config{
			StatsDumpFilePath:            "/tmp/qwerty",
			NfNetlinkBufSize:             65535,
			IngressGroup:                 1200,
			EgressGroup:                  2200,
			AgeTimeout:                   time.Duration(10) * time.Second,
			ConntrackPollingInterval:     time.Duration(1) * time.Second,
			InitialReportingDelay:        time.Duration(5) * time.Second,
			ExportingInterval:            time.Duration(1) * time.Second,
			MaxOriginalSourceIPsIncluded: 5,
		}
		rm := NewReporterManager()
		BeforeEach(func() {
			epMap := map[[16]byte]*calc.EndpointData{
				localIp1:  localEd1,
				localIp2:  localEd2,
				remoteIp1: remoteEd1,
			}
			nflogMap := map[[64]byte]*calc.RuleID{}

			for _, rid := range []*calc.RuleID{defTierPolicy1AllowEgressRuleID, defTierPolicy1AllowIngressRuleID, defTierPolicy2DenyIngressRuleID} {
				nflogMap[policyIDStrToRuleIDParts(rid)] = rid
			}

			lm := newMockLookupsCache(epMap, nflogMap, nil)
			c = newCollector(lm, rm, conf).(*collector)
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
	ProtoInfo:        nfnetlink.CtProtoInfo{State: nfnl.TCP_CONNTRACK_ESTABLISHED},
}

var alpEntryHTTPReqAllowed = 12
var alpEntryHTTPReqDenied = 130
var inALPEntry = proto.DataplaneStats{
	SrcIp:   remoteIp1Str,
	DstIp:   localIp1Str,
	SrcPort: int32(srcPort),
	DstPort: int32(dstPort),
	Protocol: &proto.Protocol{
		NumberOrName: &proto.Protocol_Number{proto_tcp},
	},
	Stats: []*proto.Statistic{
		{
			Direction:  proto.Statistic_IN,
			Relativity: proto.Statistic_DELTA,
			Kind:       proto.Statistic_HTTP_REQUESTS,
			Action:     proto.Action_ALLOWED,
			Value:      int64(alpEntryHTTPReqAllowed),
		},
		{
			Direction:  proto.Statistic_IN,
			Relativity: proto.Statistic_DELTA,
			Kind:       proto.Statistic_HTTP_REQUESTS,
			Action:     proto.Action_DENIED,
			Value:      int64(alpEntryHTTPReqDenied),
		},
	},
}

var dpStatsHTTPDataValue = 23
var dpStatsEntryWithFwdFor = proto.DataplaneStats{
	SrcIp:   remoteIp1Str,
	DstIp:   localIp1Str,
	SrcPort: int32(srcPort),
	DstPort: int32(dstPort),
	Protocol: &proto.Protocol{
		NumberOrName: &proto.Protocol_Number{proto_tcp},
	},
	Stats: []*proto.Statistic{
		{
			Direction:  proto.Statistic_IN,
			Relativity: proto.Statistic_DELTA,
			Kind:       proto.Statistic_HTTP_DATA,
			Action:     proto.Action_ALLOWED,
			Value:      int64(dpStatsHTTPDataValue),
		},
	},
	HttpData: []*proto.HTTPData{
		{
			XForwardedFor: publicIP1Str,
		},
	},
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
	ProtoInfo:        nfnetlink.CtProtoInfo{State: nfnl.TCP_CONNTRACK_ESTABLISHED},
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
	ProtoInfo:        nfnetlink.CtProtoInfo{State: nfnl.TCP_CONNTRACK_ESTABLISHED},
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
	ProtoInfo:        nfnetlink.CtProtoInfo{State: nfnl.TCP_CONNTRACK_ESTABLISHED},
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
	ProtoInfo:        nfnetlink.CtProtoInfo{State: nfnl.TCP_CONNTRACK_ESTABLISHED},
}

var _ = Describe("Conntrack Datasource", func() {
	var c *collector
	conf := &Config{
		StatsDumpFilePath:            "/tmp/qwerty",
		NfNetlinkBufSize:             65535,
		IngressGroup:                 1200,
		EgressGroup:                  2200,
		AgeTimeout:                   time.Duration(10) * time.Second,
		ConntrackPollingInterval:     time.Duration(1) * time.Second,
		InitialReportingDelay:        time.Duration(5) * time.Second,
		ExportingInterval:            time.Duration(1) * time.Second,
		MaxOriginalSourceIPsIncluded: 5,
	}
	rm := NewReporterManager()
	BeforeEach(func() {
		epMap := map[[16]byte]*calc.EndpointData{
			localIp1:  localEd1,
			localIp2:  localEd2,
			remoteIp1: remoteEd1,
		}

		nflogMap := map[[64]byte]*calc.RuleID{}

		for _, rid := range []*calc.RuleID{defTierPolicy1AllowEgressRuleID, defTierPolicy1AllowIngressRuleID, defTierPolicy2DenyIngressRuleID} {
			nflogMap[policyIDStrToRuleIDParts(rid)] = rid
		}

		lm := newMockLookupsCache(epMap, nflogMap, nil)
		c = newCollector(lm, rm, conf).(*collector)
	})
	Describe("Test local destination", func() {
		It("should create a single entry in inbound direction", func() {
			t := NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort)
			c.handleCtEntry(inCtEntry)
			Expect(c.epStats).Should(HaveKey(*t))
			data := c.epStats[*t]
			Expect(data.ConntrackPacketsCounter()).Should(Equal(*NewCounter(inCtEntry.OriginalCounters.Packets)))
			Expect(data.ConntrackPacketsCounterReverse()).Should(Equal(*NewCounter(inCtEntry.ReplyCounters.Packets)))
			Expect(data.ConntrackBytesCounter()).Should(Equal(*NewCounter(inCtEntry.OriginalCounters.Bytes)))
			Expect(data.ConntrackBytesCounterReverse()).Should(Equal(*NewCounter(inCtEntry.ReplyCounters.Bytes)))
		})
	})
	Describe("Test local source", func() {
		It("should create a single entry with outbound direction", func() {
			t := NewTuple(localIp1, remoteIp1, proto_tcp, srcPort, dstPort)
			c.handleCtEntry(outCtEntry)
			Expect(c.epStats).Should(HaveKey(*t))
			data := c.epStats[*t]
			Expect(data.ConntrackPacketsCounter()).Should(Equal(*NewCounter(outCtEntry.OriginalCounters.Packets)))
			Expect(data.ConntrackPacketsCounterReverse()).Should(Equal(*NewCounter(outCtEntry.ReplyCounters.Packets)))
			Expect(data.ConntrackBytesCounter()).Should(Equal(*NewCounter(outCtEntry.OriginalCounters.Bytes)))
			Expect(data.ConntrackBytesCounterReverse()).Should(Equal(*NewCounter(outCtEntry.ReplyCounters.Bytes)))
		})
	})
	Describe("Test local source to local destination", func() {
		It("should create a single entry with 'local' direction", func() {
			t1 := NewTuple(localIp1, localIp2, proto_tcp, srcPort, dstPort)
			c.handleCtEntry(localCtEntry)
			Expect(c.epStats).Should(HaveKey(Equal(*t1)))
			data := c.epStats[*t1]
			Expect(data.ConntrackPacketsCounter()).Should(Equal(*NewCounter(localCtEntry.OriginalCounters.Packets)))
			Expect(data.ConntrackPacketsCounterReverse()).Should(Equal(*NewCounter(localCtEntry.ReplyCounters.Packets)))
			Expect(data.ConntrackBytesCounter()).Should(Equal(*NewCounter(localCtEntry.OriginalCounters.Bytes)))
			Expect(data.ConntrackBytesCounterReverse()).Should(Equal(*NewCounter(localCtEntry.ReplyCounters.Bytes)))
		})
	})
	Describe("Test local destination with DNAT", func() {
		It("should create a single entry with inbound connection direction and with correct tuple extracted", func() {
			t := NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort)
			c.handleCtEntry(inCtEntryWithDNAT)
			Expect(c.epStats).Should(HaveKey(Equal(*t)))
			data := c.epStats[*t]
			Expect(data.ConntrackPacketsCounter()).Should(Equal(*NewCounter(inCtEntryWithDNAT.OriginalCounters.Packets)))
			Expect(data.ConntrackPacketsCounterReverse()).Should(Equal(*NewCounter(inCtEntryWithDNAT.ReplyCounters.Packets)))
			Expect(data.ConntrackBytesCounter()).Should(Equal(*NewCounter(inCtEntryWithDNAT.OriginalCounters.Bytes)))
			Expect(data.ConntrackBytesCounterReverse()).Should(Equal(*NewCounter(inCtEntryWithDNAT.ReplyCounters.Bytes)))
		})
	})
	Describe("Test local source to local destination with DNAT", func() {
		It("should create a single entry with 'local' connection direction and with correct tuple extracted", func() {
			t1 := NewTuple(localIp1, localIp2, proto_tcp, srcPort, dstPort)
			c.handleCtEntry(localCtEntryWithDNAT)
			Expect(c.epStats).Should(HaveKey(Equal(*t1)))
			data := c.epStats[*t1]
			Expect(data.ConntrackPacketsCounter()).Should(Equal(*NewCounter(localCtEntryWithDNAT.OriginalCounters.Packets)))
			Expect(data.ConntrackPacketsCounterReverse()).Should(Equal(*NewCounter(localCtEntryWithDNAT.ReplyCounters.Packets)))
			Expect(data.ConntrackBytesCounter()).Should(Equal(*NewCounter(localCtEntryWithDNAT.OriginalCounters.Bytes)))
			Expect(data.ConntrackBytesCounterReverse()).Should(Equal(*NewCounter(localCtEntryWithDNAT.ReplyCounters.Bytes)))
		})
	})
	Describe("Test conntrack TCP Protoinfo State", func() {
		It("Handle TCP conntrack entries with TCP state TIME_WAIT", func() {
			By("handling a conntrack update to start tracking stats for tuple")
			t := NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort)
			c.handleCtEntry(inCtEntry)
			Expect(c.epStats).Should(HaveKey(*t))
			data := c.epStats[*t]
			Expect(data.ConntrackPacketsCounter()).Should(Equal(*NewCounter(inCtEntry.OriginalCounters.Packets)))
			Expect(data.ConntrackPacketsCounterReverse()).Should(Equal(*NewCounter(inCtEntry.ReplyCounters.Packets)))
			Expect(data.ConntrackBytesCounter()).Should(Equal(*NewCounter(inCtEntry.OriginalCounters.Bytes)))
			Expect(data.ConntrackBytesCounterReverse()).Should(Equal(*NewCounter(inCtEntry.ReplyCounters.Bytes)))

			By("handling a conntrack update with updated counters")
			inCtEntryUpdatedCounters := inCtEntry
			inCtEntryUpdatedCounters.OriginalCounters.Packets = inCtEntry.OriginalCounters.Packets + 1
			inCtEntryUpdatedCounters.OriginalCounters.Bytes = inCtEntry.OriginalCounters.Bytes + 10
			inCtEntryUpdatedCounters.ReplyCounters.Packets = inCtEntry.ReplyCounters.Packets + 2
			inCtEntryUpdatedCounters.ReplyCounters.Bytes = inCtEntry.ReplyCounters.Bytes + 50
			c.handleCtEntry(inCtEntryUpdatedCounters)
			Expect(c.epStats).Should(HaveKey(*t))
			data = c.epStats[*t]
			Expect(data.ConntrackPacketsCounter()).Should(Equal(*NewCounter(inCtEntryUpdatedCounters.OriginalCounters.Packets)))
			Expect(data.ConntrackPacketsCounterReverse()).Should(Equal(*NewCounter(inCtEntryUpdatedCounters.ReplyCounters.Packets)))
			Expect(data.ConntrackBytesCounter()).Should(Equal(*NewCounter(inCtEntryUpdatedCounters.OriginalCounters.Bytes)))
			Expect(data.ConntrackBytesCounterReverse()).Should(Equal(*NewCounter(inCtEntryUpdatedCounters.ReplyCounters.Bytes)))

			By("handling a conntrack update with TCP CLOSE_WAIT")
			inCtEntryStateCloseWait := inCtEntryUpdatedCounters
			inCtEntryStateCloseWait.ProtoInfo.State = nfnl.TCP_CONNTRACK_CLOSE_WAIT
			inCtEntryStateCloseWait.ReplyCounters.Packets = inCtEntryUpdatedCounters.ReplyCounters.Packets + 1
			inCtEntryStateCloseWait.ReplyCounters.Bytes = inCtEntryUpdatedCounters.ReplyCounters.Bytes + 10
			c.handleCtEntry(inCtEntryStateCloseWait)
			Expect(c.epStats).Should(HaveKey(*t))
			data = c.epStats[*t]
			Expect(data.ConntrackPacketsCounter()).Should(Equal(*NewCounter(inCtEntryStateCloseWait.OriginalCounters.Packets)))
			Expect(data.ConntrackPacketsCounterReverse()).Should(Equal(*NewCounter(inCtEntryStateCloseWait.ReplyCounters.Packets)))
			Expect(data.ConntrackBytesCounter()).Should(Equal(*NewCounter(inCtEntryStateCloseWait.OriginalCounters.Bytes)))
			Expect(data.ConntrackBytesCounterReverse()).Should(Equal(*NewCounter(inCtEntryStateCloseWait.ReplyCounters.Bytes)))

			By("handling a conntrack update with TCP TIME_WAIT")
			inCtEntryStateTimeWait := inCtEntry
			inCtEntryStateTimeWait.ProtoInfo.State = nfnl.TCP_CONNTRACK_TIME_WAIT
			c.handleCtEntry(inCtEntryStateTimeWait)
			Expect(c.epStats).ShouldNot(HaveKey(*t))
		})
	})
	Describe("Test local destination combined with ALP stats", func() {
		It("should create a single entry in inbound direction", func() {
			By("Sending a conntrack update and a dataplane stats update and checking for combined values")
			t := NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort)
			c.convertDataplaneStatsAndApplyUpdate(&inALPEntry)
			c.handleCtEntry(inCtEntry)
			Expect(c.epStats).Should(HaveKey(*t))
			data := c.epStats[*t]
			Expect(data.ConntrackPacketsCounter()).Should(Equal(*NewCounter(inCtEntry.OriginalCounters.Packets)))
			Expect(data.ConntrackPacketsCounterReverse()).Should(Equal(*NewCounter(inCtEntry.ReplyCounters.Packets)))
			Expect(data.ConntrackBytesCounter()).Should(Equal(*NewCounter(inCtEntry.OriginalCounters.Bytes)))
			Expect(data.ConntrackBytesCounterReverse()).Should(Equal(*NewCounter(inCtEntry.ReplyCounters.Bytes)))
			Expect(data.HTTPRequestsAllowed()).Should(Equal(*NewCounter(alpEntryHTTPReqAllowed)))
			Expect(data.HTTPRequestsDenied()).Should(Equal(*NewCounter(alpEntryHTTPReqDenied)))

			By("Sending in another dataplane stats update and check for incremented counter")
			c.convertDataplaneStatsAndApplyUpdate(&inALPEntry)
			Expect(data.HTTPRequestsAllowed()).Should(Equal(*NewCounter(2 * alpEntryHTTPReqAllowed)))
			Expect(data.HTTPRequestsDenied()).Should(Equal(*NewCounter(2 * alpEntryHTTPReqDenied)))
		})
	})
	Describe("Test DataplaneStat with HTTPData", func() {
		It("should process DataplaneStat update with X-Forwarded-For HTTP Data", func() {
			By("Sending a conntrack update and a dataplane stats update and checking for combined values")
			t := NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort)
			expectedOrigSourceIPs := []net2.IP{net2.ParseIP(publicIP1Str)}
			c.convertDataplaneStatsAndApplyUpdate(&dpStatsEntryWithFwdFor)
			c.handleCtEntry(inCtEntry)
			Expect(c.epStats).Should(HaveKey(*t))
			data := c.epStats[*t]
			Expect(data.ConntrackPacketsCounter()).Should(Equal(*NewCounter(inCtEntry.OriginalCounters.Packets)))
			Expect(data.ConntrackPacketsCounterReverse()).Should(Equal(*NewCounter(inCtEntry.ReplyCounters.Packets)))
			Expect(data.ConntrackBytesCounter()).Should(Equal(*NewCounter(inCtEntry.OriginalCounters.Bytes)))
			Expect(data.ConntrackBytesCounterReverse()).Should(Equal(*NewCounter(inCtEntry.ReplyCounters.Bytes)))
			Expect(data.OriginalSourceIps()).Should(ConsistOf(expectedOrigSourceIPs))
			Expect(data.NumUniqueOriginalSourceIPs()).Should(Equal(dpStatsHTTPDataValue))

			By("Sending in another dataplane stats update and check for updated tracked data")
			updatedDpStatsEntryWithFwdFor := dpStatsEntryWithFwdFor
			updatedDpStatsEntryWithFwdFor.HttpData = []*proto.HTTPData{
				{
					XForwardedFor: publicIP1Str,
				},
				{
					XForwardedFor: publicIP2Str,
				},
			}
			expectedOrigSourceIPs = []net2.IP{net2.ParseIP(publicIP1Str), net2.ParseIP(publicIP2Str)}
			c.convertDataplaneStatsAndApplyUpdate(&updatedDpStatsEntryWithFwdFor)
			Expect(data.OriginalSourceIps()).Should(ConsistOf(expectedOrigSourceIPs))
			Expect(data.NumUniqueOriginalSourceIPs()).Should(Equal(2*dpStatsHTTPDataValue - 1))
		})
		It("should process DataplaneStat update with X-Real-IP HTTP Data", func() {
			By("Sending a conntrack update and a dataplane stats update and checking for combined values")
			t := NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort)
			expectedOrigSourceIPs := []net2.IP{net2.ParseIP(publicIP1Str)}
			dpStatsEntryWithRealIP := dpStatsEntryWithFwdFor
			dpStatsEntryWithRealIP.HttpData = []*proto.HTTPData{
				{
					XRealIp: publicIP1Str,
				},
			}
			c.convertDataplaneStatsAndApplyUpdate(&dpStatsEntryWithRealIP)
			c.handleCtEntry(inCtEntry)
			Expect(c.epStats).Should(HaveKey(*t))
			data := c.epStats[*t]
			Expect(data.ConntrackPacketsCounter()).Should(Equal(*NewCounter(inCtEntry.OriginalCounters.Packets)))
			Expect(data.ConntrackPacketsCounterReverse()).Should(Equal(*NewCounter(inCtEntry.ReplyCounters.Packets)))
			Expect(data.ConntrackBytesCounter()).Should(Equal(*NewCounter(inCtEntry.OriginalCounters.Bytes)))
			Expect(data.ConntrackBytesCounterReverse()).Should(Equal(*NewCounter(inCtEntry.ReplyCounters.Bytes)))
			Expect(data.OriginalSourceIps()).Should(ConsistOf(expectedOrigSourceIPs))
			Expect(data.NumUniqueOriginalSourceIPs()).Should(Equal(dpStatsHTTPDataValue))

			By("Sending a dataplane stats update with x-real-ip and check for updated tracked data")
			updatedDpStatsEntryWithRealIP := dpStatsEntryWithRealIP
			updatedDpStatsEntryWithRealIP.HttpData = []*proto.HTTPData{
				{
					XRealIp: publicIP1Str,
				},
				{
					XRealIp: publicIP2Str,
				},
			}
			expectedOrigSourceIPs = []net2.IP{net2.ParseIP(publicIP1Str), net2.ParseIP(publicIP2Str)}
			c.convertDataplaneStatsAndApplyUpdate(&updatedDpStatsEntryWithRealIP)
			Expect(data.OriginalSourceIps()).Should(ConsistOf(expectedOrigSourceIPs))
			Expect(data.NumUniqueOriginalSourceIPs()).Should(Equal(2*dpStatsHTTPDataValue - 1))
		})
		It("should process DataplaneStat update with X-Real-IP and X-Forwarded-For HTTP Data", func() {
			By("Sending a conntrack update and a dataplane stats update and checking for combined values")
			t := NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort)
			expectedOrigSourceIPs := []net2.IP{net2.ParseIP(publicIP1Str)}
			c.convertDataplaneStatsAndApplyUpdate(&dpStatsEntryWithFwdFor)
			c.handleCtEntry(inCtEntry)
			Expect(c.epStats).Should(HaveKey(*t))
			data := c.epStats[*t]
			Expect(data.ConntrackPacketsCounter()).Should(Equal(*NewCounter(inCtEntry.OriginalCounters.Packets)))
			Expect(data.ConntrackPacketsCounterReverse()).Should(Equal(*NewCounter(inCtEntry.ReplyCounters.Packets)))
			Expect(data.ConntrackBytesCounter()).Should(Equal(*NewCounter(inCtEntry.OriginalCounters.Bytes)))
			Expect(data.ConntrackBytesCounterReverse()).Should(Equal(*NewCounter(inCtEntry.ReplyCounters.Bytes)))
			Expect(data.OriginalSourceIps()).Should(ConsistOf(expectedOrigSourceIPs))
			Expect(data.NumUniqueOriginalSourceIPs()).Should(Equal(dpStatsHTTPDataValue))

			By("Sending in another dataplane stats update and check for updated tracked data")
			updatedDpStatsEntryWithFwdForAndRealIP := dpStatsEntryWithFwdFor
			updatedDpStatsEntryWithFwdForAndRealIP.HttpData = []*proto.HTTPData{
				{
					XForwardedFor: publicIP1Str,
					XRealIp:       publicIP1Str,
				},
				{
					XRealIp: publicIP2Str,
				},
			}
			expectedOrigSourceIPs = []net2.IP{net2.ParseIP(publicIP1Str), net2.ParseIP(publicIP2Str)}
			c.convertDataplaneStatsAndApplyUpdate(&updatedDpStatsEntryWithFwdForAndRealIP)
			Expect(data.OriginalSourceIps()).Should(ConsistOf(expectedOrigSourceIPs))
			// We subtract 1 because the second update contains an overlapping IP that is accounted for.
			Expect(data.NumUniqueOriginalSourceIPs()).Should(Equal(2*dpStatsHTTPDataValue - 1))
		})
	})
})

func policyIDStrToRuleIDParts(r *calc.RuleID) [64]byte {
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
	var c *collector
	const (
		ageTimeout        = time.Duration(3) * time.Second
		reportingDelay    = time.Duration(2) * time.Second
		exportingInterval = time.Duration(1) * time.Second
		pollingInterval   = time.Duration(1) * time.Second
	)
	conf := &Config{
		StatsDumpFilePath:            "/tmp/qwerty",
		NfNetlinkBufSize:             65535,
		IngressGroup:                 1200,
		EgressGroup:                  2200,
		AgeTimeout:                   ageTimeout,
		ConntrackPollingInterval:     pollingInterval,
		InitialReportingDelay:        reportingDelay,
		ExportingInterval:            exportingInterval,
		MaxOriginalSourceIPsIncluded: 5,
	}
	rm := NewReporterManager()
	mockReporter := newMockReporter()
	rm.RegisterMetricsReporter(mockReporter)
	BeforeEach(func() {
		epMap := map[[16]byte]*calc.EndpointData{
			localIp1:  localEd1,
			localIp2:  localEd2,
			remoteIp1: remoteEd1,
		}

		nflogMap := map[[64]byte]*calc.RuleID{}

		for _, rid := range []*calc.RuleID{defTierPolicy1AllowEgressRuleID, defTierPolicy1AllowIngressRuleID, defTierPolicy2DenyIngressRuleID} {
			nflogMap[policyIDStrToRuleIDParts(rid)] = rid
		}

		lm := newMockLookupsCache(epMap, nflogMap, nil)
		rm.Start()
		c = newCollector(lm, rm, conf).(*collector)
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
					srcEp:        remoteEd1,
					dstEp:        localEd1,
					ruleIDs:      []*calc.RuleID{defTierPolicy2DenyIngressRuleID},
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
					srcEp:        remoteEd1,
					dstEp:        localEd1,
					ruleIDs:      []*calc.RuleID{defTierPolicy1AllowIngressRuleID},
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
					srcEp:        remoteEd1,
					dstEp:        localEd1,
					ruleIDs:      []*calc.RuleID{defTierPolicy1AllowIngressRuleID},
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
					srcEp:        localEd1,
					dstEp:        remoteEd1,
					ruleIDs:      []*calc.RuleID{defTierPolicy1AllowEgressRuleID},
					isConnection: false,
				}
				Eventually(mockReporter.reportChan, reportingDelay*2).Should(Receive(Equal(tmu)))
			})
		})
	})
})

type mockDNSReporter struct {
	updates []DNSUpdate
}

func (c *mockDNSReporter) Start() {}

func (c *mockDNSReporter) Log(update DNSUpdate) error {
	c.updates = append(c.updates, update)
	return nil
}

var _ = Describe("DNS logging", func() {
	var c *collector
	var r *mockDNSReporter
	BeforeEach(func() {
		epMap := map[[16]byte]*calc.EndpointData{
			localIp1:  localEd1,
			localIp2:  localEd2,
			remoteIp1: remoteEd1,
		}
		nflogMap := map[[64]byte]*calc.RuleID{}
		lm := newMockLookupsCache(epMap, nflogMap, map[model.NetworkSetKey]*model.NetworkSet{netSetKey1: &netSet1})
		c = newCollector(lm, nil, &Config{
			AgeTimeout:               time.Duration(10) * time.Second,
			ConntrackPollingInterval: time.Duration(1) * time.Second,
			InitialReportingDelay:    time.Duration(5) * time.Second,
			ExportingInterval:        time.Duration(1) * time.Second,
		}).(*collector)
		r = &mockDNSReporter{}
		c.SetDNSLogReporter(r)
	})
	It("should get client and server endpoint data", func() {
		c.LogDNS(net2.ParseIP(netSetIp1Str), net2.ParseIP(localIp1Str), nil)
		Expect(r.updates).To(HaveLen(1))
		update := r.updates[0]
		Expect(update.ClientEP).NotTo(BeNil())
		Expect(update.ClientEP.Endpoint).To(BeAssignableToTypeOf(&model.WorkloadEndpoint{}))
		Expect(*(update.ClientEP.Endpoint.(*model.WorkloadEndpoint))).To(Equal(*localWlEp1))
		Expect(update.ServerEP).NotTo(BeNil())
		Expect(update.ServerEP.Networkset).To(BeAssignableToTypeOf(&model.NetworkSet{}))
		Expect(*(update.ServerEP.Networkset.(*model.NetworkSet))).To(Equal(netSet1))
	})
})

func newMockLookupsCache(
	em map[[16]byte]*calc.EndpointData,
	nm map[[64]byte]*calc.RuleID,
	ns map[model.NetworkSetKey]*model.NetworkSet,
) *calc.LookupsCache {
	l := calc.NewLookupsCache()
	l.SetMockData(em, nm, ns)
	return l
}

// Define a separate metric type that doesn't include the actual stats.  We use this
// for simpler comparisons.
type testMetricUpdate struct {
	updateType UpdateType

	// Tuple key
	tuple Tuple

	// Endpoint information.
	srcEp *calc.EndpointData
	dstEp *calc.EndpointData

	// Rules identification
	ruleIDs []*calc.RuleID

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
		srcEp:        mu.srcEp,
		dstEp:        mu.dstEp,
		ruleIDs:      mu.ruleIDs,
		isConnection: mu.isConnection,
	}
	return nil
}

func mustParseIP(s string) net.IP {
	ip := net2.ParseIP(s)
	return net.IP{ip}
}

func mustParseMac(m string) *net.MAC {
	hwAddr, err := net2.ParseMAC(m)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse MAC: %v; %v", m, err))
	}
	return &net.MAC{hwAddr}
}

func mustParseNet(n string) net.IPNet {
	_, cidr, err := net.ParseCIDR(n)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse CIDR %v; %v", n, err))
	}
	return *cidr
}

func BenchmarkNflogPktToStat(b *testing.B) {
	epMap := map[[16]byte]*calc.EndpointData{
		localIp1:  localEd1,
		localIp2:  localEd2,
		remoteIp1: remoteEd1,
	}

	nflogMap := map[[64]byte]*calc.RuleID{}

	for _, rid := range []*calc.RuleID{defTierPolicy1AllowEgressRuleID, defTierPolicy1AllowIngressRuleID, defTierPolicy2DenyIngressRuleID} {
		nflogMap[policyIDStrToRuleIDParts(rid)] = rid
	}

	conf := &Config{
		StatsDumpFilePath:            "/tmp/qwerty",
		NfNetlinkBufSize:             65535,
		IngressGroup:                 1200,
		EgressGroup:                  2200,
		AgeTimeout:                   time.Duration(10) * time.Second,
		ConntrackPollingInterval:     time.Duration(1) * time.Second,
		InitialReportingDelay:        time.Duration(5) * time.Second,
		ExportingInterval:            time.Duration(1) * time.Second,
		MaxOriginalSourceIPsIncluded: 5,
	}
	rm := NewReporterManager()
	lm := newMockLookupsCache(epMap, nflogMap, nil)
	c := newCollector(lm, rm, conf).(*collector)
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		c.convertNflogPktAndApplyUpdate(rules.RuleDirIngress, ingressPktAllow)
	}
}

func BenchmarkApplyStatUpdate(b *testing.B) {
	epMap := map[[16]byte]*calc.EndpointData{
		localIp1:  localEd1,
		localIp2:  localEd2,
		remoteIp1: remoteEd1,
	}

	nflogMap := map[[64]byte]*calc.RuleID{}
	for _, rid := range []*calc.RuleID{defTierPolicy1AllowEgressRuleID, defTierPolicy1AllowIngressRuleID, defTierPolicy2DenyIngressRuleID} {
		nflogMap[policyIDStrToRuleIDParts(rid)] = rid
	}

	conf := &Config{
		StatsDumpFilePath:            "/tmp/qwerty",
		NfNetlinkBufSize:             65535,
		IngressGroup:                 1200,
		EgressGroup:                  2200,
		AgeTimeout:                   time.Duration(10) * time.Second,
		ConntrackPollingInterval:     time.Duration(1) * time.Second,
		InitialReportingDelay:        time.Duration(5) * time.Second,
		ExportingInterval:            time.Duration(1) * time.Second,
		MaxOriginalSourceIPsIncluded: 5,
	}
	rm := NewReporterManager()
	lm := newMockLookupsCache(epMap, nflogMap, nil)
	c := newCollector(lm, rm, conf).(*collector)
	var tuples []Tuple
	MaxSrcPort := 1000
	MaxDstPort := 1000
	for sp := 1; sp < MaxSrcPort; sp++ {
		for dp := 1; dp < MaxDstPort; dp++ {
			t := NewTuple(localIp1, localIp2, proto_tcp, sp, dp)
			tuples = append(tuples, *t)
		}
	}
	var rids []*calc.RuleID
	MaxEntries := 10000
	for i := 0; i < MaxEntries; i++ {
		rid := defTierPolicy1AllowIngressRuleID
		rids = append(rids, rid)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		for i := 0; i < MaxEntries; i++ {
			c.applyNflogStatUpdate(tuples[i], rids[i], localEd1, remoteEd1, 0, 1, 2)
		}
	}
}
