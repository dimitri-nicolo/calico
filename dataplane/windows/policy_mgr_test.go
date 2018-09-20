// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package windataplane

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/projectcalico/felix/dataplane/windows/hns"
	"github.com/projectcalico/felix/dataplane/windows/policysets"
	"github.com/projectcalico/felix/proto"
)

func TestPolicyManager(t *testing.T) {
	RegisterTestingT(t)

	h := mockHNS{}

	ipsc := mockIPSetCache{
		IPSets: map[string][]string{},
	}

	ps := policysets.NewPolicySets(&h, []policysets.IPSetCache{&ipsc})
	policyMgr := newPolicyManager(ps)

	//Apply policy update
	policyMgr.OnUpdate(&proto.ActivePolicyUpdate{
		Id: &proto.PolicyID{Name: "pol1", Tier: "tier1"},
		Policy: &proto.Policy{
			InboundRules: []*proto.Rule{
				{Action: "deny"},
			},
			OutboundRules: []*proto.Rule{
				{Action: "allow"},
			},
		},
	})

	//assertion for ingress rules
	Expect(ps.GetPolicySetRules([]string{"policy-pol1"}, true)).To(Equal([]*hns.ACLPolicy{
		//policy-pol1 deny rule should be present
		{Type: hns.ACL, Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1000},
		// Default deny rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
		// Default host/pod rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.In, RuleType: hns.Host, Priority: 100},
	}), "unexpected rules returned for ingress rules update for policy-pol1")

	//assertion for egress rules
	Expect(ps.GetPolicySetRules([]string{"policy-pol1"}, false)).To(Equal([]*hns.ACLPolicy{
		//policy-pol1 allow rule should be present
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.Out, RuleType: hns.Switch, Priority: 1000},
		// Default deny rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Block, Direction: hns.Out, RuleType: hns.Switch, Priority: 1001},
		// Default host/pod rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.Out, RuleType: hns.Host, Priority: 100},
	}), "unexpected rules returned for egress rules update for policy-pol1")

	//remove policy here
	policyMgr.OnUpdate(&proto.ActivePolicyRemove{
		Id: &proto.PolicyID{Name: "pol1", Tier: "tier1"},
	})

	Expect(ps.GetPolicySetRules([]string{"policy-pol1"}, true)).To(Equal([]*hns.ACLPolicy{
		// Default deny rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
		// Default host/pod rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.In, RuleType: hns.Host, Priority: 100},
	}), "unexpected rules returned after ActivePolicyRemove event for policy-pol1")

	//Apply profile update
	policyMgr.OnUpdate(&proto.ActiveProfileUpdate{
		Id: &proto.ProfileID{Name: "prof1"},
		Profile: &proto.Profile{
			InboundRules: []*proto.Rule{
				{Action: "deny"},
			},
			OutboundRules: []*proto.Rule{
				{Action: "allow"},
			},
		},
	})

	//assertion for ingress rules
	Expect(ps.GetPolicySetRules([]string{"profile-prof1"}, true)).To(Equal([]*hns.ACLPolicy{
		//profile-prof1 deny rule should be present
		{Type: hns.ACL, Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1000},
		// Default deny rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
		// Default host/pod rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.In, RuleType: hns.Host, Priority: 100},
	}), "unexpected rules returned for ingress rules update for profile-prof1")

	//assertion for egress rules
	Expect(ps.GetPolicySetRules([]string{"profile-prof1"}, false)).To(Equal([]*hns.ACLPolicy{
		//profile-pol1 allow rule should be present
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.Out, RuleType: hns.Switch, Priority: 1000},
		// Default deny rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Block, Direction: hns.Out, RuleType: hns.Switch, Priority: 1001},
		// Default host/pod rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.Out, RuleType: hns.Host, Priority: 100},
	}), "unexpected rules returned for egress rules update for profile-prof1")

	//remove profile update
	policyMgr.OnUpdate(&proto.ActiveProfileRemove{
		Id: &proto.ProfileID{Name: "prof1"},
	})

	Expect(ps.GetPolicySetRules([]string{"profile-prof1"}, true)).To(Equal([]*hns.ACLPolicy{
		// Default deny rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Block, Direction: hns.In, RuleType: hns.Switch, Priority: 1001},
		// Default host/pod rule.
		{Type: hns.ACL, Protocol: 256, Action: hns.Allow, Direction: hns.In, RuleType: hns.Host, Priority: 100},
	}), "unexpected rules returned after ActiveProfileRemove event for profile-prof1")

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
