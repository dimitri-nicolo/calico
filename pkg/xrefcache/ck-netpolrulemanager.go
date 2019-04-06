package xrefcache

import (
	log "github.com/sirupsen/logrus"

	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
)

// This file implements a NetworkPolicyRuleSelectorManager. This acts as a bridge between the real policy resources
// and the pseudo NetworkPolicuy RuleSelector types. The manager is responsible for handling

// Callbacks for match start/stop between a selector and a policy.
type NPRSMatchStarted func(policy, selector resources.ResourceID)
type NPRSMatchStopped func(policy, selector resources.ResourceID)

// NetworkPolicyRuleSelectorManager provides a shared interface for communication between the policy and the selector
// pseudo-resource caches. It also manages the creation and deletion of the pseudo resource types based on whether
// any policy needs a particular selector to be tracked.
type NetworkPolicyRuleSelectorManager interface {
	RegisterCallbacks(onMatchStarted NPRSMatchStarted, onMatchStopped NPRSMatchStopped)
	SetPolicyRuleSelectors(policy resources.ResourceID, selectors resources.Set) (changed bool)
	DeletePolicy(policy resources.ResourceID)
}

// NewNetworkPolicyRuleSelectorManager creates a new NetworkPolicyRuleSelectorManager.
func NewNetworkPolicyRuleSelectorManager(onUpdate func(update syncer.Update)) NetworkPolicyRuleSelectorManager {
	return &networkPolicyRuleSelectorManager{
		onUpdate:           onUpdate,
		selectorsByPolicy:  make(map[resources.ResourceID]resources.Set),
		policiesBySelector: make(map[resources.ResourceID]resources.Set),
	}
}

// networkPolicyRuleSelectorManager implements the NetworkPolicyRuleSelectorManager interface.
type networkPolicyRuleSelectorManager struct {
	// The onUpdate method called to add the selector rule pseudo resource types.
	onUpdate func(syncer.Update)

	// Registered match stopped/started events.
	onMatchStarted []NPRSMatchStarted
	onMatchStopped []NPRSMatchStopped

	// Selectors by policy
	selectorsByPolicy map[resources.ResourceID]resources.Set

	// Policies by selector
	policiesBySelector map[resources.ResourceID]resources.Set
}

// RegisterCallbacks registers match start/stop callbacks with this manager.
func (m *networkPolicyRuleSelectorManager) RegisterCallbacks(onMatchStarted NPRSMatchStarted, onMatchStopped NPRSMatchStopped) {
	m.onMatchStarted = append(m.onMatchStarted, onMatchStarted)
	m.onMatchStopped = append(m.onMatchStopped, onMatchStopped)
}

// SetPolicyRuleSelectors sets the rule selectors that need to be tracked by a policy resource.
func (m *networkPolicyRuleSelectorManager) SetPolicyRuleSelectors(p resources.ResourceID, s resources.Set) bool {
	var changed bool

	// If we have not seen this policy before then add it now
	currentSelectors, ok := m.selectorsByPolicy[p]
	if !ok {
		currentSelectors = resources.EmptySet()
	}

	currentSelectors.IterDifferences(s,
		func(old resources.ResourceID) error {
			// Stop tracking old selectors for this policy.
			m.matchStopped(p, old)
			changed = true
			return nil
		},
		func(new resources.ResourceID) error {
			// Start tracking new selectors for this policy.
			m.matchStarted(p, new)
			changed = true
			return nil
		},
	)

	// Replace the set of selectors for this policy.
	m.selectorsByPolicy[p] = s

	return changed
}

func (m *networkPolicyRuleSelectorManager) matchStarted(p, s resources.ResourceID) {
	log.Debugf("NetworkPolicyRuleSelector match started: %s / %s", p, s)
	pols, ok := m.policiesBySelector[s]
	if !ok {
		pols = resources.NewSet()
		m.policiesBySelector[s] = pols
	}
	pols.Add(p)

	if !ok {
		// This is a new selector, so create a new NetworkPolicy RuleSelector pseudo resource.
		log.Debugf("First policy for selector, adding pseudo-resource %s", s)
		m.onUpdate(syncer.Update{
			Type:       syncer.UpdateTypeSet,
			ResourceID: s,
		})
	}

	// Notify our listeners of a new match.
	for _, cb := range m.onMatchStarted {
		cb(p, s)
	}
}

func (m *networkPolicyRuleSelectorManager) matchStopped(p, s resources.ResourceID) {
	log.Debugf("NetworkPolicyRuleSelector match stopped: %s / %s", p, s)
	pols := m.policiesBySelector[s]
	pols.Discard(p)

	// Notify our listeners that the match has stopped.
	for _, cb := range m.onMatchStopped {
		cb(p, s)
	}

	if pols.Len() == 0 {
		// This was the last policy associated with this selector. Delete the RuleSelector pseudo resource.
		log.Debugf("Last policy for selector, deleting pseudo-resource %s", s)
		m.onUpdate(syncer.Update{
			Type:       syncer.UpdateTypeDeleted,
			ResourceID: s,
		})

		delete(m.policiesBySelector, s)
	}
}

// DeletePolicy is called to delete a policy from the manager. This will result in match stopped callbacks for any
// selectors it was previously tracking.
func (m *networkPolicyRuleSelectorManager) DeletePolicy(policy resources.ResourceID) {
	currentSelectors, ok := m.selectorsByPolicy[policy]
	if !ok {
		return
	}

	currentSelectors.Iter(func(selector resources.ResourceID) error {
		m.matchStopped(policy, selector)
		return nil
	})

	delete(m.selectorsByPolicy, policy)
}
