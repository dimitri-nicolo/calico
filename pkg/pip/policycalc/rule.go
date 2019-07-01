package policycalc

import (
	log "github.com/sirupsen/logrus"

	"reflect"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

// CompiledRule is created from an API v3 Rule and has pre-calculated matcher functions to speed up computation.
type CompiledRule struct {
	// The action type for this rule. The will be one of:
	// - ActionFlagAllow
	// - ActionFlagDeny, or
	// - ActionFlagNextTier
	Action ActionFlag

	// The best possible match type for this rule. If any of the rules are *always* indeterminate then the best possible
	// match will be Uncertain.
	BestMatch MatchType

	// The matchers.
	Matchers []FlowMatcher
}

// Match determines whether this rule matches the supplied flow.
func (c *CompiledRule) Match(flow *Flow) MatchType {
	mt := c.BestMatch
	for i := range c.Matchers {
		log.Debugf("Invoking matcher %d", i)
		switch c.Matchers[i](flow) {
		case MatchTypeFalse:
			// We did not match this matcher, so exit immediately returning no match.
			log.Debug("No match")
			return MatchTypeFalse
		case MatchTypeUncertain:
			// The match is unknown, mark as unknown but continue enumerating the matchers because we may get a
			// negative match in which case we will never match this rule.
			log.Debug("Indeterminate match")
			mt = MatchTypeUncertain
		}
	}
	log.Debugf("Returning match type: %s", mt)
	return mt
}

// FromAPI fills in the CompiledRule from the v3 Rule.
func (c *CompiledRule) FromAPI(m *MatcherFactory, namespace EndpointMatcher, in v3.Rule) {
	// Reset matchers in case we are re-using an existing CompiledRule.
	c.Matchers = nil

	// Set the action
	switch in.Action {
	case v3.Allow:
		c.Action = ActionFlagAllow
	case v3.Deny:
		c.Action = ActionFlagDeny
	case v3.Pass:
		c.Action = ActionFlagNextTier
	default:
		// Set the best match type to MatchTypeFalse so that we don't stop enumerating when we hit this rule.
		// Exit without adding any match parameters, since we don't want to add unnecessary overhead.
		c.BestMatch = MatchTypeFalse
		return
	}

	// Assume the best match is MatchTypeTrue. If any of the match options indicates uncertainty this will be
	// changed to MatchTypeUncertain in the add() method.
	c.BestMatch = MatchTypeTrue

	// Add top level rule fields.
	c.add(m.IPVersion(in.IPVersion))
	c.add(m.Protocol(in.Protocol))
	c.add(m.ICMP(in.ICMP))
	c.add(m.HTTP(in.HTTP))
	c.add(m.Not(m.Protocol(in.NotProtocol)))
	c.add(m.Not(m.ICMP(in.NotICMP)))

	// Add source matchers
	if in.Source.Selector != "" && in.Source.NamespaceSelector == "" {
		// If a selector is specified without a namespace selector, then limit to endpoints with the same namespace as
		// the policy (for GNP which isn't namespaced, this won't add a matcher).
		c.add(m.Src(namespace))
	}
	c.add(m.Src(m.Nets(in.Source.Nets)))
	c.add(m.Src(m.Selector(in.Source.Selector)))
	c.add(m.Src(m.NamespaceSelector(in.Source.NamespaceSelector)))
	c.add(m.Src(m.Ports(in.Source.Ports)))
	c.add(m.Src(m.Domains(in.Source.Domains)))
	c.add(m.Src(m.ServiceAccounts(in.Source.ServiceAccounts)))
	c.add(m.Not(m.Src(m.Nets(in.Source.NotNets))))
	c.add(m.Not(m.Src(m.Selector(in.Source.NotSelector))))
	c.add(m.Not(m.Src(m.Ports(in.Source.NotPorts))))

	// Add destination matchers
	if in.Destination.Selector != "" && in.Destination.NamespaceSelector == "" {
		// If a selector is specified without a namespace selector, then limit to endpoints with the same namespace as
		// the policy (for GNP which isn't namespaced, this won't add a matcher).
		c.add(m.Dst(namespace))
	}
	c.add(m.Dst(m.Nets(in.Destination.Nets)))
	c.add(m.Dst(m.Selector(in.Destination.Selector)))
	c.add(m.Dst(m.NamespaceSelector(in.Destination.NamespaceSelector)))
	c.add(m.Dst(m.Ports(in.Destination.Ports)))
	c.add(m.Dst(m.Domains(in.Destination.Domains)))
	c.add(m.Dst(m.ServiceAccounts(in.Destination.ServiceAccounts)))
	c.add(m.Not(m.Dst(m.Nets(in.Destination.NotNets))))
	c.add(m.Not(m.Dst(m.Selector(in.Destination.NotSelector))))
	c.add(m.Not(m.Dst(m.Ports(in.Destination.NotPorts))))
}

// add adds the FlowMatcher to the set of matchers for the rule. It may be called with a nil matcher, in which case
// the rule is unchanged.
func (r *CompiledRule) add(fm FlowMatcher) {
	if fm == nil {
		// No matcher to add. This is fine - the short hand notation above may validly return a nil matcher.
		return
	}
	if reflect.ValueOf(fm) == reflect.ValueOf(FlowMatcherUncertain) {
		// Flow matcher will *always* return an uncertain for this rule match, so modify the best match result for this
		// rule to be "uncertain" and don't bother adding this matcher to the set of matches for the rule. This is a
		// minor finesse to avoid processing a match that will always return the same thing.
		log.Debugf("Rule matcher is always uncertain - best match for this rule is uncertain")
		r.BestMatch = MatchTypeUncertain
		return
	}

	r.Matchers = append(r.Matchers, fm)
}
