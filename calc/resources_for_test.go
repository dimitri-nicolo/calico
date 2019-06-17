// Copyright (c) 2017-2019 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package calc_test

// This file contains canned backend model values for use in tests.  Note the "." import of
// the model package.

import (
	"github.com/projectcalico/libcalico-go/lib/backend/encap"
	. "github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/net"
	"github.com/projectcalico/libcalico-go/lib/numorstring"
)

// Canned hostnames.
var (
	localHostname   = "localhostname"
	remoteHostname  = "remotehostname"
	remoteHostname2 = "remotehostname2"
)

// Canned selectors.

var (
	allSelector                 = "all()"
	allSelectorId               = selectorID(allSelector)
	allLessFoobarSelector       = "(all()) && !(foo == 'bar')"
	allLessFoobarSelectorId     = selectorID(allLessFoobarSelector)
	bEpBSelector                = "b == 'b'"
	bEqBSelectorId              = selectorID(bEpBSelector)
	tagSelector                 = "has(tag-1)"
	tagSelectorId               = selectorID(tagSelector)
	tagFoobarSelector           = "tag-1 == 'foobar'"
	tagFoobarSelectorId         = selectorID(tagFoobarSelector)
	namedPortAllTCPID           = namedPortID(allSelector, "tcp", "tcpport")
	namedPortAllLessFoobarTCPID = namedPortID(allLessFoobarSelector, "tcp", "tcpport")
	namedPortAllTCP2ID          = namedPortID(allSelector, "tcp", "tcpport2")
	namedPortAllUDPID           = namedPortID(allSelector, "udp", "udpport")
	inheritSelector             = "profile == 'prof-1'"
	namedPortInheritIPSetID     = namedPortID(inheritSelector, "tcp", "tcpport")
	httpMatchMethod             = HTTPMatch{Methods: []string{"GET"}}
	serviceAccountSelector      = "name == 'sa1'"
)

// Canned workload endpoints.

var localWlEpKey1 = WorkloadEndpointKey{localHostname, "orch", "wl1", "ep1"}
var remoteWlEpKey1 = WorkloadEndpointKey{remoteHostname, "orch", "wl1", "ep1"}
var remoteWlEpKey2 = WorkloadEndpointKey{remoteHostname2, "orch", "wl1", "ep1"}
var localWlEp1Id = "orch/wl1/ep1"
var localWlEpKey2 = WorkloadEndpointKey{localHostname, "orch", "wl2", "ep2"}
var localWlEp2Id = "orch/wl2/ep2"

var localWlEp1 = WorkloadEndpoint{
	State:      "active",
	Name:       "cali1",
	Mac:        mustParseMac("01:02:03:04:05:06"),
	ProfileIDs: []string{"prof-1", "prof-2", "prof-missing"},
	IPv4Nets: []net.IPNet{mustParseNet("10.0.0.1/32"),
		mustParseNet("10.0.0.2/32")},
	IPv6Nets: []net.IPNet{mustParseNet("fc00:fe11::1/128"),
		mustParseNet("fc00:fe11::2/128")},
	Labels: map[string]string{
		"id": "loc-ep-1",
		"a":  "a",
		"b":  "b",
	},
	Ports: []EndpointPort{
		{Name: "tcpport", Protocol: numorstring.ProtocolFromStringV1("tcp"), Port: 8080},
		{Name: "tcpport2", Protocol: numorstring.ProtocolFromStringV1("tcp"), Port: 1234},
		{Name: "udpport", Protocol: numorstring.ProtocolFromStringV1("udp"), Port: 9091},
	},
}

var localWlEp1WithLabelsButNoProfiles = WorkloadEndpoint{
	State: "active",
	Name:  "cali1",
	Mac:   mustParseMac("01:02:03:04:05:06"),
	IPv4Nets: []net.IPNet{mustParseNet("10.0.0.1/32"),
		mustParseNet("10.0.0.2/32")},
	IPv6Nets: []net.IPNet{mustParseNet("fc00:fe11::1/128"),
		mustParseNet("fc00:fe11::2/128")},
	Labels: map[string]string{
		"id": "loc-ep-1",
		"a":  "a",
		"b":  "b",
	},
	Ports: []EndpointPort{
		{Name: "tcpport", Protocol: numorstring.ProtocolFromStringV1("tcp"), Port: 8080},
		{Name: "tcpport2", Protocol: numorstring.ProtocolFromStringV1("tcp"), Port: 1234},
		{Name: "udpport", Protocol: numorstring.ProtocolFromStringV1("udp"), Port: 9091},
	},
}

var localWlEp1WithDupeNamedPorts = WorkloadEndpoint{
	State:      "active",
	Name:       "cali1",
	Mac:        mustParseMac("01:02:03:04:05:06"),
	ProfileIDs: []string{"prof-1", "prof-2", "prof-missing"},
	IPv4Nets: []net.IPNet{mustParseNet("10.0.0.1/32"),
		mustParseNet("10.0.0.2/32")},
	IPv6Nets: []net.IPNet{mustParseNet("fc00:fe11::1/128"),
		mustParseNet("fc00:fe11::2/128")},
	Labels: map[string]string{
		"id": "loc-ep-1",
		"a":  "a",
		"b":  "b",
	},
	Ports: []EndpointPort{
		{Name: "tcpport", Protocol: numorstring.ProtocolFromStringV1("tcp"), Port: 8080},
		{Name: "tcpport", Protocol: numorstring.ProtocolFromStringV1("tcp"), Port: 8081},
		{Name: "tcpport", Protocol: numorstring.ProtocolFromStringV1("tcp"), Port: 8082},
	},
}

var localWlEp1NoProfiles = WorkloadEndpoint{
	State: "active",
	Name:  "cali1",
	Mac:   mustParseMac("01:02:03:04:05:06"),
	IPv4Nets: []net.IPNet{mustParseNet("10.0.0.1/32"),
		mustParseNet("10.0.0.2/32")},
	IPv6Nets: []net.IPNet{mustParseNet("fc00:fe11::1/128"),
		mustParseNet("fc00:fe11::2/128")},
}

var localWlEp1DifferentIPs = WorkloadEndpoint{
	State:      "active",
	Name:       "cali1",
	Mac:        mustParseMac("01:02:03:04:05:06"),
	ProfileIDs: []string{"prof-1", "prof-2", "prof-missing"},
	IPv4Nets: []net.IPNet{mustParseNet("11.0.0.1/32"),
		mustParseNet("11.0.0.2/32")},
	IPv6Nets: []net.IPNet{mustParseNet("fc00:fe12::1/128"),
		mustParseNet("fc00:fe12::2/128")},
	Labels: map[string]string{
		"id": "loc-ep-1",
		"a":  "a",
		"b":  "b",
	},
}

var ep1IPs = []string{
	"10.0.0.1/32", // ep1
	"fc00:fe11::1/128",
	"10.0.0.2/32", // shared with ep2
	"fc00:fe11::2/128",
}

var localWlEp2 = WorkloadEndpoint{
	State:      "active",
	Name:       "cali2",
	ProfileIDs: []string{"prof-2", "prof-3"},
	IPv4Nets: []net.IPNet{mustParseNet("10.0.0.2/32"),
		mustParseNet("10.0.0.3/32")},
	IPv6Nets: []net.IPNet{mustParseNet("fc00:fe11::2/128"),
		mustParseNet("fc00:fe11::3/128")},
	Labels: map[string]string{
		"id": "loc-ep-2",
		"a":  "a",
		"b":  "b2",
	},
	Ports: []EndpointPort{
		{Name: "tcpport", Protocol: numorstring.ProtocolFromStringV1("tcp"), Port: 8080},
		{Name: "tcpport2", Protocol: numorstring.ProtocolFromStringV1("tcp"), Port: 2345},
		{Name: "udpport", Protocol: numorstring.ProtocolFromStringV1("udp"), Port: 9090},
	},
}

var localWlEp2WithLabelsButNoProfiles = WorkloadEndpoint{
	State: "active",
	Name:  "cali2",
	IPv4Nets: []net.IPNet{mustParseNet("10.0.0.2/32"),
		mustParseNet("10.0.0.3/32")},
	IPv6Nets: []net.IPNet{mustParseNet("fc00:fe11::2/128"),
		mustParseNet("fc00:fe11::3/128")},
	Labels: map[string]string{
		"id": "loc-ep-2",
		"a":  "a",
		"b":  "b2",
	},
	Ports: []EndpointPort{
		{Name: "tcpport", Protocol: numorstring.ProtocolFromStringV1("tcp"), Port: 8080},
		{Name: "tcpport2", Protocol: numorstring.ProtocolFromStringV1("tcp"), Port: 2345},
		{Name: "udpport", Protocol: numorstring.ProtocolFromStringV1("udp"), Port: 9090},
	},
}

var localWlEp2NoProfiles = WorkloadEndpoint{
	State: "active",
	Name:  "cali2",
	IPv4Nets: []net.IPNet{mustParseNet("10.0.0.2/32"),
		mustParseNet("10.0.0.3/32")},
	IPv6Nets: []net.IPNet{mustParseNet("fc00:fe11::2/128"),
		mustParseNet("fc00:fe11::3/128")},
}

var hostEpWithName = HostEndpoint{
	Name:       "eth1",
	ProfileIDs: []string{"prof-1", "prof-2", "prof-missing"},
	ExpectedIPv4Addrs: []net.IP{mustParseIP("10.0.0.1"),
		mustParseIP("10.0.0.2")},
	ExpectedIPv6Addrs: []net.IP{mustParseIP("fc00:fe11::1"),
		mustParseIP("fc00:fe11::2")},
	Labels: map[string]string{
		"id": "loc-ep-1",
		"a":  "a",
		"b":  "b",
	},
}

var hostEpWithNamedPorts = HostEndpoint{
	Name:       "eth1",
	ProfileIDs: []string{"prof-1"},
	ExpectedIPv4Addrs: []net.IP{mustParseIP("10.0.0.1"),
		mustParseIP("10.0.0.2")},
	ExpectedIPv6Addrs: []net.IP{mustParseIP("fc00:fe11::1"),
		mustParseIP("fc00:fe11::2")},
	Labels: map[string]string{
		"id": "loc-ep-1",
		"a":  "a",
		"b":  "b",
	},
	Ports: []EndpointPort{
		{Name: "tcpport", Protocol: numorstring.ProtocolFromStringV1("tcp"), Port: 8080},
		{Name: "tcpport2", Protocol: numorstring.ProtocolFromStringV1("tcp"), Port: 1234},
		{Name: "udpport", Protocol: numorstring.ProtocolFromStringV1("udp"), Port: 9091},
	},
}

var hostEpWithNameKey = HostEndpointKey{
	Hostname:   localHostname,
	EndpointID: "named",
}
var hostEpWithNameId = "named"

var hostEp2NoName = HostEndpoint{
	ProfileIDs: []string{"prof-2", "prof-3"},
	ExpectedIPv4Addrs: []net.IP{mustParseIP("10.0.0.2"),
		mustParseIP("10.0.0.3")},
	ExpectedIPv6Addrs: []net.IP{mustParseIP("fc00:fe11::2"),
		mustParseIP("fc00:fe11::3")},
	Labels: map[string]string{
		"id": "loc-ep-2",
		"a":  "a",
		"b":  "b2",
	},
}

var hostEp2NoNameKey = HostEndpointKey{
	Hostname:   localHostname,
	EndpointID: "unnamed",
}
var hostEpNoNameId = "unnamed"

// Canned tiers/policies.

var order10 = float64(10)
var order20 = float64(20)
var order30 = float64(30)

var policy1_order20 = Policy{
	Order:    &order20,
	Selector: "a == 'a'",
	InboundRules: []Rule{
		{SrcSelector: allSelector},
	},
	OutboundRules: []Rule{
		{SrcSelector: bEpBSelector},
	},
	Types: []string{"ingress", "egress"},
}

var protoTCP = numorstring.ProtocolFromStringV1("tcp")
var protoUDP = numorstring.ProtocolFromStringV1("udp")
var policy1_order20_with_named_port_tcpport = Policy{
	Order:    &order20,
	Selector: "a == 'a'",
	InboundRules: []Rule{
		{
			Protocol: &protoTCP,
			SrcPorts: []numorstring.Port{numorstring.NamedPort("tcpport")},
		},
	},
	OutboundRules: []Rule{
		{SrcSelector: bEpBSelector},
	},
	Types: []string{"ingress", "egress"},
}

var policy1_order20_with_named_port_tcpport_negated = Policy{
	Order:    &order20,
	Selector: "a == 'a'",
	InboundRules: []Rule{
		{
			Protocol:    &protoTCP,
			NotSrcPorts: []numorstring.Port{numorstring.NamedPort("tcpport")},
		},
	},
	OutboundRules: []Rule{
		{SrcSelector: bEpBSelector},
	},
	Types: []string{"ingress", "egress"},
}

var policy_with_named_port_inherit = Policy{
	Order:    &order20,
	Selector: "a == 'a'",
	InboundRules: []Rule{
		{
			Protocol:    &protoTCP,
			SrcSelector: "profile == 'prof-1'",
			SrcPorts:    []numorstring.Port{numorstring.NamedPort("tcpport")},
		},
	},
	OutboundRules: []Rule{},
	Types:         []string{"ingress", "egress"},
}

var policy1_order20_with_selector_and_named_port_tcpport = Policy{
	Order:    &order20,
	Selector: "a == 'a'",
	InboundRules: []Rule{
		{
			Protocol:    &protoTCP,
			SrcSelector: allSelector,
			SrcPorts:    []numorstring.Port{numorstring.NamedPort("tcpport")},
		},
	},
	OutboundRules: []Rule{
		{SrcSelector: bEpBSelector},
	},
	Types: []string{"ingress", "egress"},
}

var policy1_order20_with_selector_and_negated_named_port_tcpport = Policy{
	Order:    &order20,
	Selector: "a == 'a'",
	InboundRules: []Rule{
		{
			Protocol:       &protoTCP,
			SrcSelector:    allSelector,
			NotSrcSelector: "foo == 'bar'",
			NotSrcPorts:    []numorstring.Port{numorstring.NamedPort("tcpport")},
		},
	},
	Types: []string{"ingress"},
}

var policy1_order20_with_selector_and_negated_named_port_tcpport_dest = Policy{
	Order:    &order20,
	Selector: "a == 'a'",
	InboundRules: []Rule{
		{
			Protocol:       &protoTCP,
			DstSelector:    allSelector,
			NotDstSelector: "foo == 'bar'",
			NotDstPorts:    []numorstring.Port{numorstring.NamedPort("tcpport")},
		},
	},
	Types: []string{"ingress"},
}

var policy1_order20_with_selector_and_named_port_udpport = Policy{
	Order:    &order20,
	Selector: "a == 'a'",
	InboundRules: []Rule{
		{
			Protocol:    &protoUDP,
			SrcSelector: allSelector,
			SrcPorts:    []numorstring.Port{numorstring.NamedPort("udpport")},
		},
	},
	OutboundRules: []Rule{
		{SrcSelector: bEpBSelector},
	},
	Types: []string{"ingress", "egress"},
}

var policy1_order20_with_named_port_mismatched_protocol = Policy{
	Order:    &order20,
	Selector: "a == 'a'",
	InboundRules: []Rule{
		{
			Protocol: &protoTCP,
			SrcPorts: []numorstring.Port{numorstring.NamedPort("udpport")},
		},
	},
	OutboundRules: []Rule{
		{
			Protocol: &protoUDP,
			SrcPorts: []numorstring.Port{numorstring.NamedPort("tcpport")},
		},
	},
	Types: []string{"ingress", "egress"},
}

var policy1_order20_with_selector_and_named_port_tcpport2 = Policy{
	Order:    &order20,
	Selector: "a == 'a'",
	InboundRules: []Rule{
		{SrcSelector: bEpBSelector},
	},
	OutboundRules: []Rule{
		{
			Protocol:    &protoTCP,
			SrcSelector: allSelector,
			SrcPorts:    []numorstring.Port{numorstring.NamedPort("tcpport2")},
		},
	},
	Types: []string{"ingress", "egress"},
}

var policy1_order20_ingress_only = Policy{
	Order:    &order20,
	Selector: "a == 'a'",
	InboundRules: []Rule{
		{SrcSelector: allSelector},
	},
	Types: []string{"ingress"},
}

var policy1_order20_egress_only = Policy{
	Order:    &order20,
	Selector: "a == 'a'",
	OutboundRules: []Rule{
		{SrcSelector: bEpBSelector},
	},
	Types: []string{"egress"},
}

var policy1_order20_untracked = Policy{
	Order:    &order20,
	Selector: "a == 'a'",
	InboundRules: []Rule{
		{SrcSelector: allSelector},
	},
	OutboundRules: []Rule{
		{SrcSelector: bEpBSelector},
	},
	DoNotTrack: true,
}

var policy1_order20_pre_dnat = Policy{
	Order:    &order20,
	Selector: "a == 'a'",
	InboundRules: []Rule{
		{SrcSelector: allSelector},
	},
	PreDNAT: true,
}

var policy1_order20_http_match = Policy{
	Order:    &order20,
	Selector: "a == 'a'",
	InboundRules: []Rule{
		{HTTPMatch: &httpMatchMethod},
	},
}

var policy1_order20_src_service_account = Policy{
	Order:    &order20,
	Selector: "a == 'a'",
	InboundRules: []Rule{
		{OriginalSrcServiceAccountSelector: serviceAccountSelector},
	},
}

var policy1_order20_dst_service_account = Policy{
	Order:    &order20,
	Selector: "a == 'a'",
	OutboundRules: []Rule{
		{OriginalDstServiceAccountSelector: serviceAccountSelector},
	},
}

var profileRules1 = ProfileRules{
	InboundRules: []Rule{
		{SrcSelector: allSelector},
	},
	OutboundRules: []Rule{
		{SrcTag: "tag-1"},
	},
}

var profileRulesWithTagInherit = ProfileRules{
	InboundRules: []Rule{
		{SrcSelector: tagSelector},
	},
	OutboundRules: []Rule{
		{SrcSelector: tagFoobarSelector},
	},
}

var profileRules1TagUpdate = ProfileRules{
	InboundRules: []Rule{
		{SrcSelector: bEpBSelector},
	},
	OutboundRules: []Rule{
		{SrcTag: "tag-2"},
	},
}

var profileRules1NegatedTagSelUpdate = ProfileRules{
	InboundRules: []Rule{
		{NotSrcSelector: bEpBSelector},
	},
	OutboundRules: []Rule{
		{NotSrcTag: "tag-2"},
	},
}

var profileTags1 = []string{"tag-1"}
var profileLabels1 = map[string]string{
	"profile": "prof-1",
}
var profileLabels2 = map[string]string{
	"profile": "prof-2",
}
var profileLabelsTag1 = map[string]string{
	"tag-1": "foobar",
}

var tag1LabelID = ipSetIDForTag("tag-1")
var tag2LabelID = ipSetIDForTag("tag-2")

var netSet1Key = NetworkSetKey{Name: "netset-1"}
var netSet1 = NetworkSet{
	Nets: []net.IPNet{
		mustParseNet("12.0.0.0/24"),
		mustParseNet("12.0.0.0/24"), // A dupe, why not!
		mustParseNet("12.1.0.0/24"),
		mustParseNet("10.0.0.1/32"), // Overlaps with host endpoint.
		mustParseNet("feed:beef::/32"),
		mustParseNet("feed:beef:0::/32"), // Non-canonical dupe.
	},
	Labels: map[string]string{
		"a": "b",
	},
}
var netSet1WithBEqB = NetworkSet{
	Nets: []net.IPNet{
		mustParseNet("12.0.0.0/24"),
		mustParseNet("12.0.0.0/24"), // A dupe, why not!
		mustParseNet("12.1.0.0/24"),
		mustParseNet("10.0.0.1/32"), // Overlaps with host endpoint.
	},
	Labels: map[string]string{
		"foo": "bar",
		"b":   "b",
	},
}

var netSet2Key = NetworkSetKey{Name: "netset-2"}
var netSet2 = NetworkSet{
	Nets: []net.IPNet{
		mustParseNet("12.0.0.0/24"), // Overlaps with netset-1
		mustParseNet("13.1.0.0/24"),
	},
	Labels: map[string]string{
		"a": "b",
	},
}

// Resources for DNS Policy unit testing. We start with a basic default-deny-egress policy that only allows UDP:53 out.
var policyDNSBasic = Policy{
	Selector: "name == 'dnspolicy'",
	Order:    &order30,
	Types:    []string{"egress"},
	OutboundRules: []Rule{
		{
			Action:      "allow",
			SrcSelector: allSelector,
			Protocol:    &protoUDP,
			DstPorts:    []numorstring.Port{numorstring.SinglePort(53)},
		},
		{
			Action: "deny",
		},
	},
}

// Two GlobalNetworkSets, one for microsoft.com and one for google.com.
var allowedEgressDomains = []string{"microsoft.com", "www.microsoft.com"}
var allowedEgressDomains2 = []string{"google.com", "www.google.com"}

var netSetDNSKey = NetworkSetKey{Name: "netset-domains"}
var netSetDNSKey2 = NetworkSetKey{Name: "netset-domains-2"}

var netSetDNS = NetworkSet{
	AllowedEgressDomains: allowedEgressDomains,
	Labels:               map[string]string{"external-service-name": "microsoft"},
}

var netSetDNS2 = NetworkSet{
	AllowedEgressDomains: allowedEgressDomains2,
	Labels:               map[string]string{"external-service-name": "google"},
}

// Two GlobalNetworkPolicies, the first allows external access to microsoft.com and the second to any resource that
// specifies a "external-service-name" label.
var dstSelectorDNSExternal = "external-service-name == 'microsoft'"
var dstSelectorDNSExternal2 = "has(external-service-name)"

var selectorIdDNSExternal = domainSelectorID(dstSelectorDNSExternal, allowedEgressDomains)
var selectorIdDNSExternal2 = domainSelectorID(dstSelectorDNSExternal2, allowedEgressDomains2)
var selectorIdDNSExternal3 = domainSelectorID("", allowedEgressDomains)

var selectorIdDNSEmpty = selectorID(dstSelectorDNSExternal)
var selectorIdDNSEmpty2 = selectorID(dstSelectorDNSExternal2)

var policyDNSExternal = Policy{
	Selector: allSelector,
	Order:    &order20,
	Types:    []string{"egress"},
	OutboundRules: []Rule{
		{
			Action:      "allow",
			DstSelector: dstSelectorDNSExternal,
		},
	},
}

var policyDNSExternal2 = Policy{
	Selector: allSelector,
	Order:    &order20,
	Types:    []string{"egress"},
	OutboundRules: []Rule{
		{
			Action:      "allow",
			DstSelector: dstSelectorDNSExternal2,
		},
	},
}

var policyDNSExternal3 = Policy{
	Selector: allSelector,
	Order:    &order20,
	Types:    []string{"egress"},
	OutboundRules: []Rule{
		{
			Action:     "allow",
			DstDomains: allowedEgressDomains,
		},
	},
}

// One simple workload endpoint with v4 and v6 addresses, with a label to match on.
var localWlEpDNS = WorkloadEndpoint{
	State: "active",
	Name:  "cali1",
	Mac:   mustParseMac("01:02:03:04:05:06"),
	IPv4Nets: []net.IPNet{mustParseNet("10.0.0.1/32"),
		mustParseNet("10.0.0.2/32")},
	IPv6Nets: []net.IPNet{mustParseNet("fc00:fe11::1/128"),
		mustParseNet("fc00:fe11::2/128")},
	Labels: map[string]string{
		"name": "dnspolicy",
	},
}

var localHostIP = mustParseIP("192.168.0.1")
var remoteHostIP = mustParseIP("192.168.0.2")
var remoteHost2IP = mustParseIP("192.168.0.3")

var localHostVXLANTunnelConfigKey = HostConfigKey{
	Hostname: localHostname,
	Name:     "IPv4VXLANTunnelAddr",
}
var remoteHostVXLANTunnelConfigKey = HostConfigKey{
	Hostname: remoteHostname,
	Name:     "IPv4VXLANTunnelAddr",
}
var remoteHost2VXLANTunnelConfigKey = HostConfigKey{
	Hostname: remoteHostname2,
	Name:     "IPv4VXLANTunnelAddr",
}

var remoteHostVXLANTunnelMACConfigKey = HostConfigKey{
	Hostname: remoteHostname,
	Name:     "VXLANTunnelMACAddr",
}

var ipPoolKey = IPPoolKey{
	CIDR: mustParseNet("10.0.0.0/16"),
}

var ipPoolNoEncap = IPPool{
	CIDR: mustParseNet("10.0.0.0/16"),
}

var ipPoolWithIPIP = IPPool{
	CIDR:     mustParseNet("10.0.0.0/16"),
	IPIPMode: encap.Always,
}

var ipPoolWithVXLAN = IPPool{
	CIDR:      mustParseNet("10.0.0.0/16"),
	VXLANMode: encap.Always,
}

var remoteIPAMBlockKey = BlockKey{
	CIDR: mustParseNet("10.0.1.0/29"),
}

var localIPAMBlockKey = BlockKey{
	CIDR: mustParseNet("10.0.0.0/29"),
}

var localHostAffinity = "host:" + localHostname
var remoteHostAffinity = "host:" + remoteHostname
var remoteHost2Affinity = "host:" + remoteHostname2
var remoteIPAMBlock = AllocationBlock{
	CIDR:        mustParseNet("10.0.1.0/29"),
	Affinity:    &remoteHostAffinity,
	Allocations: make([]*int, 8),
	Unallocated: []int{0, 1, 2, 3, 4, 5, 6, 7},
}
var remoteIPAMBlockWithBorrows = AllocationBlock{
	CIDR:     mustParseNet("10.0.1.0/29"),
	Affinity: &remoteHostAffinity,
	Allocations: []*int{
		intPtr(0),
		intPtr(1),
		intPtr(2),
		nil,
		nil,
		nil,
		nil,
		nil,
	},
	Unallocated: []int{3, 4, 5, 6, 7},
	Attributes: []AllocationAttribute{
		{},
		{AttrSecondary: map[string]string{
			IPAMBlockAttributeNode: remoteHostname,
		}},
		{AttrSecondary: map[string]string{
			IPAMBlockAttributeNode: remoteHostname2,
		}},
	},
}
var remoteIPAMBlockWithBorrowsSwitched = AllocationBlock{
	CIDR:     mustParseNet("10.0.1.0/29"),
	Affinity: &remoteHost2Affinity,
	Allocations: []*int{
		intPtr(0),
		intPtr(1),
		intPtr(2),
		nil,
		nil,
		nil,
		nil,
		nil,
	},
	Unallocated: []int{3, 4, 5, 6, 7},
	Attributes: []AllocationAttribute{
		{},
		{AttrSecondary: map[string]string{
			IPAMBlockAttributeNode: remoteHostname2,
		}},
		{AttrSecondary: map[string]string{
			IPAMBlockAttributeNode: remoteHostname,
		}},
	},
}

var localIPAMBlockWithBorrows = AllocationBlock{
	CIDR:     mustParseNet("10.0.0.0/29"),
	Affinity: &localHostAffinity,
	Allocations: []*int{
		intPtr(0),
		intPtr(1),
		intPtr(2),
		nil,
		nil,
		nil,
		nil,
		nil,
	},
	Unallocated: []int{3, 4, 5, 6, 7},
	Attributes: []AllocationAttribute{
		{},
		{AttrSecondary: map[string]string{
			IPAMBlockAttributeNode: localHostname,
		}},
		{AttrSecondary: map[string]string{
			IPAMBlockAttributeNode: remoteHostname,
		}},
	},
}

func intPtr(i int) *int {
	return &i
}

var localHostVXLANTunnelIP = "10.0.0.0"
var remoteHostVXLANTunnelIP = "10.0.1.0"
var remoteHostVXLANTunnelIP2 = "10.0.1.1"
var remoteHost2VXLANTunnelIP = "10.0.2.0"
var remoteHostVXLANTunnelMAC = "66:74:c5:72:3f:01"
