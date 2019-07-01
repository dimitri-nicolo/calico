package policycalc

import (
	log "github.com/sirupsen/logrus"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

// CompiledPolicy contains the compiled policy matchers for either ingress _or_ egress policy rules.
type CompiledPolicy struct {
	Matchers []FlowMatcher
	Rules    []CompiledRule
}

// Applies determines whether the policy applies to the flow.
func (p *CompiledPolicy) Applies(flow *Flow) MatchType {
	mt := MatchTypeTrue
	for i := range p.Matchers {
		switch p.Matchers[i](flow) {
		case MatchTypeFalse:
			return MatchTypeFalse
		case MatchTypeUncertain:
			mt = MatchTypeUncertain
		}
	}
	return mt
}

// add adds the FlowMatcher to the set of matchers for the policy. It may be called with a nil matcher, in which case
// the policy is unchanged.
func (p *CompiledPolicy) add(fm FlowMatcher) {
	if fm == nil {
		// No matcher to add.
		return
	}
	p.Matchers = append(p.Matchers, fm)
}

// Action determines the action of this policy on the flow. It is assumed Applies() has already been invoked to
// determine if this policy actually applies to the flow.
func (c *CompiledPolicy) Action(flow *Flow, af ActionFlag) (finalActionType ActionFlag, gotoNextPolicy bool) {
	for i := range c.Rules {
		log.Debugf("Processing rule %d", i)
		switch c.Rules[i].Match(flow) {
		case MatchTypeTrue:
			// This rule matches exactly, so store off the action type for this rule and exit. No need to enumerate the
			// next policy since this was an exact match.
			log.Debug("Rule matches exactly")
			return af | c.Rules[i].Action, false
		case MatchTypeUncertain:
			// If the match type is unknown, then at this point we bifurcate by assuming we both matched and did not
			// match - we track that we would use this rules action, but continue enumerating until we either get
			// conflicting possible actions (at which point we deem the impact to be indeterminate), or we end up with
			// same action through all possible match paths.
			log.Debug("Rule is indeterminate")
			af |= c.Rules[i].Action

			// If the action is now indeterminate then exit immediately.
			if af.Indeterminate() {
				return af, false
			}
		}
	}

	// We got to the end of the rules, so enumerate the next policy in the tier.
	log.Debugf("Reached end of rules. ActionFlags=%v", af)
	return af, true
}

// CompiledRulesFromAPI creates a slice of compiled rules from the supplied v3 parameters.
func CompiledRulesFromAPI(m *MatcherFactory, namespace EndpointMatcher, rules []v3.Rule) []CompiledRule {
	compiled := make([]CompiledRule, len(rules))
	for i := range rules {
		compiled[i].FromAPI(m, namespace, rules[i])
	}
	return compiled
}
