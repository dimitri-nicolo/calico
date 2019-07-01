package policycalc

import log "github.com/sirupsen/logrus"

// CompiledTiersAndPolicies contains the compiled set of ingress/egress tiers and tracks the ingress and egress
// policies impacted by the configuration update.
type CompiledTiersAndPolicies struct {
	IngressTiers    CompiledTiers
	EgressTiers     CompiledTiers
	IngressImpacted []*CompiledPolicy
	EgressImpacted  []*CompiledPolicy
}

// FlowImpacted returns whether the supplied flow is potentially impacted by the configuration update.
func (c *CompiledTiersAndPolicies) FlowImpacted(flow *Flow) bool {
	// If source is a Calico managed endpoint then check if flow matches impacted egress policies.
	if flow.Source.Labels != nil {
		for i := range c.EgressImpacted {
			if c.EgressImpacted[i].Applies(flow) == MatchTypeTrue {
				return true
			}
		}
	}

	// If destination is a Calico managed endpoint then check if flow matches impacted ingress policies.
	if flow.Destination.Labels != nil {
		for i := range c.IngressImpacted {
			if c.IngressImpacted[i].Applies(flow) == MatchTypeTrue {
				return true
			}
		}
	}
	return false
}

// Action returns the calculated action for a specific flow on this compiled set of tiers and policies.
func (c *CompiledTiersAndPolicies) Action(flow *Flow) Action {
	// Assume ingress/egress are allowed by default.
	ingress, egress := ActionFlagAllow, ActionFlagAllow

	// If source is a Calico managed endpoint then calculate egress action.
	if flow.Source.isCalicoEndpoint() {
		log.Debug("Checking egress action")
		egress = c.EgressTiers.Action(flow)
	}

	// If destination is a Calico managed endpoint then calculate ingress action.
	if flow.Destination.isCalicoEndpoint() {
		log.Debug("Checking ingress action")
		ingress = c.IngressTiers.Action(flow)
	}

	// If either ingress or egress actions are deny then the flow is denied.
	if ingress.Deny() || egress.Deny() {
		log.Debug("Flow denied")
		return ActionDeny
	}

	// Otherwise, if either of the ingress or egress actions is indeterminate then the flow is indeterminate.
	if ingress.Indeterminate() || egress.Indeterminate() {
		log.Debug("Flow indeterminate")
		return ActionIndeterminate
	}
	// If not denied or indeterminate, then the flow must be allowed.
	log.Debug("Flow allowed")
	return ActionAllow
}

// CompiledTiers contains a set of compiled tiers and policies for either ingress or egress.
type CompiledTiers []CompiledTier

// Action returns the calculated action from the tiers for the supplied flow.
func (ts CompiledTiers) Action(flow *Flow) ActionFlag {
	var af ActionFlag
	for i := range ts {
		// Calculate the set of action flags for the tier.
		af = ts[i].Action(flow, af)

		if af.Indeterminate() {
			// The flags now indicate the action is indeterminate, exit immediately.
			return af
		}

		if af&ActionFlagNextTier == 0 {
			// The next tier flag was not set, so we are now done. Since the action is not unknown, then we should
			// have a concrete allow or deny action at this point. Note that whilst the uncertain flag may be set
			// all of the possible paths have resulted in the same action.
			return af
		}

		// Clear the pass flag before we skip to the next tier.
		af &^= ActionFlagNextTier
	}

	// -- END OF TIERS ALLOW? --
	af |= ActionFlagAllow
	return af
}

// CompiledTier contains a set of compiled policies for a specific tier, for either ingress _or_ egress.
type CompiledTier []*CompiledPolicy

// Action returns the calculated action for the tier for the supplied flow.
// A previous tier/policy may have specified a possible match action which could not be confirmed due to lack of
// information. We supply the current action flags so that further enumeration can exit as soon as we either find
// an identical action with confirmed match, or a different action (confirmed or unconfirmed) that means we cannot
// determine the result with certainty.
func (tier CompiledTier) Action(flow *Flow, af ActionFlag) ActionFlag {
	var matchedTier bool
	var nextPolicy bool
	for _, p := range tier {
		// If the policy does not apply to this Endpoint then skip to the next policy.
		if p.Applies(flow) != MatchTypeTrue {
			log.Debugf("Policy does not apply - skipping")
			continue
		}
		// Track that at least one policy in this tier matched.
		log.Debugf("Policy applies - matches tier")
		matchedTier = true

		// Calculate the set of action flags from the policy.
		af, nextPolicy = p.Action(flow, af)

		if af.Indeterminate() || !nextPolicy {
			// The action flags either indicate that the action is indeterminate, or the policy had an explicit match.
			log.Debugf("No need to enumerate next policy. ActionFlags=%v", af)
			return af
		}
	}

	// -- END OF TIER DROP --
	if matchedTier {
		// We matched at least one policy in the tier, but matched no rules. Set to deny.
		log.Debug("Hit end of tier drop")
		af |= ActionFlagDeny
	} else {
		// Otherwise this flow didn't apply to any policy in this tier, so go to the next tier.
		log.Debug("Did not match tier - enumerate next tier")
		af |= ActionFlagNextTier
	}

	log.Debugf("Calculated action from tier. ActionFlags=%v", af)

	return af
}
