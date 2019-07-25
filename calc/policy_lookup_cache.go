// Copyright (c) 2018-2019 Tigera, Inc. All rights reserved.

package calc

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"

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
	pc := &PolicyLookupsCache{
		nflogPrefixesPolicy:  map[model.PolicyKey]set.Set{},
		nflogPrefixesProfile: map[model.ProfileRulesKey]set.Set{},
		nflogPrefixHash:      map[[64]byte]*RuleID{},
		nflogMutex:           sync.RWMutex{},
		tierRefs:             map[string]int{},
	}
	// Add NFLog mappings for the no-profile match.
	pc.addNFLogPrefixEntry(
		rules.CalculateNoMatchProfileNFLOGPrefixStr(rules.RuleDirIngress),
		NewRuleID("", "", "", 0, rules.RuleDirIngress, rules.RuleActionDeny),
	)

	pc.addNFLogPrefixEntry(
		rules.CalculateNoMatchProfileNFLOGPrefixStr(rules.RuleDirEgress),
		NewRuleID("", "", "", 0, rules.RuleDirEgress, rules.RuleActionDeny),
	)
	return pc
}

func (pc *PolicyLookupsCache) OnPolicyActive(key model.PolicyKey, policy *model.Policy) {
	pc.updatePolicyRulesNFLOGPrefixes(key, policy)
}

func (pc *PolicyLookupsCache) OnPolicyInactive(key model.PolicyKey) {
	pc.removePolicyRulesNFLOGPrefixes(key)
}

func (pc *PolicyLookupsCache) OnProfileActive(key model.ProfileRulesKey, profile *model.ProfileRules) {
	pc.updateProfileRulesNFLOGPrefixes(key, profile)
}

func (pc *PolicyLookupsCache) OnProfileInactive(key model.ProfileRulesKey) {
	pc.removeProfileRulesNFLOGPrefixes(key)
}

// addNFLogPrefixEntry adds a single NFLOG prefix entry to our internal cache.
func (pc *PolicyLookupsCache) addNFLogPrefixEntry(prefix string, ruleIDs *RuleID) {
	var bph [64]byte
	copy(bph[:], []byte(prefix[:]))
	pc.nflogMutex.Lock()
	defer pc.nflogMutex.Unlock()
	pc.nflogPrefixHash[bph] = ruleIDs
}

// deleteNFLogPrefixEntry deletes a single NFLOG prefix entry to our internal cache.
func (pc *PolicyLookupsCache) deleteNFLogPrefixEntry(prefix string) {
	var bph [64]byte
	copy(bph[:], []byte(prefix[:]))
	pc.nflogMutex.Lock()
	defer pc.nflogMutex.Unlock()
	delete(pc.nflogPrefixHash, bph)
}

// updatePolicyRulesNFLOGPrefixes stores the required prefix to RuleID maps for a policy, deleting any
// stale entries if the number of rules or action types have changed.
func (pc *PolicyLookupsCache) updatePolicyRulesNFLOGPrefixes(key model.PolicyKey, policy *model.Policy) {
	// If this is the first time we have seen this tier, add the default deny entries for the tier.
	count, ok := pc.tierRefs[key.Tier]
	if !ok {
		pc.addNFLogPrefixEntry(
			rules.CalculateNoMatchPolicyNFLOGPrefixStr(rules.RuleDirIngress, key.Tier),
			NewRuleID(key.Tier, "", "", 0, rules.RuleDirIngress, rules.RuleActionDeny),
		)
		pc.addNFLogPrefixEntry(
			rules.CalculateNoMatchPolicyNFLOGPrefixStr(rules.RuleDirEgress, key.Tier),
			NewRuleID(key.Tier, "", "", 0, rules.RuleDirEgress, rules.RuleActionDeny),
		)
	}
	pc.tierRefs[key.Tier] = count + 1

	namespace, tier, name, err := deconstructPolicyName(key.Name)
	if err != nil {
		log.WithError(err).Error("Unable to parse policy name")
		return
	}

	oldPrefixes := pc.nflogPrefixesPolicy[key]
	pc.nflogPrefixesPolicy[key] = pc.updateRulesNFLOGPrefixes(
		key.Name,
		namespace,
		tier,
		name,
		oldPrefixes,
		policy.InboundRules,
		policy.OutboundRules,
	)
}

// removePolicyRulesNFLOGPrefixes removes the prefix to RuleID maps for a policy.
func (pc *PolicyLookupsCache) removePolicyRulesNFLOGPrefixes(key model.PolicyKey) {
	// If this is the last entry for the tier, remove the default deny entries for the tier.
	// Increment the reference count so that we don't keep adding tiers.
	count := pc.tierRefs[key.Tier]
	if count == 1 {
		delete(pc.tierRefs, key.Tier)
		pc.deleteNFLogPrefixEntry(
			rules.CalculateNoMatchPolicyNFLOGPrefixStr(rules.RuleDirIngress, key.Tier),
		)
		pc.deleteNFLogPrefixEntry(
			rules.CalculateNoMatchPolicyNFLOGPrefixStr(rules.RuleDirEgress, key.Tier),
		)
	} else {
		pc.tierRefs[key.Tier] = count - 1
	}

	oldPrefixes := pc.nflogPrefixesPolicy[key]
	pc.deleteRulesNFLOGPrefixes(oldPrefixes)
	delete(pc.nflogPrefixesPolicy, key)
}

// updateProfileRulesNFLOGPrefixes stores the required prefix to RuleID maps for a profile, deleting any
// stale entries if the number of rules or action types have changed.
func (pc *PolicyLookupsCache) updateProfileRulesNFLOGPrefixes(key model.ProfileRulesKey, profile *model.ProfileRules) {
	oldPrefixes := pc.nflogPrefixesProfile[key]
	pc.nflogPrefixesProfile[key] = pc.updateRulesNFLOGPrefixes(
		key.Name,
		"",
		"",
		key.Name,
		oldPrefixes,
		profile.InboundRules,
		profile.OutboundRules,
	)
}

// removeProfileRulesNFLOGPrefixes removes the prefix to RuleID maps for a profile.
func (pc *PolicyLookupsCache) removeProfileRulesNFLOGPrefixes(key model.ProfileRulesKey) {
	oldPrefixes := pc.nflogPrefixesProfile[key]
	pc.deleteRulesNFLOGPrefixes(oldPrefixes)
	delete(pc.nflogPrefixesProfile, key)
}

// updateRulesNFLOGPrefixes updates our NFLOG prefix to RuleID map based on the supplied set of
// ingress and egress rules, and the old set of prefixes associated with the previous resource
// settings. This method adds any new rules and removes any obsolete rules.
// TODO (rlb): Maybe we should do a lazy clean up of rules?
func (pc *PolicyLookupsCache) updateRulesNFLOGPrefixes(
	v1Name, namespace, tier, name string, oldPrefixes set.Set, ingress []model.Rule, egress []model.Rule,
) set.Set {
	newPrefixes := set.New()

	convertAction := func(a string) rules.RuleAction {
		switch a {
		case "allow":
			return rules.RuleActionAllow
		case "deny":
			return rules.RuleActionDeny
		case "pass", "next-tier":
			return rules.RuleActionPass
		}
		return rules.RuleActionDeny
	}
	owner := rules.RuleOwnerTypePolicy
	if tier == "" {
		owner = rules.RuleOwnerTypeProfile
	}
	for ii, rule := range ingress {
		action := convertAction(rule.Action)
		prefix := rules.CalculateNFLOGPrefixStr(action, owner, rules.RuleDirIngress, ii, v1Name)
		pc.addNFLogPrefixEntry(
			prefix,
			NewRuleID(tier, name, namespace, ii, rules.RuleDirIngress, action),
		)
		newPrefixes.Add(prefix)
	}
	for ii, rule := range egress {
		action := convertAction(rule.Action)
		prefix := rules.CalculateNFLOGPrefixStr(action, owner, rules.RuleDirEgress, ii, v1Name)
		pc.addNFLogPrefixEntry(
			prefix,
			NewRuleID(tier, name, namespace, ii, rules.RuleDirEgress, action),
		)
		newPrefixes.Add(prefix)
	}

	// Delete the stale prefixes.
	if oldPrefixes != nil {
		oldPrefixes.Iter(func(item interface{}) error {
			if !newPrefixes.Contains(item) {
				pc.deleteNFLogPrefixEntry(item.(string))
			}
			return nil
		})
	}

	return newPrefixes
}

// deleteRulesNFLOGPrefixes deletes the supplied set of prefixes.
func (pc *PolicyLookupsCache) deleteRulesNFLOGPrefixes(prefixes set.Set) {
	if prefixes != nil {
		prefixes.Iter(func(item interface{}) error {
			pc.deleteNFLogPrefixEntry(item.(string))
			return nil
		})
	}
}

// GetRuleIDFromNFLOGPrefix returns the RuleID associated with the supplied NFLOG prefix.
func (pc *PolicyLookupsCache) GetRuleIDFromNFLOGPrefix(prefix [64]byte) *RuleID {
	pc.nflogMutex.RLock()
	defer pc.nflogMutex.RUnlock()
	return pc.nflogPrefixHash[prefix]
}

const (
	// String values used in the string representation of the RuleID. These are used
	// in some of the external APIs and therefore should not be modified.
	RuleDirIngressStr  string = "ingress"
	RuleDirEgressStr          = "egress"
	ActionAllowStr            = "allow"
	ActionDenyStr             = "deny"
	ActionNextTierStr         = "pass"
	GlobalNamespaceStr        = "__GLOBAL__"
	ProfileTierStr            = "__PROFILE__"
	NoMatchNameStr            = "__NO_MATCH__"
	UnknownStr                = "__UNKNOWN__"

	// Special rule index that specifies that a policy has selected traffic that
	// has implicitly denied traffic.
	RuleIDIndexImplicitDrop int = -1
	RuleIDIndexUnknown      int = -2
)

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
	fpName string
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
	rid.setFlowLogPolicyName()
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

func (r *RuleID) String() string {
	return fmt.Sprintf(
		"Rule(Tier=%s,Name=%s,Namespace=%s,Direction=%s,Index=%s,Action=%s)",
		r.TierString(), r.NameString(), r.NamespaceString(), r.DirectionString(), r.IndexStr, r.ActionString(),
	)
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

func (r *RuleID) IsImplicitDropRule() bool {
	return r.Index == RuleIDIndexImplicitDrop
}

// TierString returns either the Tier name or the Profile indication string.
func (r *RuleID) TierString() string {
	if len(r.Tier) == 0 {
		return ProfileTierStr
	}
	return r.Tier
}

// NameString returns either the resource name or the No-match indication string.
func (r *RuleID) NameString() string {
	if len(r.Name) == 0 {
		return NoMatchNameStr
	}
	return r.Name
}

// NamespaceString returns either the resource namespace or the Global indication string.
func (r *RuleID) NamespaceString() string {
	if len(r.Namespace) == 0 {
		return GlobalNamespaceStr
	}
	return r.Namespace
}

// ActionString converts the action to a string value.
func (r *RuleID) ActionString() string {
	switch r.Action {
	case rules.RuleActionDeny:
		return ActionDenyStr
	case rules.RuleActionAllow:
		return ActionAllowStr
	case rules.RuleActionPass:
		return ActionNextTierStr
	}
	return ""
}

// DirectionString converts the direction to a string value.
func (r *RuleID) DirectionString() string {
	switch r.Direction {
	case rules.RuleDirIngress:
		return RuleDirIngressStr
	case rules.RuleDirEgress:
		return RuleDirEgressStr
	}
	return ""
}

func (r *RuleID) setDeniedPacketRuleName() {
	if r.Action != rules.RuleActionDeny {
		return
	}
	if !r.IsNamespaced() {
		r.dpName = fmt.Sprintf(
			"%s|%s|%s|%s",
			r.TierString(),
			r.NameString(),
			r.IndexStr,
			r.ActionString(),
		)
		return
	}
	r.dpName = fmt.Sprintf(
		"%s|%s/%s|%s|%s",
		r.TierString(),
		r.Namespace,
		r.NameString(),
		r.IndexStr,
		r.ActionString(),
	)
}

func (r *RuleID) GetDeniedPacketRuleName() string {
	if r == nil {
		return ""
	}
	return r.dpName
}

func (r *RuleID) setFlowLogPolicyName() {
	if !r.IsNamespaced() {
		r.fpName = fmt.Sprintf(
			"%s|%s.%s|%s",
			r.TierString(),
			r.TierString(),
			r.NameString(),
			r.ActionString(),
		)
	} else if strings.HasPrefix(r.Name, "knp.default") {
		r.fpName = fmt.Sprintf(
			"%s|%s/%s|%s",
			r.TierString(),
			r.Namespace,
			r.NameString(),
			r.ActionString(),
		)
	} else {
		r.fpName = fmt.Sprintf(
			"%s|%s/%s.%s|%s",
			r.TierString(),
			r.Namespace,
			r.TierString(),
			r.NameString(),
			r.ActionString(),
		)
	}
}

func (r *RuleID) GetFlowLogPolicyName() string {
	if r == nil {
		return ""
	}
	return r.fpName
}

// deconstructPolicyName deconstructs the v1 policy name that is constructed by the SyncerUpdateProcessors in
// libcalico-go and extracts the v3 fields: namespace, tier, name.
//
// The v1 policy name is of the format:
// -  <namespace>/<tier>.<name> for a namespaced NetworkPolicies
// -  <tier>.<name> for GlobalNetworkPolicies.
// -  <namespace>/knp.default.k8spolicy for a k8s NetworkPolicies
//
// The namespace is returned blank for GlobalNetworkPolicies.
// For k8s network policies, the tier is always "default" and the name will be returned including the
// knp.default prefix.
func deconstructPolicyName(name string) (string, string, string, error) {
	parts := strings.Split(name, "/")
	switch len(parts) {
	case 1: // GlobalNetworkPolicy
		nameParts := strings.Split(parts[0], ".")
		if len(nameParts) != 2 {
			return "", "", "", fmt.Errorf("Could not parse policy %s", name)
		}
		return "", nameParts[0], nameParts[1], nil
	case 2: // NetworkPolicy
		nameParts := strings.Split(parts[1], ".")
		switch len(nameParts) {
		case 2: // Non-k8s
			return parts[0], nameParts[0], nameParts[1], nil
		case 3: // K8s
			if nameParts[0] == "knp" && nameParts[1] == "default" {
				return parts[0], nameParts[1], parts[1], nil
			}
		}
	}
	return "", "", "", fmt.Errorf("Could not parse policy %s", name)
}

// Dump returns the contents of important structures in the LookupManager used for
// logging purposes in the test code. This should not be used in any mainline code.
func (pc *PolicyLookupsCache) Dump() string {
	pc.nflogMutex.RLock()
	defer pc.nflogMutex.RUnlock()
	lines := []string{}
	for p, r := range pc.nflogPrefixHash {
		lines = append(lines, string(p[:])+": "+r.String())
	}
	return strings.Join(lines, "\n")
}
