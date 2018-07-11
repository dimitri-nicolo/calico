// Copyright (c) 2018 Tigera, Inc. All rights reserved.
// windows stub implementation for policy lookup cache

package calc

import (
	"strconv"
	"sync"

	"github.com/projectcalico/felix/rules"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/set"
)

// PolicyLookupsCache provides an API to lookup policy to NFLOG prefix mapping.
// To do this, the PolicyLookupsCache hooks into the calculation graph
// by handling callbacks for policy and profile updates.
type PolicyLookupsCache struct {
	nflogPrefixesPolicy  map[model.PolicyKey]set.Set
	nflogPrefixesProfile map[model.ProfileRulesKey]set.Set
	nflogPrefixHash      map[[64]byte]*RuleID
	nflogMutex           sync.RWMutex

	tierRefs map[string]int
}

func NewPolicyLookupsCache() *PolicyLookupsCache {
	return nil
}

func (pc *PolicyLookupsCache) OnPolicyActive(key model.PolicyKey, policy *model.Policy) {
	return
}

func (pc *PolicyLookupsCache) OnPolicyInactive(key model.PolicyKey) {
	return
}

func (pc *PolicyLookupsCache) OnProfileActive(key model.ProfileRulesKey, profile *model.ProfileRules) {
	return
}

func (pc *PolicyLookupsCache) OnProfileInactive(key model.ProfileRulesKey) {
	return
}

// GetRuleIDFromNFLOGPrefix returns the RuleID associated with the supplied NFLOG prefix.
func (pc *PolicyLookupsCache) GetRuleIDFromNFLOGPrefix(prefix [64]byte) *RuleID {
	pc.nflogMutex.RLock()
	defer pc.nflogMutex.RUnlock()
	return pc.nflogPrefixHash[prefix]
}

// RuleID contains the complete identifiers for a particular rule. This is a breakdown of the
// Felix v1 representation into the v3 representation used by the API and the collector.
type RuleID struct {
	// The tier name. If this is blank this represents a Profile backed rule.
	Tier string
	// The policy or profile name. This has the tier removed from the name. If this is blank, this represents
	// a "no match" rule. For k8s policies, this will be the full v3 name (knp.default.<k8s name>) - this avoids
	// name conflicts with Calico policies.
	Name string
	// The namespace. This is only non-blank for a NetworkPolicy type. For Tiers, GlobalNetworkPolicies and the
	// no match rules this will be blank.
	Namespace string
	// The rule direction.
	Direction rules.RuleDir
	// The index into the rule slice.
	Index int
	// A stringified version of the above index (stored to avoid frequent conversion)
	IndexStr string
	// The rule action.
	Action rules.RuleAction

	// Optimization so that the hot path doesn't need to create strings.
	dpName string
}

func NewRuleID(tier, policy, namespace string, ruleIndex int, ruleDirection rules.RuleDir, ruleAction rules.RuleAction) *RuleID {
	rid := &RuleID{
		Tier:      tier,
		Name:      policy,
		Namespace: namespace,
		Direction: ruleDirection,
		Index:     ruleIndex,
		IndexStr:  strconv.Itoa(ruleIndex),
		Action:    ruleAction,
	}
	rid.setDeniedPacketRuleName()
	return rid
}

func (r *RuleID) Equals(r2 *RuleID) bool {
	return r.Tier == r2.Tier &&
		r.Name == r2.Name &&
		r.Namespace == r2.Namespace &&
		r.Direction == r2.Direction &&
		r.Index == r2.Index &&
		r.Action == r2.Action
}

func (r *RuleID) IsNamespaced() bool {
	return len(r.Namespace) != 0
}

func (r *RuleID) IsProfile() bool {
	return len(r.Tier) == 0
}

func (r *RuleID) IsNoMatchRule() bool {
	return len(r.Name) == 0
}

// TierString returns either the Tier name or the Profile indication string.
func (r *RuleID) TierString() string {
	return r.Tier
}

// NameString returns either the resource name or the No-match indication string.
func (r *RuleID) NameString() string {
	return ""
}

func (r *RuleID) setDeniedPacketRuleName() {
	return
}

func (r *RuleID) GetDeniedPacketRuleName() error {
	return nil
}

// Dump returns the contents of important structures in the LookupManager used for
// logging purposes in the test code. This should not be used in any mainline code.
func (pc *PolicyLookupsCache) Dump() {
	return
}
