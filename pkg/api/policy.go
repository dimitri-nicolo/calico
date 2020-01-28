// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package api

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

const (
	ActionInvalid       = ""
	ActionUnknown       = "unknown"
	ActionAllow         = "allow"
	ActionDeny          = "deny"
	ActionEndOfTierDeny = "eot-deny"
	ActionNextTier      = "pass"
	knpString           = "knp.default."
)

// PolicyHitKey identifies a policy.
type PolicyHit struct {
	// The tier name (or __PROFILE__ for profile match)
	Tier string

	// The policy name. This will include the tier prefix for calico policy and the "knp.default" prefix for Kubernetes
	// policies.
	Name string

	// The policy namespace (if namespaced).
	Namespace string

	// Whether this is a staged policy.
	Staged bool

	// The action flag(s) for this policy hit.
	Action ActionFlag

	// The match index for this hit.
	MatchIndex int

	// The document count.
	Count int64
}

// ToFlowLogPolicyStrings converts a PolicyHit to a slice of flow log policy strings. This is used to convert a
// PIP response to multiple possible entries.
func (p PolicyHit) ToFlowLogPolicyStrings() []string {
	// Calculate the set of action strings for this policy.
	actions := p.Action.ToActionStrings()

	// Tweak the action strings to the policy hit string required for the flow log.
	for i := range actions {
		actions[i] = fmt.Sprintf("%d|%s|%s|%s", p.MatchIndex, p.Tier, p.FlowLogName(), actions[i])
	}
	return actions
}

// ToFlowLogPolicyString returns a single flow policy string. This assumes the action flag contains a single valid
// action. Returns an empty string if not valid.
func (p PolicyHit) ToFlowLogPolicyString() string {
	if s := p.ToFlowLogPolicyStrings(); len(s) == 1 {
		return s[0]
	}
	return ""
}

// IsKubernetes returns true if this is a k8s network policy or staged network policy
func (p PolicyHit) IsKubernetes() bool {
	return strings.HasPrefix(p.Name, knpString)
}

// FlowLogName returns the name as it would appear in the flow log. This is unique for a specific policy instance.
// -  <name>
// -  staged:<name>
// -  <namespace>/<name>
// -  <namespace>/staged:<name>
// -  <namespace>/knp.default.<name>
// -  <namespace>/staged:knp.default.<name>
func (p PolicyHit) FlowLogName() string {
	if len(p.Namespace) == 0 {
		if p.Staged {
			return model.PolicyNamePrefixStaged + p.Name
		} else {
			return p.Name
		}
	}
	if p.Staged {
		return p.Namespace + "/" + model.PolicyNamePrefixStaged + p.Name
	}
	return p.Namespace + "/" + p.Name
}

// PolicyHitFromFlowLogPolicyString creates a PolicyHit from a flow log policy string.
func PolicyHitFromFlowLogPolicyString(n string, count int64) (PolicyHit, bool) {
	p := PolicyHit{
		Count: count,
	}

	parts := strings.Split(n, "|")
	if len(parts) != 4 {
		return p, false
	}

	// Extract match index.
	var err error
	p.MatchIndex, err = strconv.Atoi(parts[0])
	if err != nil {
		return p, false
	}

	// Extract tier
	p.Tier = parts[1]

	// Extract namespace and name.
	nameparts := strings.SplitN(parts[2], "/", 2)
	if len(nameparts) == 2 {
		p.Namespace = nameparts[0]
		p.Name = nameparts[1]
	} else {
		p.Name = nameparts[0]
	}

	// Remove the staged prefix, if staged.
	if strings.HasPrefix(p.Name, model.PolicyNamePrefixStaged) {
		p.Staged = true
		p.Name = p.Name[len(model.PolicyNamePrefixStaged):]
	}

	// Extract action.
	p.Action = ActionFlagFromString(parts[3])
	return p, p.Action != 0
}

// SortablePolicyHits is a sortable slice of PolicyHits.
type SortablePolicyHits []PolicyHit

func (s SortablePolicyHits) Len() int { return len(s) }

func (s SortablePolicyHits) Less(i, j int) bool {
	if s[i].MatchIndex != s[j].MatchIndex {
		return s[i].MatchIndex < s[j].MatchIndex
	}
	if s[i].Namespace != s[j].Namespace {
		return s[i].Namespace < s[j].Namespace
	}
	if s[i].Name != s[j].Name {
		return s[i].Name < s[j].Name
	}
	if s[i].Action != s[j].Action {
		return s[i].Action < s[j].Action
	}
	return s[i].Staged && !s[j].Staged
}

func (s SortablePolicyHits) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// SortAndRenumber sorts the PolicyHit slice and renumbers to be monotonically increasing.
func (s SortablePolicyHits) SortAndRenumber() {
	sort.Sort(s)
	for i := range s {
		s[i].MatchIndex = i
	}
}

// PolicyHitsEqual compares two sets of PolicyHits to see if both order and values are identical.
func PolicyHitsEqual(p1, p2 []PolicyHit) bool {
	if len(p1) != len(p2) {
		return false
	}

	for i := range p1 {
		if p1[i] != p2[i] {
			return false
		}
	}

	return true
}

// ObfuscatedPolicyString creates the flow log policy string indicating an obfuscated policy.
func ObfuscatedPolicyString(matchIdx int, flag ActionFlag) string {
	var action string
	if actions := flag.ToActionStrings(); len(actions) == 1 {
		action = actions[0]
	}
	return fmt.Sprintf("%d|*|*|%s", matchIdx, action)
}
