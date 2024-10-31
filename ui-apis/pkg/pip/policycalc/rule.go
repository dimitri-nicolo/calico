package policycalc

import (
	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/lma/pkg/api"
)

// CompiledRule is created from an API v3 Rule and has pre-calculated matcher functions to speed up computation.
type CompiledRule struct {
	// The action type for this rule. The will be one of:
	// - ActionFlagAllow
	// - ActionFlagDeny, or
	// - ActionFlagNextTier
	ActionFlag api.ActionFlag

	// The matchers.
	Matchers []FlowMatcher
}

// Match determines whether this rule matches the supplied flow.
func (c *CompiledRule) Match(flow *api.Flow, cache *flowCache) MatchType {
	mt := MatchTypeTrue
	for i := range c.Matchers {
		log.Debugf("Invoking matcher %d", i)
		switch c.Matchers[i](flow, cache) {
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

// compileRules creates a slice of compiled rules from the supplied v3 parameters.
func compileRules(m *MatcherFactory, namespace EndpointMatcher, rules []v3.Rule) []CompiledRule {
	compiled := make([]CompiledRule, 0, len(rules))
	for i := range rules {
		if c := compileRule(m, namespace, rules[i]); c != nil {
			compiled = append(compiled, *c)
		}
	}
	return compiled
}

// compileRule creates a CompiledRule from the v3 Rule.
func compileRule(m *MatcherFactory, namespace EndpointMatcher, in v3.Rule) *CompiledRule {
	c := &CompiledRule{}

	// Set the action
	switch in.Action {
	case v3.Allow:
		c.ActionFlag = api.ActionFlagAllow
	case v3.Deny:
		c.ActionFlag = api.ActionFlagDeny
	case v3.Pass:
		c.ActionFlag = api.ActionFlagNextTier
	default:
		return nil
	}

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
	c.add(m.Src(m.ServiceSelector(in.Source.Services)))
	c.add(m.Src(m.Ports(in.Source.Ports)))
	c.add(m.Src(m.Domains(in.Source.Domains)))
	c.add(m.Src(m.ServiceAccounts(in.Source.ServiceAccounts)))
	c.add(m.Src(m.NotNets(in.Source.NotNets)))
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
	c.add(m.Dst(m.ServiceSelector(in.Destination.Services)))
	c.add(m.Dst(m.Ports(in.Destination.Ports)))
	c.add(m.Dst(m.Domains(in.Destination.Domains)))
	c.add(m.Dst(m.ServiceAccounts(in.Destination.ServiceAccounts)))
	c.add(m.Dst(m.NotNets(in.Destination.NotNets)))
	c.add(m.Not(m.Dst(m.Selector(in.Destination.NotSelector))))
	c.add(m.Not(m.Dst(m.Ports(in.Destination.NotPorts))))

	return c
}

// add adds the FlowMatcher to the set of matchers for the rule. It may be called with a nil matcher, in which case
// the rule is unchanged.
func (r *CompiledRule) add(fm FlowMatcher) {
	if fm == nil {
		// No matcher to add. This is fine - the short hand notation above may validly return a nil matcher.
		return
	}
	r.Matchers = append(r.Matchers, fm)
}
