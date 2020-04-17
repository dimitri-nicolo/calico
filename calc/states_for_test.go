// Copyright (c) 2017-2020 Tigera, Inc. All rights reserved.

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

import (
	"fmt"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/dataplane/mock"
	"github.com/projectcalico/felix/proto"
	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	. "github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/net"
)

// Pre-defined datastore states.  Each State object wraps up the complete state
// of the datastore as well as the expected state of the dataplane.  The state
// of the dataplane *should* depend only on the current datastore state, not on
// the path taken to get there.  Therefore, it's always a valid test to move
// from any state to any other state (by feeding in the corresponding
// datastore updates) and then assert that the dataplane matches the resulting
// state.
//
// Notice that most of these pre-defined states are compounded. A small test
// might prefer to start with a simpler state instead.

// empty is the base state, with nothing in the datastore or dataplane.
var empty = NewState().withName("<empty>")

// initialisedStore builds on empty, adding in the ready flag and global config.
var initialisedStore = empty.withKVUpdates(
	KVPair{Key: GlobalConfigKey{Name: "InterfacePrefix"}, Value: "cali"},
	KVPair{Key: ReadyFlagKey{}, Value: true},
).withName("<initialised>")

// withPolicy adds a tier and policy containing selectors for all and b=="b"
var pol1KVPair = KVPair{Key: PolicyKey{Name: "pol-1", Tier: "default"}, Value: &policy1_order20}
var withPolicy = initialisedStore.withKVUpdates(
	pol1KVPair,
).withName("with policy")

// withPolicyIngressOnly adds a tier and ingress policy containing selectors for all
var withPolicyIngressOnly = initialisedStore.withKVUpdates(
	KVPair{Key: PolicyKey{Name: "pol-1", Tier: "default"}, Value: &policy1_order20_ingress_only},
).withName("with ingress-only policy")

// withPolicyEgressOnly adds a tier and egress policy containing selectors for b=="b"
var withPolicyEgressOnly = initialisedStore.withKVUpdates(
	KVPair{Key: PolicyKey{Name: "pol-1", Tier: "default"}, Value: &policy1_order20_egress_only},
).withName("with egress-only policy")

// withUntrackedPolicy adds a tier and policy containing selectors for all and b=="b"
var withUntrackedPolicy = initialisedStore.withKVUpdates(
	KVPair{Key: PolicyKey{Name: "pol-1", Tier: "default"}, Value: &policy1_order20_untracked},
).withName("with untracked policy")

// withPreDNATPolicy adds a tier and policy containing selectors for all and a=="a"
var withPreDNATPolicy = initialisedStore.withKVUpdates(
	KVPair{Key: PolicyKey{Name: "pre-dnat-pol-1", Tier: "default"}, Value: &policy1_order20_pre_dnat},
).withName("with pre-DNAT policy")

// withHttpMethodPolicy adds a policy containing http method selector.
var withHttpMethodPolicy = initialisedStore.withKVUpdates(
	KVPair{Key: PolicyKey{Name: "pol-1"}, Value: &policy1_order20_http_match},
).withTotalALPPolicies(
	1,
).withName("with http-method policy")

// DNS Policy state(s)
// withDNSPolicy tests the base use case of limiting egress traffic of a WEP to a single external domain.
var withDNSPolicy = initialisedStore.withKVUpdates(
	KVPair{Key: localWlEpKey1, Value: &localWlEpDNS},
	KVPair{Key: netSetDNSKey, Value: &netSetDNS},
	KVPair{Key: PolicyKey{Tier: "default", Name: "default.dns-basic"}, Value: &policyDNSBasic},
	KVPair{Key: PolicyKey{Tier: "default", Name: "default.ext-service"}, Value: &policyDNSExternal},
).withActivePolicies(
	proto.PolicyID{"default", "default.dns-basic"},
	proto.PolicyID{"default", "default.ext-service"},
).withEndpoint(
	localWlEp1Id,
	[]mock.TierInfo{
		{"default", nil, []string{"default.ext-service", "default.dns-basic"}},
	},
).withIPSet(allSelectorId, []string{
	"fc00:fe11::1/128",
	"fc00:fe11::2/128",
	"10.0.0.1/32",
	"10.0.0.2/32",
}).withIPSet(selectorIdDNSExternal, allowedEgressDomains).withIPSet(selectorIdDNSEmpty, []string{}).withName("with DNS Policy")

// Same as withDNSPolicy, but with no duplication in the network set domain names.
var withDNSPolicyNoDupe = withDNSPolicy.withKVUpdates(
	KVPair{Key: netSetDNSKey, Value: &netSetDNSNoDupe},
).withIPSet(selectorIdDNSExternal, allowedEgressDomainsNoDupe)

// withDNSPolicy2 verifies that when two GlobalNetworkSets, each with its own allowedEgressDomains, are selected by an appropriate
// GlobalNetworkPolicy, the resulting IPSet will contain the domain names from both Sets.
var withDNSPolicy2 = initialisedStore.withKVUpdates(
	KVPair{Key: localWlEpKey1, Value: &localWlEpDNS},
	KVPair{Key: netSetDNSKey, Value: &netSetDNS},
	KVPair{Key: netSetDNSKey2, Value: &netSetDNS2},
	KVPair{Key: PolicyKey{Tier: "default", Name: "default.dns-basic"}, Value: &policyDNSBasic},
	KVPair{Key: PolicyKey{Tier: "default", Name: "default.ext-service-2"}, Value: &policyDNSExternal2},
).withActivePolicies(
	proto.PolicyID{"default", "default.dns-basic"},
	proto.PolicyID{"default", "default.ext-service-2"},
).withEndpoint(
	localWlEp1Id,
	[]mock.TierInfo{
		{"default", nil, []string{"default.ext-service-2", "default.dns-basic"}},
	},
).withIPSet(allSelectorId, []string{
	"fc00:fe11::1/128",
	"fc00:fe11::2/128",
	"10.0.0.1/32",
	"10.0.0.2/32",
}).withIPSet(selectorIdDNSExternal2, append(allowedEgressDomains, allowedEgressDomains2...)).withIPSet(selectorIdDNSEmpty2, []string{}).withName("with DNS Policy 2")

// withDNSPolicy3 verifies that a GlobalNetworkSet with allowedEgressDomains and a Policy that matches the domains directly but
// without a selector will contain an IPSet with the correct domains.
var withDNSPolicy3 = initialisedStore.withKVUpdates(
	KVPair{Key: localWlEpKey1, Value: &localWlEpDNS},
	KVPair{Key: netSetDNSKey, Value: &netSetDNS},
	KVPair{Key: PolicyKey{Tier: "default", Name: "default.dns-basic"}, Value: &policyDNSBasic},
	KVPair{Key: PolicyKey{Tier: "default", Name: "default.destination-domains"}, Value: &policyDNSExternal3},
).withActivePolicies(
	proto.PolicyID{"default", "default.dns-basic"},
	proto.PolicyID{"default", "default.destination-domains"},
).withEndpoint(
	localWlEp1Id,
	[]mock.TierInfo{
		{"default", nil, []string{"default.destination-domains", "default.dns-basic"}},
	},
).withIPSet(allSelectorId, []string{
	"fc00:fe11::1/128",
	"fc00:fe11::2/128",
	"10.0.0.1/32",
	"10.0.0.2/32",
}).withIPSet(selectorIdDNSExternal3, allowedEgressDomains).withName("with DNS Policy 3")

// withServiceAccountPolicy adds two policies containing service account selector.
var withServiceAccountPolicy = initialisedStore.withKVUpdates(
	KVPair{Key: PolicyKey{Name: "pol-1"}, Value: &policy1_order20_src_service_account},
	KVPair{Key: PolicyKey{Name: "pol-2"}, Value: &policy1_order20_dst_service_account},
).withTotalALPPolicies(
	0,
).withName("with service-account policy")

// withNonALPPolicy adds a non ALP policy.
var withNonALPPolicy = withPolicy.withTotalALPPolicies(
	0,
).withName("with non-ALP policy")

// localEp1WithPolicy adds a local endpoint to the mix.  It matches all and b=="b".
var localEp1WithPolicy = withPolicy.withKVUpdates(
	KVPair{Key: localWlEpKey1, Value: &localWlEp1},
).withIPSet(allSelectorId, []string{
	"10.0.0.1/32", // ep1
	"fc00:fe11::1/128",
	"10.0.0.2/32", // ep1 and ep2
	"fc00:fe11::2/128",
}).withIPSet(bEqBSelectorId, []string{
	"10.0.0.1/32",
	"fc00:fe11::1/128",
	"10.0.0.2/32",
	"fc00:fe11::2/128",
}).withActivePolicies(
	proto.PolicyID{Tier: "default", Name: "pol-1"},
).withActiveProfiles(
	proto.ProfileID{Name: "prof-1"},
	proto.ProfileID{Name: "prof-2"},
	proto.ProfileID{Name: "prof-missing"},
).withEndpoint(
	localWlEp1Id,
	[]mock.TierInfo{
		{Name: "default", IngressPolicyNames: []string{"pol-1"}, EgressPolicyNames: []string{"pol-1"}},
	},
).withName("ep1 local, policy")

// localEp1WithNamedPortPolicy as above but with named port in the policy.
var localEp1WithNamedPortPolicy = localEp1WithPolicy.withKVUpdates(
	KVPair{Key: PolicyKey{Tier: "default", Name: "pol-1"}, Value: &policy1_order20_with_selector_and_named_port_tcpport},
).withIPSet(namedPortAllTCPID, []string{
	"10.0.0.1,tcp:8080",
	"10.0.0.2,tcp:8080",
	"fc00:fe11::1,tcp:8080",
	"fc00:fe11::2,tcp:8080",
}).withIPSet(allSelectorId, nil).withName("ep1 local, named port policy")

// localEp1WithNamedPortPolicy as above but with negated named port in the policy.
var localEp1WithNegatedNamedPortPolicy = empty.withKVUpdates(
	KVPair{Key: localWlEpKey1, Value: &localWlEp1},
	KVPair{Key: PolicyKey{Name: "pol-1", Tier: "default"}, Value: &policy1_order20_with_selector_and_negated_named_port_tcpport},
).withIPSet(namedPortAllLessFoobarTCPID, []string{
	"10.0.0.1,tcp:8080",
	"10.0.0.2,tcp:8080",
	"fc00:fe11::1,tcp:8080",
	"fc00:fe11::2,tcp:8080",
}).withIPSet(allLessFoobarSelectorId, []string{
	// The selector gets filled in because it's needed when doing the negation.
	"10.0.0.1/32",
	"10.0.0.2/32",
	"fc00:fe11::1/128",
	"fc00:fe11::2/128",
}).withActivePolicies(
	proto.PolicyID{Tier: "default", Name: "pol-1"},
).withActiveProfiles(
	proto.ProfileID{Name: "prof-1"},
	proto.ProfileID{Name: "prof-2"},
	proto.ProfileID{Name: "prof-missing"},
).withEndpoint(
	localWlEp1Id,
	[]mock.TierInfo{
		{
			Name:               "default",
			IngressPolicyNames: []string{"pol-1"},
		},
	},
).withName("ep1 local, negated named port policy")

// As above but using the destination fields in the policy instead of source.
var localEp1WithNegatedNamedPortPolicyDest = localEp1WithNegatedNamedPortPolicy.withKVUpdates(
	KVPair{
		Key:   PolicyKey{Name: "pol-1", Tier: "default"},
		Value: &policy1_order20_with_selector_and_negated_named_port_tcpport_dest,
	},
).withName("ep1 local, negated named port policy in destination fields")

// A host endpoint with a named port
var localHostEp1WithNamedPortPolicy = empty.withKVUpdates(
	KVPair{Key: hostEpWithNameKey, Value: &hostEpWithNamedPorts},
	KVPair{Key: PolicyKey{Tier: "default", Name: "pol-1"}, Value: &policy1_order20_with_selector_and_named_port_tcpport},
).withIPSet(namedPortAllTCPID, []string{
	"10.0.0.1,tcp:8080",
	"10.0.0.2,tcp:8080",
	"fc00:fe11::1,tcp:8080",
	"fc00:fe11::2,tcp:8080",
}).withIPSet(bEqBSelectorId, []string{
	"10.0.0.1/32",
	"fc00:fe11::1/128",
	"10.0.0.2/32",
	"fc00:fe11::2/128",
}).withActivePolicies(
	proto.PolicyID{Tier: "default", Name: "pol-1"},
).withActiveProfiles(
	proto.ProfileID{Name: "prof-1"},
).withEndpoint(
	"named",
	[]mock.TierInfo{
		{Name: "default", IngressPolicyNames: []string{"pol-1"}, EgressPolicyNames: []string{"pol-1"}},
	},
).withName("Host endpoint, named port policy")

// As above but with no selector in the rules.
var localEp1WithNamedPortPolicyNoSelector = localEp1WithNamedPortPolicy.withKVUpdates(
	KVPair{Key: PolicyKey{Tier: "default", Name: "pol-1"}, Value: &policy1_order20_with_named_port_tcpport},
).withName("ep1 local, named port only")

// As above but with negated named port.
var localEp1WithNegatedNamedPortPolicyNoSelector = localEp1WithNamedPortPolicy.withKVUpdates(
	KVPair{Key: PolicyKey{Tier: "default", Name: "pol-1"}, Value: &policy1_order20_with_named_port_tcpport_negated},
).withName("ep1 local, negated named port only")

// localEp1WithIngressPolicy is as above except ingress policy only.
var localEp1WithIngressPolicy = withPolicyIngressOnly.withKVUpdates(
	KVPair{Key: localWlEpKey1, Value: &localWlEp1},
).withIPSet(allSelectorId, []string{
	"10.0.0.1/32", // ep1
	"fc00:fe11::1/128",
	"10.0.0.2/32", // ep1 and ep2
	"fc00:fe11::2/128",
}).withActivePolicies(
	proto.PolicyID{Tier: "default", Name: "pol-1"},
).withActiveProfiles(
	proto.ProfileID{Name: "prof-1"},
	proto.ProfileID{Name: "prof-2"},
	proto.ProfileID{Name: "prof-missing"},
).withEndpoint(
	localWlEp1Id,
	[]mock.TierInfo{
		{Name: "default", IngressPolicyNames: []string{"pol-1"}, EgressPolicyNames: nil},
	},
).withName("ep1 local, ingress-only policy")

// localEp1WithNamedPortPolicy as above but with UDP named port in the policy.
var localEp1WithNamedPortPolicyUDP = localEp1WithPolicy.withKVUpdates(
	KVPair{Key: PolicyKey{Tier: "default", Name: "pol-1"}, Value: &policy1_order20_with_selector_and_named_port_udpport},
).withIPSet(namedPortAllUDPID, []string{
	"10.0.0.1,udp:9091",
	"10.0.0.2,udp:9091",
	"fc00:fe11::1,udp:9091",
	"fc00:fe11::2,udp:9091",
}).withIPSet(allSelectorId, nil).withName("ep1 local, named port policy")

var hostEp1WithPolicy = withPolicy.withKVUpdates(
	KVPair{Key: hostEpWithNameKey, Value: &hostEpWithName},
).withIPSet(allSelectorId, []string{
	"10.0.0.1/32", // ep1
	"fc00:fe11::1/128",
	"10.0.0.2/32", // ep1 and ep2
	"fc00:fe11::2/128",
}).withIPSet(bEqBSelectorId, []string{
	"10.0.0.1/32",
	"fc00:fe11::1/128",
	"10.0.0.2/32",
	"fc00:fe11::2/128",
}).withActivePolicies(
	proto.PolicyID{Tier: "default", Name: "pol-1"},
).withActiveProfiles(
	proto.ProfileID{Name: "prof-1"},
	proto.ProfileID{Name: "prof-2"},
	proto.ProfileID{Name: "prof-missing"},
).withEndpoint(
	hostEpWithNameId,
	[]mock.TierInfo{
		{Name: "default", IngressPolicyNames: []string{"pol-1"}, EgressPolicyNames: []string{"pol-1"}},
	},
).withName("host ep1, policy")

var hostEp1WithIngressPolicy = withPolicyIngressOnly.withKVUpdates(
	KVPair{Key: hostEpWithNameKey, Value: &hostEpWithName},
).withIPSet(allSelectorId, []string{
	"10.0.0.1/32", // ep1
	"fc00:fe11::1/128",
	"10.0.0.2/32", // ep1 and ep2
	"fc00:fe11::2/128",
}).withActivePolicies(
	proto.PolicyID{Tier: "default", Name: "pol-1"},
).withActiveProfiles(
	proto.ProfileID{Name: "prof-1"},
	proto.ProfileID{Name: "prof-2"},
	proto.ProfileID{Name: "prof-missing"},
).withEndpoint(
	hostEpWithNameId,
	[]mock.TierInfo{
		{Name: "default", IngressPolicyNames: []string{"pol-1"}, EgressPolicyNames: nil},
	},
).withName("host ep1, ingress-only policy")

var hostEp1WithEgressPolicy = withPolicyEgressOnly.withKVUpdates(
	KVPair{Key: hostEpWithNameKey, Value: &hostEpWithName},
).withIPSet(bEqBSelectorId, []string{
	"10.0.0.1/32",
	"fc00:fe11::1/128",
	"10.0.0.2/32",
	"fc00:fe11::2/128",
}).withActivePolicies(
	proto.PolicyID{Tier: "default", Name: "pol-1"},
).withActiveProfiles(
	proto.ProfileID{Name: "prof-1"},
	proto.ProfileID{Name: "prof-2"},
	proto.ProfileID{Name: "prof-missing"},
).withEndpoint(
	hostEpWithNameId,
	[]mock.TierInfo{
		{Name: "default", IngressPolicyNames: nil, EgressPolicyNames: []string{"pol-1"}},
	},
).withName("host ep1, egress-only policy")

var hostEp1WithUntrackedPolicy = withUntrackedPolicy.withKVUpdates(
	KVPair{Key: hostEpWithNameKey, Value: &hostEpWithName},
).withIPSet(allSelectorId, []string{
	"10.0.0.1/32", // ep1
	"fc00:fe11::1/128",
	"10.0.0.2/32", // ep1 and ep2
	"fc00:fe11::2/128",
}).withIPSet(bEqBSelectorId, []string{
	"10.0.0.1/32",
	"fc00:fe11::1/128",
	"10.0.0.2/32",
	"fc00:fe11::2/128",
}).withActivePolicies(
	proto.PolicyID{Tier: "default", Name: "pol-1"},
).withUntrackedPolicies(
	proto.PolicyID{Tier: "default", Name: "pol-1"},
).withActiveProfiles(
	proto.ProfileID{Name: "prof-1"},
	proto.ProfileID{Name: "prof-2"},
	proto.ProfileID{Name: "prof-missing"},
).withEndpointUntracked(
	hostEpWithNameId,
	[]mock.TierInfo{},
	[]mock.TierInfo{
		{Name: "default", IngressPolicyNames: []string{"pol-1"}, EgressPolicyNames: []string{"pol-1"}},
	},
	[]mock.TierInfo{},
).withName("host ep1, untracked policy")

var hostEp1WithPreDNATPolicy = withPreDNATPolicy.withKVUpdates(
	KVPair{Key: hostEpWithNameKey, Value: &hostEpWithName},
).withIPSet(allSelectorId, []string{
	"10.0.0.1/32", // ep1
	"fc00:fe11::1/128",
	"10.0.0.2/32", // ep1 and ep2
	"fc00:fe11::2/128",
}).withActivePolicies(
	proto.PolicyID{Tier: "default", Name: "pre-dnat-pol-1"},
).withPreDNATPolicies(
	proto.PolicyID{Tier: "default", Name: "pre-dnat-pol-1"},
).withActiveProfiles(
	proto.ProfileID{Name: "prof-1"},
	proto.ProfileID{Name: "prof-2"},
	proto.ProfileID{Name: "prof-missing"},
).withEndpointUntracked(
	hostEpWithNameId,
	[]mock.TierInfo{},
	[]mock.TierInfo{},
	[]mock.TierInfo{
		{Name: "default", IngressPolicyNames: []string{"pre-dnat-pol-1"}, EgressPolicyNames: nil},
	},
).withName("host ep1, pre-DNAT policy")

var hostEp1WithTrackedAndUntrackedPolicy = hostEp1WithUntrackedPolicy.withKVUpdates(
	KVPair{Key: PolicyKey{Name: "pol-2", Tier: "default"}, Value: &policy1_order20},
).withActivePolicies(
	proto.PolicyID{Tier: "default", Name: "pol-1"},
	proto.PolicyID{Tier: "default", Name: "pol-2"},
).withEndpointUntracked(
	hostEpWithNameId,
	[]mock.TierInfo{
		{Name: "default", IngressPolicyNames: []string{"pol-2"}, EgressPolicyNames: []string{"pol-2"}},
	},
	[]mock.TierInfo{
		{Name: "default", IngressPolicyNames: []string{"pol-1"}, EgressPolicyNames: []string{"pol-1"}},
	},
	[]mock.TierInfo{},
).withName("host ep1, tracked+untracked policy")

var hostEp2WithPolicy = withPolicy.withKVUpdates(
	KVPair{Key: hostEp2NoNameKey, Value: &hostEp2NoName},
).withIPSet(allSelectorId, []string{
	"10.0.0.2/32", // ep1 and ep2
	"fc00:fe11::2/128",
	"10.0.0.3/32", // ep2
	"fc00:fe11::3/128",
}).withIPSet(bEqBSelectorId, []string{}).withActivePolicies(
	proto.PolicyID{Tier: "default", Name: "pol-1"},
).withActiveProfiles(
	proto.ProfileID{Name: "prof-2"},
	proto.ProfileID{Name: "prof-3"},
).withEndpoint(
	hostEpNoNameId,
	[]mock.TierInfo{
		{Name: "default", IngressPolicyNames: []string{"pol-1"}, EgressPolicyNames: []string{"pol-1"}},
	},
).withName("host ep2, policy")

// Policy ordering tests.  We keep the names of the policies the same but we
// change their orders to check that order trumps name.
var localEp1WithOneTierPolicy123 = policyOrderState(
	[3]float64{order10, order20, order30},
	[3]string{"pol-1", "pol-2", "pol-3"},
)
var localEp1WithOneTierPolicy321 = policyOrderState(
	[3]float64{order30, order20, order10},
	[3]string{"pol-3", "pol-2", "pol-1"},
)
var localEp1WithOneTierPolicyAlpha = policyOrderState(
	[3]float64{order10, order10, order10},
	[3]string{"pol-1", "pol-2", "pol-3"},
)

func policyOrderState(policyOrders [3]float64, expectedOrder [3]string) State {
	policies := [3]Policy{}
	for i := range policies {
		policies[i] = Policy{
			Order:         &policyOrders[i],
			Selector:      "a == 'a'",
			InboundRules:  []Rule{{SrcSelector: allSelector}},
			OutboundRules: []Rule{{SrcSelector: bEpBSelector}},
		}
	}
	state := initialisedStore.withKVUpdates(
		KVPair{Key: localWlEpKey1, Value: &localWlEp1},
		KVPair{Key: PolicyKey{Name: "pol-1", Tier: "default"}, Value: &policies[0]},
		KVPair{Key: PolicyKey{Name: "pol-2", Tier: "default"}, Value: &policies[1]},
		KVPair{Key: PolicyKey{Name: "pol-3", Tier: "default"}, Value: &policies[2]},
	).withIPSet(allSelectorId, []string{
		"10.0.0.1/32", // ep1
		"fc00:fe11::1/128",
		"10.0.0.2/32", // ep1 and ep2
		"fc00:fe11::2/128",
	}).withIPSet(bEqBSelectorId, []string{
		"10.0.0.1/32",
		"fc00:fe11::1/128",
		"10.0.0.2/32",
		"fc00:fe11::2/128",
	}).withActivePolicies(
		proto.PolicyID{Tier: "default", Name: "pol-1"},
		proto.PolicyID{Tier: "default", Name: "pol-2"},
		proto.PolicyID{Tier: "default", Name: "pol-3"},
	).withActiveProfiles(
		proto.ProfileID{Name: "prof-1"},
		proto.ProfileID{Name: "prof-2"},
		proto.ProfileID{Name: "prof-missing"},
	).withEndpoint(
		localWlEp1Id,
		[]mock.TierInfo{
			{Name: "default", IngressPolicyNames: expectedOrder[:], EgressPolicyNames: expectedOrder[:]},
		},
	).withName(fmt.Sprintf("ep1 local, 1 tier, policies %v", expectedOrder[:]))
	return state
}

// localEp2WithPolicy adds a different endpoint that doesn't match b=="b".
// This tests an empty IP set.
var localEp2WithPolicy = withPolicy.withKVUpdates(
	KVPair{Key: localWlEpKey2, Value: &localWlEp2},
).withIPSet(allSelectorId, []string{
	"10.0.0.2/32", // ep1 and ep2
	"fc00:fe11::2/128",
	"10.0.0.3/32", // ep2
	"fc00:fe11::3/128",
}).withIPSet(
	bEqBSelectorId, []string{},
).withActivePolicies(
	proto.PolicyID{Tier: "default", Name: "pol-1"},
).withActiveProfiles(
	proto.ProfileID{Name: "prof-2"},
	proto.ProfileID{Name: "prof-3"},
).withEndpoint(
	localWlEp2Id,
	[]mock.TierInfo{
		{Name: "default", IngressPolicyNames: []string{"pol-1"}, EgressPolicyNames: []string{"pol-1"}},
	},
).withName("ep2 local, policy")

// localEpsWithPolicy contains both of the above endpoints, which have some
// overlapping IPs.  When we sequence this with the states above, we test
// overlapping IP addition and removal.
var localEpsWithPolicy = withPolicy.withKVUpdates(
	// Two local endpoints with overlapping IPs.
	KVPair{Key: localWlEpKey1, Value: &localWlEp1},
	KVPair{Key: localWlEpKey2, Value: &localWlEp2},
).withIPSet(allSelectorId, []string{
	"10.0.0.1/32", // ep1
	"fc00:fe11::1/128",
	"10.0.0.2/32", // ep1 and ep2
	"fc00:fe11::2/128",
	"10.0.0.3/32", // ep2
	"fc00:fe11::3/128",
}).withIPSet(bEqBSelectorId, []string{
	"10.0.0.1/32",
	"fc00:fe11::1/128",
	"10.0.0.2/32",
	"fc00:fe11::2/128",
}).withActivePolicies(
	proto.PolicyID{Tier: "default", Name: "pol-1"},
).withActiveProfiles(
	proto.ProfileID{Name: "prof-1"},
	proto.ProfileID{Name: "prof-2"},
	proto.ProfileID{Name: "prof-3"},
	proto.ProfileID{Name: "prof-missing"},
).withEndpoint(
	localWlEp1Id,
	[]mock.TierInfo{
		{Name: "default", IngressPolicyNames: []string{"pol-1"}, EgressPolicyNames: []string{"pol-1"}},
	},
).withEndpoint(
	localWlEp2Id,
	[]mock.TierInfo{
		{Name: "default", IngressPolicyNames: []string{"pol-1"}, EgressPolicyNames: []string{"pol-1"}},
	},
).withName("2 local, overlapping IPs & a policy")

var localEpsWithNamedPortsPolicy = localEpsWithPolicy.withKVUpdates(
	KVPair{Key: PolicyKey{Tier: "default", Name: "pol-1"}, Value: &policy1_order20_with_selector_and_named_port_tcpport},
).withIPSet(
	allSelectorId, nil,
).withIPSet(namedPortAllTCPID, []string{
	"10.0.0.1,tcp:8080", // ep1
	"fc00:fe11::1,tcp:8080",
	"10.0.0.2,tcp:8080", // ep1 and ep2
	"fc00:fe11::2,tcp:8080",
	"10.0.0.3,tcp:8080", // ep2
	"fc00:fe11::3,tcp:8080",
}).withName("2 local, overlapping IPs & a named port policy")

var localEpsWithNamedPortsPolicyTCPPort2 = localEpsWithPolicy.withKVUpdates(
	KVPair{Key: PolicyKey{Tier: "default", Name: "pol-1"}, Value: &policy1_order20_with_selector_and_named_port_tcpport2},
).withIPSet(
	allSelectorId, nil,
).withIPSet(namedPortAllTCP2ID, []string{
	"10.0.0.1,tcp:1234", // ep1
	"fc00:fe11::1,tcp:1234",

	"10.0.0.2,tcp:1234", // IP shared between ep1 and ep2 but different port no
	"10.0.0.2,tcp:2345",
	"fc00:fe11::2,tcp:1234",
	"fc00:fe11::2,tcp:2345",

	"10.0.0.3,tcp:2345", // ep2
	"fc00:fe11::3,tcp:2345",
}).withName("2 local, overlapping IPs & a named port policy")

// localEpsWithMismatchedNamedPortsPolicy contains a policy that has named port matches where the
// rule has a protocol that doesn't match that in the named port definitions in the endpoint.
var localEpsWithMismatchedNamedPortsPolicy = localEpsWithPolicy.withKVUpdates(
	KVPair{Key: PolicyKey{Tier: "default", Name: "pol-1"}, Value: &policy1_order20_with_named_port_mismatched_protocol},
).withIPSet(
	allSelectorId, nil,
).withIPSet(
	bEqBSelectorId, nil,
).withIPSet(
	namedPortID(allSelector, "udp", "tcpport"), []string{},
).withIPSet(
	namedPortID(allSelector, "tcp", "udpport"), []string{},
).withName("Named ports policy with protocol not matching endpoints")

// In this state, we have a couple of endpoints.  EP1 has a profile, through which it inherits
// a label.
var localEpsWithOverlappingIPsAndInheritedLabels = empty.withKVUpdates(
	// Two local endpoints with overlapping IPs.
	KVPair{Key: localWlEpKey1, Value: &localWlEp1},
	KVPair{Key: localWlEpKey2, Value: &localWlEp2},
	KVPair{Key: ProfileLabelsKey{ProfileKey: ProfileKey{Name: "prof-1"}}, Value: profileLabels1},
).withEndpoint(
	localWlEp1Id,
	[]mock.TierInfo{},
).withEndpoint(
	localWlEp2Id,
	[]mock.TierInfo{},
).withActiveProfiles(
	proto.ProfileID{Name: "prof-1"},
	proto.ProfileID{Name: "prof-2"},
	proto.ProfileID{Name: "prof-3"},
	proto.ProfileID{Name: "prof-missing"},
)

// Building on the above, we add a policy to match on the inherited label, which should produce
// a named port.
var localEpsAndNamedPortPolicyMatchingInheritedLabelOnEP1 = localEpsWithOverlappingIPsAndInheritedLabels.withKVUpdates(
	KVPair{Key: PolicyKey{Tier: "default", Name: "inherit-pol"}, Value: &policy_with_named_port_inherit},
).withActivePolicies(
	proto.PolicyID{Tier: "default", Name: "inherit-pol"},
).withEndpoint(
	localWlEp1Id,
	[]mock.TierInfo{{Name: "default",
		IngressPolicyNames: []string{"inherit-pol"},
		EgressPolicyNames:  []string{"inherit-pol"},
	}},
).withEndpoint(
	localWlEp2Id,
	[]mock.TierInfo{{Name: "default",
		IngressPolicyNames: []string{"inherit-pol"},
		EgressPolicyNames:  []string{"inherit-pol"},
	}},
).withIPSet(namedPortInheritIPSetID, []string{
	"10.0.0.1,tcp:8080", // ep1
	"fc00:fe11::1,tcp:8080",
	"10.0.0.2,tcp:8080", // ep1 and ep2
	"fc00:fe11::2,tcp:8080",
	// ep2 doesn't match because it doesn't inherit the profile.
}).withName("2 local WEPs with policy matching inherited label on WEP1")

// Add a second profile with the same labels so that both endpoints now match.
var localEpsAndNamedPortPolicyMatchingInheritedLabelBothEPs = localEpsAndNamedPortPolicyMatchingInheritedLabelOnEP1.withKVUpdates(
	KVPair{Key: ProfileLabelsKey{ProfileKey: ProfileKey{Name: "prof-2"}}, Value: profileLabels1},
).withIPSet(namedPortInheritIPSetID, []string{
	"10.0.0.1,tcp:8080", // ep1
	"fc00:fe11::1,tcp:8080",
	"10.0.0.2,tcp:8080", // ep1 and ep2
	"fc00:fe11::2,tcp:8080",
	"10.0.0.3,tcp:8080", // ep2
	"fc00:fe11::3,tcp:8080",
}).withName("2 local WEPs with policy matching inherited label on both WEPs")

// Adjust workload 1 so it has duplicate named ports.
var localEpsAndNamedPortPolicyDuplicatePorts = localEpsAndNamedPortPolicyMatchingInheritedLabelBothEPs.withKVUpdates(
	KVPair{Key: localWlEpKey1, Value: &localWlEp1WithDupeNamedPorts},
).withIPSet(namedPortInheritIPSetID, []string{
	"10.0.0.1,tcp:8080", // ep1
	"fc00:fe11::1,tcp:8080",
	"10.0.0.1,tcp:8081", // ep1
	"fc00:fe11::1,tcp:8081",
	"10.0.0.1,tcp:8082", // ep1
	"fc00:fe11::1,tcp:8082",
	"10.0.0.2,tcp:8081", // ep1
	"fc00:fe11::2,tcp:8081",
	"10.0.0.2,tcp:8082", // ep1
	"fc00:fe11::2,tcp:8082",

	"10.0.0.2,tcp:8080", // ep1 and ep2
	"fc00:fe11::2,tcp:8080",

	"10.0.0.3,tcp:8080", // ep2
	"fc00:fe11::3,tcp:8080",
}).withName("2 local WEPs with policy and duplicate named port on WEP1")

// Then, change the label on EP2 so it no-longer matches.
var localEpsAndNamedPortPolicyNoLongerMatchingInheritedLabelOnEP2 = localEpsAndNamedPortPolicyMatchingInheritedLabelBothEPs.withKVUpdates(
	KVPair{Key: ProfileLabelsKey{ProfileKey: ProfileKey{Name: "prof-2"}}, Value: profileLabels2},
).withIPSet(namedPortInheritIPSetID, []string{
	"10.0.0.1,tcp:8080", // ep1
	"fc00:fe11::1,tcp:8080",
	"10.0.0.2,tcp:8080", // ep1 and ep2
	"fc00:fe11::2,tcp:8080",
	// ep2 no longer matches
}).withName("2 local WEPs with policy matching inherited label on WEP1; WEP2 has different label")

// Then, change the label on EP1 so it no-longer matches.
var localEpsAndNamedPortPolicyNoLongerMatchingInheritedLabelOnEP1 = localEpsAndNamedPortPolicyNoLongerMatchingInheritedLabelOnEP2.withKVUpdates(
	KVPair{Key: ProfileLabelsKey{ProfileKey: ProfileKey{Name: "prof-1"}}, Value: profileLabels2},
).withIPSet(namedPortInheritIPSetID, []string{
	// No longer any matches.
}).withName("2 local WEPs with policy not matching inherited labels")

// Alternatively, prevent EP2 from matching by removing its profiles.
var localEpsAndNamedPortPolicyEP2ProfileRemoved = localEpsAndNamedPortPolicyMatchingInheritedLabelBothEPs.withKVUpdates(
	KVPair{Key: localWlEpKey2, Value: &localWlEp2WithLabelsButNoProfiles},
).withIPSet(namedPortInheritIPSetID, []string{
	"10.0.0.1,tcp:8080", // ep1
	"fc00:fe11::1,tcp:8080",
	"10.0.0.2,tcp:8080", // ep1 and ep2
	"fc00:fe11::2,tcp:8080",
	// ep2 no longer matches
}).withActiveProfiles(
	proto.ProfileID{Name: "prof-1"},
	proto.ProfileID{Name: "prof-2"},
	proto.ProfileID{Name: "prof-missing"},
).withName("2 local WEPs with policy matching inherited label on WEP1; WEP2 has no profile")

// Then do the same for EP1.
var localEpsAndNamedPortPolicyBothEPsProfilesRemoved = localEpsAndNamedPortPolicyEP2ProfileRemoved.withKVUpdates(
	KVPair{Key: localWlEpKey1, Value: &localWlEp1WithLabelsButNoProfiles},
).withIPSet(namedPortInheritIPSetID, []string{
	// Neither EP matches.
}).withActiveProfiles().withName("2 local WEPs with no matches due to removing profiles from endpoints")

// localEpsWithPolicyUpdatedIPs, when used with localEpsWithPolicy checks
// correct handling of IP address updates.  We add and remove some IPs from
// endpoint 1 and check that only its non-shared IPs are removed from the IP
// sets.
var localEpsWithPolicyUpdatedIPs = localEpsWithPolicy.withKVUpdates(
	KVPair{Key: localWlEpKey1, Value: &localWlEp1DifferentIPs},
	KVPair{Key: localWlEpKey2, Value: &localWlEp2},
).withIPSet(allSelectorId, []string{
	"11.0.0.1/32", // ep1
	"fc00:fe12::1/128",
	"11.0.0.2/32",
	"fc00:fe12::2/128",
	"10.0.0.2/32", // now ep2 only
	"fc00:fe11::2/128",
	"10.0.0.3/32", // ep2
	"fc00:fe11::3/128",
}).withIPSet(bEqBSelectorId, []string{
	"11.0.0.1/32", // ep1
	"fc00:fe12::1/128",
	"11.0.0.2/32",
	"fc00:fe12::2/128",
})

// withProfile adds a profile to the initialised state.
var withProfile = initialisedStore.withKVUpdates(
	KVPair{Key: ProfileRulesKey{ProfileKey: ProfileKey{Name: "prof-1"}}, Value: &profileRules1},
	KVPair{Key: ProfileTagsKey{ProfileKey: ProfileKey{Name: "prof-1"}}, Value: profileTags1},
	KVPair{Key: ProfileLabelsKey{ProfileKey: ProfileKey{Name: "prof-1"}}, Value: profileLabels1},
).withName("profile")

// localEpsWithProfile contains a pair of overlapping IP endpoints and a profile
// that matches them both.
var localEpsWithProfile = withProfile.withKVUpdates(
	// Two local endpoints with overlapping IPs.
	KVPair{Key: localWlEpKey1, Value: &localWlEp1},
	KVPair{Key: localWlEpKey2, Value: &localWlEp2},
).withIPSet(allSelectorId, []string{
	"10.0.0.1/32", // ep1
	"fc00:fe11::1/128",
	"10.0.0.2/32", // ep1 and ep2
	"fc00:fe11::2/128",
	"10.0.0.3/32", // ep2
	"fc00:fe11::3/128",
}).withIPSet(tag1LabelID, []string{
	"10.0.0.1/32",
	"fc00:fe11::1/128",
	"10.0.0.2/32",
	"fc00:fe11::2/128",
}).withActiveProfiles(
	proto.ProfileID{Name: "prof-1"},
	proto.ProfileID{Name: "prof-2"},
	proto.ProfileID{Name: "prof-3"},
	proto.ProfileID{Name: "prof-missing"},
).withEndpoint(
	localWlEp1Id,
	[]mock.TierInfo{},
).withEndpoint(
	localWlEp2Id,
	[]mock.TierInfo{},
).withName("2 local, overlapping IPs & a profile")

// localEpsWithNonMatchingProfile contains a pair of overlapping IP endpoints and a profile
// that matches them both.
var localEpsWithNonMatchingProfile = withProfile.withKVUpdates(
	// Two local endpoints with overlapping IPs.
	KVPair{Key: localWlEpKey1, Value: &localWlEp1NoProfiles},
	KVPair{Key: localWlEpKey2, Value: &localWlEp2NoProfiles},
).withEndpoint(
	localWlEp1Id,
	[]mock.TierInfo{},
).withEndpoint(
	localWlEp2Id,
	[]mock.TierInfo{},
).withName("2 local, overlapping IPs & a non-matching profile")

// localEpsWithUpdatedProfile Follows on from localEpsWithProfile, changing the
// profile to use a different tag and selector.
var localEpsWithUpdatedProfile = localEpsWithProfile.withKVUpdates(
	KVPair{Key: ProfileRulesKey{ProfileKey: ProfileKey{Name: "prof-1"}}, Value: &profileRules1TagUpdate},
).withIPSet(
	tag1LabelID, nil,
).withIPSet(
	allSelectorId, nil,
).withIPSet(bEqBSelectorId, []string{
	"10.0.0.1/32",
	"fc00:fe11::1/128",
	"10.0.0.2/32",
	"fc00:fe11::2/128",
}).withIPSet(
	tag2LabelID, []string{},
).withEndpoint(
	localWlEp1Id,
	[]mock.TierInfo{},
).withEndpoint(
	localWlEp2Id,
	[]mock.TierInfo{},
).withName("2 local, overlapping IPs & updated profile")

var localEpsWithUpdatedProfileNegatedTags = localEpsWithUpdatedProfile.withKVUpdates(
	KVPair{Key: ProfileRulesKey{ProfileKey: ProfileKey{Name: "prof-1"}}, Value: &profileRules1NegatedTagSelUpdate},
)

// withProfileTagInherit adds a profile that includes rules that match on
// tags as labels.  I.e. a tag of name foo should be equivalent to label foo=""
var withProfileTagInherit = initialisedStore.withKVUpdates(
	KVPair{Key: ProfileRulesKey{ProfileKey: ProfileKey{Name: "prof-1"}}, Value: &profileRulesWithTagInherit},
	KVPair{Key: ProfileTagsKey{ProfileKey: ProfileKey{Name: "prof-1"}}, Value: profileTags1},
	KVPair{Key: ProfileLabelsKey{ProfileKey: ProfileKey{Name: "prof-1"}}, Value: profileLabels1},
).withName("profile")

var localEpsWithTagInheritProfile = withProfileTagInherit.withKVUpdates(
	// Two local endpoints with overlapping IPs.
	KVPair{Key: localWlEpKey1, Value: &localWlEp1},
	KVPair{Key: localWlEpKey2, Value: &localWlEp2},
).withIPSet(tagSelectorId, []string{
	"10.0.0.1/32", // ep1
	"fc00:fe11::1/128",
	"10.0.0.2/32", // ep1 and ep2
	"fc00:fe11::2/128",
}).withIPSet(
	tagFoobarSelectorId, []string{},
).withActiveProfiles(
	proto.ProfileID{Name: "prof-1"},
	proto.ProfileID{Name: "prof-2"},
	proto.ProfileID{Name: "prof-3"},
	proto.ProfileID{Name: "prof-missing"},
).withEndpoint(
	localWlEp1Id, []mock.TierInfo{},
).withEndpoint(
	localWlEp2Id, []mock.TierInfo{},
).withName("2 local, overlapping IPs & a tag inherit profile")

var withProfileTagOverriden = initialisedStore.withKVUpdates(
	KVPair{Key: ProfileRulesKey{ProfileKey: ProfileKey{Name: "prof-1"}}, Value: &profileRulesWithTagInherit},
	KVPair{Key: ProfileTagsKey{ProfileKey: ProfileKey{Name: "prof-1"}}, Value: profileTags1},
	KVPair{Key: ProfileLabelsKey{ProfileKey: ProfileKey{Name: "prof-1"}}, Value: profileLabelsTag1},
).withName("profile")

// localEpsWithTagOverriddenProfile Checks that tags-inherited labels can be
// overridden by explicit labels on the profile.
var localEpsWithTagOverriddenProfile = withProfileTagOverriden.withKVUpdates(
	// Two local endpoints with overlapping IPs.
	KVPair{Key: localWlEpKey1, Value: &localWlEp1},
	KVPair{Key: localWlEpKey2, Value: &localWlEp2},
).withIPSet(tagSelectorId, []string{
	"10.0.0.1/32", // ep1
	"fc00:fe11::1/128",
	"10.0.0.2/32", // ep1 and ep2
	"fc00:fe11::2/128",
}).withIPSet(tagFoobarSelectorId, []string{
	"10.0.0.1/32", // ep1
	"fc00:fe11::1/128",
	"10.0.0.2/32", // ep1 and ep2
	"fc00:fe11::2/128",
}).withActiveProfiles(
	proto.ProfileID{Name: "prof-1"},
	proto.ProfileID{Name: "prof-2"},
	proto.ProfileID{Name: "prof-3"},
	proto.ProfileID{Name: "prof-missing"},
).withEndpoint(
	localWlEp1Id,
	[]mock.TierInfo{},
).withEndpoint(
	localWlEp2Id,
	[]mock.TierInfo{},
).withName("2 local, overlapping IPs & a tag inherit profile")

var hostEp1WithPolicyAndANetworkSet = hostEp1WithPolicy.withKVUpdates(
	KVPair{Key: netSet1Key, Value: &netSet1},
).withIPSet(allSelectorId, []string{
	"10.0.0.1/32", // ep1 and net set.
	"fc00:fe11::1/128",
	"10.0.0.2/32", // ep1 and ep2
	"fc00:fe11::2/128",
	"12.0.0.0/24",
	"12.1.0.0/24",
	"feed:beef::/32",
}).withIPSet(bEqBSelectorId, []string{
	"10.0.0.1/32",
	"fc00:fe11::1/128",
	"10.0.0.2/32",
	"fc00:fe11::2/128",
})

var hostEp1WithPolicyAndTwoNetworkSets = hostEp1WithPolicyAndANetworkSet.withKVUpdates(
	KVPair{Key: netSet2Key, Value: &netSet2},
).withIPSet(allSelectorId, []string{
	"10.0.0.1/32",
	"fc00:fe11::1/128",
	"10.0.0.2/32",
	"fc00:fe11::2/128",
	"12.0.0.0/24", // Shared by both net sets.
	"12.1.0.0/24",
	"feed:beef::/32",
	"13.1.0.0/24", // Unique to netset-2
}).withIPSet(bEqBSelectorId, []string{
	"10.0.0.1/32",
	"fc00:fe11::1/128",
	"10.0.0.2/32",
	"fc00:fe11::2/128",
})

var hostEp1WithPolicyAndANetworkSetMatchingBEqB = hostEp1WithPolicy.withKVUpdates(
	KVPair{Key: netSet1Key, Value: &netSet1WithBEqB},
).withIPSet(allSelectorId, []string{
	"10.0.0.1/32", // ep1 and net set.
	"fc00:fe11::1/128",
	"10.0.0.2/32", // ep1 and ep2
	"fc00:fe11::2/128",
	"12.0.0.0/24",
	"12.1.0.0/24",
}).withIPSet(bEqBSelectorId, []string{
	"10.0.0.1/32",
	"fc00:fe11::1/128",
	"10.0.0.2/32",
	"fc00:fe11::2/128",
	"12.0.0.0/24",
	"12.1.0.0/24",
})

// Minimal VXLAN set-up, all the data needed for a remote VTEP, a pool and a block.
var vxlanWithBlock = empty.withKVUpdates(
	KVPair{Key: ipPoolKey, Value: &ipPoolWithVXLAN},
	KVPair{Key: remoteIPAMBlockKey, Value: &remoteIPAMBlock},
	KVPair{Key: remoteHostIPKey, Value: &remoteHostIP},
	KVPair{Key: remoteHostVXLANTunnelConfigKey, Value: remoteHostVXLANTunnelIP},
).withName("VXLAN").withVTEPs(
	// VTEP for the remote node.
	proto.VXLANTunnelEndpointUpdate{
		Node:           remoteHostname,
		Mac:            "66:3e:ca:a4:db:65",
		Ipv4Addr:       remoteHostVXLANTunnelIP,
		ParentDeviceIp: remoteHostIP.String(),
	},
).withRoutes(
	// Single route for the block.
	proto.RouteUpdate{
		Node: remoteHostname,
		Dst:  "10.0.1.0/29",
		Gw:   remoteHostVXLANTunnelIP,
		Type: proto.RouteType_VXLAN,
	},
	proto.RouteUpdate{
		Node: remoteHostname,
		Dst:  "10.0.1.0/29",
		Gw:   remoteHostIP.String(),
		Type: proto.RouteType_WORKLOADS_NODE,
	},
)

// Minimal VXLAN set-up with a MAC address.
var vxlanWithMAC = vxlanWithBlock.withKVUpdates(
	KVPair{Key: remoteHostVXLANTunnelMACConfigKey, Value: remoteHostVXLANTunnelMAC},
).withName("VXLAN MAC").withVTEPs(
	// VTEP for the remote node.
	proto.VXLANTunnelEndpointUpdate{
		Node:           remoteHostname,
		Mac:            remoteHostVXLANTunnelMAC,
		Ipv4Addr:       remoteHostVXLANTunnelIP,
		ParentDeviceIp: remoteHostIP.String(),
	},
)

// As above but with a more complex block.  The block has some allocated IPs on the same
// node as well as one that's borrowed by a second node.  We add the extra VTEP config for the
// other node.
var vxlanWithBlockAndBorrows = vxlanWithBlock.withKVUpdates(
	KVPair{Key: remoteIPAMBlockKey, Value: &remoteIPAMBlockWithBorrows},
	KVPair{Key: remoteHost2IPKey, Value: &remoteHost2IP},
	KVPair{Key: remoteHost2VXLANTunnelConfigKey, Value: remoteHost2VXLANTunnelIP},
).withName("VXLAN borrow").withVTEPs(
	proto.VXLANTunnelEndpointUpdate{
		Node:           remoteHostname,
		Mac:            "66:3e:ca:a4:db:65",
		Ipv4Addr:       remoteHostVXLANTunnelIP,
		ParentDeviceIp: remoteHostIP.String(),
	},
	proto.VXLANTunnelEndpointUpdate{
		Node:           remoteHostname2,
		Mac:            "66:40:18:59:1f:16",
		Ipv4Addr:       remoteHost2VXLANTunnelIP,
		ParentDeviceIp: remoteHost2IP.String(),
	},
).withRoutes(
	proto.RouteUpdate{
		Type: proto.RouteType_VXLAN,
		Node: remoteHostname,
		Gw:   remoteHostVXLANTunnelIP,
		Dst:  "10.0.1.0/29",
	},
	proto.RouteUpdate{
		Type: proto.RouteType_VXLAN,
		Node: remoteHostname2,
		Gw:   remoteHost2VXLANTunnelIP,
		Dst:  "10.0.1.2/32",
	},
	proto.RouteUpdate{
		Type: proto.RouteType_WORKLOADS_NODE,
		Node: remoteHostname,
		Gw:   remoteHostIP.String(),
		Dst:  "10.0.1.0/29",
	},
	proto.RouteUpdate{
		Type: proto.RouteType_WORKLOADS_NODE,
		Node: remoteHostname2,
		Gw:   remoteHost2IP.String(),
		Dst:  "10.0.1.2/32",
	},
)

// vxlanWithBlock but with a different tunnel IP.
var vxlanWithBlockAndDifferentTunnelIP = vxlanWithBlock.withKVUpdates(
	KVPair{Key: remoteHostVXLANTunnelConfigKey, Value: remoteHostVXLANTunnelIP2},
).withName("VXLAN different tunnel IP").withVTEPs(
	// VTEP for the remote node.
	proto.VXLANTunnelEndpointUpdate{
		Node:           remoteHostname,
		Mac:            "66:3e:ca:a4:db:65",
		Ipv4Addr:       remoteHostVXLANTunnelIP2,
		ParentDeviceIp: remoteHostIP.String(),
	},
).withRoutes(
	// Single route for the block.
	proto.RouteUpdate{
		Node: remoteHostname,
		Dst:  "10.0.1.0/29",
		Gw:   remoteHostVXLANTunnelIP2,
		Type: proto.RouteType_VXLAN,
	},
	proto.RouteUpdate{
		Node: remoteHostname,
		Dst:  "10.0.1.0/29",
		Gw:   remoteHostIP.String(),
		Type: proto.RouteType_WORKLOADS_NODE,
	},
)

// vxlanWithBlock but with a different node IP.
var vxlanWithBlockAndDifferentNodeIP = vxlanWithBlock.withKVUpdates(
	KVPair{Key: remoteHostIPKey, Value: &remoteHost2IP},
).withName("VXLAN different node IP").withVTEPs(
	// VTEP for the remote node.
	proto.VXLANTunnelEndpointUpdate{
		Node:           remoteHostname,
		Mac:            "66:3e:ca:a4:db:65",
		Ipv4Addr:       remoteHostVXLANTunnelIP,
		ParentDeviceIp: remoteHost2IP.String(),
	},
).withRoutes(
	// Single route for the block.
	proto.RouteUpdate{
		Node: remoteHostname,
		Dst:  "10.0.1.0/29",
		Gw:   remoteHostVXLANTunnelIP,
		Type: proto.RouteType_VXLAN,
	},
	proto.RouteUpdate{
		Node: remoteHostname,
		Dst:  "10.0.1.0/29",
		Gw:   remoteHost2IP.String(),
		Type: proto.RouteType_WORKLOADS_NODE,
	},
)

// As above but with the owner of the block and the borrows switched.
var vxlanBlockOwnerSwitch = vxlanWithBlockAndBorrows.withKVUpdates(
	KVPair{Key: remoteIPAMBlockKey, Value: &remoteIPAMBlockWithBorrowsSwitched},
).withRoutes(
	proto.RouteUpdate{
		Type: proto.RouteType_VXLAN,
		Node: remoteHostname2,
		Gw:   remoteHost2VXLANTunnelIP,
		Dst:  "10.0.1.0/29",
	},
	proto.RouteUpdate{
		Type: proto.RouteType_VXLAN,
		Node: remoteHostname,
		Gw:   remoteHostVXLANTunnelIP,
		Dst:  "10.0.1.2/32",
	},
	proto.RouteUpdate{
		Type: proto.RouteType_WORKLOADS_NODE,
		Node: remoteHostname2,
		Gw:   remoteHost2IP.String(),
		Dst:  "10.0.1.0/29",
	},
	proto.RouteUpdate{
		Type: proto.RouteType_WORKLOADS_NODE,
		Node: remoteHostname,
		Gw:   remoteHostIP.String(),
		Dst:  "10.0.1.2/32",
	},
).withName("VXLAN owner switch")

// As above but with the owner of the block and the borrows switched.
var vxlanLocalBlockWithBorrows = empty.withKVUpdates(
	KVPair{Key: ipPoolKey, Value: &ipPoolWithVXLAN},

	KVPair{Key: localHostIPKey, Value: &localHostIP},
	KVPair{Key: localHostVXLANTunnelConfigKey, Value: localHostVXLANTunnelIP},

	KVPair{Key: remoteHostIPKey, Value: &remoteHostIP},
	KVPair{Key: remoteHostVXLANTunnelConfigKey, Value: remoteHostVXLANTunnelIP},

	KVPair{Key: localIPAMBlockKey, Value: &localIPAMBlockWithBorrows},
).withVTEPs(
	proto.VXLANTunnelEndpointUpdate{
		Node:           remoteHostname,
		Mac:            "66:3e:ca:a4:db:65",
		Ipv4Addr:       remoteHostVXLANTunnelIP,
		ParentDeviceIp: remoteHostIP.String(),
	},
	proto.VXLANTunnelEndpointUpdate{
		Node:           localHostname,
		Mac:            "66:48:f6:56:dc:f1",
		Ipv4Addr:       localHostVXLANTunnelIP,
		ParentDeviceIp: localHostIP.String(),
	},
).withRoutes(
	proto.RouteUpdate{
		Type: proto.RouteType_VXLAN,
		Node: remoteHostname,
		Gw:   remoteHostVXLANTunnelIP,
		Dst:  "10.0.0.2/32",
	},
	proto.RouteUpdate{
		Type: proto.RouteType_WORKLOADS_NODE,
		Node: remoteHostname,
		Gw:   remoteHostIP.String(),
		Dst:  "10.0.0.2/32",
	},
).withName("VXLAN local with borrows")

// vxlanWithBlockAndBorrows but missing hte VTEP information for the first host.
var vxlanWithBlockAndBorrowsAndMissingFirstVTEP = vxlanWithBlockAndBorrows.withKVUpdates(
	KVPair{Key: remoteHostIPKey, Value: nil},
).withName("VXLAN borrow missing VTEP").withVTEPs(
	proto.VXLANTunnelEndpointUpdate{
		Node:           remoteHostname2,
		Mac:            "66:40:18:59:1f:16",
		Ipv4Addr:       remoteHost2VXLANTunnelIP,
		ParentDeviceIp: remoteHost2IP.String(),
	},
).withRoutes(
	proto.RouteUpdate{
		Node: remoteHostname2,
		Dst:  "10.0.1.2/32",
		Gw:   remoteHost2VXLANTunnelIP,
		Type: proto.RouteType_VXLAN,
	},
	proto.RouteUpdate{
		Node: remoteHostname2,
		Dst:  "10.0.1.2/32",
		Gw:   remoteHost2IP.String(),
		Type: proto.RouteType_WORKLOADS_NODE,
	},
)

var vxlanToIPIPSwitch = vxlanWithBlock.withKVUpdates(
	KVPair{Key: ipPoolKey, Value: &ipPoolWithIPIP},
).withName("VXLAN switched to IPIP").withRoutes(
	// VXLAN route removed but still get the simple route.
	proto.RouteUpdate{
		Node: remoteHostname,
		Dst:  "10.0.1.0/29",
		Gw:   remoteHostIP.String(),
		Type: proto.RouteType_WORKLOADS_NODE,
	},
)

var vxlanBlockDelete = vxlanWithBlock.withKVUpdates(
	KVPair{Key: remoteIPAMBlockKey, Value: nil},
).withName("VXLAN block removed").withRoutes()

var vxlanHostIPDelete = vxlanWithBlock.withKVUpdates(
	KVPair{Key: remoteHostIPKey, Value: nil},
).withName("VXLAN host IP removed").withRoutes().withVTEPs()

var vxlanTunnelIPDelete = vxlanWithBlock.withKVUpdates(
	KVPair{Key: remoteHostVXLANTunnelConfigKey, Value: nil},
).withName("VXLAN host tunnel IP removed").withRoutes(
	// VXLAN route removed but still get the simple route.
	proto.RouteUpdate{
		Node: remoteHostname,
		Dst:  "10.0.1.0/29",
		Gw:   remoteHostIP.String(),
		Type: proto.RouteType_WORKLOADS_NODE,
	},
).withVTEPs()

// Egress IP.
var (
	egressSelector        = "egress-provider == 'true'"
	egressProfileSelector = "(projectcalico.org/namespace == 'egress') && (egress-provider == 'true')"
	gatewayKey            = WorkloadEndpointKey{
		Hostname:       remoteHostname,
		WorkloadID:     "gw1",
		EndpointID:     "ep1",
		OrchestratorID: "orch",
	}
	gatewayEndpoint = &WorkloadEndpoint{
		Name:     "gw1",
		IPv4Nets: []net.IPNet{mustParseNet("137.0.0.1/32")},
		Labels: map[string]string{
			"egress-provider":             "true",
			"projectcalico.org/namespace": "egress",
		},
	}

	endpointWithOwnEgressGatewayID = WorkloadEndpointKey{
		Hostname:       localHostname,
		WorkloadID:     "wep1o",
		EndpointID:     "ep1",
		OrchestratorID: "orch",
	}
	endpointWithOwnEgressGateway = initialisedStore.withKVUpdates(
		KVPair{
			Key: endpointWithOwnEgressGatewayID,
			Value: &WorkloadEndpoint{
				Name:           "wep1o",
				EgressSelector: egressSelector,
			},
		},
		KVPair{
			Key:   gatewayKey,
			Value: gatewayEndpoint,
		},
	).withRemoteEndpoint(
		&calc.EndpointData{
			Key:      gatewayKey,
			Endpoint: gatewayEndpoint,
		},
	).withEndpoint(
		"orch/wep1o/ep1",
		[]mock.TierInfo{},
	).withEndpointEgressIPSetID(
		"orch/wep1o/ep1",
		egressSelectorID(egressSelector),
	).withIPSet(egressSelectorID(egressSelector), []string{
		"137.0.0.1/32",
	},
	).withName("endpointWithOwnEgressGateway")

	endpointWithProfileEgressGatewayID = WorkloadEndpointKey{
		Hostname:       localHostname,
		WorkloadID:     "wep1p",
		EndpointID:     "ep1",
		OrchestratorID: "orch",
	}
	endpointWithProfileEgressGateway = initialisedStore.withKVUpdates(
		KVPair{
			Key: ResourceKey{Name: "egress", Kind: v3.KindProfile},
			Value: &v3.Profile{
				ObjectMeta: v1.ObjectMeta{
					Name: "egress",
				},
				Spec: v3.ProfileSpec{
					EgressGateway: &v3.EgressSpec{
						Selector: "egress-provider == 'true'",
					},
				},
			},
		},
		KVPair{
			Key: endpointWithProfileEgressGatewayID,
			Value: &WorkloadEndpoint{
				Name:       "wep1p",
				ProfileIDs: []string{"egress"},
			},
		},
		KVPair{
			Key:   gatewayKey,
			Value: gatewayEndpoint,
		},
	).withRemoteEndpoint(
		&calc.EndpointData{
			Key:      gatewayKey,
			Endpoint: gatewayEndpoint,
		},
	).withEndpoint(
		"orch/wep1p/ep1",
		[]mock.TierInfo{},
	).withEndpointEgressIPSetID(
		"orch/wep1p/ep1",
		egressSelectorID(egressProfileSelector),
	).withIPSet(egressSelectorID(egressProfileSelector), []string{
		"137.0.0.1/32",
	},
	).withActiveProfiles(
		proto.ProfileID{Name: "egress"},
	).withName("endpointWithProfileEgressGateway")

	endpointWithoutOwnEgressGateway = initialisedStore.withKVUpdates(
		KVPair{
			Key: endpointWithOwnEgressGatewayID,
			Value: &WorkloadEndpoint{
				Name: "wep1o",
			},
		},
		KVPair{
			Key:   gatewayKey,
			Value: gatewayEndpoint,
		},
	).withRemoteEndpoint(
		&calc.EndpointData{
			Key:      gatewayKey,
			Endpoint: gatewayEndpoint,
		},
	).withEndpoint(
		"orch/wep1o/ep1",
		[]mock.TierInfo{},
	).withName("endpointWithoutOwnEgressGateway")

	endpointWithoutProfileEgressGateway = initialisedStore.withKVUpdates(
		KVPair{
			Key: ResourceKey{Name: "egress", Kind: v3.KindProfile},
			Value: &v3.Profile{
				ObjectMeta: v1.ObjectMeta{
					Name: "egress",
				},
				Spec: v3.ProfileSpec{},
			},
		},
		KVPair{
			Key: endpointWithProfileEgressGatewayID,
			Value: &WorkloadEndpoint{
				Name:       "wep1p",
				ProfileIDs: []string{"egress"},
			},
		},
		KVPair{
			Key:   gatewayKey,
			Value: gatewayEndpoint,
		},
	).withRemoteEndpoint(
		&calc.EndpointData{
			Key:      gatewayKey,
			Endpoint: gatewayEndpoint,
		},
	).withEndpoint(
		"orch/wep1p/ep1",
		[]mock.TierInfo{},
	).withActiveProfiles(
		proto.ProfileID{Name: "egress"},
	).withName("endpointWithoutProfileEgressGateway")
)

type StateList []State

func (l StateList) String() string {
	names := make([]string, 0)
	for _, state := range l {
		names = append(names, state.String())
	}
	return "[" + strings.Join(names, ", ") + "]"
}

// identity is a test expander that returns the test unaltered.
func identity(baseTest StateList) (string, []StateList) {
	return "in normal ordering", []StateList{baseTest}
}

// reverseStateOrder returns a StateList containing the same states in
// reverse order.
func reverseStateOrder(baseTest StateList) (desc string, mappedTests []StateList) {
	desc = "with order of states reversed"
	palindrome := true
	mappedTest := StateList{}
	for ii := 0; ii < len(baseTest); ii++ {
		mappedTest = append(mappedTest, baseTest[len(baseTest)-ii-1])
		if &baseTest[len(baseTest)-1-ii] != &baseTest[ii] {
			palindrome = false
		}
	}
	if palindrome {
		// Test was a palindrome so there's no point in reversing it.
		return
	}
	mappedTests = []StateList{mappedTest}
	return
}

// reverseKVOrder returns a StateList containing the states in the same order
// but with their DataStore key order reversed.
func reverseKVOrder(baseTests StateList) (desc string, mappedTests []StateList) {
	desc = "with order of KVs reversed within each state"
	mappedTest := StateList{}
	for _, test := range baseTests {
		mappedState := test.Copy()
		state := mappedState.DatastoreState
		for ii := 0; ii < len(state)/2; ii++ {
			jj := len(state) - ii - 1
			state[ii], state[jj] = state[jj], state[ii]
		}
		mappedTest = append(mappedTest, mappedState)
	}
	mappedTests = []StateList{mappedTest}
	return
}

// insertEmpties inserts an empty state between each state in the base test.
func insertEmpties(baseTest StateList) (desc string, mappedTests []StateList) {
	desc = "with empty state inserted between each state"
	mappedTest := StateList{}
	first := true
	for _, state := range baseTest {
		if !first {
			mappedTest = append(mappedTest, empty)
		} else {
			first = false
		}
		mappedTest = append(mappedTest, state)
	}
	mappedTests = append(mappedTests, mappedTest)
	return
}

func splitStates(baseTest StateList) (desc string, mappedTests []StateList) {
	desc = "with individual states broken out"
	if len(baseTest) <= 1 {
		// No point in splitting a single-item test.
		return
	}
	for _, state := range baseTest {
		mappedTests = append(mappedTests, StateList{state})
	}
	return
}

// squash returns a StateList with all the states squashed into one (which may
// include some deletions in the DatastoreState.
func squashStates(baseTests StateList) (desc string, mappedTests []StateList) {
	mappedTest := StateList{}
	desc = "all states squashed into one"
	if len(baseTests) == 0 {
		return
	}
	kvs := make([]KVPair, 0)
	mappedState := baseTests[len(baseTests)-1].Copy()
	lastTest := empty
	for _, test := range baseTests {
		for _, update := range test.KVDeltas(lastTest) {
			kvs = append(kvs, update.KVPair)
		}
		lastTest = test
	}
	mappedState.DatastoreState = kvs
	mappedState.ExpectedEndpointPolicyOrder = lastTest.ExpectedEndpointPolicyOrder
	mappedState.Name = fmt.Sprintf("squashed(%v)", baseTests)
	mappedTest = append(mappedTest, mappedState)
	mappedTests = []StateList{mappedTest}
	return
}
