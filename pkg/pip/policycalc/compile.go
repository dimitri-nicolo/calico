package policycalc

import (
	log "github.com/sirupsen/logrus"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/compliance/pkg/resources"
)

// compile compiles the Tiers into CompiledTiersAndPolicies.
func compile(cfg *Config, rd *ResourceData, modified ModifiedResources, sel *EndpointSelectorHandler) *CompiledTiersAndPolicies {
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
			// compile the policy to get the ingress and egress versions of the policy as appropriate.
			ingressPol, egressPol := compilePolicy(matcherFactory, pol)

			// Was this policy resource one of the resources modified in the proposed config change.
			isModified := modified.IsModified(pol)

			// Add the ingress and egress policies to their respective slices. If this is a modified policy, also
			// track it - we'll use this as a shortcut to determine if a flow is affected by the configuration change
			// or not.
			if ingressPol != nil {
				ingressTier = append(ingressTier, ingressPol)
				if isModified {
					c.IngressImpacted = append(c.IngressImpacted, ingressPol)
				}
			}
			if egressPol != nil {
				egressTier = append(egressTier, egressPol)
				if isModified {
					c.EgressImpacted = append(c.EgressImpacted, egressPol)
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

// compilePolicy compiles the Calico v3 policy resource into a CompiledPolicy for both ingress and egress flows.
// If the policy does not contain ingress or egress matches then the corresponding CompiledPolicy will be nil.
func compilePolicy(m *MatcherFactory, r resources.Resource) (ingressPol, egressPol *CompiledPolicy) {
	log.Debugf("Compiling policy %s", resources.GetResourceID(r))

	// Determine from the resource type, the namespace matcher, selector matcher and set of rules to use.
	//
	// The resource type here will either be a Calico NetworkPolicy or GlobalNetworkPolicy. Any Kubernetes
	// NetworkPolicies will have been converted to Calico NetworkPolicies prior to this point.
	var namespace EndpointMatcher
	var selector EndpointMatcher
	var ingress, egress []v3.Rule
	var types []v3.PolicyType
	switch res := r.(type) {
	case *v3.NetworkPolicy:
		namespace = m.Namespace(res.Namespace)
		selector = m.Selector(res.Spec.Selector)
		ingress, egress = res.Spec.Ingress, res.Spec.Egress
		types = res.Spec.Types
	case *v3.GlobalNetworkPolicy:
		selector = m.Selector(res.Spec.Selector)
		ingress, egress = res.Spec.Ingress, res.Spec.Egress
		types = res.Spec.Types
	default:
		log.WithField("res", res).Fatal("Unexpected policy resource type")
	}

	// Handle ingress policy matchers
	if policyTypesContains(types, v3.PolicyTypeIngress) {
		ingressPol = &CompiledPolicy{
			Rules: CompiledRulesFromAPI(m, namespace, ingress),
		}
		ingressPol.add(m.Dst(namespace))
		ingressPol.add(m.Dst(selector))
	}

	// Handle egress policy matchers
	if policyTypesContains(types, v3.PolicyTypeEgress) {
		egressPol = &CompiledPolicy{
			Rules: CompiledRulesFromAPI(m, namespace, egress),
		}
		egressPol.add(m.Src(namespace))
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
