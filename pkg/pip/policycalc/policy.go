package policycalc

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/projectcalico/libcalico-go/lib/resources"
)

// CompiledPolicy contains the compiled policy matchers for either ingress _or_ egress policy rules.
type CompiledPolicy struct {
	// Name of the policy in flow log format. It does not include the tier index which needs to be pre-pended, nor the
	// action which needs to appended. This will be of the format:
	// -  |<tier>|<namespace>/<policy>|
	// -  |<tier>|<policy>|
	// for namespaces or non-namespaces policies respectively.
	Name string

	// Flow matchers for the main selector of the policy.
	MainSelectorMatchers []FlowMatcher

	// Endpoint matchers for the policy.
	Rules []CompiledRule

	// Whether this policy was modified.
	Modified bool
}

// Applies determines whether the policy applies to the flow.
func (p *CompiledPolicy) Applies(flow *Flow) MatchType {
	mt := MatchTypeTrue
	for i := range p.MainSelectorMatchers {
		switch p.MainSelectorMatchers[i](flow) {
		case MatchTypeFalse:
			return MatchTypeFalse
		case MatchTypeUncertain:
			mt = MatchTypeUncertain
		}
	}
	return mt
}

// Action determines the action of this policy on the flow. It is assumed Applies() has already been invoked to
// determine if this policy actually applies to the flow.
// It returns:
// -  The computed action for this policy. This may have multiple action flags set.
// -  Whether we need to enumerate the next policy in the tier.
func (c *CompiledPolicy) Action(flow *Flow, af ActionFlag, tierIdx int, r *EndpointResponse) (ActionFlag, bool) {
	var flagsThisPolicy ActionFlag
	defer func() {
		// Once finished with this policy add any matching policy actions to the set of policies.
		if flagsThisPolicy&ActionFlagAllow != 0 {
			r.Policies = append(r.Policies, c.calculateFlowLogName(tierIdx, ActionFlagAllow))
		}
		if flagsThisPolicy&ActionFlagDeny != 0 {
			r.Policies = append(r.Policies, c.calculateFlowLogName(tierIdx, ActionFlagDeny))
		}
		if flagsThisPolicy&ActionFlagNextTier != 0 {
			r.Policies = append(r.Policies, c.calculateFlowLogName(tierIdx, ActionFlagNextTier))
		}
	}()

	nextPolicy := true
	uncertain := false
	for i := range c.Rules {
		log.Debugf("Processing rule %d", i)
		switch c.Rules[i].Match(flow) {
		case MatchTypeTrue:
			// This rule matches exactly, so store off the action type for this rule and exit. No need to enumerate the
			// next policy, or rule since this was an exact match.
			log.Debug("Rule matches exactly")
			flagsThisPolicy |= c.Rules[i].Action
			nextPolicy = false
			break
		case MatchTypeUncertain:
			// If the match type is unknown, then at this point we bifurcate by assuming we both matched and did not
			// match - we track that we would use this rules action, but continue enumerating until we either get
			// conflicting possible actions (at which point we deem the impact to be indeterminate), or we end up with
			// same action through all possible match paths.
			log.Debug("Rule match is uncertain")
			flagsThisPolicy |= c.Rules[i].Action
			uncertain = true

			// If the action is now indeterminate then exit loop.
			if (af | flagsThisPolicy).Indeterminate() {
				nextPolicy = false
				break
			}
		}
	}

	// If we hit an unknown rule match then check the original flow to see if there is a definitive match for this
	// policy. We can only do this for policies that are not modified.
	if uncertain && !c.Modified {
		log.Debug("Check if flow contains exact match for this policy")
		if flowPolicyFlags := c.policyActionFlagInFlowLog(flow); flowPolicyFlags&flagsThisPolicy != 0 {
			// The flow contains a definitive match for this policy, and the policy flags match one of the enumerated
			// uncertain flags that we calculated above.
			log.Debug("Using exact match flags from flow log to break uncertainty")
			flagsThisPolicy = flowPolicyFlags
			nextPolicy = false
		}
	}

	// We got to the end of the rules, so enumerate the next policy in the tier.
	log.Debugf("Reached end of rules. ActionFlags=%v; nextPolicy=%v", af, nextPolicy)
	return af | flagsThisPolicy, nextPolicy
}

// policyActionFlagInFlowLog returns the policy action flag if present in the flow log.
func (c *CompiledPolicy) policyActionFlagInFlowLog(flow *Flow) ActionFlag {
	var af ActionFlag
	for _, p := range flow.Policies {
		// Split the flow log policy name by our policy name (which does not have the tier index or action). If this
		// flow log policy represents our policy then we'll get two parts - the tier index and action.
		parts := strings.Split(p, c.Name)
		if len(parts) == 2 {
			// We have found the policy
			af |= ActionFlagFromAction(Action(parts[1]))
		}
	}

	if af == ActionFlagAllow || af == ActionFlagDeny || af == ActionFlagNextTier {
		// Only return the flag if we found a definitive (single) result for this policy.
		return af
	}
	return 0
}

// compilePolicy compiles the Calico v3 policy resource into separate ingress and egress CompiledPolicy structs.
// If the policy does not contain ingress or egress matches then the corresponding result will be nil.
func compilePolicy(m *MatcherFactory, r resources.Resource, modified bool) (ingressPol, egressPol *CompiledPolicy) {
	log.Debugf("Compiling policy %s", resources.GetResourceID(r))

	// From the resource type, determine the namespace matcher, selector matcher and set of rules to use.
	//
	// The resource type here will either be a Calico NetworkPolicy or GlobalNetworkPolicy. Any Kubernetes
	// NetworkPolicies will have been converted to Calico NetworkPolicies prior to this point.
	var namespace EndpointMatcher
	var selector EndpointMatcher
	var serviceAccount EndpointMatcher
	var ingress, egress []v3.Rule
	var types []v3.PolicyType
	var name string
	switch res := r.(type) {
	case *v3.NetworkPolicy:
		namespace = m.Namespace(res.Namespace)
		// borrow the ServiceAccounts matcher factory since it's functionality is a superset of what we need
		serviceAccount = m.ServiceAccounts(&v3.ServiceAccountMatch{Selector: res.Spec.ServiceAccountSelector})
		selector = m.Selector(res.Spec.Selector)
		ingress, egress = res.Spec.Ingress, res.Spec.Egress
		types = res.Spec.Types
		name = fmt.Sprintf("|%s|%s/%s|", res.Spec.Tier, res.Namespace, res.Name)
	case *v3.GlobalNetworkPolicy:
		namespace = m.NamespaceSelector(res.Spec.NamespaceSelector)
		serviceAccount = m.ServiceAccounts(&v3.ServiceAccountMatch{Selector: res.Spec.ServiceAccountSelector})
		selector = m.Selector(res.Spec.Selector)
		ingress, egress = res.Spec.Ingress, res.Spec.Egress
		types = res.Spec.Types
		name = fmt.Sprintf("|%s|%s|", res.Spec.Tier, res.Name)
	default:
		log.WithField("res", res).Fatal("Unexpected policy resource type")
	}

	// Handle ingress policy matchers
	if policyTypesContains(types, v3.PolicyTypeIngress) {
		ingressPol = &CompiledPolicy{
			Name:     name,
			Rules:    compileRules(m, namespace, ingress),
			Modified: modified,
		}
		ingressPol.add(m.Dst(namespace))
		ingressPol.add(m.Dst(serviceAccount))
		ingressPol.add(m.Dst(selector))
	}

	// Handle egress policy matchers
	if policyTypesContains(types, v3.PolicyTypeEgress) {
		egressPol = &CompiledPolicy{
			Name:     name,
			Rules:    compileRules(m, namespace, egress),
			Modified: modified,
		}
		egressPol.add(m.Src(namespace))
		egressPol.add(m.Src(serviceAccount))
		egressPol.add(m.Src(selector))
	}

	return
}

// policyTypesContains checks if the supplied policy type is in the policy type slice
func policyTypesContains(s []v3.PolicyType, e v3.PolicyType) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// add adds the FlowMatcher to the set of matchers for the policy. It may be called with a nil matcher, in which case
// the policy is unchanged.
func (p *CompiledPolicy) add(fm FlowMatcher) {
	if fm == nil {
		// No matcher to add.
		return
	}
	p.MainSelectorMatchers = append(p.MainSelectorMatchers, fm)
}

// calculateFlowLogName calculates the flow log name for this policy. This name is endpoint specific since the tier
// index is dependent on the policy.
func (p *CompiledPolicy) calculateFlowLogName(tierIdx int, af ActionFlag) string {
	return fmt.Sprintf("%d%s%s", tierIdx, p.Name, af.ToAction())
}
