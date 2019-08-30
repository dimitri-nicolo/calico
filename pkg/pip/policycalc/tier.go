package policycalc

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	pipcfg "github.com/tigera/es-proxy/pkg/pip/config"
)

// newCompiledTiersAndPolicies compiles the Tiers into CompiledTiersAndPolicies.
func newCompiledTiersAndPolicies(cfg *pipcfg.Config, rd *ResourceData, modified ModifiedResources, sel *EndpointSelectorHandler) *CompiledTiersAndPolicies {
	// Create the namespace handler, and populate from the Namespaces and ServiceAccounts.
	log.Debugf("Creating namespace handler with %d namespaces and %d service accounts", len(rd.Namespaces), len(rd.ServiceAccounts))
	namespaces := NewNamespaceHandler(rd.Namespaces, rd.ServiceAccounts)

	// Create a new matcher factory which is used to create Matcher functions for the compiled policies.
	matcherFactory := NewMatcherFactory(cfg, namespaces, sel)

	// Iterate through tiers.
	c := &CompiledTiersAndPolicies{}
	for i, tier := range rd.Tiers {
		log.Debugf("Compiling tier (idx %d)", i)
		var ingressTier, egressTier CompiledTier

		// Iterate through the policies in a tier.
		for _, pol := range tier {
			// Compile the policy to get the ingress and egress versions of the policy as appropriate.
			ingressPol, egressPol := compilePolicy(matcherFactory, pol, modified.IsModified(pol))

			// Add the ingress and egress policies to their respective slices. If this is a modified policy, also
			// track it - we'll use this as a shortcut to determine if a flow is affected by the configuration change
			// or not.
			if ingressPol != nil {
				ingressTier = append(ingressTier, ingressPol)
				if ingressPol.Modified {
					c.ModifiedIngressPolicies = append(c.ModifiedIngressPolicies, ingressPol)
				}
			}
			if egressPol != nil {
				egressTier = append(egressTier, egressPol)
				if egressPol.Modified {
					c.ModifiedEgressPolicies = append(c.ModifiedEgressPolicies, egressPol)
				}
			}
		}

		// Append the ingress and egress tiers if any policies were added to them.
		if ingressTier != nil {
			c.IngressTiers = append(c.IngressTiers, ingressTier)
		}
		if egressTier != nil {
			c.EgressTiers = append(c.EgressTiers, egressTier)
		}
	}

	return c
}

// CompiledTiersAndPolicies contains the compiled set of ingress/egress tiers and tracks the ingress and egress
// policies impacted by the configuration update.
type CompiledTiersAndPolicies struct {
	// IngressTiers is the set of compiled tiers and policies, containing only ingress rules. Policies that do not
	// apply to ingress flows are filtered out, and tiers are omitted if all policies were filtered out.
	IngressTiers CompiledTiers

	// EgressTiers is the set of compiled tiers and policies, containing only egress rules. Policies that do not
	// apply to egress flows are filtered out, and tiers are omitted if all policies were filtered out.
	EgressTiers CompiledTiers

	// ModifiedIngressPolicies is the set of compiled policies containing ingress rules that were modified by the
	// resource update.
	ModifiedIngressPolicies []*CompiledPolicy

	// ModifiedEgressPolicies is the set of compiled policies containing egress rules that were modified by the
	// resource update.
	ModifiedEgressPolicies []*CompiledPolicy
}

// FlowSelectedByModifiedPolicies returns whether the flow is selected by any of the policies that were modified.
// Flows that are not selected cannot be impacted by the policy updates and therefore do not need to be run through
// the policy calculator.
func (c *CompiledTiersAndPolicies) FlowSelectedByModifiedPolicies(flow *Flow) bool {
	// If the source is a calico-managed endpoint, check whether egress policies were modified. We do this for both
	// source and dest reported flows - it's possible a flow is now denied at source which means the the dest-reported
	// flow should no longer be included.
	if flow.Source.isCalicoEndpoint() {
		for i := range c.ModifiedEgressPolicies {
			if c.ModifiedEgressPolicies[i].Applies(flow) == MatchTypeTrue {
				return true
			}
		}
	}

	// For dest-reported flows, we check the ingress policies were modified. No need to check if the destination is a
	// calico-managed endpoint, it has to have been for it to be reported.
	if flow.Reporter == ReporterTypeDestination {
		for i := range c.ModifiedIngressPolicies {
			if c.ModifiedIngressPolicies[i].Applies(flow) == MatchTypeTrue {
				return true
			}
		}
	}

	return false
}

// Calculate determines the action and policy path for a specific flow on this compiled set of tiers and policies.
func (c *CompiledTiersAndPolicies) Calculate(flow *Flow) *Response {
	r := &Response{}

	// Determine if we need to include the destination flow. Initialize to whether the flow is a destination reported
	// flow for a Calico endpoint - we may adjust this in the source processing.
	includeDestIngress := flow.Destination.isCalicoEndpoint() && flow.Reporter == ReporterTypeDestination

	// If the source endpoint is a Calico endpoint, then calculate the policy impact for egress. We do this for both
	// source and destination reported flows. The reason we do this for source flows is because we may need to remove
	// the destination flow completely if the source flow is now denied (and previously was not).
	if flow.Source.isCalicoEndpoint() {
		// Calculate egress.
		log.Debug("Calculating egress action")
		c.EgressTiers.Calculate(flow, &flow.Source, &r.Source)

		// If the reporter is the source, then make sure we include this result.
		if flow.Reporter == ReporterTypeSource {
			r.Source.Include = true

			// Furthermore, if the action was originally Deny and now it isn't, and the destination is also a Calico
			// endpoint, then we will need to include the equivalent destination flow since this will not be in the
			// flow logs.
			if flow.Action == ActionDeny && r.Source.Action != ActionDeny && flow.Destination.isCalicoEndpoint() {
				log.Debug("Source action moved from Deny, so need to include Dest action")
				includeDestIngress = true
			}
		} else if r.Source.Action == ActionDeny {
			log.Debug("Source egress action is deny, do not calculate destination ingress action")
			includeDestIngress = false
		}
	}

	if includeDestIngress {
		log.Debug("Calculating ingress action")
		r.Destination.Include = true
		c.IngressTiers.Calculate(flow, &flow.Destination, &r.Destination)
	}

	return r
}

// CompiledTiers contains a set of compiled tiers and policies for either ingress or egress.
type CompiledTiers []CompiledTier

// Calculate determines the policy impact of the tiers for the supplied flow.
func (ts CompiledTiers) Calculate(flow *Flow, ep *FlowEndpointData, epr *EndpointResponse) {
	var af ActionFlag
	var tierIdx int
	for i := range ts {
		// Calculate the set of action flags for the tier.
		af = ts[i].Action(flow, af, tierIdx, epr)

		if af.Indeterminate() {
			// The flags now indicate the action is indeterminate, exit immediately. Store the action in the response
			// object.
			epr.Action = ActionUnknown
			return
		}

		if af&ActionFlagNextTier == 0 {
			// The next tier flag was not set, so we are now done. Since the action is not unknown, then we should
			// have a concrete allow or deny action at this point. Note that whilst the uncertain flag may be set
			// all of the possible paths have resulted in the same action. Store the action in the response object.
			log.Debug("Not required to enumerate next tier - must have final action")
			epr.Action = af.ToAction()
			return
		}

		if af&ActionFlagDidNotMatchTier == 0 {
			// There was a policy that applied to the endpoint, so increment the tier index. Felixes tier ordering
			// is endpoint specific, and only includes tiers that have policies which apply the endpoint.
			//TODO(rlb): This still isn't quite correct. IIUC Felix does not separately order ingress and egress policies
			//           so it's possible for ingress/egress tier numbering to be different for the same endpoint. Not
			//           sure it actually matters though.
			tierIdx++
		}

		// Clear the pass and tier match flags before we skip to the next tier.
		af &^= ActionFlagNextTier | ActionFlagDidNotMatchTier
	}

	// -- END OF TIERS --
	// This is Allow for Pods, and Deny for HEPs.
	log.Debug("Hit end of tiers")
	if ep.Type == EndpointTypeWep {
		// End of tiers allow is handled by the namespace profile. Add the policy name for this and set the allow flag.
		epr.Policies = append(epr.Policies, fmt.Sprintf("%d|__PROFILE__|__PROFILE__.kns.%s|allow", tierIdx, ep.Namespace))
		af |= ActionFlagAllow
	} else {
		// End of tiers deny is handled implicitly by Felix and has a very specific pseudo-profile name.
		epr.Policies = append(epr.Policies, fmt.Sprintf("%d|__PROFILE__|__PROFILE__.__NO_MATCH__|deny", tierIdx))
		af |= ActionFlagDeny
	}

	// Store the action in the response object.
	epr.Action = af.ToAction()
}

// CompiledTier contains a set of compiled policies for a specific tier, for either ingress _or_ egress.
type CompiledTier []*CompiledPolicy

// Action returns the calculated action for the tier for the supplied flow.
// A previous tier/policy may have specified a possible match action which could not be confirmed due to lack of
// information. We supply the current action flags so that further enumeration can exit as soon as we either find
// an identical action with confirmed match, or a different action (confirmed or unconfirmed) that means we cannot
// determine the result with certainty.
func (tier CompiledTier) Action(flow *Flow, af ActionFlag, tierIdx int, r *EndpointResponse) ActionFlag {
	var lastMatchedPolicy *CompiledPolicy
	var nextPolicy bool
	for _, p := range tier {
		// If the policy does not apply to this Endpoint then skip to the next policy.
		if p.Applies(flow) != MatchTypeTrue {
			log.Debugf("Policy %s does not apply - skipping", p.Name)
			continue
		}
		// Track that at least one policy in this tier matched. This influences end-of-tier behavior (see below).
		log.Debugf("Policy %s applies - matches tier", p.Name)
		lastMatchedPolicy = p

		// Calculate the set of action flags from the policy.
		af, nextPolicy = p.Action(flow, af, tierIdx, r)

		if af.Indeterminate() || !nextPolicy {
			// The action flags either indicate that the action is indeterminate, or the policy had an explicit match.
			log.Debugf("No need to enumerate next policy. ActionFlags=%v", af)
			return af
		}
	}

	// -- END OF TIER --
	if lastMatchedPolicy != nil {
		// We matched at least one policy in the tier, but matched no rules. Set to deny, and add the implicit end of
		// tier drop policy.
		log.Debug("Hit end of tier drop")
		r.Policies = append(r.Policies, lastMatchedPolicy.calculateFlowLogName(tierIdx, ActionFlagDeny))
		af |= ActionFlagDeny
	} else {
		// Otherwise this flow didn't apply to any policy in this tier, so go to the next tier.
		log.Debug("Did not match tier - enumerate next tier")
		af |= ActionFlagNextTier | ActionFlagDidNotMatchTier
	}

	log.Debugf("Calculated action from tier. ActionFlags=%v", af)

	return af
}
