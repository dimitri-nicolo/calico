// Copyright (c) 2018-2019 Tigera, Inc. All rights reserved.

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

package checker

import (
	"testing"

	authz "github.com/envoyproxy/data-plane-api/envoy/service/auth/v2"
	. "github.com/onsi/gomega"

	"github.com/gogo/googleapis/google/rpc"

	"github.com/projectcalico/app-policy/policystore"
	"github.com/projectcalico/app-policy/proto"
)

// actionFromString should parse strings in case insensitive mode.
func TestActionFromString(t *testing.T) {
	RegisterTestingT(t)

	Expect(actionFromString("allow")).To(Equal(ALLOW))
	Expect(actionFromString("Allow")).To(Equal(ALLOW))
	Expect(actionFromString("deny")).To(Equal(DENY))
	Expect(actionFromString("Deny")).To(Equal(DENY))
	Expect(actionFromString("pass")).To(Equal(PASS))
	Expect(actionFromString("Pass")).To(Equal(PASS))
	Expect(actionFromString("log")).To(Equal(LOG))
	Expect(actionFromString("Log")).To(Equal(LOG))
	Expect(actionFromString("next-tier")).To(Equal(PASS))
	Expect(func() { actionFromString("no_match") }).To(Panic())
}

// A policy with no rules does not match.
func TestCheckPolicyNoRules(t *testing.T) {
	RegisterTestingT(t)

	policy := &proto.Policy{}
	store := policystore.NewPolicyStore()
	req := &authz.CheckRequest{Attributes: &authz.AttributeContext{
		Source: &authz.AttributeContext_Peer{
			Principal: "spiffe://cluster.local/ns/default/sa/steve",
		},
		Destination: &authz.AttributeContext_Peer{
			Principal: "spiffe://cluster.local/ns/default/sa/sue",
		},
	}}
	reqCache, err := NewRequestCache(store, req)
	Expect(err).To(Succeed())
	Expect(checkPolicy(policy, reqCache)).To(Equal(NO_MATCH))
}

// If rules exist, but none match, we should get NO_MATCH
// Rules that do match should return their Action.
// Log rules should continue processing.
func TestCheckPolicyRules(t *testing.T) {
	RegisterTestingT(t)

	policy := &proto.Policy{InboundRules: []*proto.Rule{
		{
			Action: "log",
			HttpMatch: &proto.HTTPMatch{
				Methods: []string{"GET", "POST"},
			},
		},
		{
			Action: "allow",
			HttpMatch: &proto.HTTPMatch{
				Methods: []string{"POST"},
			},
		},
		{
			Action: "deny",
			HttpMatch: &proto.HTTPMatch{
				Methods: []string{"GET"},
			},
		},
	}}
	req := &authz.CheckRequest{Attributes: &authz.AttributeContext{
		Source: &authz.AttributeContext_Peer{
			Principal: "spiffe://cluster.local/ns/default/sa/steve",
		},
		Destination: &authz.AttributeContext_Peer{
			Principal: "spiffe://cluster.local/ns/default/sa/sue",
		},
		Request: &authz.AttributeContext_Request{
			Http: &authz.AttributeContext_HttpRequest{Method: "HEAD"},
		},
	}}
	reqCache, err := NewRequestCache(policystore.NewPolicyStore(), req)
	Expect(err).To(Succeed())
	Expect(checkPolicy(policy, reqCache)).To(Equal(NO_MATCH))

	http := req.GetAttributes().GetRequest().GetHttp()
	http.Method = "POST"
	Expect(checkPolicy(policy, reqCache)).To(Equal(ALLOW))

	http.Method = "GET"
	Expect(checkPolicy(policy, reqCache)).To(Equal(DENY))
}

// If tiers have no ingress policies, we should not get NO_MATCH.
func TestCheckNoIngressPolicyRulesInTier(t *testing.T) {
	RegisterTestingT(t)

	store := policystore.NewPolicyStore()
	store.Endpoint = &proto.WorkloadEndpoint{
		Tiers: []*proto.TierInfo{
			{
				Name:           "tier1",
				EgressPolicies: []string{"policy1", "policy2"},
			},
		},
		ProfileIds: []string{"profile1"},
	}
	store.PolicyByID[proto.PolicyID{Tier: "tier1", Name: "policy1"}] = &proto.Policy{
		OutboundRules: []*proto.Rule{
			{
				Action: "allow",
			},
		},
	}
	store.PolicyByID[proto.PolicyID{Tier: "tier1", Name: "policy2"}] = &proto.Policy{
		OutboundRules: []*proto.Rule{
			{
				Action: "allow",
			},
		},
	}
	store.ProfileByID[proto.ProfileID{Name: "profile1"}] = &proto.Profile{
		InboundRules: []*proto.Rule{
			{
				Action:    "allow",
				HttpMatch: &proto.HTTPMatch{Methods: []string{"GET"}},
			},
		},
	}
	req := &authz.CheckRequest{Attributes: &authz.AttributeContext{
		Source: &authz.AttributeContext_Peer{
			Principal: "spiffe://cluster.local/ns/default/sa/steve",
		},
		Destination: &authz.AttributeContext_Peer{
			Principal: "spiffe://cluster.local/ns/default/sa/sue",
		},
		Request: &authz.AttributeContext_Request{
			Http: &authz.AttributeContext_HttpRequest{Method: "GET"},
		},
	}}
	Expect(checkTiers(store, req)).To(Equal(rpc.Status{Code: OK}))
}

// CheckStore when the store has no endpoint should deny requests.
func TestCheckStoreNoEndpoint(t *testing.T) {
	RegisterTestingT(t)

	store := policystore.NewPolicyStore()
	req := &authz.CheckRequest{Attributes: &authz.AttributeContext{
		Source: &authz.AttributeContext_Peer{
			Principal: "spiffe://cluster.local/ns/default/sa/steve",
		},
		Destination: &authz.AttributeContext_Peer{
			Principal: "spiffe://cluster.local/ns/default/sa/steve",
		},
		Request: &authz.AttributeContext_Request{
			Http: &authz.AttributeContext_HttpRequest{Method: "HEAD"},
		},
	}}
	status := checkStore(store, req)
	Expect(status.Code).To(Equal(PERMISSION_DENIED))
}

// CheckStore with no Tiers and no Profiles on the endpoint should deny.
func TestCheckStoreNoTiers(t *testing.T) {
	RegisterTestingT(t)

	store := policystore.NewPolicyStore()
	store.Endpoint = &proto.WorkloadEndpoint{
		Tiers: []*proto.TierInfo{},
	}
	req := &authz.CheckRequest{Attributes: &authz.AttributeContext{
		Source: &authz.AttributeContext_Peer{
			Principal: "spiffe://cluster.local/ns/default/sa/steve",
		},
		Destination: &authz.AttributeContext_Peer{
			Principal: "spiffe://cluster.local/ns/default/sa/steve",
		},
		Request: &authz.AttributeContext_Request{
			Http: &authz.AttributeContext_HttpRequest{Method: "HEAD"},
		},
	}}
	status := checkStore(store, req)
	Expect(status.Code).To(Equal(PERMISSION_DENIED))
}

// If a Policy matches, the action on the matched rule is the result.
func TestCheckStorePolicyMatch(t *testing.T) {
	RegisterTestingT(t)

	store := policystore.NewPolicyStore()
	store.Endpoint = &proto.WorkloadEndpoint{
		Tiers: []*proto.TierInfo{
			{
				Name:            "tier1",
				IngressPolicies: []string{"policy1", "policy2"},
			},
		},
	}
	store.PolicyByID[proto.PolicyID{Tier: "tier1", Name: "policy1"}] = &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action:    "deny",
				HttpMatch: &proto.HTTPMatch{Methods: []string{"HEAD"}},
			},
		},
	}
	store.PolicyByID[proto.PolicyID{Tier: "tier1", Name: "policy2"}] = &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action:    "allow",
				HttpMatch: &proto.HTTPMatch{Methods: []string{"GET"}},
			},
		},
	}

	req := &authz.CheckRequest{Attributes: &authz.AttributeContext{
		Source: &authz.AttributeContext_Peer{
			Principal: "spiffe://cluster.local/ns/default/sa/steve",
		},
		Destination: &authz.AttributeContext_Peer{
			Principal: "spiffe://cluster.local/ns/default/sa/sally",
		},
		Request: &authz.AttributeContext_Request{
			Http: &authz.AttributeContext_HttpRequest{Method: "GET"},
		},
	}}

	status := checkStore(store, req)
	Expect(status.Code).To(Equal(OK))

	http := req.GetAttributes().GetRequest().GetHttp()
	http.Method = "HEAD"

	status = checkStore(store, req)
	Expect(status.Code).To(Equal(PERMISSION_DENIED))
}

// And endpoint with no Tiers should evaluate Profiles.
func TestCheckStoreProfileOnly(t *testing.T) {
	RegisterTestingT(t)

	store := policystore.NewPolicyStore()
	store.Endpoint = &proto.WorkloadEndpoint{
		Tiers:      []*proto.TierInfo{},
		ProfileIds: []string{"profile1", "profile2"},
	}
	store.ProfileByID[proto.ProfileID{Name: "profile1"}] = &proto.Profile{
		InboundRules: []*proto.Rule{
			{
				Action:    "Deny",
				HttpMatch: &proto.HTTPMatch{Methods: []string{"HEAD"}},
			},
		},
	}
	store.ProfileByID[proto.ProfileID{Name: "profile2"}] = &proto.Profile{
		InboundRules: []*proto.Rule{
			{
				Action:    "allow",
				HttpMatch: &proto.HTTPMatch{Methods: []string{"GET"}},
			},
		},
	}

	req := &authz.CheckRequest{Attributes: &authz.AttributeContext{
		Source: &authz.AttributeContext_Peer{
			Principal: "spiffe://cluster.local/ns/default/sa/steve",
		},
		Destination: &authz.AttributeContext_Peer{
			Principal: "spiffe://cluster.local/ns/default/sa/quinn",
		},
		Request: &authz.AttributeContext_Request{
			Http: &authz.AttributeContext_HttpRequest{Method: "GET"},
		},
	}}

	status := checkStore(store, req)
	Expect(status.Code).To(Equal(OK))

	http := req.GetAttributes().GetRequest().GetHttp()
	http.Method = "HEAD"

	status = checkStore(store, req)
	Expect(status.Code).To(Equal(PERMISSION_DENIED))
}

// And endpoint with a Tier should not evaluate profiles; there is a default deny on the tier.
func TestCheckStorePolicyDefaultDeny(t *testing.T) {
	RegisterTestingT(t)

	store := policystore.NewPolicyStore()
	store.Endpoint = &proto.WorkloadEndpoint{
		Tiers: []*proto.TierInfo{
			{
				Name:            "tier1",
				IngressPolicies: []string{"policy1"},
			},
		},
		ProfileIds: []string{"profile1"},
	}
	store.PolicyByID[proto.PolicyID{Tier: "tier1", Name: "policy1"}] = &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action:    "deny",
				HttpMatch: &proto.HTTPMatch{Methods: []string{"HEAD"}},
			},
		},
	}
	store.ProfileByID[proto.ProfileID{Name: "profile1"}] = &proto.Profile{
		InboundRules: []*proto.Rule{
			{
				Action:    "allow",
				HttpMatch: &proto.HTTPMatch{Methods: []string{"GET"}},
			},
		},
	}

	req := &authz.CheckRequest{Attributes: &authz.AttributeContext{
		Source: &authz.AttributeContext_Peer{
			Principal: "spiffe://cluster.local/ns/default/sa/steve",
		},
		Destination: &authz.AttributeContext_Peer{
			Principal: "spiffe://cluster.local/ns/default/sa/quinn",
		},
		Request: &authz.AttributeContext_Request{
			Http: &authz.AttributeContext_HttpRequest{Method: "GET"},
		},
	}}

	status := checkStore(store, req)
	Expect(status.Code).To(Equal(PERMISSION_DENIED))
}

// Ensure policy action of "Pass" ends policy evaluation and moves to profiles.
func TestCheckStorePass(t *testing.T) {
	RegisterTestingT(t)

	store := policystore.NewPolicyStore()
	store.Endpoint = &proto.WorkloadEndpoint{
		Tiers: []*proto.TierInfo{{
			Name:            "tier1",
			IngressPolicies: []string{"policy1", "policy2"},
		}},
		ProfileIds: []string{"profile1"},
	}

	// Policy1 matches and has action PASS, which means policy2 is not evaluated.
	store.PolicyByID[proto.PolicyID{Tier: "tier1", Name: "policy1"}] = &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action:    "next-tier",
				HttpMatch: &proto.HTTPMatch{Methods: []string{"GET"}},
			},
		},
	}
	store.PolicyByID[proto.PolicyID{Tier: "tier1", Name: "policy2"}] = &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action:    "deny",
				HttpMatch: &proto.HTTPMatch{Methods: []string{"GET"}},
			},
		},
	}

	// Profile1 matches and allows the traffic.
	store.ProfileByID[proto.ProfileID{Name: "profile1"}] = &proto.Profile{
		InboundRules: []*proto.Rule{
			{
				Action:    "allow",
				HttpMatch: &proto.HTTPMatch{Methods: []string{"HEAD", "GET"}},
			},
		},
	}

	req := &authz.CheckRequest{Attributes: &authz.AttributeContext{
		Source: &authz.AttributeContext_Peer{
			Principal: "spiffe://cluster.local/ns/default/sa/steve",
		},
		Destination: &authz.AttributeContext_Peer{
			Principal: "spiffe://cluster.local/ns/default/sa/molly",
		},
		Request: &authz.AttributeContext_Request{
			Http: &authz.AttributeContext_HttpRequest{Method: "GET"},
		},
	}}

	status := checkStore(store, req)
	Expect(status.Code).To(Equal(OK))
}

func TestCheckStoreInitFails(t *testing.T) {
	RegisterTestingT(t)

	store := policystore.NewPolicyStore()
	store.Endpoint = &proto.WorkloadEndpoint{
		Tiers: []*proto.TierInfo{{
			Name:            "tier1",
			IngressPolicies: []string{"policy1", "policy2"},
		}},
		ProfileIds: []string{"profile1"},
	}

	req := &authz.CheckRequest{Attributes: &authz.AttributeContext{
		Source: &authz.AttributeContext_Peer{
			Principal: "spiffe://malformed",
		},
		Request: &authz.AttributeContext_Request{
			Http: &authz.AttributeContext_HttpRequest{Method: "GET"},
		},
	}}

	//Check that we get PERMISSION_DENIED for DROP and LOG_AND_DROP values
	// for DropActionOverride, and OK for ACCEPT and LOG_AND_ACCEPT. Default
	// value is DROP.
	Expect(store.DropActionOverride).To(Equal(policystore.DROP))
	status := checkStore(store, req)
	Expect(status.Code).To(Equal(PERMISSION_DENIED))

	store.DropActionOverride = policystore.LOG_AND_DROP
	status = checkStore(store, req)
	Expect(status.Code).To(Equal(PERMISSION_DENIED))

	store.DropActionOverride = policystore.ACCEPT
	status = checkStore(store, req)
	Expect(status.Code).To(Equal(OK))

	store.DropActionOverride = policystore.LOG_AND_ACCEPT
	status = checkStore(store, req)
	Expect(status.Code).To(Equal(OK))
}

// Ensure checkStore returns INVALID_ARGUMENT on invalid input
func TestCheckStoreWithInvalidData(t *testing.T) {
	RegisterTestingT(t)

	store := policystore.NewPolicyStore()
	store.Endpoint = &proto.WorkloadEndpoint{
		Tiers: []*proto.TierInfo{{
			Name:            "tier1",
			IngressPolicies: []string{"policy1", "policy2"},
		}},
		ProfileIds: []string{"profile1"},
	}
	store.PolicyByID[proto.PolicyID{Tier: "tier1", Name: "policy1"}] = &proto.Policy{InboundRules: []*proto.Rule{
		{
			Action: "allow",
			HttpMatch: &proto.HTTPMatch{
				Methods: []string{"GET", "POST"},
				Paths: []*proto.HTTPMatch_PathMatch{
					{PathMatch: &proto.HTTPMatch_PathMatch_Exact{Exact: "/foo"}}},
			},
		},
	}}
	req := &authz.CheckRequest{Attributes: &authz.AttributeContext{
		Source: &authz.AttributeContext_Peer{
			Principal: "spiffe://cluster.local/ns/default/sa/steve",
		},
		Destination: &authz.AttributeContext_Peer{
			Principal: "spiffe://cluster.local/ns/default/sa/sue",
		},
		Request: &authz.AttributeContext_Request{
			// the path is invalid data as it does not have the `/' prefix
			Http: &authz.AttributeContext_HttpRequest{Method: "GET", Path: "foo"},
		},
	}}

	// Check that we get INVALID_ARGUMENT for DROP and LOG_AND_DROP values
	// for DropActionOverride, and OK for ACCEPT and LOG_AND_ACCEPT.
	Expect(store.DropActionOverride).To(Equal(policystore.DROP))
	status := checkStore(store, req)
	Expect(status.Code).To(Equal(INVALID_ARGUMENT))

	store.DropActionOverride = policystore.LOG_AND_DROP
	status = checkStore(store, req)
	Expect(status.Code).To(Equal(INVALID_ARGUMENT))

	store.DropActionOverride = policystore.ACCEPT
	status = checkStore(store, req)
	Expect(status.Code).To(Equal(OK))

	store.DropActionOverride = policystore.LOG_AND_ACCEPT
	status = checkStore(store, req)
	Expect(status.Code).To(Equal(OK))
}

// Check multiple tiers with next-tier (pass) to next tier and match the action on the matched rule in the next tier is the result.
func TestCheckStorePolicyMultiTierMatch(t *testing.T) {
	RegisterTestingT(t)

	store := policystore.NewPolicyStore()
	store.Endpoint = &proto.WorkloadEndpoint{
		Tiers: []*proto.TierInfo{
			{
				Name:            "tier1",
				IngressPolicies: []string{"policy1"},
			},
			{
				Name:            "tier2",
				IngressPolicies: []string{"policy2", "policy3"},
			},
		},
	}
	store.PolicyByID[proto.PolicyID{Tier: "tier1", Name: "policy1"}] = &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action:    "next-tier",
				HttpMatch: &proto.HTTPMatch{Methods: []string{"GET", "HEAD"}},
			},
		},
	}
	store.PolicyByID[proto.PolicyID{Tier: "tier2", Name: "policy2"}] = &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action: "deny",
				HttpMatch: &proto.HTTPMatch{
					Paths: []*proto.HTTPMatch_PathMatch{{PathMatch: &proto.HTTPMatch_PathMatch_Exact{Exact: "/bad"}}},
				},
			},
		},
	}
	store.PolicyByID[proto.PolicyID{Tier: "tier2", Name: "policy3"}] = &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action: "allow",
				HttpMatch: &proto.HTTPMatch{
					Paths: []*proto.HTTPMatch_PathMatch{{PathMatch: &proto.HTTPMatch_PathMatch_Exact{Exact: "/foo"}}},
				},
			},
		},
	}

	req := &authz.CheckRequest{Attributes: &authz.AttributeContext{
		Source: &authz.AttributeContext_Peer{
			Principal: "spiffe://cluster.local/ns/default/sa/steve",
		},
		Destination: &authz.AttributeContext_Peer{
			Principal: "spiffe://cluster.local/ns/default/sa/sally",
		},
		Request: &authz.AttributeContext_Request{
			Http: &authz.AttributeContext_HttpRequest{Method: "GET", Path: "/foo"},
		},
	}}

	// Check request is OK for all values of DropActionOverride.
	Expect(store.DropActionOverride).To(Equal(policystore.DROP))
	status := checkStore(store, req)
	Expect(status.Code).To(Equal(OK))

	store.DropActionOverride = policystore.LOG_AND_DROP
	status = checkStore(store, req)
	Expect(status.Code).To(Equal(OK))

	store.DropActionOverride = policystore.ACCEPT
	status = checkStore(store, req)
	Expect(status.Code).To(Equal(OK))

	store.DropActionOverride = policystore.LOG_AND_ACCEPT
	status = checkStore(store, req)
	Expect(status.Code).To(Equal(OK))

	// Change to a bad path, and check that we get PERMISSION_DENIED for
	// DROP and LOG_AND_DROP values for DropActionOverride, and OK for
	// ACCEPT and LOG_AND_ACCEPT.
	http := req.GetAttributes().GetRequest().GetHttp()
	http.Path = "/bad"

	store.DropActionOverride = policystore.DROP
	status = checkStore(store, req)
	Expect(status.Code).To(Equal(PERMISSION_DENIED))

	store.DropActionOverride = policystore.LOG_AND_DROP
	status = checkStore(store, req)
	Expect(status.Code).To(Equal(PERMISSION_DENIED))

	store.DropActionOverride = policystore.ACCEPT
	status = checkStore(store, req)
	Expect(status.Code).To(Equal(OK))

	store.DropActionOverride = policystore.LOG_AND_ACCEPT
	status = checkStore(store, req)
	Expect(status.Code).To(Equal(OK))
}

// Check multiple tiers with next-tier (pass) or deny in first tier and an allow in next tier.
func TestCheckStorePolicyMultiTierDiffTierMatch(t *testing.T) {
	RegisterTestingT(t)

	store := policystore.NewPolicyStore()
	store.Endpoint = &proto.WorkloadEndpoint{
		Tiers: []*proto.TierInfo{
			{
				Name:            "tier1",
				IngressPolicies: []string{"policy1", "policy2"},
			},
			{
				Name:            "tier2",
				IngressPolicies: []string{"policy3"},
			},
		},
	}
	store.PolicyByID[proto.PolicyID{Tier: "tier1", Name: "policy1"}] = &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action:    "deny",
				HttpMatch: &proto.HTTPMatch{Methods: []string{"HEAD"}},
			},
		},
	}
	store.PolicyByID[proto.PolicyID{Tier: "tier1", Name: "policy2"}] = &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action:    "next-tier",
				HttpMatch: &proto.HTTPMatch{Methods: []string{"GET"}},
			},
		},
	}
	store.PolicyByID[proto.PolicyID{Tier: "tier2", Name: "policy3"}] = &proto.Policy{
		InboundRules: []*proto.Rule{
			{
				Action: "allow",
				HttpMatch: &proto.HTTPMatch{
					Methods: []string{"GET"},
					Paths:   []*proto.HTTPMatch_PathMatch{{PathMatch: &proto.HTTPMatch_PathMatch_Exact{Exact: "/foo"}}},
				},
			},
		},
	}

	req := &authz.CheckRequest{Attributes: &authz.AttributeContext{
		Source: &authz.AttributeContext_Peer{
			Principal: "spiffe://cluster.local/ns/default/sa/steve",
		},
		Destination: &authz.AttributeContext_Peer{
			Principal: "spiffe://cluster.local/ns/default/sa/sally",
		},
		Request: &authz.AttributeContext_Request{
			Http: &authz.AttributeContext_HttpRequest{Method: "HEAD", Path: "/foo"},
		},
	}}

	status := checkStore(store, req)
	Expect(status.Code).To(Equal(PERMISSION_DENIED))

	http := req.GetAttributes().GetRequest().GetHttp()
	http.Method = "GET"

	status = checkStore(store, req)
	Expect(status.Code).To(Equal(OK))
}
