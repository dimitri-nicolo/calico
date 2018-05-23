// Copyright (c) 2016-2018 Tigera, Inc. All rights reserved.

package calc_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"

	"github.com/projectcalico/felix/dataplane/mock"
	"github.com/projectcalico/felix/proto"
	. "github.com/projectcalico/libcalico-go/lib/backend/model"
	calinet "github.com/projectcalico/libcalico-go/lib/net"
)

// Canned tiers/policies.

var tier1_order20 = Tier{
	Order: &order20,
}

// Pre-defined datastore states.  Each State object wraps up the complete state
// of the datastore as well as the expected state of the dataplane.  The state
// of the dataplane *should* depend only on the current datastore state, not on
// the path taken to get there.  Therefore, it's always a valid test to move
// from any state to any other state (by feeding in the corresponding
// datastore updates) and then assert that the dataplane matches the resulting
// state.

// withPolicyAndTier adds a tier and policy containing selectors for all and b=="b"
var withPolicyAndTier = initialisedStore.withKVUpdates(
	KVPair{Key: TierKey{"tier-1"}, Value: &tier1_order20},
	KVPair{Key: PolicyKey{Tier: "tier-1", Name: "pol-1"}, Value: &policy1_order20},
).withName("with policy")

// localEp1WithPolicyAndTier adds a local endpoint to the mix.  It matches all and b=="b".
var localEp1WithPolicyAndTier = withPolicyAndTier.withKVUpdates(
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
	proto.PolicyID{"tier-1", "pol-1"},
).withActiveProfiles(
	proto.ProfileID{"prof-1"},
	proto.ProfileID{"prof-2"},
	proto.ProfileID{"prof-missing"},
).withEndpoint(
	localWlEp1Id,
	[]mock.TierInfo{
		{"tier-1", []string{"pol-1"}, []string{"pol-1"}},
	},
).withName("ep1 local, policy")

var hostEp1WithPolicyAndTier = withPolicyAndTier.withKVUpdates(
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
	proto.PolicyID{"tier-1", "pol-1"},
).withActiveProfiles(
	proto.ProfileID{"prof-1"},
	proto.ProfileID{"prof-2"},
	proto.ProfileID{"prof-missing"},
).withEndpoint(
	hostEpWithNameId,
	[]mock.TierInfo{
		{"tier-1", []string{"pol-1"}, []string{"pol-1"}},
	},
).withName("host ep1, policy")

var hostEp2WithPolicyAndTier = withPolicyAndTier.withKVUpdates(
	KVPair{Key: hostEp2NoNameKey, Value: &hostEp2NoName},
).withIPSet(allSelectorId, []string{
	"10.0.0.2/32", // ep1 and ep2
	"fc00:fe11::2/128",
	"10.0.0.3/32", // ep2
	"fc00:fe11::3/128",
}).withIPSet(bEqBSelectorId, []string{}).withActivePolicies(
	proto.PolicyID{"tier-1", "pol-1"},
).withActiveProfiles(
	proto.ProfileID{"prof-2"},
	proto.ProfileID{"prof-3"},
).withEndpoint(
	hostEpNoNameId,
	[]mock.TierInfo{
		{"tier-1", []string{"pol-1"}, []string{"pol-1"}},
	},
).withName("host ep2, policy")

// Policy ordering tests.  We keep the names of the policies the same but we
// change their orders to check that order trumps name.
var commLocalEp1WithOneTierPolicy123 = commercialPolicyOrderState(
	[3]float64{order10, order20, order30},
	[3]string{"pol-1", "pol-2", "pol-3"},
)
var commLocalEp1WithOneTierPolicy321 = commercialPolicyOrderState(
	[3]float64{order30, order20, order10},
	[3]string{"pol-3", "pol-2", "pol-1"},
)
var commLocalEp1WithOneTierPolicyAlpha = commercialPolicyOrderState(
	[3]float64{order10, order10, order10},
	[3]string{"pol-1", "pol-2", "pol-3"},
)

func commercialPolicyOrderState(policyOrders [3]float64, expectedOrder [3]string) State {
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
		KVPair{Key: TierKey{"tier-1"}, Value: &tier1_order20},
		KVPair{Key: PolicyKey{Tier: "tier-1", Name: "pol-1"}, Value: &policies[0]},
		KVPair{Key: PolicyKey{Tier: "tier-1", Name: "pol-2"}, Value: &policies[1]},
		KVPair{Key: PolicyKey{Tier: "tier-1", Name: "pol-3"}, Value: &policies[2]},
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
		proto.PolicyID{"tier-1", "pol-1"},
		proto.PolicyID{"tier-1", "pol-2"},
		proto.PolicyID{"tier-1", "pol-3"},
	).withActiveProfiles(
		proto.ProfileID{"prof-1"},
		proto.ProfileID{"prof-2"},
		proto.ProfileID{"prof-missing"},
	).withEndpoint(
		localWlEp1Id,
		[]mock.TierInfo{
			{"tier-1", expectedOrder[:], expectedOrder[:]},
		},
	).withName(fmt.Sprintf("ep1 local, 1 tier, policies %v", expectedOrder[:]))
	return state
}

// Tier ordering tests.  We keep the names of the tiers constant but adjust
// their orders.
var localEp1WithTiers123 = tierOrderState(
	[3]float64{order10, order20, order30},
	[3]string{"tier-1", "tier-2", "tier-3"},
)
var localEp1WithTiers321 = tierOrderState(
	[3]float64{order30, order20, order10},
	[3]string{"tier-3", "tier-2", "tier-1"},
)

// These tests use the same order for each tier, checking that the name is
// used as a tie breaker.
var localEp1WithTiersAlpha = tierOrderState(
	[3]float64{order10, order10, order10},
	[3]string{"tier-1", "tier-2", "tier-3"},
)
var localEp1WithTiersAlpha2 = tierOrderState(
	[3]float64{order20, order20, order20},
	[3]string{"tier-1", "tier-2", "tier-3"},
)
var localEp1WithTiersAlpha3 = tierOrderState(
	[3]float64{order20, order20, order10},
	[3]string{"tier-3", "tier-1", "tier-2"},
)

func tierOrderState(tierOrders [3]float64, expectedOrder [3]string) State {
	tiers := [3]Tier{}
	for i := range tiers {
		tiers[i] = Tier{
			Order: &tierOrders[i],
		}
	}
	state := initialisedStore.withKVUpdates(
		KVPair{Key: localWlEpKey1, Value: &localWlEp1},
		KVPair{Key: TierKey{"tier-1"}, Value: &tiers[0]},
		KVPair{Key: PolicyKey{Tier: "tier-1", Name: "tier-1-pol"}, Value: &policy1_order20},
		KVPair{Key: TierKey{"tier-2"}, Value: &tiers[1]},
		KVPair{Key: PolicyKey{Tier: "tier-2", Name: "tier-2-pol"}, Value: &policy1_order20},
		KVPair{Key: TierKey{"tier-3"}, Value: &tiers[2]},
		KVPair{Key: PolicyKey{Tier: "tier-3", Name: "tier-3-pol"}, Value: &policy1_order20},
	).withIPSet(
		allSelectorId, ep1IPs,
	).withIPSet(
		bEqBSelectorId, ep1IPs,
	).withActivePolicies(
		proto.PolicyID{"tier-1", "tier-1-pol"},
		proto.PolicyID{"tier-2", "tier-2-pol"},
		proto.PolicyID{"tier-3", "tier-3-pol"},
	).withActiveProfiles(
		proto.ProfileID{"prof-1"},
		proto.ProfileID{"prof-2"},
		proto.ProfileID{"prof-missing"},
	).withEndpoint(
		localWlEp1Id,
		[]mock.TierInfo{
			{expectedOrder[0], []string{expectedOrder[0] + "-pol"}, []string{expectedOrder[0] + "-pol"}},
			{expectedOrder[1], []string{expectedOrder[1] + "-pol"}, []string{expectedOrder[1] + "-pol"}},
			{expectedOrder[2], []string{expectedOrder[2] + "-pol"}, []string{expectedOrder[2] + "-pol"}},
		},
	).withName(fmt.Sprintf("tier-order-state%v", expectedOrder[:]))
	return state
}

// localEp2WithPolicyAndTier adds a different endpoint that doesn't match b=="b".
// This tests an empty IP set.
var localEp2WithPolicyAndTier = withPolicyAndTier.withKVUpdates(
	KVPair{Key: localWlEpKey2, Value: &localWlEp2},
).withIPSet(allSelectorId, []string{
	"10.0.0.2/32", // ep1 and ep2
	"fc00:fe11::2/128",
	"10.0.0.3/32", // ep2
	"fc00:fe11::3/128",
}).withIPSet(
	bEqBSelectorId, []string{},
).withActivePolicies(
	proto.PolicyID{"tier-1", "pol-1"},
).withActiveProfiles(
	proto.ProfileID{"prof-2"},
	proto.ProfileID{"prof-3"},
).withEndpoint(
	localWlEp2Id,
	[]mock.TierInfo{
		{"tier-1", []string{"pol-1"}, []string{"pol-1"}},
	},
).withName("ep2 local, policy")

// localEpsWithPolicyAndTier contains both of the above endpoints, which have some
// overlapping IPs.  When we sequence this with the states above, we test
// overlapping IP addition and removal.
var localEpsWithPolicyAndTier = withPolicyAndTier.withKVUpdates(
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
	proto.PolicyID{"tier-1", "pol-1"},
).withActiveProfiles(
	proto.ProfileID{"prof-1"},
	proto.ProfileID{"prof-2"},
	proto.ProfileID{"prof-3"},
	proto.ProfileID{"prof-missing"},
).withEndpoint(
	localWlEp1Id,
	[]mock.TierInfo{
		{"tier-1", []string{"pol-1"}, []string{"pol-1"}},
	},
).withEndpoint(
	localWlEp2Id,
	[]mock.TierInfo{
		{"tier-1", []string{"pol-1"}, []string{"pol-1"}},
	},
).withName("2 local, overlapping IPs & a policy")

// One local endpoint with a host IP, should generate an IPsec binding for each IP of the endpoint.
var localEp1WithNode = localEp1WithPolicy.withKVUpdates(
	KVPair{Key: HostIPKey{Hostname: localHostname}, Value: calinet.ParseIP("192.168.0.1")},
).withIPSecBinding(
	"192.168.0.1", "10.0.0.1",
).withIPSecBinding(
	"192.168.0.1", "10.0.0.2",
).withName("Local endpoint 1 with a host IP")

var localEp1WithNodeDiffIP = localEp1WithPolicy.withKVUpdates(
	KVPair{Key: HostIPKey{Hostname: localHostname}, Value: calinet.ParseIP("192.168.0.2")},
).withIPSecBinding(
	"192.168.0.2", "10.0.0.1",
).withIPSecBinding(
	"192.168.0.2", "10.0.0.2",
).withName("Local endpoint 1 with a (different) host IP")

var localEp1WithNodesSharingIP = localEp1WithPolicy.withKVUpdates(
	KVPair{Key: HostIPKey{Hostname: localHostname}, Value: calinet.ParseIP("192.168.0.1")},
	KVPair{Key: HostIPKey{Hostname: remoteHostname}, Value: calinet.ParseIP("192.168.0.1")},
).withName("Local endpoint 1 with pair of hosts sharing IP")

const remoteHostname2 = "remotehostname2"

var localEp1With3NodesSharingIP = localEp1WithPolicy.withKVUpdates(
	KVPair{Key: HostIPKey{Hostname: localHostname}, Value: calinet.ParseIP("192.168.0.1")},
	KVPair{Key: HostIPKey{Hostname: remoteHostname}, Value: calinet.ParseIP("192.168.0.1")},
	KVPair{Key: HostIPKey{Hostname: remoteHostname2}, Value: calinet.ParseIP("192.168.0.1")},
).withName("Local endpoint 1 with triple of hosts sharing IP")

// Different local endpoint with a host IP, should generate an IPsec binding for each IP of the endpoint.
var localEp2WithNode = localEp2WithPolicy.withKVUpdates(
	KVPair{Key: HostIPKey{Hostname: localHostname}, Value: calinet.ParseIP("192.168.0.1")},
).withIPSecBinding(
	"192.168.0.1", "10.0.0.2",
).withIPSecBinding(
	"192.168.0.1", "10.0.0.3",
).withName("Local endpoint 2 with a host IP")

// Endpoint 2 using endpoint 1's key (so we can simulate changing an endpoint's IPs.
var localEp2AsEp1WithNode = localEp2WithNode.withKVUpdates(
	KVPair{Key: localWlEpKey2},
	KVPair{Key: localWlEpKey1, Value: &localWlEp2},
).withIPSecBinding(
	"192.168.0.1", "10.0.0.2",
).withIPSecBinding(
	"192.168.0.1", "10.0.0.3",
).withEndpoint(
	localWlEp1Id,
	[]mock.TierInfo{
		{"default", []string{"pol-1"}, []string{"pol-1"}},
	},
).withEndpoint(localWlEp2Id, nil).withName("Local endpoint 2 (using key for ep 1) with a host IP")

// Endpoint 1 and 2 sharing an IP with a node too.
var localWlEpKey3 = WorkloadEndpointKey{localHostname, "orch", "wl3", "ep3"}
var localWlEp3 = WorkloadEndpoint{
	State: "active",
	Name:  "cali3",
	IPv4Nets: []calinet.IPNet{
		mustParseNet("10.0.0.2/32"), // Shared with all endpoints
		mustParseNet("10.0.0.4/32"), // unique to this endpoint
	},
}

const localWlEp3Id = "orch/wl3/ep3"

var localEp1And2WithNode = localEpsWithPolicy.withKVUpdates(
	KVPair{Key: HostIPKey{Hostname: localHostname}, Value: calinet.ParseIP("192.168.0.1")},
).withIPSecBinding(
	"192.168.0.1", "10.0.0.1",
).withIPSecBinding(
	"192.168.0.1", "10.0.0.3",
).withName("Local endpoints 1 and 2 sharing an IP with a host IP defined")

// Endpoint 1, 2 and 3 sharing an IP with a node too.
var threeEndpointsSharingIPWithNode = localEpsWithPolicy.withKVUpdates(
	KVPair{Key: HostIPKey{Hostname: localHostname}, Value: calinet.ParseIP("192.168.0.1")},
	KVPair{Key: localWlEpKey1, Value: &localWlEp1},
	KVPair{Key: localWlEpKey2, Value: &localWlEp2},
	KVPair{Key: localWlEpKey3, Value: &localWlEp3},
).withIPSecBinding(
	"192.168.0.1", "10.0.0.1",
).withIPSecBinding(
	"192.168.0.1", "10.0.0.3",
).withIPSecBinding(
	"192.168.0.1", "10.0.0.4",
).withIPSet(allSelectorId, []string{
	"10.0.0.1/32", // ep1
	"fc00:fe11::1/128",
	"10.0.0.2/32", // ep1, ep2 and ep3
	"fc00:fe11::2/128",
	"10.0.0.3/32", // ep2
	"fc00:fe11::3/128",
	"10.0.0.4/32", // ep3
}).withEndpoint(
	localWlEp3Id,
	[]mock.TierInfo{},
).withName("3 endpoints sharing an IP with a host IP defined")

var threeEndpointsSharingIPWithDulicateNodeIP = localEpsWithPolicy.withKVUpdates(
	KVPair{Key: HostIPKey{Hostname: localHostname}, Value: calinet.ParseIP("192.168.0.1")},
	KVPair{Key: HostIPKey{Hostname: remoteHostname}, Value: calinet.ParseIP("192.168.0.1")},
	KVPair{Key: localWlEpKey1, Value: &localWlEp1},
	KVPair{Key: localWlEpKey2, Value: &localWlEp2},
	KVPair{Key: localWlEpKey3, Value: &localWlEp3},
).withIPSet(allSelectorId, []string{
	"10.0.0.1/32", // ep1
	"fc00:fe11::1/128",
	"10.0.0.2/32", // ep1, ep2 and ep3
	"fc00:fe11::2/128",
	"10.0.0.3/32", // ep2
	"fc00:fe11::3/128",
	"10.0.0.4/32", // ep3
}).withEndpoint(
	localWlEp3Id,
	[]mock.TierInfo{},
).withName("3 endpoints sharing an IP with a duplicate host IP defined")

var commercialTests = []StateList{
	// Empty should be empty!
	{},
	// Add one endpoint then remove it and add another with overlapping IP.
	{localEp1WithPolicyAndTier, localEp2WithPolicyAndTier},

	// Add one endpoint then another with an overlapping IP, then remove
	// first.
	{localEp1WithPolicyAndTier, localEpsWithPolicyAndTier, localEp2WithPolicyAndTier},

	// Add both endpoints, then return to empty, then add them both back.
	{localEpsWithPolicyAndTier, initialisedStore, localEpsWithPolicyAndTier},

	// Add a profile and a couple of endpoints.  Then update the profile to
	// use different tags and selectors.
	{localEpsWithProfile, localEpsWithUpdatedProfile},

	// Tests of policy ordering.  Each state has one tier but we shuffle
	// the order of the policies within it.
	{commLocalEp1WithOneTierPolicy123,
		commLocalEp1WithOneTierPolicy321,
		commLocalEp1WithOneTierPolicyAlpha},

	// Test mutating the profile list of some endpoints.
	{localEpsWithNonMatchingProfile, localEpsWithProfile},

	// And tier ordering.
	{localEp1WithTiers123,
		localEp1WithTiers321,
		localEp1WithTiersAlpha,
		localEp1WithTiersAlpha2,
		localEp1WithTiers321,
		localEp1WithTiersAlpha3},

	// String together some complex updates with profiles and policies
	// coming and going.
	{localEpsWithProfile,
		commLocalEp1WithOneTierPolicy123,
		localEp1WithTiers321,
		localEpsWithNonMatchingProfile,
		localEpsWithPolicyAndTier,
		localEpsWithUpdatedProfile,
		localEpsWithNonMatchingProfile,
		localEpsWithUpdatedProfileNegatedTags,
		localEp1WithPolicyAndTier,
		localEp1WithTiersAlpha2,
		localEpsWithProfile},

	// Host endpoint tests.
	{hostEp1WithPolicyAndTier, hostEp2WithPolicyAndTier},

	// IPsec basic tests.
	{localEp1WithNode},
	{localEp2WithNode},
	{localEp2AsEp1WithNode},

	// IPsec mutation tests (changing IPs etc)
	{localEp1WithNode, localEp2WithNode},
	{localEp1WithNode, localEp2AsEp1WithNode, localEp2WithNode},
	{localEp1WithNode, localEp1WithNodeDiffIP, localEp2AsEp1WithNode, localEp2WithNode},
	{localEp1WithNode, localEp2AsEp1WithNode, localEp1WithNodeDiffIP, localEp2WithNode},

	// IPSec Ambiguous binding tests: hosts sharing IP.
	{localEp1WithNodesSharingIP},
	{localEp1WithNode, localEp1WithNodesSharingIP, localEp1WithNode, localEp1WithNodesSharingIP},
	{localEp1WithNode, localEp1With3NodesSharingIP, localEp1WithNode},

	// IPSec ambiguous binding tests: endpoints sharing IP.
	{localEp1And2WithNode},
	{localEp1WithNode, localEp1And2WithNode, localEp1WithNode},
	{localEp1WithNode, localEp1And2WithNode, localEp2WithNode},
	{localEp1And2WithNode, localEp1WithNodesSharingIP, localEp1WithNode},
	{localEp1And2WithNode, localEp1WithNodesSharingIP, localEp2WithNode},
	{threeEndpointsSharingIPWithNode},
	{threeEndpointsSharingIPWithNode, localEp1And2WithNode, localEp1WithNode},
	{threeEndpointsSharingIPWithDulicateNodeIP, threeEndpointsSharingIPWithNode, localEp1And2WithNode},
	{threeEndpointsSharingIPWithDulicateNodeIP, localEp1WithNodesSharingIP, localEp1And2WithNode},

	// IPsec deletion tests (removing host IPs).
	{localEp1WithNode, localEp1WithPolicy},
	{localEp2WithNode, localEp2WithPolicy},

	// TODO(smc): Test config calculation
	// TODO(smc): Test mutation of endpoints
	// TODO(smc): Test mutation of host endpoints
	// TODO(smc): Test validation
	// TODO(smc): Test rule conversions
}

var _ = Describe("COMMERCIAL: Calculation graph state sequencing tests:", func() {
	describeSyncTests(commercialTests)
})
var _ = Describe("COMMERCIAL: Async calculation graph state sequencing tests:", func() {
	describeAsyncTests(commercialTests)
})

type tierInfo struct {
	Name               string
	IngressPolicyNames []string
	EgressPolicyNames  []string
}
