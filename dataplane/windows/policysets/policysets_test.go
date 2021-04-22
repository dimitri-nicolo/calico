// Copyright (c) 2017-2021 Tigera, Inc. All rights reserved.
//
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

package policysets

import (
	"testing"

	log "github.com/sirupsen/logrus"

	. "github.com/onsi/gomega"

	"github.com/projectcalico/felix/dataplane/windows/hns"
	"github.com/projectcalico/felix/proto"
)

func TestRuleRenderingWithStaticRules(t *testing.T) {
	RegisterTestingT(t)

	h := mockHNS{}

	// Windows 1803/RS4
	h.SupportedFeatures.Acl.AclRuleId = true
	h.SupportedFeatures.Acl.AclNoHostRulePriority = true

	log.SetLevel(log.DebugLevel)

	ipsc := mockIPSetCache{
		IPSets: map[string][]string{},
	}

	ps := NewPolicySets(&h, []IPSetCache{&ipsc}, mockReader(staticRules))

	// Unknown policy should result in default drop.
	Expect(ps.GetPolicySetRules([]string{"unknown"}, true)).To(Equal([]*hns.ACLPolicy{
		// Static inbound rule.
		{Type: hns.ACL, Id: "MyPlatform-block-client", Protocol: 17, Action: hns.Allow, Direction: hns.In,
			RuleType: hns.Host, Priority: 300, RemoteAddresses: "10.0.0.2/32", RemotePorts: "90"},
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rules returned for unknown policy")

	// Empty policy should return no rules (apart from the default drop).
	ps.AddOrReplacePolicySet("empty", &proto.Policy{
		InboundRules:  []*proto.Rule{},
		OutboundRules: []*proto.Rule{},
	})

	Expect(ps.GetPolicySetRules([]string{"empty"}, true)).To(Equal([]*hns.ACLPolicy{
		// Static inbound rule.
		{Type: hns.ACL, Id: "MyPlatform-block-client", Protocol: 17, Action: hns.Allow, Direction: hns.In,
			RuleType: hns.Host, Priority: 300, RemoteAddresses: "10.0.0.2/32", RemotePorts: "90"},
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rules returned for empty policy")

	Expect(ps.GetPolicySetRules([]string{"empty"}, false)).To(Equal([]*hns.ACLPolicy{
		// Static outbound rule.
		{Type: hns.ACL, Id: "MyPlatform-block-server", Protocol: 6, Action: hns.Block, Direction: hns.Out,
			RuleType: hns.Switch, Priority: 200, RemoteAddresses: "10.0.0.1/32", RemotePorts: "80"},
		// Default deny rule.
		{Type: hns.ACL, Id: "DRE", Protocol: 256, Action: hns.Block, Direction: hns.Out, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rules returned for empty policy")

	// Tests of basic policy matches: CIDRs, protocol, ports.
	ps.AddOrReplacePolicySet("policy-basic", &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action:   "Allow",
				SrcNet:   []string{"10.0.0.0/24"},
				Protocol: &proto.Protocol{NumberOrName: &proto.Protocol_Name{Name: "tcp"}},
				SrcPorts: []*proto.PortRange{{First: 1234, Last: 1234}},
				DstPorts: []*proto.PortRange{{First: 80, Last: 80}},
				RuleId:   "rule-1",
			},
			{
				Action:   "Allow",
				Protocol: &proto.Protocol{NumberOrName: &proto.Protocol_Number{Number: 17}},
				RuleId:   "rule-2",
			},
			{
				Action: "Deny",
				DstNet: []string{"10.0.0.0/24"},
				RuleId: "rule-3",
			},
			{
				Action: "Deny",
				DstNet: []string{"11.0.0.0/24"},
				RuleId: "rule-4",
			},
		},
		OutboundRules: []*proto.Rule{
			{
				Action:   "Allow",
				Protocol: &proto.Protocol{NumberOrName: &proto.Protocol_Number{Number: 17}},
				RuleId:   "rule-5",
			},
		},
	})

	Expect(ps.GetPolicySetRules([]string{"policy-basic"}, true)).To(Equal([]*hns.ACLPolicy{
		// Static inbound rule.
		{Type: hns.ACL, Id: "MyPlatform-block-client", Protocol: 17, Action: hns.Allow, Direction: hns.In,
			RuleType: hns.Host, Priority: 300, RemoteAddresses: "10.0.0.2/32", RemotePorts: "90"},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch,
			Priority:        1000,
			Protocol:        6,
			Id:              "API0|basic---rule-1---0",
			RemoteAddresses: "10.0.0.0/24",
			RemotePorts:     "1234",
			LocalPorts:      "80",
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch,
			Priority: 1001,
			Protocol: 17,
			Id:       "API1|basic---rule-2---0",
		},
		{
			Type: hns.ACL, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch,
			Priority:       1002,
			Protocol:       256,
			Id:             "DPI2|basic---rule-3---0",
			LocalAddresses: "10.0.0.0/24",
		},
		{
			Type: hns.ACL, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch,
			Priority:       1003,
			Protocol:       256,
			Id:             "DPI3|basic---rule-4---0",
			LocalAddresses: "11.0.0.0/24",
		},
		// Default deny rule.
		{Type: hns.ACL, Protocol: 256, Id: "DRI", Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1004},
	}), "unexpected rules returned for basic policy")

	Expect(ps.GetPolicySetRules([]string{"policy-basic"}, false)).To(Equal([]*hns.ACLPolicy{
		// Static outbound rule.
		{Type: hns.ACL, Id: "MyPlatform-block-server", Protocol: 6, Action: hns.Block, Direction: hns.Out,
			RuleType: hns.Switch, Priority: 200, RemoteAddresses: "10.0.0.1/32", RemotePorts: "80"},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.Out, RuleType: hns.Switch,
			Priority: 1000,
			Protocol: 17,
			Id:       "APE0|basic---rule-5---0",
		},
		// Default deny rule.
		{Type: hns.ACL, Protocol: 256, Id: "DRE", Action: hns.Block, Direction: hns.Out, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rules returned for basic policy")
}

func TestRuleRenderingExceedingPriorityLimit(t *testing.T) {
	RegisterTestingT(t)

	h := mockHNS{}

	// Windows 1803/RS4
	h.SupportedFeatures.Acl.AclRuleId = true
	h.SupportedFeatures.Acl.AclNoHostRulePriority = true

	log.SetLevel(log.DebugLevel)

	ipsc := mockIPSetCache{
		IPSets: map[string][]string{},
	}

	ps := NewPolicySets(&h, []IPSetCache{&ipsc}, mockReader(""))
	ps.priorityLimit = 1002

	// Test a set of rules that exceeds the priority limit. Rule priority should
	// be reused when action and direction do not change.
	ps.AddOrReplacePolicySet("policy-basic", &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action:   "Allow",
				SrcNet:   []string{"10.0.0.0/24"},
				Protocol: &proto.Protocol{NumberOrName: &proto.Protocol_Name{Name: "tcp"}},
				SrcPorts: []*proto.PortRange{{First: 1234, Last: 1234}},
				DstPorts: []*proto.PortRange{{First: 80, Last: 80}},
				RuleId:   "rule-1",
			},
			{
				Action:   "Allow",
				Protocol: &proto.Protocol{NumberOrName: &proto.Protocol_Number{Number: 17}},
				RuleId:   "rule-2",
			},
			{
				Action: "Deny",
				DstNet: []string{"10.0.0.0/24"},
				RuleId: "rule-3",
			},
			{
				Action: "Deny",
				DstNet: []string{"11.0.0.0/24"},
				RuleId: "rule-4",
			},
		},
		OutboundRules: []*proto.Rule{},
	})

	Expect(ps.GetPolicySetRules([]string{"policy-basic"}, true)).To(Equal([]*hns.ACLPolicy{
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch,
			Priority:        1000,
			Protocol:        6,
			Id:              "API0|basic---rule-1---0",
			RemoteAddresses: "10.0.0.0/24",
			RemotePorts:     "1234",
			LocalPorts:      "80",
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch,
			Priority: 1000,
			Protocol: 17,
			Id:       "API1|basic---rule-2---0",
		},
		{
			Type: hns.ACL, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch,
			Priority:       1001, // Switch from Allow to Deny triggers increment to priority.
			Protocol:       256,
			Id:             "DPI2|basic---rule-3---0",
			LocalAddresses: "10.0.0.0/24",
		},
		{
			Type: hns.ACL, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch,
			Priority:       1001, // No change of action so priority stays the same.
			Protocol:       256,
			Id:             "DPI3|basic---rule-4---0",
			LocalAddresses: "11.0.0.0/24",
		},
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1002},
	}), "unexpected rules returned for basic policy")

}

func TestRuleRendering(t *testing.T) {
	RegisterTestingT(t)

	h := mockHNS{}

	// Windows 1803/RS4
	h.SupportedFeatures.Acl.AclRuleId = true
	h.SupportedFeatures.Acl.AclNoHostRulePriority = true

	log.SetLevel(log.DebugLevel)

	ipsc := mockIPSetCache{
		IPSets: map[string][]string{},
	}

	ps := NewPolicySets(&h, []IPSetCache{&ipsc}, mockReader(""))
	Expect(ps.priorityLimit).To(BeEquivalentTo(PolicyRuleMaxPriority))

	// Unknown policy should result in default drop.
	Expect(ps.GetPolicySetRules([]string{"unknown"}, true)).To(Equal([]*hns.ACLPolicy{
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rules returned for unknown policy")

	// Empty policy should return no rules (apart from the default drop).
	ps.AddOrReplacePolicySet("empty", &proto.Policy{
		InboundRules:  []*proto.Rule{},
		OutboundRules: []*proto.Rule{},
	})

	Expect(ps.GetPolicySetRules([]string{"empty"}, true)).To(Equal([]*hns.ACLPolicy{
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rules returned for empty policy")

	// Tests of basic policy matches: CIDRs, protocol, ports.
	ps.AddOrReplacePolicySet("policy-basic", &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action:   "Allow",
				SrcNet:   []string{"10.0.0.0/24"},
				Protocol: &proto.Protocol{NumberOrName: &proto.Protocol_Name{Name: "tcp"}},
				SrcPorts: []*proto.PortRange{{First: 1234, Last: 1234}},
				DstPorts: []*proto.PortRange{{First: 80, Last: 80}},
				RuleId:   "rule-1",
			},
			{
				Action:   "Allow",
				Protocol: &proto.Protocol{NumberOrName: &proto.Protocol_Number{Number: 17}},
				RuleId:   "rule-2",
			},
			{
				Action: "Deny",
				DstNet: []string{"10.0.0.0/24"},
				RuleId: "rule-3",
			},
			{
				Action: "Deny",
				DstNet: []string{"11.0.0.0/24"},
				RuleId: "rule-4",
			},
		},
		OutboundRules: []*proto.Rule{},
	})

	Expect(ps.GetPolicySetRules([]string{"policy-basic"}, true)).To(Equal([]*hns.ACLPolicy{
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch,
			Priority:        1000,
			Protocol:        6,
			Id:              "API0|basic---rule-1---0",
			RemoteAddresses: "10.0.0.0/24",
			RemotePorts:     "1234",
			LocalPorts:      "80",
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch,
			Priority: 1001,
			Protocol: 17,
			Id:       "API1|basic---rule-2---0",
		},
		{
			Type: hns.ACL, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch,
			Priority:       1002,
			Protocol:       256,
			Id:             "DPI2|basic---rule-3---0",
			LocalAddresses: "10.0.0.0/24",
		},
		{
			Type: hns.ACL, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch,
			Priority:       1003,
			Protocol:       256,
			Id:             "DPI3|basic---rule-4---0",
			LocalAddresses: "11.0.0.0/24",
		},
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1004},
	}), "unexpected rules returned for basic policy")

	// Tests for Profile
	// Empty profile should return no rules (apart from the default drop).
	ps.AddOrReplacePolicySet("profile-empty", &proto.Profile{
		InboundRules:  []*proto.Rule{},
		OutboundRules: []*proto.Rule{},
	})

	Expect(ps.GetPolicySetRules([]string{"profile-empty"}, true)).To(Equal([]*hns.ACLPolicy{
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rules returned for empty profile")

	// Test for Rule with profile
	ps.AddOrReplacePolicySet("profile-rule-with", &proto.Profile{
		InboundRules: []*proto.Rule{
			{
				Action: "Allow",
				RuleId: "rule-1",
			},
		},
		OutboundRules: []*proto.Rule{},
	})

	Expect(ps.GetPolicySetRules([]string{"profile-rule-with"}, true)).To(Equal([]*hns.ACLPolicy{
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id: "ARI0|rule-with---rule-1---0",
		},
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rules returned for profile")

	//Test with Mixed CIDR to filterout IpV4
	ps.AddOrReplacePolicySet("policy-mixed-cidr", &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action:    "Allow",
				SrcNet:    []string{"0:0:0:0:0:ffff:af4:301"},
				IpVersion: 4,
				RuleId:    "rule-1",
			},
			{
				Action:    "Allow",
				DstNet:    []string{"0:0:0:0:0:ffff:af4:301"},
				IpVersion: 4,
				RuleId:    "rule-2",
			},
			{
				Action:    "Allow",
				NotSrcNet: []string{"0:0:0:0:0:ffff:af4:301"},
				IpVersion: 4,
				RuleId:    "rule-3",
			},
			{
				Action:    "Allow",
				NotDstNet: []string{"0:0:0:0:0:ffff:af4:301"},
				IpVersion: 4,
				RuleId:    "rule-4",
			},
		},
		OutboundRules: []*proto.Rule{},
	})

	// We expect the rules to be skipped as there isn't any ip of type Ipv4
	Expect(ps.GetPolicySetRules([]string{"policy-mixed-cidr"}, true)).To(Equal([]*hns.ACLPolicy{
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rules returned for mixed-cidr")

	//Outbound policy with SrcNet
	ps.AddOrReplacePolicySet("policy-out-srcnet", &proto.Policy{
		InboundRules: []*proto.Rule{},
		OutboundRules: []*proto.Rule{
			{
				Action: "Allow",
				SrcNet: []string{"10.0.0.0/24"},
				RuleId: "rule-1",
			},
		},
	})

	Expect(ps.GetPolicySetRules([]string{"policy-out-srcnet"}, false)).To(Equal([]*hns.ACLPolicy{
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.Out, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id:             "APE0|out-srcnet---rule-1---0",
			LocalAddresses: "10.0.0.0/24",
		},
		// Default deny rule.
		{Type: hns.ACL, Id: "DRE", Protocol: 256, Action: hns.Block, Direction: hns.Out, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rules returned for outbound policy with SrcNet")

}

func TestNegativeTestCases(t *testing.T) {

	RegisterTestingT(t)

	h := mockHNS{}

	// Windows 1803/RS4
	h.SupportedFeatures.Acl.AclRuleId = true
	h.SupportedFeatures.Acl.AclNoHostRulePriority = true

	ipsc := mockIPSetCache{
		IPSets: map[string][]string{},
	}

	ps := NewPolicySets(&h, []IPSetCache{&ipsc}, mockReader(""))
	Expect(ps.priorityLimit).To(BeEquivalentTo(PolicyRuleMaxPriority))

	//Test Negative scenarios
	//look up ip set that doesn't exist.
	ps.AddOrReplacePolicySet("policy-ipset-that-does-not-exist", &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action:      "Allow",
				SrcIpSetIds: []string{"i", "j"},
				RuleId:      "rule-1",
			},
			{
				Action:      "Allow",
				DstIpSetIds: []string{"k"},
				RuleId:      "rule-2",
			},
		},
		OutboundRules: []*proto.Rule{},
	})

	Expect(ps.GetPolicySetRules([]string{"policy-ipset-that-does-not-exist"}, true)).To(Equal([]*hns.ACLPolicy{
		// Rules should be skipped
		// Only the Default rules should exist.
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rules returned for ipset-that-does-not-exist")

	//Negative test: Unsupported protocol
	ps.AddOrReplacePolicySet("policy-unsupported-protocol", &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action:   "Allow",
				Protocol: &proto.Protocol{NumberOrName: &proto.Protocol_Name{Name: "gre"}},
				RuleId:   "rule-1",
			},
		},
		OutboundRules: []*proto.Rule{},
	})

	Expect(ps.GetPolicySetRules([]string{"policy-unsupported-protocol"}, true)).NotTo(Equal([]*hns.ACLPolicy{
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch,
			Priority: 1000,
			Protocol: 47,
			Id:       "API0|unsupported-protocol---rule-1---0",
		},
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rule returned for Unsupported protocol")

	//Negative test: Unsupported IP version (IP v6)
	ps.AddOrReplacePolicySet("policy-unsupported-ip-version", &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action:    "Allow",
				IpVersion: 6,
				SrcNet:    []string{"0:0:0:0:0:ffff:af4:301"},
				RuleId:    "rule-1",
			},
		},
		OutboundRules: []*proto.Rule{},
	})

	Expect(ps.GetPolicySetRules([]string{"policy-unsupported-ip-version"}, true)).To(Equal([]*hns.ACLPolicy{
		//The rule with IP v6 should be skipped
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rule returned for unsupported IP version")

	//Negative test: Named port
	ps.AddOrReplacePolicySet("policy-named-port", &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action:               "Allow",
				Protocol:             &proto.Protocol{NumberOrName: &proto.Protocol_Name{Name: "tcp"}},
				SrcNamedPortIpSetIds: []string{"ipset-1"},
				RuleId:               "rule-1",
			},
		},
		OutboundRules: []*proto.Rule{},
	})

	Expect(ps.GetPolicySetRules([]string{"policy-named-port"}, true)).To(Equal([]*hns.ACLPolicy{
		//The rule with named port should be skipped
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rule with named port")

	//Negative test: ICMP type
	ps.AddOrReplacePolicySet("policy-icmp-type", &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action: "Allow",
				Icmp:   &proto.Rule_IcmpType{IcmpType: 10},
				RuleId: "rule-1",
			},
		},
		OutboundRules: []*proto.Rule{},
	})

	Expect(ps.GetPolicySetRules([]string{"policy-icmp-type"}, true)).To(Equal([]*hns.ACLPolicy{
		//The rule with ICMP type should be skipped
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rule with ICMP Type")

	//Negative test: With Negative Matches
	ps.AddOrReplacePolicySet("policy-negative-match", &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action:    "Allow",
				NotSrcNet: []string{"10.0.0.0/24"},
				RuleId:    "rule-1",
			},
		},
		OutboundRules: []*proto.Rule{},
	})

	Expect(ps.GetPolicySetRules([]string{"policy-negative-match"}, true)).To(Equal([]*hns.ACLPolicy{
		//The rule with negative match should be skipped
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rule with Negative match")

	//Test with invalid argument to AddOrReplacePolicySet (Other than Profile/Policy)
	ps.AddOrReplacePolicySet("policy-invalid-arg", &proto.ProfileID{
		Name: "abc",
	})

	Expect(ps.GetPolicySetRules([]string{"policy-invalid-arg"}, true)).To(Equal([]*hns.ACLPolicy{
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rules returned when invalid argument is passed to addOrReplacePolicySet function")

	//Negative test for protoRuleToHnsRules
	chunkSize := 2

	var aclPolicy []*hns.ACLPolicy
	//Negative test: Policy with NotSrcNet
	aclPolicy, _ = ps.protoRuleToHnsRules("policy-with-NotSrcNet",
		&proto.Rule{
			Action:    "Allow",
			NotSrcNet: []string{"10.0.0.0/24"},
			RuleId:    "rule-1",
		}, 0, true, chunkSize)

	//Rule should be skipped
	Expect(aclPolicy).To(Equal([]*hns.ACLPolicy(nil)), "incorrect rules returned for policy with NotSrcNet")

	//Negative test: Policy with NotDstNet
	aclPolicy, _ = ps.protoRuleToHnsRules("policy-with-NotDstNet",
		&proto.Rule{
			Action:    "Allow",
			NotDstNet: []string{"10.0.0.0/24"},
			RuleId:    "rule-1",
		}, 0, true, chunkSize)

	//Rule should be skipped
	Expect(aclPolicy).To(Equal([]*hns.ACLPolicy(nil)), "incorrect rules returned for NotDstNet")

	//Negative test: Policy where Action is pass/next-tier/log
	aclPolicy, _ = ps.protoRuleToHnsRules("policy-with-unsupported-action",
		&proto.Rule{
			Action:    "pass",
			NotDstNet: []string{"10.0.0.0/24"},
			RuleId:    "rule-1",
		}, 0, true, chunkSize)

	// Rule should be skipped
	Expect(aclPolicy).To(Equal([]*hns.ACLPolicy(nil)), "incorrect rules returned for Policy with unsupported action")

	//Negative test: Policy with invalid Action
	aclPolicy, _ = ps.protoRuleToHnsRules("policy-with-invalid-action",
		&proto.Rule{
			Action:    "abc",
			NotDstNet: []string{"10.0.0.0/24"},
			RuleId:    "rule-1",
		}, 0, true, chunkSize)

	//Rule should be skipped
	Expect(aclPolicy).To(Equal([]*hns.ACLPolicy(nil)), "incorrect rules returned for Policy with invalid action")

}

func TestMultiIpPortChunks(t *testing.T) {
	RegisterTestingT(t)

	h := mockHNS{}

	// Windows 1803/RS4
	h.SupportedFeatures.Acl.AclRuleId = true
	h.SupportedFeatures.Acl.AclNoHostRulePriority = true

	ipsc := mockIPSetCache{
		IPSets: map[string][]string{
			"a": {"10.0.0.1", "10.0.0.2"},
			"b": {"10.0.0.2", "10.0.0.3"},
			"d": {"10.1.0.1", "10.1.0.2"},
			"e": {"10.1.0.2", "10.1.0.3"},
			"f": {"10.0.0.3", "10.1.0.1"},
		},
	}

	ps := NewPolicySets(&h, []IPSetCache{&ipsc}, mockReader(""))
	ps.priorityLimit = 1000

	chunkSize := 2
	//check for empty portrange
	Expect(SplitPortList([]*proto.PortRange{}, chunkSize)).To(Equal([][]*proto.PortRange{{}}), "incorrect chunks returned for empty PortRange")

	//check with multi port number and range
	portChunks := SplitPortList([]*proto.PortRange{{First: 1234, Last: 1234}, {First: 22, Last: 24}, {First: 80, Last: 80}}, chunkSize)
	Expect(portChunks).To(Equal([][]*proto.PortRange{
		{
			{First: 1234, Last: 1234},
			{First: 22, Last: 24},
		},
		{
			{First: 80, Last: 80},
		},
	}), "incorrect chunks returned for multi ports")

	//Now verify that each chunk should be converted into HCS format
	var portList string
	results := []string{"1234,22-24", "80"}
	i := 0
	for _, ports := range portChunks {
		portList = appendPortsinList(ports)
		Expect(portList).To(Equal(results[i]), "incorrect portList returned for multi ports")
		i++
	}

	//check with empty string
	Expect(SplitIPList([]string{}, chunkSize)).To(Equal([][]string{{}}), "incorrect chunks returned for empty string")

	//check with multi ip addresses
	Expect(SplitIPList([]string{"10.1.1.1/32", "10.2.2.2/32", "10.3.3.3/32"}, chunkSize)).To(Equal([][]string{
		{"10.1.1.1/32", "10.2.2.2/32"},
		{"10.3.3.3/32"},
	}), "incorrect chunks returned for multi IPs")

	//verify aclpolicy for empty egress rule
	Expect(ps.protoRuleToHnsRules("policy-empty-egress-1", &proto.Rule{}, 0, false, chunkSize)).To(Equal([]*hns.ACLPolicy{
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.Out, RuleType: hns.Switch,
			Id:              "APE0|empty-egress-1---0",
			Protocol:        256,
			LocalAddresses:  "",
			RemoteAddresses: "",
			LocalPorts:      "",
			RemotePorts:     "",
			Priority:        1000,
		},
	}), "incorrect hns rules returned for empty egress rules")

	//verify aclpolicy for empty ingress rule
	Expect(ps.protoRuleToHnsRules("policy-empty-ingress-1", &proto.Rule{}, 0, true, chunkSize)).To(Equal([]*hns.ACLPolicy{
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch,
			Id:              "API0|empty-ingress-1---0",
			Protocol:        256,
			LocalAddresses:  "",
			RemoteAddresses: "",
			LocalPorts:      "",
			RemotePorts:     "",
			Priority:        1000,
		},
	}), "incorrect hns rules returned for empty egress rules")

	//verify aclPolicy for multiple ips and port in a sigle inbound rule with chunksize 2
	var aclPolicy []*hns.ACLPolicy
	aclPolicy, _ = ps.protoRuleToHnsRules("policy-Multi-ips-ports-1",
		&proto.Rule{
			Action:   "Allow",
			SrcNet:   []string{"10.0.0.0/24", "10.1.1.0/24", "10.2.2.0/24"},
			Protocol: &proto.Protocol{NumberOrName: &proto.Protocol_Name{Name: "tcp"}},
			SrcPorts: []*proto.PortRange{{First: 1234, Last: 1234}, {First: 22, Last: 24}, {First: 81, Last: 81}},
			DstPorts: []*proto.PortRange{{First: 80, Last: 80}, {First: 81, Last: 81}, {First: 85, Last: 85}},
			RuleId:   "rule-1",
		}, 0, true, chunkSize)

	Expect(aclPolicy).To(Equal([]*hns.ACLPolicy{
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch,
			Id:              "API0|Multi-ips-ports-1---rule-1---0",
			Protocol:        6,
			Protocols:       "",
			RemoteAddresses: "10.0.0.0/24,10.1.1.0/24",
			LocalPorts:      "80,81",
			RemotePorts:     "1234,22-24",
			Priority:        1000,
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch,
			Id:              "API0|Multi-ips-ports-1---rule-1---1",
			Protocol:        6,
			RemoteAddresses: "10.0.0.0/24,10.1.1.0/24",
			LocalPorts:      "80,81",
			RemotePorts:     "81",
			Priority:        1000,
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch,
			Id:              "API0|Multi-ips-ports-1---rule-1---2",
			Protocol:        6,
			RemoteAddresses: "10.2.2.0/24",
			LocalPorts:      "80,81",
			RemotePorts:     "1234,22-24",
			Priority:        1000,
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch,
			Id:              "API0|Multi-ips-ports-1---rule-1---3",
			Protocol:        6,
			RemoteAddresses: "10.2.2.0/24",
			LocalPorts:      "80,81",
			RemotePorts:     "81",
			Priority:        1000,
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch,
			Id:              "API0|Multi-ips-ports-1---rule-1---4",
			Protocol:        6,
			RemoteAddresses: "10.0.0.0/24,10.1.1.0/24",
			LocalPorts:      "85",
			RemotePorts:     "1234,22-24",
			Priority:        1000,
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch,
			Id:              "API0|Multi-ips-ports-1---rule-1---5",
			Protocol:        6,
			RemoteAddresses: "10.0.0.0/24,10.1.1.0/24",
			LocalPorts:      "85",
			RemotePorts:     "81",
			Priority:        1000,
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch,
			Id:              "API0|Multi-ips-ports-1---rule-1---6",
			Protocol:        6,
			RemoteAddresses: "10.2.2.0/24",
			LocalPorts:      "85",
			RemotePorts:     "1234,22-24",
			Priority:        1000,
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch,
			Id:              "API0|Multi-ips-ports-1---rule-1---7",
			Protocol:        6,
			RemoteAddresses: "10.2.2.0/24",
			LocalPorts:      "85",
			RemotePorts:     "81",
			Priority:        1000,
		},
	},
	), "incorrect hns rules returned for multi IPs")

	//verify aclPolicy for multiple ips and port in a sigle outbound rule with chunksize 2
	aclPolicy, _ = ps.protoRuleToHnsRules("policy-Multi-ips-ports-out-1",
		&proto.Rule{
			Action:   "Allow",
			DstNet:   []string{"10.0.0.0/24", "10.1.1.0/24", "10.2.2.0/24"},
			Protocol: &proto.Protocol{NumberOrName: &proto.Protocol_Name{Name: "tcp"}},
			SrcPorts: []*proto.PortRange{{First: 1234, Last: 1234}, {First: 22, Last: 24}, {First: 81, Last: 81}},
			DstPorts: []*proto.PortRange{{First: 80, Last: 80}, {First: 81, Last: 81}, {First: 85, Last: 85}},
			RuleId:   "rule-1",
		}, 0, false, chunkSize)

	Expect(aclPolicy).To(Equal([]*hns.ACLPolicy{
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.Out, RuleType: hns.Switch,
			Id:              "APE0|Multi-ips-ports-out-1---rule-1---0",
			Protocol:        6,
			RemoteAddresses: "10.0.0.0/24,10.1.1.0/24",
			LocalPorts:      "1234,22-24",
			RemotePorts:     "80,81",
			Priority:        1000,
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.Out, RuleType: hns.Switch,
			Id:              "APE0|Multi-ips-ports-out-1---rule-1---1",
			Protocol:        6,
			RemoteAddresses: "10.0.0.0/24,10.1.1.0/24",
			LocalPorts:      "1234,22-24",
			RemotePorts:     "85",
			Priority:        1000,
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.Out, RuleType: hns.Switch,
			Id:              "APE0|Multi-ips-ports-out-1---rule-1---2",
			Protocol:        6,
			RemoteAddresses: "10.2.2.0/24",
			LocalPorts:      "1234,22-24",
			RemotePorts:     "80,81",
			Priority:        1000,
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.Out, RuleType: hns.Switch,
			Id:              "APE0|Multi-ips-ports-out-1---rule-1---3",
			Protocol:        6,
			RemoteAddresses: "10.2.2.0/24",
			LocalPorts:      "1234,22-24",
			RemotePorts:     "85",
			Priority:        1000,
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.Out, RuleType: hns.Switch,
			Id:              "APE0|Multi-ips-ports-out-1---rule-1---4",
			Protocol:        6,
			RemoteAddresses: "10.0.0.0/24,10.1.1.0/24",
			LocalPorts:      "81",
			RemotePorts:     "80,81",
			Priority:        1000,
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.Out, RuleType: hns.Switch,
			Id:              "APE0|Multi-ips-ports-out-1---rule-1---5",
			Protocol:        6,
			RemoteAddresses: "10.0.0.0/24,10.1.1.0/24",
			LocalPorts:      "81",
			RemotePorts:     "85",
			Priority:        1000,
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.Out, RuleType: hns.Switch,
			Id:              "APE0|Multi-ips-ports-out-1---rule-1---6",
			Protocol:        6,
			RemoteAddresses: "10.2.2.0/24",
			LocalPorts:      "81",
			RemotePorts:     "80,81",
			Priority:        1000,
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.Out, RuleType: hns.Switch,
			Id:              "APE0|Multi-ips-ports-out-1---rule-1---7",
			Protocol:        6,
			RemoteAddresses: "10.2.2.0/24",
			LocalPorts:      "81",
			RemotePorts:     "85",
			Priority:        1000,
		},
	},
	), "incorrect hns rules returned for multi IPs with outbound policy")

	// Tests that look up an IP set.
	ps.AddOrReplacePolicySet("policy-selector", &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action:      "Allow",
				SrcIpSetIds: []string{"a", "b"},
				RuleId:      "rule-1",
			},
			{
				Action:      "Allow",
				DstIpSetIds: []string{"d"},
				RuleId:      "rule-2",
			},
		},
		OutboundRules: []*proto.Rule{},
	})

	//check multi ips hns rules should be created using ipsets
	Expect(ps.GetPolicySetRules([]string{"policy-selector"}, true)).To(Equal([]*hns.ACLPolicy{
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id: "API0|selector---rule-1---0", RemoteAddresses: "10.0.0.1,10.0.0.2,10.0.0.2,10.0.0.3",
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id: "API1|selector---rule-2---0", LocalAddresses: "10.1.0.1,10.1.0.2",
		},
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rules returned for selector multi ips policy")

	// Source and dest IP sets should be converted into hns rule with multi ips.
	ps.AddOrReplacePolicySet("policy-selector-ipsets", &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action:      "Allow",
				SrcIpSetIds: []string{"a"},
				DstIpSetIds: []string{"d"},
				RuleId:      "rule-1",
			},
		},
		OutboundRules: []*proto.Rule{},
	})

	Expect(ps.GetPolicySetRules([]string{"policy-selector-ipsets"}, true)).To(Equal([]*hns.ACLPolicy{
		// We expect the source/dest IP sets to be expressed as the cross product.
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id: "API0|selector-ipsets---rule-1---0", LocalAddresses: "10.1.0.1,10.1.0.2", RemoteAddresses: "10.0.0.1,10.0.0.2",
		},
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rules returned for selector ipset multi ips")

	// The source IP set should be intersected with the source CIDR.
	ps.AddOrReplacePolicySet("policy-selector-cidr", &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action:      "Allow",
				SrcIpSetIds: []string{"a", "f"},
				SrcNet:      []string{"10.0.0.0/24"},
				RuleId:      "rule-1",
			},
			{
				Action:      "Allow",
				SrcIpSetIds: []string{"a", "f"},
				SrcNet:      []string{"10.1.0.0/24"},
				RuleId:      "rule-2",
			},
			{
				Action:      "Allow",
				SrcIpSetIds: []string{"a", "f"},
				SrcNet:      []string{"10.0.0.0/24", "10.1.0.0/24"},
				RuleId:      "rule-3",
			},
			{
				Action:      "Allow",
				SrcIpSetIds: []string{"a", "f"},
				SrcNet:      []string{"12.0.0.0/24"},
				RuleId:      "rule-4",
			},
		},
		OutboundRules: []*proto.Rule{},
	})

	Expect(ps.GetPolicySetRules([]string{"policy-selector-cidr"}, true)).To(Equal([]*hns.ACLPolicy{
		// Intersection with first CIDR, picks up some IPs from each IP set.
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id: "API0|selector-cidr---rule-1---0", RemoteAddresses: "10.0.0.1/32,10.0.0.2/31",
		},
		// Intersection with second CIDr picks up only one IP.
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id: "API1|selector-cidr---rule-2---0", RemoteAddresses: "10.1.0.1/32",
		},

		// Intersection with both picks up everything.
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id: "API2|selector-cidr---rule-3---0", RemoteAddresses: "10.0.0.1/32,10.0.0.2/31,10.1.0.1/32",
		},
		// Rule 4 becomes a no-op since intersection is empty.

		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rules returned for selector CIDR filtering policy")

	//Complete coverage of IPSetUpdate
	ipsc = mockIPSetCache{
		IPSets: map[string][]string{
			"a": {"10.0.0.1", "10.0.0.2", "10.0.0.3"},
			"b": {"10.0.0.2", "10.0.0.3"},
			"d": {"10.1.0.1", "10.1.0.2"},
			"e": {"10.1.0.2", "10.1.0.3"},
			"f": {"10.0.0.3", "10.1.0.1"},
		},
	}

	//Updates the policies those use ipset a
	ps.ProcessIpSetUpdate("a")

	Expect(ps.GetPolicySetRules([]string{"policy-selector-ipsets"}, true)).To(Equal([]*hns.ACLPolicy{
		// We expect the policy to reflect the updated ipset
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id: "API0|selector-ipsets---rule-1---0", LocalAddresses: "10.1.0.1,10.1.0.2", RemoteAddresses: "10.0.0.1,10.0.0.2,10.0.0.3",
		},
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rules returned after ipset update")

	// Test for the ProcessIpSetUpdate while applying policy after updating the ipset
	ps.AddOrReplacePolicySet("policy-ipset-update", &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action:      "Allow",
				SrcIpSetIds: []string{"i"},
				RuleId:      "rule-1",
			},
		},
		OutboundRules: []*proto.Rule{},
	})

	Expect(ps.GetPolicySetRules([]string{"policy-ipset-update"}, true)).To(Equal([]*hns.ACLPolicy{
		// Rule should be skipped
		// Only the Default rules should exist.
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rules returned for ipset-update")

	//Updating the ipset
	ipsc = mockIPSetCache{
		IPSets: map[string][]string{
			"a": {"10.0.0.1", "10.0.0.2", "10.0.0.3"},
			"b": {"10.0.0.2", "10.0.0.3"},
			"d": {"10.1.0.1", "10.1.0.2"},
			"e": {"10.1.0.2", "10.1.0.3"},
			"f": {"10.0.0.3", "10.1.0.1"},
			"i": {"10.0.0.3", "10.1.0.4"},
			"k": {"10.0.0.5", "10.1.0.6"},
		},
	}

	ps.AddOrReplacePolicySet("policy-ipset-update", &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action:      "Allow",
				SrcIpSetIds: []string{"i"},
				RuleId:      "rule-1",
			},
		},
		OutboundRules: []*proto.Rule{},
	})

	Expect(ps.GetPolicySetRules([]string{"policy-ipset-update"}, true)).To(Equal([]*hns.ACLPolicy{
		//We expect the ipset updates would reflect in the rule
		{
			Type:            hns.ACL,
			Id:              "API0|ipset-update---rule-1---0",
			Protocol:        256,
			Action:          hns.Allow,
			Direction:       hns.In,
			RemoteAddresses: "10.0.0.3,10.1.0.4",
			RuleType:        hns.Switch,
			Priority:        1000,
		},
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rules returned for ipset-update")

	//Test where ProcessIpSetUpdate() receives an ipset-Id that doesn't have any policies linked
	//We expect the ProcessIpSetUpdate() should return nil
	Expect(ps.ProcessIpSetUpdate("policy-k")).To(Equal([]string(nil)), "Unexpected result returned by ProcessIpSetUpdate")

	//No overlapping as between SrcIpSetId and SrcNet & DstIpSetId and DstNet
	ps.AddOrReplacePolicySet("policy-no-overlapping", &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action:      "Allow",
				SrcNet:      []string{"10.5.5.5"},
				SrcIpSetIds: []string{"b"},
				RuleId:      "rule-1",
			},
			{
				Action:      "Allow",
				DstNet:      []string{"10.5.5.6"},
				DstIpSetIds: []string{"d"},
				RuleId:      "rule-2",
			},
		},
		OutboundRules: []*proto.Rule{},
	})

	Expect(ps.GetPolicySetRules([]string{"policy-no-overlapping"}, true)).To(Equal([]*hns.ACLPolicy{
		// We expect the rules to be skipped as no overlapping exist
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rules returned for no-overlapping")

}

func TestRuleRenderingWithDomainIPSets(t *testing.T) {
	RegisterTestingT(t)

	h := mockHNS{}

	// Windows 1803/RS4
	h.SupportedFeatures.Acl.AclRuleId = true
	h.SupportedFeatures.Acl.AclNoHostRulePriority = true

	//Updating the ipset
	ipsc := mockIPSetCache{
		IPSets: map[string][]string{
			"a": {"10.0.0.1", "10.0.0.2"},
			"d": {"10.1.0.1", "10.1.0.2"},
			"s": {"12.0.0.1"},
		},
	}

	ps := NewPolicySets(&h, []IPSetCache{&ipsc}, mockReader(""))

	// Policy should handle domain IpSets.
	ps.AddOrReplacePolicySet("policy-domain-ipset-update", &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action:      "Allow",
				SrcIpSetIds: []string{"s"},
				RuleId:      "rule-1",
			},
		},
		OutboundRules: []*proto.Rule{
			{
				Action:            "Allow",
				DstIpSetIds:       []string{"a"},
				DstDomainIpSetIds: []string{"d"},
				RuleId:            "rule-1",
			},
		},
	})

	Expect(ps.GetPolicySetRules([]string{"policy-domain-ipset-update"}, true)).To(Equal([]*hns.ACLPolicy{
		// Inbound rule.
		{Type: hns.ACL, Id: "API0|domain-ipset-update---rule-1---0", Protocol: 256, Action: hns.Allow, Direction: hns.In,
			RuleType: hns.Switch, Priority: 1000, RemoteAddresses: "12.0.0.1", RemotePorts: ""},
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rules returned for domain ipset policy")

	Expect(ps.GetPolicySetRules([]string{"policy-domain-ipset-update"}, false)).To(Equal([]*hns.ACLPolicy{
		// Outbound rule.
		{Type: hns.ACL, Id: "APE0|domain-ipset-update---rule-1---0", Protocol: 256, Action: hns.Allow, Direction: hns.Out,
			RuleType: hns.Switch, Priority: 1000, RemoteAddresses: "10.0.0.1,10.0.0.2,10.1.0.1,10.1.0.2", RemotePorts: ""},
		// Default deny rule.
		{Type: hns.ACL, Id: "DRE", Protocol: 256, Action: hns.Block, Direction: hns.Out, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rules returned for domain ipset policy")

	//Policy should handle empty domain IpSets.
	ipsc.IPSets["a"] = []string{}
	ipsc.IPSets["d"] = []string{}

	ps.ProcessIpSetUpdate("a")
	ps.ProcessIpSetUpdate("d")

	Expect(ps.GetPolicySetRules([]string{"policy-domain-ipset-update"}, false)).To(Equal([]*hns.ACLPolicy{
		// Default deny rule.
		{Type: hns.ACL, Id: "DRE", Protocol: 256, Action: hns.Block, Direction: hns.Out, RuleType: hns.Switch, Priority: 1001},
	}), "unexpected rules returned for domain ipset policy")

}

func TestPolicyOrderingExceedingPriorityLimit(t *testing.T) {
	RegisterTestingT(t)

	h := mockHNS{}
	// Windows 1803/RS4
	h.SupportedFeatures.Acl.AclRuleId = true
	h.SupportedFeatures.Acl.AclNoHostRulePriority = true

	ipsc := mockIPSetCache{
		IPSets: map[string][]string{},
	}

	ps := NewPolicySets(&h, []IPSetCache{&ipsc}, mockReader(""))
	ps.priorityLimit = 1000

	// Empty policy should return no rules (apart from the default drop).
	ps.AddOrReplacePolicySet("policy-allow", &proto.Policy{
		InboundRules: []*proto.Rule{{Action: "Allow"}},
	})
	ps.AddOrReplacePolicySet("policy-deny", &proto.Policy{
		InboundRules: []*proto.Rule{{Action: "Deny"}},
	})

	Expect(ps.GetPolicySetRules([]string{"policy-allow", "policy-deny"}, true)).To(Equal([]*hns.ACLPolicy{
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000,
			Id: "API0|allow---0"},
		{Type: hns.ACL, Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001,
			Id: "DPI0|deny---0"},
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1002},
	}), "incorrect rules returned for allow,deny")

	Expect(ps.GetPolicySetRules([]string{"policy-allow", "policy-allow"}, true)).To(Equal([]*hns.ACLPolicy{
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000,
			Id: "API0|allow---0"},
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000,
			Id: "API0|allow---0"},
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
	}), "incorrect rules returned for allow,allow")

	Expect(ps.GetPolicySetRules([]string{"policy-deny", "policy-allow"}, true)).To(Equal([]*hns.ACLPolicy{
		{Type: hns.ACL, Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1000,
			Id: "DPI0|deny---0"},
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1001,
			Id: "API0|allow---0"},
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1002},
	}), "incorrect rules returned for deny,allow")
}

func TestPolicyOrdering(t *testing.T) {
	RegisterTestingT(t)

	h := mockHNS{}
	// Windows 1803/RS4
	h.SupportedFeatures.Acl.AclRuleId = true
	h.SupportedFeatures.Acl.AclNoHostRulePriority = true

	ipsc := mockIPSetCache{
		IPSets: map[string][]string{},
	}

	ps := NewPolicySets(&h, []IPSetCache{&ipsc}, mockReader(""))
	Expect(ps.priorityLimit).To(BeEquivalentTo(PolicyRuleMaxPriority))

	// Empty policy should return no rules (apart from the default drop).
	ps.AddOrReplacePolicySet("policy-allow", &proto.Policy{
		InboundRules: []*proto.Rule{{Action: "Allow"}},
	})
	ps.AddOrReplacePolicySet("policy-deny", &proto.Policy{
		InboundRules: []*proto.Rule{{Action: "Deny"}},
	})

	Expect(ps.GetPolicySetRules([]string{"policy-allow", "policy-deny"}, true)).To(Equal([]*hns.ACLPolicy{
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000,
			Id: "API0|allow---0"},
		{Type: hns.ACL, Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001,
			Id: "DPI0|deny---0"},
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1002},
	}), "incorrect rules returned for allow,deny")

	Expect(ps.GetPolicySetRules([]string{"policy-allow", "policy-allow"}, true)).To(Equal([]*hns.ACLPolicy{
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000,
			Id: "API0|allow---0"},
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1001,
			Id: "API0|allow---0"},
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1002},
	}), "incorrect rules returned for allow,allow")

	Expect(ps.GetPolicySetRules([]string{"policy-deny", "policy-allow"}, true)).To(Equal([]*hns.ACLPolicy{
		{Type: hns.ACL, Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1000,
			Id: "DPI0|deny---0"},
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1001,
			Id: "API0|allow---0"},
		// Default deny rule.
		{Type: hns.ACL, Id: "DRI", Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1002},
	}), "incorrect rules returned for deny,allow")
}

// Test of ruleHasNegativeMatches()
func TestRuleHasNegativeMatches(t *testing.T) {

	RegisterTestingT(t)

	Expect(ruleHasNegativeMatches(&proto.Rule{
		Action:    "Allow",
		NotSrcNet: []string{"10.0.0.0/24"},
	})).To(Equal(true), "Unexpected result with NotSrcNet")

	Expect(ruleHasNegativeMatches(&proto.Rule{
		Action:      "Allow",
		NotSrcPorts: []*proto.PortRange{{First: 1234, Last: 1234}},
	})).To(Equal(true), "Unexpected result with NotSrcPort")

	Expect(ruleHasNegativeMatches(&proto.Rule{
		Action:         "Allow",
		NotSrcIpSetIds: []string{"a"},
	})).To(Equal(true), "Unexpected result with NotSrcIpSetIds")

	Expect(ruleHasNegativeMatches(&proto.Rule{
		Action:                  "Allow",
		NotSrcNamedPortIpSetIds: []string{"ipset-1"},
	})).To(Equal(true), "Unexpected result with NotSrcNamedPortIpSetIds")

	Expect(ruleHasNegativeMatches(&proto.Rule{
		Action:      "Allow",
		NotProtocol: &proto.Protocol{NumberOrName: &proto.Protocol_Name{Name: "tcp"}},
	})).To(Equal(true), "Unexpected result with NotProtocol")

	Expect(ruleHasNegativeMatches(&proto.Rule{
		Action:  "Allow",
		NotIcmp: &proto.Rule_NotIcmpType{NotIcmpType: 10},
	})).To(Equal(true), "Unexpected result with NotIcmp")

}

// Test of ProtocolNameToNumber()
func TestProtocolNameToNumber(t *testing.T) {

	RegisterTestingT(t)

	Expect(protocolNameToNumber("tcp")).To(Equal(uint16(6)), "Unexpected result for protocolNameToNumber with tcp")

	Expect(protocolNameToNumber("udp")).To(Equal(uint16(17)), "Unexpected result for protocolNameToNumber with udp")

	Expect(protocolNameToNumber("icmp")).To(Equal(uint16(1)), "Unexpected result for protocolNameToNumber with icmp")

	Expect(protocolNameToNumber("icmpv6")).To(Equal(uint16(58)), "Unexpected result for protocolNameToNumber with icmpv6")

	Expect(protocolNameToNumber("sctp")).To(Equal(uint16(132)), "Unexpected result for protocolNameToNumber with sctp")

	Expect(protocolNameToNumber("udplite")).To(Equal(uint16(136)), "Unexpected result for protocolNameToNumber with udplite")

	Expect(protocolNameToNumber("gre")).To(Equal(uint16(256)), "Unexpected result for protocolNameToNumber with any")

}

// Test of filterNets()
func TestFilterNets(t *testing.T) {

	RegisterTestingT(t)

	Expect(filterNets([]string{}, uint8(4))).To(Equal([]string(nil)), "Unexpected result for filterNets with empty argument")

	Expect(filterNets([]string{"10.0.0.1", "10.0.0.2", "0:0:0:0:0:ffff:af4:301"}, uint8(6))).To(Equal([]string{"0:0:0:0:0:ffff:af4:301"}), "Unexpected result for filterNets with ip v6 filtering")

	Expect(filterNets([]string{"10.0.0.1", "10.0.0.2", "0:0:0:0:0:ffff:af4:301"}, uint8(4))).To(Equal([]string{"10.0.0.1", "10.0.0.2"}), "Unexpected result for filterNets with ip v4 filtering")
}

type mockHNS struct {
	SupportedFeatures hns.HNSSupportedFeatures
}

func (h *mockHNS) GetHNSSupportedFeatures() hns.HNSSupportedFeatures {
	return h.SupportedFeatures
}

type mockIPSetCache struct {
	IPSets map[string][]string
}

func (c *mockIPSetCache) GetIPSetMembers(ipsetID string) []string {
	if len(c.IPSets[ipsetID]) == 0 {
		return nil
	}

	return c.IPSets[ipsetID]
}
