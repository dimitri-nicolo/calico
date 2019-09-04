package policycalc

import (
	"strings"

	"github.com/projectcalico/libcalico-go/lib/backend/model"
	log "github.com/sirupsen/logrus"

	pipcfg "github.com/tigera/es-proxy/pkg/pip/config"
)

// calculateCompiledTiersAndImpactedPolicies compiles the Tiers and policies and returns ingress and egress sets of
// -  The compiled tiers (With policies)
// -  The set of policies that are impacted by a resource update.
func calculateCompiledTiersAndImpactedPolicies(
	cfg *pipcfg.Config, rd *ResourceData, impacted ImpactedResources, sel *EndpointSelectorHandler, changesApplied bool,
) (ingress, egress CompiledTiersAndImpactedPolicies) {
	// Create the namespace handler, and populate from the Namespaces and ServiceAccounts.
	log.Debugf("Creating namespace handler with %d namespaces and %d service accounts", len(rd.Namespaces), len(rd.ServiceAccounts))
	namespaces := NewNamespaceHandler(rd.Namespaces, rd.ServiceAccounts)

	// Create a new matcher factory which is used to create Matcher functions for the compiled policies.
	matcherFactory := NewMatcherFactory(cfg, namespaces, sel)

	// Iterate through tiers.
	for i, tier := range rd.Tiers {
		log.Debugf("Compiling tier (idx %d)", i)
		var ingressTier, egressTier CompiledTier

		// Iterate through the policies in a tier.
		for _, pol := range tier {
			// Determine the impact of this policy.
			impact, isImpacted := impacted.Impact(pol.Policy)

			// If this is set of data with changes applied, pass the impact associated with the policy when compiling
			// it.
			var actualImpact Impact
			if changesApplied {
				actualImpact = impact
			}

			// Compile the policy to get the ingress and egress versions of the policy as appropriate.
			ingressPol, egressPol := compilePolicy(matcherFactory, pol, actualImpact)

			// Add the ingress and egress policies to their respective slices. If this is a impacted policy, also
			// track it - we'll use this as a shortcut to determine if a flow is possibly affected by the configuration
			// change or not. We do this for both the pre and post updated resources.
			if ingressPol != nil {
				ingressTier = append(ingressTier, ingressPol)
				if isImpacted {
					// If resource is impacted include this policy. We do this for original and changed resources.
					ingress.ImpactedPolicies = append(ingress.ImpactedPolicies, ingressPol)
				}
			}
			if egressPol != nil {
				egressTier = append(egressTier, egressPol)
				if isImpacted {
					// If resource is impacted include this policy. We do this for original and changed resources.
					egress.ImpactedPolicies = append(egress.ImpactedPolicies, egressPol)
				}
			}
		}

		// Append the ingress and egress tiers if any policies were added to them.
		if ingressTier != nil {
			ingress.Tiers = append(ingress.Tiers, ingressTier)
		}
		if egressTier != nil {
			egress.Tiers = append(egress.Tiers, egressTier)
		}
	}

	return
}

// CompiledTiersAndImpactedPolicies contains a set of compiled tiers and impacted policies for a single flow direction
// (i.e. ingress or egress).
type CompiledTiersAndImpactedPolicies struct {
	// IngressTiers is the set of compiled tiers and policies, containing only ingress rules. Policies that do not
	// apply to ingress flows are filtered out, and tiers are omitted if all policies were filtered out.
	Tiers CompiledTiers

	// ImpactedEgressPolicies is the set of compiled policies containing egress rules that were impacted by the
	// resource update, either through resource modification, or previewing a staged policy.
	ImpactedPolicies []*CompiledPolicy
}

// FlowSelectedByModifiedEgressPolicies returns whether the flow is selected by any of the impacted policies.
func (c *CompiledTiersAndImpactedPolicies) FlowSelectedByImpactedPolicies(flow *Flow, cache *flowCache) bool {
	for i := range c.ImpactedPolicies {
		if c.ImpactedPolicies[i].Applies(flow, cache) == MatchTypeTrue {
			return true
		}
	}
	return false
}

// Calculate determines the action and policy path for a specific flow on this compiled set of tiers and policies.
func (c *CompiledTiersAndImpactedPolicies) Calculate(flow *Flow, cache *flowCache, before bool) (r EndpointResponse) {
	// If the source endpoint is a Calico endpoint and this is reported by source then calculate egress. If the
	// action changes from deny in the original flow to either allow or unknown then we need to calculate for the
	// destination ingress too.
	if flow.Source.IsCalicoManagedEndpoint() && flow.Reporter == ReporterTypeSource {
		// Calculate egress.
		log.Debug("Calculating egress action")
		r = c.Tiers.Calculate(flow, &flow.Source, cache, before)
	} else if flow.Destination.IsCalicoManagedEndpoint() && flow.Reporter == ReporterTypeDestination {
		log.Debug("Calculating ingress action")
		r = c.Tiers.Calculate(flow, &flow.Destination, cache, before)
	}
	return
}

// CompiledTiers contains a set of compiled tiers and policies for either ingress or egress.
type CompiledTiers []CompiledTier

// Calculate determines the policy impact of the tiers for the supplied flow.
func (ts CompiledTiers) Calculate(flow *Flow, ep *FlowEndpointData, cache *flowCache, before bool) EndpointResponse {
	var af ActionFlag
	epr := EndpointResponse{Include: true}
	for i := range ts {
		// Calculate the set of action flags for the tier. The before/after calculation are handled separately so fan
		// off accordingly.
		if before {
			af |= ts[i].ActionBefore(flow, &epr, cache)
		} else {
			af |= ts[i].ActionAfter(flow, &epr, cache)
		}

		if af&ActionFlagsAllowAndDeny == ActionFlagsAllowAndDeny {
			// The flags now indicate the action is indeterminate, exit immediately. Store the action in the response
			// object.
			log.Debug("Indeterminate action from this tier - stop further processing")
			epr.Action = af
			return epr
		}

		if af&ActionFlagNextTier == 0 {
			// The next tier flag was not set, so we are now done. Since the action is not unknown, then we should
			// have a concrete allow or deny action at this point. Note that whilst the uncertain flag may be set
			// all of the possible paths have resulted in the same action. Store the action in the response object.
			log.Debug("Not required to enumerate next tier - must have final action")
			epr.Action = af
			return epr
		}

		// Clear the pass and tier match flags before we skip to the next tier.
		af &^= ActionFlagNextTier
	}

	// -- END OF TIERS --
	// This is Allow for Pods, and Deny for HEPs.
	log.Debug("Hit end of tiers")
	if ep.Type == EndpointTypeWep {
		// End of tiers allow is handled by the namespace profile. Add the policy name for this and set the allow flag.
		addPolicyToResponse("__PROFILE__", "__PROFILE__.kns."+ep.Namespace, ActionFlagAllow, &epr)
		af |= ActionFlagAllow
	} else {
		// End of tiers deny is handled implicitly by Felix and has a very specific pseudo-profile name.
		addPolicyToResponse("__PROFILE__", "__PROFILE__.__NO_MATCH__", ActionFlagDeny, &epr)
		af |= ActionFlagDeny
	}

	// Store the action in the response object.
	epr.Action = af
	return epr
}

// CompiledTier contains a set of compiled policies for a specific tier, for either ingress _or_ egress.
type CompiledTier []*CompiledPolicy

// ActionBefore returns the calculated action for the tier for the supplied flow on the initial set of config.
//
// In this "before" processing, calculated flow hits are cross referenced against the flow log policy hits to provide
// additional certainty, and in the cases where the the calculation was not possible to infer the result from the
// measured data.
func (tier CompiledTier) ActionBefore(flow *Flow, r *EndpointResponse, cache *flowCache) ActionFlag {
	// In the before branch we track uncertain matches which we can convert to no-matches if we get an exact hit
	// corroborated by the flow log.
	var noMatchPolicies []string
	var lastEnforcedPolicy *CompiledPolicy
	var lastEnforcedActions ActionFlag
	for _, p := range tier {
		log.Debugf("Process policy: %s", p.FlowLogName)

		// If the policy does not apply to this Endpoint then skip to the next policy.
		if p.Applies(flow, cache) != MatchTypeTrue {
			log.Debug("Policy does not apply - skipping")
			continue
		}
		//TODO(rlb): We may want to handle unknown selector matches if we decide to be a little more clever about our
		//           label aggregation.

		if !p.Staged {
			// Track the last enforced policy - we use this for end-of-tier drop processing.
			log.Debug("Policy is enforced")
			lastEnforcedPolicy = p
		}

		// Calculate the policy action. This will set at least one action flag.
		policyActions := p.Action(flow, cache)

		// This is the before run, so assumed the policy is not modified in relation to the flow logs data. Use the flow
		// log data to augment the calculated action.
		if flagsFromFlowLog, ok := getFlagsFromFlowLog(p.FlowLogName, flow); ok {
			log.Debugf("Policy action flags found in flow log %d", flagsFromFlowLog)

			// An end-of-tier deny flag actually means the policy was a no-match, so convert the flag to be no match
			// since that's what we need to cache (in the after processing it may no longer be an end-of-tier drop).
			if flagsFromFlowLog == ActionFlagEndOfTierDeny {
				log.Debug("Found end of tier drop matching policy - cache as no-match")
				flagsFromFlowLog = ActionFlagNoMatch
			}

			if flagsFromFlowLog&policyActions == 0 {
				// The action in the flow log does not agree with any of the calculated actions in the policy.
				log.Debugf("Policy action found in flow log conflicts with calculated: flagsFromFlowLog: %d; policyActions: %d",
					flagsFromFlowLog, policyActions)
				policyActions |= ActionFlagFlowLogConflictsWithCalculated
			} else if policyActions&ActionFlagsAllCalculatedPolicyActions == flagsFromFlowLog {
				log.Debugf("Policy action found in flow log exactly matches calculated")
				policyActions |= ActionFlagFlowLogMatchesCalculated
			} else {
				log.Debugf("Policy action found in flow log agrees with calculated - use to break uncertainty")
				policyActions = flagsFromFlowLog | ActionFlagFlowLogRemovedUncertainty
			}
		}

		// Cache the value so that we don't have to recalculate in the "after" processing.
		cache.policies[p.FlowLogName] = policyActions

		if !p.Staged {
			// Track the last enforced policy actions that we calculate.
			log.Debugf("Policy is not staged - store action flags: %d", policyActions)
			lastEnforcedActions = policyActions
		}

		if policyActions&ActionFlagNoMatch == 0 {
			// Policy has an exactly matching rule so no need to enumerate further.
			log.Debug("Policy has an exactly matching rule")
			break
		}

		// Track the set of policies that provide no match.
		noMatchPolicies = append(noMatchPolicies, p.FlowLogName)
	}

	if lastEnforcedPolicy == nil {
		// This flow didn't apply to any policy in this tier, so go to the next tier.
		log.Debug("Did not match tier - enumerate next tier")
		return ActionFlagNextTier
	}

	if lastEnforcedActions&(ActionFlagFlowLogMatchesCalculated|ActionFlagFlowLogRemovedUncertainty) != 0 {
		// The last enforced policy calculation was corroborated the flow log data. This means any uncertain
		// no-matches can be assumed to really be a no-match where the uncertainty has been removed by the flow log
		// data, and any certain no-matches can be flagged as exactly matching the flow log.
		log.Debug("Final enforced action was corroborated by the flow log policies")
		for _, n := range noMatchPolicies {
			prev := cache.policies[n]
			if prev == ActionFlagNoMatch {
				cache.policies[n] = ActionFlagNoMatch | ActionFlagFlowLogMatchesCalculated
			} else {
				cache.policies[n] = ActionFlagNoMatch | ActionFlagFlowLogRemovedUncertainty
			}
		}

		// Given this was corroborated by the flow log, we can finish processing here.
		addPolicyToResponse(lastEnforcedPolicy.Tier, lastEnforcedPolicy.FlowLogName, lastEnforcedActions, r)
		return lastEnforcedActions
	}

	// At this point, we have a match that has not been corroborated by the flow logs. All we can do is include all of
	// the possible hits from this tier which may include multiple hits from the same policy and multiple policies. We
	// do not include staged policies nor do we include the last enforced policy which we handle explicitly.
	log.Debug("Final enforced policy action was not corroborated by the flow log policies")
	var combinedPolicyActions ActionFlag
	for _, n := range noMatchPolicies {
		if n == lastEnforcedPolicy.FlowLogName || strings.Contains(n, model.PolicyNamePrefixStaged) {
			// This is either the last enforced policy (handled separately below), or a staged policy. In either case
			// skip.
			continue
		}

		// Update previous uncertain no-matches to just be no-match inferred from flow log.
		noMatchPolicyActions := cache.policies[n]
		if noMatchPolicyActions == ActionFlagNoMatch {
			// This was an exact no match, so we don't include that.
			continue
		}

		// Add each possible action to the flow policy hits.
		addPolicyToResponse(lastEnforcedPolicy.Tier, n, noMatchPolicyActions, r)

		// Collate all of the action flags, but omit the no-match, we don't need to pass that up the stack since a
		// no match will never actually happen (due to end-of-tier drop).
		combinedPolicyActions |= noMatchPolicyActions &^ ActionFlagNoMatch
	}

	if lastEnforcedActions&ActionFlagNoMatch != 0 {
		// The last enforced policy included a no-match flag. This must be the last enforced policy in the tier because
		// we'd otherwise exit as soon as we get an exactly matched rule. We need to convert the no-match to an
		// end-of-tier drop.
		log.Debug("Final policy included a no-match, include the end of tier action")
		lastEnforcedActions = (lastEnforcedActions | ActionFlagEndOfTierDeny) &^ ActionFlagNoMatch
	}

	// And add the final verdict policy.
	addPolicyToResponse(lastEnforcedPolicy.Tier, lastEnforcedPolicy.FlowLogName, lastEnforcedActions, r)
	return lastEnforcedActions | combinedPolicyActions
}

// ActionAfter returns the calculated action for the tier for the supplied flow.
// A previous tier/policy may have specified a possible match action which could not be confirmed due to lack of
// information. We supply the current action flags so that further enumeration can exit as soon as we either find
// an identical action with confirmed match, or a different action (confirmed or unconfirmed) that means we cannot
// determine the result with certainty.
func (tier CompiledTier) ActionAfter(flow *Flow, r *EndpointResponse, cache *flowCache) ActionFlag {
	var lastMatchedPolicy *CompiledPolicy
	var lastMatchedActions ActionFlag
	var uncertainNoMatchPolicies []string
	uncertainNoMatchActions := make(map[string]ActionFlag)
	for _, p := range tier {
		log.Debugf("Process policy: %s", p.FlowLogName)

		// If the policy does not apply to this Endpoint then skip to the next policy.
		if p.Applies(flow, cache) != MatchTypeTrue {
			log.Debug("Policy does not apply - skipping")
			continue
		}
		//TODO(rlb): We may want to handle unknown selector matches if we decide to be a little more clever about our
		//           label aggregation.

		// Store the last matched policy. For the "after" processing we treat staged as enforced, so it's not necessary
		// to treat them differntly.
		lastMatchedPolicy = p

		// If the policy is not modified use the cached value if there is one, otherwise calculate.
		if p.Modified {
			log.Debug("Policy is modified - calculate action")
			lastMatchedActions = p.Action(flow, cache)
		} else if cachedActions, ok := cache.policies[p.FlowLogName]; ok {
			log.Debug("Use cached policy action")
			lastMatchedActions = cachedActions
		} else {
			log.Debug("No cached policy action - calculate")
			lastMatchedActions = p.Action(flow, cache)
		}

		if lastMatchedActions&ActionFlagNoMatch == 0 {
			// Policy has an exactly matching rule so no need to enumerate further.
			log.Debug("Policy has an exactly matching rule")
			break
		}

		// One of the possible results of the policy is a no-match. If, in addition, one of the real policy action
		// flags is set (i.e. pass, deny, allow) then this is an uncertain no-match.
		if lastMatchedActions&ActionFlagsAllPolicyActions != 0 {
			log.Debug("Policy is an uncertain no-match")
			uncertainNoMatchPolicies = append(uncertainNoMatchPolicies, p.FlowLogName)
			uncertainNoMatchActions[p.FlowLogName] = lastMatchedActions
		}
	}

	if lastMatchedPolicy == nil {
		// This flow didn't apply to any policy in this tier, so go to the next tier.
		log.Debug("Did not match tier - enumerate next tier")
		return ActionFlagNextTier
	}

	var combinedPolicyActions ActionFlag
	for _, n := range uncertainNoMatchPolicies {
		if n == lastMatchedPolicy.FlowLogName {
			// This is the last policy (handled separately below) so skip.
			continue
		}

		// Update previous uncertain no-matches to just be no-match inferred from flow log.
		noMatchPolicyActions := uncertainNoMatchActions[n]

		// Add each possible action to the flow policy hits.
		addPolicyToResponse(lastMatchedPolicy.Tier, n, noMatchPolicyActions, r)

		// Collate all of the action flags, but omit the no-match, we don't need to pass that up the stack since a
		// no match will never actually happen (due to end-of-tier drop).
		combinedPolicyActions |= noMatchPolicyActions &^ ActionFlagNoMatch
	}

	if lastMatchedActions&ActionFlagNoMatch != 0 {
		// The last enforced policy included a no-match flag. This must be the last enforced policy in the tier because
		// we'd otherwise exit as soon as we get an exactly matched rule. We need to convert the no-match to an
		// end-of-tier drop.
		log.Debug("Final policy included a no-match, include the end of tier action")
		lastMatchedActions = (lastMatchedActions | ActionFlagEndOfTierDeny) &^ ActionFlagNoMatch
	}

	// And add the final verdict policy.
	addPolicyToResponse(lastMatchedPolicy.Tier, lastMatchedPolicy.FlowLogName, lastMatchedActions, r)
	return lastMatchedActions | combinedPolicyActions
}

// addPolicyToResponse adds a policy to the endpoint response. This may add multiple entries for the same policy
// if there is uncertainty in the match.
func addPolicyToResponse(tier, name string, flags ActionFlag, r *EndpointResponse) {
	r.Policies = append(r.Policies, PolicyHit{
		MatchIndex: len(r.Policies),
		Tier:       tier,
		Name:       name,
		Action:     flags,
	})
}

// getFlagsFromFlowLog extracts the policy action flag from the flow log data.
func getFlagsFromFlowLog(flowLogName string, flow *Flow) (ActionFlag, bool) {
	var flagsFromFlowLog ActionFlag
	for _, p := range flow.Policies {
		// We have a match if the policy name (which will include the staged: for staged policies) is the same.
		if p.Name == flowLogName {
			thisActionFlag := p.Action
			log.Debugf("Policy %s in flow log has action flags %d", p.Name, thisActionFlag)
			if flagsFromFlowLog != 0 && flagsFromFlowLog != thisActionFlag {
				return 0, false
			}
			flagsFromFlowLog = thisActionFlag
		}
	}
	return flagsFromFlowLog, flagsFromFlowLog != 0
}
