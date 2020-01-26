// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package policyrec

import (
	"fmt"
	"strings"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/numorstring"
	"github.com/projectcalico/libcalico-go/lib/set"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/lma/pkg/api"
)

// Labels to ignore. These labels are known ones that are generated by controllers or can change.
var labelKeysToIgnore = set.FromArray([]string{
	"pod-template-hash",
	"job-name",
	"controller-uid",
	"pod-template-generation",
	"controller-revision-hash",
	"statefulset.kubernetes.io/pod-name",
	"app.kubernetes.io/instance",
})

const (
	noNamespace = ""

	// TODO(doublek): Import these from libcalico-go when we bump pins.
	namespaceByNameLabel    = "projectcalico.org/name"
	globalNamespaceSelector = "global()"
)

// The rules for recommend policy are aggregated per endpoint + protocol with
// a list of ports and computed selector.

// endpointRulePerProtocol is the key that represents this.
type endpointRulePerProtocol struct {
	endpointName      string
	endpointNamespace string
	protocol          numorstring.Protocol
}

// entityRule is the value that holds the list of ports and computed selectors.
type entityRule struct {
	selector selectorBuilder
	ports    set.Set
	// TODO(doublek): Add service account string
}

// endpointRecommendationEngine implements the RecommendationEngine interface.
// Policies are recommended for a given endpoint in a namespace.
type endpointRecommendationEngine struct {
	// endpointName is the name of endpoint for which the policy recommendation
	// was requested for.
	endpointName string
	// endpointNamespace is the namespace of the endpoint for which the
	// policy recommendation was requested for.
	endpointNamespace string

	// The following fields are generated fields by the recommendation engine.
	// name of the recommended policy.
	policyName string
	// namespace of the recommended policy.
	policyNamespace string
	// The tier of the policy - obtained from observation point.
	policyTier string
	// The order of the policy - obtained from observation point.
	policyOrder *float64
	// selector of the recommended policy.
	// We currently only support AND-ed selectors. All the key:value
	// pairs of the map are AND-ed in the final selector string.
	policySelector selectorBuilder
	// ingress rules of the recommended policy.
	// TODO(doublek): This doesn't need to be here. It can probably be constructed
	// and used on the fly.
	ingressRules []v3.Rule
	// egress rules of the recommended policy.
	// TODO(doublek): This doesn't need to be here. It can probably be constructed
	// and used on the fly.
	egressRules []v3.Rule
	// ingressTraffic tracks the ingress traffic to this endpoint
	ingressTraffic map[endpointRulePerProtocol]entityRule
	// egressTraffic tracks the egress traffic to this endpoint
	egressTraffic map[endpointRulePerProtocol]entityRule
}

func NewEndpointRecommendationEngine(name, namespace, policyName, policyTier string, policyOrder *float64) *endpointRecommendationEngine {
	return &endpointRecommendationEngine{
		endpointName:      name,
		endpointNamespace: namespace,
		policyName:        policyName,
		policyNamespace:   namespace,
		policyTier:        policyTier,
		policyOrder:       policyOrder,
		ingressRules:      []v3.Rule{},
		egressRules:       []v3.Rule{},
		ingressTraffic:    make(map[endpointRulePerProtocol]entityRule),
		egressTraffic:     make(map[endpointRulePerProtocol]entityRule),
	}
}

// ProcessFlow takes a flow log and updates the recommendation engine policies.
func (ere *endpointRecommendationEngine) ProcessFlow(flow api.Flow) error {
	// We only support allowed flows.
	if flow.Action != api.ActionAllow {
		return fmt.Errorf("%v isn't an allowed flow", flow)
	}

	// Make sure we only process flows that have either source or destination endpoint name/namespace
	// that we expect.
	if !ere.matchesSourceEndpoint(flow) && !ere.matchesDestinationEndpoint(flow) {
		return fmt.Errorf("namespace/name of flow %v don't match request or endpoint isn't a Workload Endpoint", flow)
	}

	// Update selector from flow.
	ere.updateSelectorFromFlow(flow)

	// Next up is constructing rules.
	ere.processRuleFromFlow(flow)

	return nil
}

// Recommend returns a recommendation containing network policies based on the processing flow logs.
func (ere *endpointRecommendationEngine) Recommend() (*Recommendation, error) {
	recommendation := &Recommendation{
		NetworkPolicies:       make([]*v3.StagedNetworkPolicy, 0),
		GlobalNetworkPolicies: make([]*v3.StagedGlobalNetworkPolicy, 0),
	}
	ere.constructRulesFromTraffic()

	if len(ere.ingressRules) == 0 && len(ere.egressRules) == 0 {
		return nil, fmt.Errorf("Could not calculate any rules for namespace/name %v/%v", ere.endpointNamespace, ere.endpointName)
	}

	policyTypes := []v3.PolicyType{}
	if len(ere.ingressRules) > 0 {
		policyTypes = append(policyTypes, v3.PolicyTypeIngress)
	}
	if len(ere.egressRules) > 0 {
		policyTypes = append(policyTypes, v3.PolicyTypeEgress)
	}
	if ere.policyNamespace == noNamespace {
		// If the engine is for an endpoint with no namespace, then we
		// recommend a globalnetworkpolicy.
		gnp := v3.NewStagedGlobalNetworkPolicy()
		gnp.ObjectMeta = metav1.ObjectMeta{
			Name: fmt.Sprintf("%s.%s", ere.policyTier, ere.policyName),
		}
		policySelector := ere.policySelector.Expression()
		if policySelector == "" {
			log.Errorf("Could not compute selector for namespace/name: %v/%v", ere.endpointNamespace, ere.endpointName)
			return nil, fmt.Errorf("Could not compute selector for namespace/name: %v/%v", ere.endpointNamespace, ere.endpointName)
		}
		gnp.Spec = v3.StagedGlobalNetworkPolicySpec{
			StagedAction: v3.StagedActionSet,
			Tier:         ere.policyTier,
			Types:        policyTypes,
			Selector:     policySelector,
			Ingress:      ere.ingressRules,
			Egress:       ere.egressRules,
		}
		recommendation.GlobalNetworkPolicies = append(recommendation.GlobalNetworkPolicies, gnp)
	} else {
		np := v3.NewStagedNetworkPolicy()
		np.ObjectMeta = metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s.%s", ere.policyTier, ere.policyName),
			Namespace: ere.policyNamespace,
		}
		policySelector := ere.policySelector.Expression()
		if policySelector == "" {
			log.Errorf("Could not compute selector for namespace/name: %v/%v", ere.endpointNamespace, ere.endpointName)
			return nil, fmt.Errorf("Could not compute selector for namespace/name: %v/%v", ere.endpointNamespace, ere.endpointName)
		}
		np.Spec = v3.StagedNetworkPolicySpec{
			StagedAction: v3.StagedActionSet,
			Tier:         ere.policyTier,
			Types:        policyTypes,
			Selector:     policySelector,
			Ingress:      ere.ingressRules,
			Egress:       ere.egressRules,
		}
		recommendation.NetworkPolicies = append(recommendation.NetworkPolicies, np)
	}
	return recommendation, nil
}

// updateSelectorFromFlow extracts labels from relevant flows and updates the
// main policy selector.
func (ere *endpointRecommendationEngine) updateSelectorFromFlow(flow api.Flow) {
	var labels map[string]string
	if ere.matchesSourceEndpoint(flow) {
		labels = flow.Source.Labels
	}
	if ere.matchesDestinationEndpoint(flow) {
		labels = flow.Destination.Labels
	}
	if ere.policySelector != nil {
		ere.policySelector.IntersectLabels(labels)
	} else {
		ere.policySelector = NewSelectorBuilder(labels)
	}
}

// processRuleFromFlow collects traffic information for contructing policy rules.
func (ere *endpointRecommendationEngine) processRuleFromFlow(flow api.Flow) {
	// Process a flow and append to appropriate rule. Rules are processed
	// in the following order:
	//   1. Service account rules go first. TODO(doublek).
	//   2. Named ports matches go next. TODO(doublek).
	//   3. Finally, port + protocol matches.
	// The current assumption is that a single flow will yield a single rule.

	if flow.Reporter == api.ReporterTypeSource && ere.matchesSourceEndpoint(flow) {
		// A flow reported at the source will add a egress rule to an destination port + protocol.
		erpp := endpointRulePerProtocol{
			endpointName:      flow.Destination.Name,
			endpointNamespace: flow.Destination.Namespace,
			protocol:          api.GetProtocol(*flow.Proto),
		}
		rule, ok := ere.egressTraffic[erpp]
		if ok {
			rule.selector.IntersectLabels(flow.Destination.Labels)
			rule.ports.Add(numorstring.SinglePort(*flow.Destination.Port))
		} else {
			rule = entityRule{
				selector: NewSelectorBuilder(flow.Destination.Labels),
				ports:    set.From(numorstring.SinglePort(*flow.Destination.Port)),
			}
		}
		ere.egressTraffic[erpp] = rule
		log.Debugf("Adding egress traffic %+v with labels %+v", erpp, rule)
	} else if flow.Reporter == api.ReporterTypeDestination && ere.matchesDestinationEndpoint(flow) {
		// A flow reported at the destination will add an ingress rule to the destination port + protocol.
		erpp := endpointRulePerProtocol{
			endpointName:      flow.Source.Name,
			endpointNamespace: flow.Source.Namespace,
			protocol:          api.GetProtocol(*flow.Proto),
		}

		rule, ok := ere.ingressTraffic[erpp]
		if ok {
			rule.selector.IntersectLabels(flow.Source.Labels)
			rule.ports.Add(numorstring.SinglePort(*flow.Destination.Port))
		} else {
			rule = entityRule{
				selector: NewSelectorBuilder(flow.Source.Labels),
				ports:    set.From(numorstring.SinglePort(*flow.Destination.Port)),
			}
		}
		ere.ingressTraffic[erpp] = rule
		log.Debugf("Adding ingress traffic %+v with labels %+v", erpp, rule)
	}
}

// constructRulesFromTraffic creates ingress and egress rules for use in a policy.
func (ere *endpointRecommendationEngine) constructRulesFromTraffic() {
	if len(ere.egressTraffic) != 0 {
		ere.egressRules = ere.rulesFromTraffic(v3.PolicyTypeEgress, ere.egressTraffic)
	}
	if len(ere.ingressTraffic) != 0 {
		ere.ingressRules = ere.rulesFromTraffic(v3.PolicyTypeIngress, ere.ingressTraffic)
	}
}

// rulesFromTraffic is a convenience method for converting intermediate traffic representation
// to libcalico-go v3 Rule object.
func (ere *endpointRecommendationEngine) rulesFromTraffic(policyType v3.PolicyType, trafficAndSelector map[endpointRulePerProtocol]entityRule) []v3.Rule {
	log.WithField("policyType", policyType).Debugf("Processing rules for traffic %+v", trafficAndSelector)
	rules := make([]v3.Rule, 0)
	for erpp, rule := range trafficAndSelector {
		ports := make([]numorstring.Port, 0)
		rule.ports.Iter(func(item interface{}) error {
			ports = append(ports, item.(numorstring.Port))
			return nil
		})
		proto := erpp.protocol
		newRule := v3.Rule{
			Action:   v3.Allow,
			Protocol: &proto,
			Destination: v3.EntityRule{
				Ports: ports,
			},
		}
		// The entityRule is translated to an ingress or egress policy depending on the
		// policyType specified. Rule selectors are constructed based on the following.
		// Rule selectors only select endpoints within the main policy selectors namespace.
		// To workaround this issue, we do the following:
		// 1. For namespaced endpoints not belonging to the current policy's endpoint, we
		//    include the "all()" as the namespaceSelector to select all namespaced endpoints
		//    and then specify the current policy's namespace using the hidden
		//    "projectcalico.org/namespace == '<namespace>'" selector.
		// 2. For global/non-namespaced endpoints, we specify a "!all()" selector which
		//    will select all non namespaced endpoints.
		// TODO(doublek): Fix this when we have nicer namespaced selectors.
		switch policyType {
		case v3.PolicyTypeEgress:
			if erpp.endpointNamespace == noNamespace {
				// Only include the !all() selector if we want to actually select a Calico
				// HostEndpoint. For all other non namespaced non-Calico endpoints, we just leave
				// out all selectors.
				if !rule.selector.IsEmpty() {
					newRule.Destination.NamespaceSelector = globalNamespaceSelector
				}
			} else if erpp.endpointNamespace != ere.endpointNamespace {
				newRule.Destination.NamespaceSelector = selectNamespaceByName(erpp.endpointNamespace)
			}
			newRule.Destination.Selector = rule.selector.Expression()
		case v3.PolicyTypeIngress:
			if erpp.endpointNamespace == noNamespace {
				// Only include the !all() selector if we want to actually select a Calico
				// HostEndpoint. For all other non namespaced non-Calico endpoints, we just leave
				// out all selectors.
				if !rule.selector.IsEmpty() {
					newRule.Source.NamespaceSelector = globalNamespaceSelector
				}
			} else if erpp.endpointNamespace != ere.endpointNamespace {
				newRule.Source.NamespaceSelector = selectNamespaceByName(erpp.endpointNamespace)
			}
			newRule.Source.Selector = rule.selector.Expression()
		}
		rules = append(rules, newRule)
	}
	log.WithField("policyType", policyType).Debugf("Rules for traffic %+v", rules)
	return rules
}

// Check if the flow matches the source endpoint.
func (ere *endpointRecommendationEngine) matchesSourceEndpoint(flow api.Flow) bool {
	return flow.Source.Name == ere.endpointName &&
		flow.Source.Namespace == ere.endpointNamespace &&
		flow.Source.Type == api.EndpointTypeWep &&
		flow.Reporter == api.ReporterTypeSource
}

// Check if the flow matches the destination endpoint.
func (ere *endpointRecommendationEngine) matchesDestinationEndpoint(flow api.Flow) bool {
	return flow.Destination.Name == ere.endpointName &&
		flow.Destination.Namespace == ere.endpointNamespace &&
		flow.Destination.Type == api.EndpointTypeWep &&
		flow.Reporter == api.ReporterTypeDestination
}

// selectorBuilder wraps a map to provide convenience methods selector construction.
type selectorBuilder map[string]string

// Creates and initializes a selectorBuilder with the provided labels.
func NewSelectorBuilder(labels map[string]string) selectorBuilder {
	sb := make(selectorBuilder)
	for k, v := range labels {
		sb[k] = v
	}
	return sb
}

// IntersectLabels creates the intersection of current labels with additional
// labels provided.
func (sb selectorBuilder) IntersectLabels(labels map[string]string) {
	// If there are no labels present, intersection should be empty as well.
	if len(sb) == 0 {
		return
	}
	for k, v := range labels {
		value, ok := sb[k]
		if ok && value != v {
			delete(sb, k)
		}
	}
	for k := range sb {
		_, ok := labels[k]
		if !ok {
			delete(sb, k)
		}
	}
}

// Expression constructs a selector expression for the labels stored in the
// selectorBuilder. Currently only "&&"-ed selector expressions are supported.
func (sb selectorBuilder) Expression() string {
	if len(sb) == 0 {
		return ""
	}
	expressionParts := []string{}
	for k, v := range sb {
		if labelKeysToIgnore.Contains(k) {
			continue
		}
		expressionParts = append(expressionParts, fmt.Sprintf("%s == '%s'", k, v))
	}
	return strings.Join(expressionParts, " && ")
}

// IsEmpty returns if there are no labels present in the label map.
func (sb selectorBuilder) IsEmpty() bool {
	return len(sb) == 0
}

// selectNamespaceByName constructs a selector of the form
// "projectcalico.org/name == 'passed-in-namespace'". This is suitable for use
// in a namespaceSelector to select a namespace by name.
func selectNamespaceByName(namespace string) string {
	return fmt.Sprintf("%s == '%s'", namespaceByNameLabel, namespace)
}
