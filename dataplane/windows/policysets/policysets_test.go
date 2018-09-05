// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package policysets

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/projectcalico/felix/dataplane/windows/hns"
	"github.com/projectcalico/felix/proto"
)

func TestRuleRendering(t *testing.T) {
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

	ps := NewPolicySets(&h, []IPSetCache{&ipsc})

	// Unknown policy should result in default drop.
	Expect(ps.GetPolicySetRules([]string{"unknown"}, true)).To(Equal([]*hns.ACLPolicy{
		// Default deny rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
		// Default host/pod rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.In, RuleType: hns.Host},
	}), "unexpected rules returned for unknown policy")

	// Empty policy should return no rules (apart from the default drop).
	ps.AddOrReplacePolicySet("empty", &proto.Policy{
		InboundRules:  []*proto.Rule{},
		OutboundRules: []*proto.Rule{},
	})

	Expect(ps.GetPolicySetRules([]string{"empty"}, true)).To(Equal([]*hns.ACLPolicy{
		// Default deny rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
		// Default host/pod rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.In, RuleType: hns.Host},
	}), "unexpected rules returned for empty policy")

	// Tests of basic policy matches: CIDRs, protocol, ports.
	ps.AddOrReplacePolicySet("basic", &proto.Policy{
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
				Protocol: &proto.Protocol{NumberOrName: &proto.Protocol_Name{Name: "udp"}},
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

	Expect(ps.GetPolicySetRules([]string{"basic"}, true)).To(Equal([]*hns.ACLPolicy{
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch,
			Priority:        1000,
			Protocol:        6,
			Id:              "basic-rule-1-0",
			RemoteAddresses: "10.0.0.0/24",
			RemotePort:      1234,
			LocalPort:       80,
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch,
			Priority: 1000,
			Protocol: 17,
			Id:       "basic-rule-2-0",
		},
		{
			Type: hns.ACL, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch,
			Priority:       1001, // Switch from Allow to Deny triggers increment to priority.
			Protocol:       256,
			Id:             "basic-rule-3-0",
			LocalAddresses: "10.0.0.0/24",
		},
		{
			Type: hns.ACL, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch,
			Priority:       1001, // No change of action so priority stays the same.
			Protocol:       256,
			Id:             "basic-rule-4-0",
			LocalAddresses: "11.0.0.0/24",
		},
		// Default deny rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1002},
		// Default host/pod rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.In, RuleType: hns.Host},
	}), "unexpected rules returned for basic policy")

	// Tests that look up an IP set.
	ps.AddOrReplacePolicySet("selector", &proto.Policy{
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

	Expect(ps.GetPolicySetRules([]string{"selector"}, true)).To(Equal([]*hns.ACLPolicy{
		// We expect the source/dest IP sets to be expressed as the cross product.
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id: "selector-rule-1-0", RemoteAddresses: "10.0.0.1",
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id: "selector-rule-1-1", RemoteAddresses: "10.0.0.2",
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id: "selector-rule-1-2", RemoteAddresses: "10.0.0.2", // Note: no deduplication yet.
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id: "selector-rule-1-3", RemoteAddresses: "10.0.0.3",
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id: "selector-rule-2-0", LocalAddresses: "10.1.0.1",
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id: "selector-rule-2-1", LocalAddresses: "10.1.0.2",
		},
		// Default deny rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
		// Default host/pod rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.In, RuleType: hns.Host},
	}), "unexpected rules returned for selector cross-product policy")

	// Source and dest IP sets should be expressed as a cross-product.
	ps.AddOrReplacePolicySet("selector-cp", &proto.Policy{
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

	Expect(ps.GetPolicySetRules([]string{"selector-cp"}, true)).To(Equal([]*hns.ACLPolicy{
		// We expect the source/dest IP sets to be expressed as the cross product.
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id: "selector-cp-rule-1-0", LocalAddresses: "10.1.0.1", RemoteAddresses: "10.0.0.1",
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id: "selector-cp-rule-1-1", LocalAddresses: "10.1.0.1", RemoteAddresses: "10.0.0.2",
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id: "selector-cp-rule-1-2", LocalAddresses: "10.1.0.2", RemoteAddresses: "10.0.0.1",
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id: "selector-cp-rule-1-3", LocalAddresses: "10.1.0.2", RemoteAddresses: "10.0.0.2",
		},
		// Default deny rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
		// Default host/pod rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.In, RuleType: hns.Host},
	}), "unexpected rules returned for selector cross-product policy")

	// The source IP set should be intersected with the source CIDR.
	ps.AddOrReplacePolicySet("selector-cidr", &proto.Policy{
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

	Expect(ps.GetPolicySetRules([]string{"selector-cidr"}, true)).To(Equal([]*hns.ACLPolicy{
		// Intersection with first CIDR, picks up some IPs from each IP set.
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id: "selector-cidr-rule-1-0", RemoteAddresses: "10.0.0.1/32",
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id: "selector-cidr-rule-1-1", RemoteAddresses: "10.0.0.2/32",
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id: "selector-cidr-rule-1-2", RemoteAddresses: "10.0.0.3/32",
		},

		// Intersection with second CIDr picks up only one IP.
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id: "selector-cidr-rule-2-0", RemoteAddresses: "10.1.0.1/32",
		},

		// Intersection with both picks up everything.
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id: "selector-cidr-rule-3-0", RemoteAddresses: "10.0.0.1/32",
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id: "selector-cidr-rule-3-1", RemoteAddresses: "10.0.0.2/32",
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id: "selector-cidr-rule-3-2", RemoteAddresses: "10.0.0.3/32",
		},
		{
			Type: hns.ACL, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000, Protocol: 256,
			Id: "selector-cidr-rule-3-3", RemoteAddresses: "10.1.0.1/32",
		},

		// Rule 4 becomes a no-op since intersection is empty.

		// Default deny rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
		// Default host/pod rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.In, RuleType: hns.Host},
	}), "unexpected rules returned for selector CIDR filtering policy")
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

	ps := NewPolicySets(&h, []IPSetCache{&ipsc})

	// Empty policy should return no rules (apart from the default drop).
	ps.AddOrReplacePolicySet("allow", &proto.Policy{
		InboundRules: []*proto.Rule{{Action: "Allow"}},
	})
	ps.AddOrReplacePolicySet("deny", &proto.Policy{
		InboundRules: []*proto.Rule{{Action: "Deny"}},
	})

	Expect(ps.GetPolicySetRules([]string{"allow", "deny"}, true)).To(Equal([]*hns.ACLPolicy{
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000,
			Id: "allow--0"},
		{Type: hns.ACL, Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001,
			Id: "deny--0"},
		// Default deny rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1002},
		// Default host/pod rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.In, RuleType: hns.Host},
	}), "incorrect rules returned for allow,deny")

	Expect(ps.GetPolicySetRules([]string{"allow", "allow"}, true)).To(Equal([]*hns.ACLPolicy{
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000,
			Id: "allow--0"},
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1000,
			Id: "allow--0"},
		// Default deny rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
		// Default host/pod rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.In, RuleType: hns.Host},
	}), "incorrect rules returned for allow,allow")

	Expect(ps.GetPolicySetRules([]string{"deny", "allow"}, true)).To(Equal([]*hns.ACLPolicy{
		{Type: hns.ACL, Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1000,
			Id: "deny--0"},
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.In, RuleType: hns.Switch, Priority: 1001,
			Id: "allow--0"},
		// Default deny rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1002},
		// Default host/pod rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.In, RuleType: hns.Host},
	}), "incorrect rules returned for deny,allow")
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
	return c.IPSets[ipsetID]
}
