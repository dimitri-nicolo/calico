// Copyright (c) 2023 Tigera Inc. All rights reserved.

package engine

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
	"github.com/tigera/api/pkg/lib/numorstring"

	"github.com/projectcalico/calico/libcalico-go/lib/set"
	linseed "github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/lma/pkg/api"
	calicores "github.com/projectcalico/calico/policy-recommendation/pkg/calico-resources"
	"github.com/projectcalico/calico/policy-recommendation/pkg/flows"
)

// Clock is an interface added for testing purposes.
type Clock interface {
	NowRFC3339() string
}

// recommendationEngine implements the RecommendationEngine interface.
// Policies are recommended for a given endpoint in a namespace.
type recommendationEngine struct {
	// Name of the recommended policy.
	name string
	// Namespace of the recommended policy.
	namespace string
	// Tier of the policy - obtained from observation point.
	tier string
	// Order of the policy - obtained from observation point.
	order *float64

	// Egress engine incoming rules.
	egress *engineRules
	// Ingress engine incoming rules.
	ingress *engineRules

	// Network sets
	nets set.Set[NetworkSet]

	// Clock used for formatting and testing purposes.
	clock Clock
	// Recommendation interval duration.
	interval time.Duration
	// Stabilization interval.
	stabilization time.Duration

	// log entry
	clog log.Entry
}

type NetworkSet struct {
	Name      string
	Namespace string
}

// RunEngine queries flows logs, processes and updates staged network policies with new
// recommendations, if those exist
func RunEngine(
	ctx context.Context,
	calico calicoclient.ProjectcalicoV3Interface,
	linseedClient linseed.Client,
	lookback time.Duration,
	order *float64,
	clusterID string,
	clock Clock,
	recInterval time.Duration,
	stabilizationPeriod time.Duration,
	owner *metav1.OwnerReference,
	snp *v3.StagedNetworkPolicy,
) {
	clog := log.WithField("cluster", clusterID)

	if snp == nil {
		clog.Debugf("empty staged network policy")
		return
	}

	if snp.Spec.StagedAction != v3.StagedActionLearn {
		// Engine only processes 'Learn' policies
		clog.Debugf("Ignoring %s policy", snp.Name)
		return
	}

	tier := snp.Spec.Tier
	if err := calicores.MaybeCreateTier(ctx, calico, tier, order); err != nil {
		// If a tier does not exist create it. Recommendation policy needs a recommendation tier
		clog.WithError(err).Debugf("failed to create tier: %s", tier)
		return
	}

	// Update the status annotation, if necessary
	emptyRules := len(snp.Spec.Egress) == 0 && len(snp.Spec.Ingress) == 0
	updateStatusAnnotation(snp, emptyRules, clock.NowRFC3339(), recInterval, stabilizationPeriod)

	namespace := snp.Namespace

	// Define flow log query params
	params := getNamespacePolicyRecParams(lookback, namespace, clusterID)

	// Query flows
	query := flows.NewPolicyRecommendationQuery(ctx, linseedClient, clusterID)
	flows, err := query.QueryFlows(params)
	if err != nil {
		clog.WithError(err).WithField("params", params).Debug("Error querying flows")
		return
	}
	if len(flows) == 0 {
		clog.WithField("params", params).Debug("No matching flows found")
		return
	}

	// Update staged network policy
	engine := getRecommendationEngine(*snp, clock, recInterval, stabilizationPeriod, *clog)
	engine.processRecommendation(flows, snp)

	// If private network flows have generated new rules, then create a 'private-network' global
	// network set. The global network set must be updated manually by the user to include new
	// subnets.
	if len(engine.egress.privateNetworkRules) > 0 || len(engine.ingress.privateNetworkRules) > 0 {
		clog.Infof("Creating global network set: 'private-network'")
		if err := calicores.MaybeCreatePrivateNetworkSet(ctx, calico, *owner); err != nil {
			clog.WithError(err).Errorf("failed to create private network set: %s", calicores.PrivateNetworkSetName)
			return
		}
	}
}

// Recommendation Engine

// NewRecommendationEngine returns a new recommendation engine.
func newRecommendationEngine(
	name, namespace, tier string,
	order *float64,
	clock Clock,
	interval time.Duration,
	stabilization time.Duration,
	clog log.Entry,
) *recommendationEngine {
	return &recommendationEngine{
		name:          name,
		namespace:     namespace,
		tier:          tier,
		order:         order,
		egress:        NewEngineRules(),
		ingress:       NewEngineRules(),
		nets:          set.New[NetworkSet](),
		clock:         clock,
		interval:      interval,
		stabilization: stabilization,
		clog:          clog,
	}
}

func (ere *recommendationEngine) buildRules(dir calicores.DirectionType, rules []v3.Rule) {
	var scope string
	var ok bool
	for _, rule := range rules {
		if rule.Metadata == nil {
			ere.clog.Warn("recommended rule metadata is empty")
			continue
		}
		scope, ok = rule.Metadata.Annotations[calicores.ScopeKey]
		if !ok {
			ere.clog.Warn("recommended rule does not contain a scope")
			continue
		}
		switch scope {
		case string(calicores.EgressToDomainScope):
			ere.buildEgressToDomain(rule)
		case string(calicores.EgressToDomainSetScope):
			// TODO(dimitrin): Create buildEgressToDomainSet
		case string(calicores.EgressToServiceScope):
			ere.buildEgressToService(rule)
		case string(calicores.NamespaceScope):
			ere.buildNamespace(dir, rule)
		case string(calicores.NetworkSetScope):
			ere.buildNetworkSet(dir, rule)
		case string(calicores.PrivateNetworkScope):
			ere.buildPrivate(dir, rule)
		case string(calicores.PublicNetworkScope):
			ere.buildPublic(dir, rule)
		default:
			ere.clog.Warnf("Invalid scope: %s", scope)
		}
	}
}

// ProcessFlow adds the flow the recommendation engine rules. Actions other than allow, namespace
// mismatches and non WEP destination flows are skipped.
func (ere *recommendationEngine) processFlow(flow *api.Flow) {
	if flow == nil {
		return
	}
	ere.clog.Debugf("Processing flow: %+v", flow)

	// Only allowed flows are used to recommend policy
	if flow.ActionFlag&api.ActionFlagAllow == 0 {
		ere.clog.Debug("Skipping flow, only allow action processed")
		return
	}
	// Make sure we only process flows that have either source or destination in the expected
	// namespace
	if !ere.matchesSourceNamespace(*flow) && !ere.matchesDestinationNamespace(*flow) {
		// Skip this flow, as it doesn't match the namespace, or is not WEP
		ere.clog.Debug("Skipping flow, namespace mismatch or destination isn't WEP")
		return
	}
	// Construct rule
	ere.processEngineRuleFromFlow(*flow)
}

// buildEgressToDomain creates a new EgressToDomain engine rule key. It is assumed that each new
// rule will generate a new key.
func (ere *recommendationEngine) buildEgressToDomain(rule v3.Rule) {
	if len(rule.Destination.Ports) == 0 {
		err := fmt.Errorf("no ports in this rule")
		ere.clog.WithError(err)
		return
	}
	// 'Domains' rules should only contain one port
	port := rule.Destination.Ports[0]
	key := egressToDomainRuleKey{
		port:     port,
		protocol: *rule.Protocol,
	}
	val := &egressToDomainRule{
		domains:   rule.Destination.Domains,
		port:      port,
		protocol:  *rule.Protocol,
		timestamp: rule.Metadata.Annotations[calicores.LastUpdatedKey],
	}
	ere.egress.egressToDomainRules[key] = val
	ere.egress.size++
}

// buildEgressToService creates a new EgressToService engine rule key. It is assumed that each new
// rule will generate a new key.
func (ere *recommendationEngine) buildEgressToService(rule v3.Rule) {
	key := egressToServiceRuleKey{
		name:      rule.Destination.Services.Name,
		namespace: rule.Destination.Services.Namespace,
		protocol:  *rule.Protocol,
	}
	val := &egressToServiceRule{
		name:      rule.Destination.Services.Name,
		namespace: rule.Destination.Services.Namespace,
		ports:     rule.Destination.Ports,
		protocol:  *rule.Protocol,
		timestamp: rule.Metadata.Annotations[calicores.LastUpdatedKey],
	}
	ere.egress.egressToServiceRules[key] = val
	ere.egress.size++
}

// buildNamespace creates a new Namespace engine rule key. It is assumed that each new
// rule will generate a new key.
func (ere *recommendationEngine) buildNamespace(dir calicores.DirectionType, rule v3.Rule) {
	nsKey := fmt.Sprintf("%s/namespace", calicores.PolicyRecKeyName)
	ns, ok := getRuleMetadata(nsKey, rule)
	if !ok {
		return
	}
	ts, ok := getRuleMetadata(calicores.LastUpdatedKey, rule)
	if !ok {
		return
	}

	key := namespaceRuleKey{
		namespace: ns,
		protocol:  *rule.Protocol,
	}
	val := &namespaceRule{
		namespace: ns,
		ports:     rule.Destination.Ports, // Always destination
		protocol:  *rule.Protocol,
		timestamp: ts,
	}

	var erules *engineRules
	if dir == calicores.EgressTraffic {
		erules = ere.egress
	} else {
		erules = ere.ingress
	}
	erules.namespaceRules[key] = val
	erules.size++
}

// buildNetworkSet creates a new NetworkSet engine rule key. It is assumed that each new rule will
// generate a new key.
func (ere *recommendationEngine) buildNetworkSet(dir calicores.DirectionType, rule v3.Rule) {
	entity := getEntityRule(dir, &rule)
	gl := false
	var ns string
	if entity.NamespaceSelector == "global()" {
		gl = true
		ns = ""
	} else {
		nsKey := fmt.Sprintf("%s/namespace", calicores.PolicyRecKeyName)
		ok := false
		if ns, ok = getRuleMetadata(nsKey, rule); !ok {
			return
		}
	}

	nameKey := fmt.Sprintf("%s/name", calicores.PolicyRecKeyName)
	name, ok := getRuleMetadata(nameKey, rule)
	if !ok {
		return
	}
	ts, ok := getRuleMetadata(calicores.LastUpdatedKey, rule)
	if !ok {
		return
	}

	key := networkSetRuleKey{
		global:    gl,
		name:      name,
		namespace: ns,
		protocol:  *rule.Protocol,
	}
	val := &networkSetRule{
		global:    gl,
		name:      name,
		namespace: ns,
		ports:     rule.Destination.Ports, // Always destination
		protocol:  *rule.Protocol,
		timestamp: ts,
	}

	var erules *engineRules
	if dir == calicores.EgressTraffic {
		erules = ere.egress
	} else {
		erules = ere.ingress
	}
	erules.networkSetRules[key] = val
	erules.size++
}

// buildPrivate creates a new PrivateNetwork engine rule key. It is assumed that each new
// rule will generate a new key.
func (ere *recommendationEngine) buildPrivate(dir calicores.DirectionType, rule v3.Rule) {
	if len(rule.Destination.Ports) == 0 {
		err := errors.New("no ports in private rule")
		ere.clog.WithError(err)
		return
	}

	nameKey := fmt.Sprintf("%s/name", calicores.PolicyRecKeyName)
	name, ok := getRuleMetadata(nameKey, rule)
	if !ok {
		return
	}
	ts, ok := getRuleMetadata(calicores.LastUpdatedKey, rule)
	if !ok {
		return
	}

	key := privateNetworkRuleKey{
		protocol: *rule.Protocol,
	}
	val := &privateNetworkRule{
		name:      name,
		protocol:  *rule.Protocol,
		ports:     rule.Destination.Ports, // Always destination
		timestamp: ts,
	}

	var erules *engineRules
	if dir == calicores.EgressTraffic {
		erules = ere.egress
	} else {
		erules = ere.ingress
	}
	erules.privateNetworkRules[key] = val
	erules.size++
}

// buildPublic creates a new PublicNetwork engine rule key. It is assumed that each new
// rule will generate a new key.
func (ere *recommendationEngine) buildPublic(dir calicores.DirectionType, rule v3.Rule) {
	ts, ok := getRuleMetadata(calicores.LastUpdatedKey, rule)
	if !ok {
		return
	}

	key := publicNetworkRuleKey{
		protocol: *rule.Protocol,
	}
	val := &publicNetworkRule{
		ports:     rule.Destination.Ports,
		protocol:  *rule.Protocol,
		timestamp: ts,
	}

	var engRules *engineRules
	if dir == calicores.EgressTraffic {
		engRules = ere.egress
	} else {
		engRules = ere.ingress
	}
	engRules.publicNetworkRules[key] = val

	engRules.size++
}

// getSortedEngineAsV3Rules returns a list of sorted v3 rules. Each v3 rules is sorted by the keys
// that define the equivalent recommendation engine rule. The engine rules are first converted into
//
//	v3 rules, which are subsequently sorted.
func (ere *recommendationEngine) getSortedEngineAsV3Rules(direction calicores.DirectionType) []v3.Rule {
	var engRules *engineRules
	if direction == calicores.EgressTraffic {
		engRules = ere.egress
	} else {
		engRules = ere.ingress
	}

	// Maintain order for each engine rules type
	rules := []v3.Rule{}

	// EgressToDomain
	edRules := []v3.Rule{}
	for _, er := range engRules.egressToDomainRules {
		rule := calicores.GetEgressToDomainV3Rule(er.domains, er.port, &er.protocol, er.timestamp)
		edRules = append(edRules, *rule)
	}
	sort.Slice(edRules, func(i, j int) bool {
		return lessEgressToDomain(edRules[i], edRules[j])
	})
	rules = append(rules, edRules...)

	// EgressToService
	esRules := []v3.Rule{}
	for _, er := range engRules.egressToServiceRules {
		rule := calicores.GetEgressToServiceV3Rule(er.name, er.namespace, er.ports, &er.protocol, er.timestamp)
		esRules = append(esRules, *rule)
	}
	sort.Slice(esRules, func(i, j int) bool {
		return lessEgressToService(esRules[i], esRules[j])
	})
	rules = append(rules, esRules...)

	// Namespace
	nsRules := []v3.Rule{}
	for _, er := range engRules.namespaceRules {
		rule := calicores.GetNamespaceV3Rule(direction, er.namespace, er.ports, &er.protocol, er.timestamp)
		nsRules = append(nsRules, *rule)
	}
	sort.Slice(nsRules, func(i, j int) bool {
		return lessNamespace(direction, nsRules[i], nsRules[j])
	})
	rules = append(rules, nsRules...)

	// NetworkSet
	netsetRules := []v3.Rule{}
	for _, er := range engRules.networkSetRules {
		rule := calicores.GetNetworkSetV3Rule(direction, er.name, er.namespace, er.global, er.ports, &er.protocol, er.timestamp)
		netsetRules = append(netsetRules, *rule)
	}
	sort.Slice(netsetRules, func(i, j int) bool {
		return lessNetworkSet(direction, netsetRules[i], netsetRules[j])
	})
	rules = append(rules, netsetRules...)

	// PrivateNetwork
	prnRules := []v3.Rule{}
	for _, er := range engRules.privateNetworkRules {
		rule := calicores.GetPrivateNetworkSetV3Rule(direction, er.ports, &er.protocol, er.timestamp)
		prnRules = append(prnRules, *rule)
	}
	sort.Slice(prnRules, func(i, j int) bool {
		return lessPrivateNetwork(prnRules[i], prnRules[j])
	})
	rules = append(rules, prnRules...)

	// PublicNetwork
	pbnRules := []v3.Rule{}
	for _, er := range engRules.publicNetworkRules {
		rule := calicores.GetPublicV3Rule(er.ports, &er.protocol, er.timestamp)
		pbnRules = append(pbnRules, *rule)
	}
	sort.Slice(pbnRules, func(i, j int) bool {
		return lessPublicNetwork(pbnRules[i], pbnRules[j])
	})
	rules = append(rules, pbnRules...)

	return rules
}

// ProcessRecommendation updates a staged network policy's rules with the recommendation engine
// incoming rules.
// The staged network policy's (snp) egress/ingress v3 rules are assumed to be in order and are
// split into scopes. For each scope, the method merges the snp's with the incoming rules. Current
// rules are updated accordingly and incoming rules are added in order. A new slice is generated
// for the egress/ingress rules each time ProcessRecommendation is executed, with the assumption
// the snp's are compared by value with reflect.DeepEqual(), and a cache is considered altered
// only if the values of the rules have changed.
func (ere *recommendationEngine) processRecommendation(flows []*api.Flow, snp *v3.StagedNetworkPolicy) {
	if snp == nil {
		ere.clog.Warn("Empty staged network policy")
		return
	}
	ere.clog.Debugf("Processing recommendation: %s", snp.Name)

	// Process flows into egress/ingress rules, and the policy selector.
	for _, flow := range flows {
		ere.clog.WithField("flow", flow).Debug("Calling recommendation engine with flow")

		ere.processFlow(flow)
	}

	// Get sorted v3 rules
	egress := ere.getSortedEngineAsV3Rules(calicores.EgressTraffic)
	ingress := ere.getSortedEngineAsV3Rules(calicores.IngressTraffic)

	// If the egress or ingress private network contains rules, create the 'private-network' global()
	// network set if it doesn't already exist

	emptyRules := len(egress) == 0 && len(ingress) == 0
	if calicores.UpdateStagedNetworkPolicyRules(snp, egress, ingress) {
		snp.Annotations[calicores.LastUpdatedKey] = ere.clock.NowRFC3339()
	}
	updateStatusAnnotation(snp, emptyRules, ere.clock.NowRFC3339(), ere.interval, ere.stabilization)
}

// Check if the flow matches the destination namespace.
func (ere *recommendationEngine) matchesDestinationNamespace(flow api.Flow) bool {
	return flow.Reporter == api.ReporterTypeDestination &&
		flow.Destination.Namespace == ere.namespace &&
		flow.Destination.Type == api.FlowLogEndpointTypeWEP
}

// Check if the flow matches the source namespace.
func (ere *recommendationEngine) matchesSourceNamespace(flow api.Flow) bool {
	return flow.Reporter == api.ReporterTypeSource &&
		flow.Source.Namespace == ere.namespace &&
		flow.Source.Type == api.FlowLogEndpointTypeWEP
}

// processEngineRuleFromFlow converts a flow into an engine rule. In case of an unsupported flow type, the
// flow is not added to the engine rules, is logged as a warning, and the process continuous
// uninterrupted.
// Rules are added to the recommended policy by their scope and in the following
// order:
// 1. Egress to domains
// 2. Egress to domain sets
// 3. Egress to services
// 4. Namespaces
// 5. NetworkSets or GlobalNetworkSet
// 6. Private network
// 7. Public network
func (ere *recommendationEngine) processEngineRuleFromFlow(apiFlow api.Flow) {
	// Get the flow's type and direction.
	var flowType flowType
	var direction calicores.DirectionType
	if ere.matchesSourceNamespace(apiFlow) {
		if flowType = getFlowType(calicores.EgressTraffic, apiFlow); flowType == unsupportedFlowType {
			ere.clog.Debug("Unsupported flow type")
			return
		}
		direction = calicores.EgressTraffic
	} else if ere.matchesDestinationNamespace(apiFlow) {
		if flowType = getFlowType(calicores.IngressTraffic, apiFlow); flowType == unsupportedFlowType {
			ere.clog.Debug("Unsupported flow type")
			return
		}
		direction = calicores.IngressTraffic
	} else {
		ere.clog.Warnf("Staged network policy namespace does not match flow. Cannot process flow: %+v",
			apiFlow)
		return
	}

	// Add flow to Ingress or Egress rules
	var engRules *engineRules
	if direction == calicores.EgressTraffic {
		engRules = ere.egress
	} else {
		engRules = ere.ingress
	}

	// Add the flow to the existing set of engine rules, or log a warning if unsupported
	switch flowType {
	case egressToDomainFlowType:
		engRules.addFlowToEgressToDomainRules(direction, apiFlow, ere.clock)
	case egressToServiceFlowType:
		engRules.addFlowToEgressToServiceRules(direction, apiFlow, ere.clock)
	case namespaceFlowType:
		engRules.addFlowToNamespaceRules(direction, apiFlow, ere.clock)
	case networkSetFlowType:
		engRules.addFlowToNetworkSetRules(direction, apiFlow, ere.clock)
	case privateNetworkFlowType:
		engRules.addFlowToPrivateNetworkRules(direction, apiFlow, ere.clock)
	case publicNetworkFlowType:
		engRules.addFlowToPublicNetworkRules(direction, apiFlow, ere.clock)
	}
}

// Helpers

// compPorts compares two slices of ports. It returns 0 if the two lists contains the same elements.
// otherwise it returns -1 if slice 'a' contains an element that is lesser than the minPort, maxPort
// or portName than that of slice 'b' has at the same index.
func compPorts(a, b []numorstring.Port) int {
	for i := range a {
		if i >= len(b) {
			return 1
		}
		if a[i].MinPort != b[i].MinPort {
			if a[i].MinPort < b[i].MinPort {
				return -1
			}
			return 1
		}
		if a[i].MaxPort != b[i].MaxPort {
			if a[i].MaxPort < b[i].MaxPort {
				return -1
			}
			return 1
		}
		if a[i].PortName != b[i].PortName {
			if a[i].PortName < b[i].PortName {
				return 1
			}
			return -1
		}
	}

	// Length of port a is less than b, and all ports compared up to this point have been found to
	// be equal
	if len(a) != len(b) {
		return -1
	}

	return 0
}

// compStrArrays returns true if slice a contains an element that at the same index has a lesser
// alphabetical ordering than that of slice b.
func compStrArrays(a, b []string) bool {
	for i := range a {
		if i >= len(b) {
			return false
		}
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return len(a) < len(b)
}

func getEntityRule(dir calicores.DirectionType, rule *v3.Rule) *v3.EntityRule {
	var entity *v3.EntityRule
	if dir == calicores.EgressTraffic {
		entity = &rule.Destination
	} else {
		entity = &rule.Source
	}

	return entity
}

func getRuleMetadata(key string, rule v3.Rule) (string, bool) {
	val, ok := rule.Metadata.Annotations[key]
	if !ok {
		log.WithError(fmt.Errorf("rule metadata does not contain key: %s", key))
		return "", ok
	}

	return val, ok
}

// getNamespacePolicyRecParams returns the policy parameters of a namespaces based policy
// recommendation query to flow logs
func getNamespacePolicyRecParams(st time.Duration, ns, cl string) *flows.PolicyRecommendationParams {
	return &flows.PolicyRecommendationParams{
		StartTime:   st,
		EndTime:     time.Duration(0), // Now
		Namespace:   ns,
		Unprotected: true,
	}
}

// getRecommendationEngine returns a recommendation engine. Instantiated a new recommendation
// engine and uses any existing staged network policy rules for instantiation.
func getRecommendationEngine(
	snp v3.StagedNetworkPolicy, clock Clock, interval, stabilization time.Duration, clog log.Entry,
) recommendationEngine {
	eng := newRecommendationEngine(
		snp.Name, snp.Namespace, snp.Spec.Tier, snp.Spec.Order, clock, interval, stabilization, clog)
	eng.buildRules(calicores.EgressTraffic, snp.Spec.Egress)
	eng.buildRules(calicores.IngressTraffic, snp.Spec.Ingress)

	return *eng
}

// lessEgressToDomain compares two egress to domain rules by the alphabetical order of the protocol
// the order of ports, and the alphabetical order of the rule's domains. It is assumed that the
// rules will differ in at least one field.
func lessEgressToDomain(left, right v3.Rule) bool {
	if left.Protocol.StrVal != right.Protocol.StrVal {
		return left.Protocol.StrVal < right.Protocol.StrVal
	}
	cp := compPorts(left.Destination.Ports, right.Destination.Ports)
	if cp != 0 {
		return cp == -1
	}
	return compStrArrays(left.Destination.Domains, right.Destination.Domains)
}

// lessEgressToService compares two egress to service rules by the alphabetical order of the protocol
// the order of ports, and the alphabetical order of the destination name and namespace. It is
// assumed that the rules will differ in at least one field.
func lessEgressToService(left, right v3.Rule) bool {
	if left.Protocol.StrVal != right.Protocol.StrVal {
		return left.Protocol.StrVal < right.Protocol.StrVal
	}
	cp := compPorts(left.Destination.Ports, right.Destination.Ports)
	if cp != 0 {
		return cp == -1
	}
	if left.Destination.Services.Name != right.Destination.Services.Name {
		return left.Destination.Services.Name < right.Destination.Services.Name
	}
	return left.Destination.Services.Namespace < right.Destination.Services.Namespace
}

// lessNamespace compares two namespace rules by the alphabetical order of the protocol
// the order of ports, and the alphabetical order of the destination name and namespace. It is
// assumed that the rules will differ in at least one field.
func lessNamespace(direction calicores.DirectionType, left, right v3.Rule) bool {
	if left.Protocol.StrVal != right.Protocol.StrVal {
		return left.Protocol.StrVal < right.Protocol.StrVal
	}
	cp := compPorts(left.Destination.Ports, right.Destination.Ports)
	if cp != 0 {
		return cp == -1
	}
	if direction == calicores.EgressTraffic {
		return left.Destination.NamespaceSelector < right.Destination.NamespaceSelector
	} else {
		return left.Source.NamespaceSelector < right.Source.NamespaceSelector
	}
}

// lessNetworkSet compares two network set rules by the alphabetical order of the protocol
// the order of ports, and the alphabetical order of the destination name and namespace. It is
// assumed that the rules will differ in at least one field.
func lessNetworkSet(direction calicores.DirectionType, left, right v3.Rule) bool {
	if left.Protocol.StrVal != right.Protocol.StrVal {
		return left.Protocol.StrVal < right.Protocol.StrVal
	}
	cp := compPorts(left.Destination.Ports, right.Destination.Ports)
	if cp != 0 {
		return cp == -1
	}
	if direction == calicores.EgressTraffic {
		if left.Destination.NamespaceSelector != right.Destination.NamespaceSelector {
			return left.Destination.NamespaceSelector < right.Destination.NamespaceSelector
		}
		return left.Destination.Selector < right.Destination.Selector
	} else {
		if left.Source.NamespaceSelector != right.Source.NamespaceSelector {
			return left.Source.NamespaceSelector < right.Source.NamespaceSelector
		}
		return left.Source.Selector < right.Source.Selector
	}
}

// lessPrivateNetwork compares two namespace rules by the alphabetical order of the protocol. It is
// assumed that no two rules will not have the same protocol.
func lessPrivateNetwork(left, right v3.Rule) bool {
	return left.Protocol.StrVal < right.Protocol.StrVal
}

// lessPublicNetwork compares two namespace rules by the alphabetical order of the protocol. It is
// assumed that no two rules will not have the same protocol.
func lessPublicNetwork(left, right v3.Rule) bool {
	return left.Protocol.StrVal < right.Protocol.StrVal
}

// updateStatusAnnotation updates the learning annotation of a staged network policy given
// the time since the last update.
//
//   - Learning
//     Policy rule was updated <= 2 x recommendation interval ago
//   - Stale
//     Policy was updated > stabilization period ago, and the flows contain policy matches that do
//     not match the expected policy hits. This is usually the result of long-running connections
//     that were established before the recommended staged policy was created or modified.
//     Resolving this may require the connections to be restarted by cycling the impacted pods
//   - Stabilizing
//     Policy was updated > 2 x recommendation interval ago. The flows contain policy matches that
//     match the expected policy hits for the recommended policy, and may still contain some logs
//     that do not. The flows that do not match are fully covered by the existing rules in the
//     recommended policy (i.e. no further changes are required to the policy)
//   - Stable
//     Policy was updated > stabilization period ago. The flows all contain the expected
//     recommended policy hits
func updateStatusAnnotation(
	snp *v3.StagedNetworkPolicy,
	emptyRules bool,
	timeNowFRC3339 string,
	interval, stabilization time.Duration,
) {
	if emptyRules {
		// No update to status annotation necessary
		return
	}

	lastUpdateStr, ok := snp.Annotations[calicores.LastUpdatedKey]
	if !ok {
		// Fist time creating the last update key
		snp.Annotations[calicores.StatusKey] = calicores.LearningStatus
		return
	}
	snpLastUpdateTime, err := time.Parse(time.RFC3339, lastUpdateStr)
	if err != nil {
		log.WithError(err).Debugf("Failed to parse snp last update time using the RFC3339 format")
		return
	}
	nowTime, err := time.Parse(time.RFC3339, timeNowFRC3339)
	if err != nil {
		log.WithError(err).Debugf("Failed to parse the time now using the RFC3339 format")
		return
	}
	durationSinceLastUpdate := nowTime.Sub(snpLastUpdateTime)

	// Update status
	switch {
	case durationSinceLastUpdate <= 2*interval:
		snp.Annotations[calicores.StatusKey] = calicores.LearningStatus
	case durationSinceLastUpdate > 2*interval && durationSinceLastUpdate <= stabilization:
		snp.Annotations[calicores.StatusKey] = calicores.StabilizingStatus
	case durationSinceLastUpdate > stabilization:
		snp.Annotations[calicores.StatusKey] = calicores.StableStatus
	default:
		log.Warnf("Invalid status")
		snp.Annotations[calicores.StatusKey] = calicores.NoDataStatus
	}
}
