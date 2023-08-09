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
	"github.com/projectcalico/calico/policy-recommendation/pkg/types"
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

	// passIntraNamespaceTraffic passes intra-namespace traffic to the next policy.
	passIntraNamespaceTraffic bool

	// serviceNameSuffix is the server name suffix of the local domain.
	serviceNameSuffix string

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
	serviceNameSuffix string,
	clock Clock,
	recInterval time.Duration,
	stabilizationPeriod time.Duration,
	owner *metav1.OwnerReference,
	passIntraNamespaceTraffic bool,
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

	// Define flow log query params
	params := getNamespacePolicyRecParams(lookback, snp.Namespace, clusterID)

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
	engine := getRecommendationEngine(*snp, clock, recInterval, stabilizationPeriod, passIntraNamespaceTraffic, serviceNameSuffix, *clog)
	engine.processRecommendation(flows, snp)
}

// Recommendation Engine

// newRecommendationEngine returns a new recommendation engine.
func newRecommendationEngine(
	name, namespace, tier string,
	order *float64,
	clock Clock,
	interval, stabilization time.Duration,
	passIntraNamespaceTraffic bool,
	serviceNameSuffix string,
	clog log.Entry,
) *recommendationEngine {
	return &recommendationEngine{
		name:                      name,
		namespace:                 namespace,
		tier:                      tier,
		order:                     order,
		egress:                    NewEngineRules(),
		ingress:                   NewEngineRules(),
		nets:                      set.New[NetworkSet](),
		clock:                     clock,
		interval:                  interval,
		passIntraNamespaceTraffic: passIntraNamespaceTraffic,
		stabilization:             stabilization,
		serviceNameSuffix:         serviceNameSuffix,
		clog:                      clog,
	}
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
	key := engineRuleKey{
		port:     port,
		protocol: *rule.Protocol,
	}
	val := &types.FlowLogData{
		Action:    rule.Action,
		Domains:   rule.Destination.Domains,
		Ports:     []numorstring.Port{port},
		Protocol:  *rule.Protocol,
		Timestamp: rule.Metadata.Annotations[calicores.LastUpdatedKey],
	}
	ere.egress.egressToDomainRules[key] = val
	ere.egress.size++
}

// buildEgressToService creates a new EgressToService engine rule key. It is assumed that each new
// rule will generate a new key.
func (ere *recommendationEngine) buildEgressToService(rule v3.Rule) {
	key := engineRuleKey{
		name:      rule.Destination.Services.Name,
		namespace: rule.Destination.Services.Namespace,
		protocol:  *rule.Protocol,
	}
	val := &types.FlowLogData{
		Action:    rule.Action,
		Name:      rule.Destination.Services.Name,
		Namespace: rule.Destination.Services.Namespace,
		Ports:     rule.Destination.Ports,
		Protocol:  *rule.Protocol,
		Timestamp: rule.Metadata.Annotations[calicores.LastUpdatedKey],
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

	key := engineRuleKey{
		namespace: ns,
		protocol:  *rule.Protocol,
	}
	val := &types.FlowLogData{
		Action:    rule.Action,
		Namespace: ns,
		Ports:     rule.Destination.Ports, // Always destination
		Protocol:  *rule.Protocol,
		Timestamp: ts,
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

	key := engineRuleKey{
		global:    gl,
		name:      name,
		namespace: ns,
		protocol:  *rule.Protocol,
	}
	val := &types.FlowLogData{
		Action:    rule.Action,
		Global:    gl,
		Name:      name,
		Namespace: ns,
		Ports:     rule.Destination.Ports, // Always destination
		Protocol:  *rule.Protocol,
		Timestamp: ts,
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

	ts, ok := getRuleMetadata(calicores.LastUpdatedKey, rule)
	if !ok {
		return
	}

	key := engineRuleKey{
		protocol: *rule.Protocol,
	}
	val := &types.FlowLogData{
		Action:    rule.Action,
		Protocol:  *rule.Protocol,
		Ports:     rule.Destination.Ports, // Always destination
		Timestamp: ts,
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

	key := engineRuleKey{
		protocol: *rule.Protocol,
	}
	val := &types.FlowLogData{
		Action:    rule.Action,
		Ports:     rule.Destination.Ports,
		Protocol:  *rule.Protocol,
		Timestamp: ts,
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

// buildRules builds the engine rules from a list of v3 rules.
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

// getV3Rules returns the engine rules as a sorted list v3 rules.
func (ere *recommendationEngine) getV3Rules(direction calicores.DirectionType) []v3.Rule {
	var engRules *engineRules
	if direction == calicores.EgressTraffic {
		engRules = ere.egress
	} else {
		engRules = ere.ingress
	}

	rules := []v3.Rule{}

	egressToDomainRules := processRules(direction, engRules.egressToDomainRules, calicores.GetEgressToDomainV3Rule, compEgressToDomain)
	rules = append(rules, egressToDomainRules...)

	egressToServiceRules := processRules(direction, engRules.egressToServiceRules, calicores.GetEgressToServiceV3Rule, compEgressToService)
	rules = append(rules, egressToServiceRules...)

	namespaceRules := processRules(direction, engRules.namespaceRules, calicores.GetNamespaceV3Rule, compNamespace)
	rules = append(rules, namespaceRules...)

	networkSetRules := processRules(direction, engRules.networkSetRules, calicores.GetNetworkSetV3Rule, compNetworkSet)
	rules = append(rules, networkSetRules...)

	privateNetworkRules := processRules(direction, engRules.privateNetworkRules, calicores.GetPrivateNetworkV3Rule, compPrivateNetwork)
	rules = append(rules, privateNetworkRules...)

	publicNetworkRules := processRules(direction, engRules.publicNetworkRules, calicores.GetPublicNetworkV3Rule, compPublicNetwork)
	rules = append(rules, publicNetworkRules...)

	return rules
}

// processEngineRuleFromFlow converts a flow into an engine rule. In case of an unsupported flow type, the
// flow is not added to the engine rules, is logged as a warning, and the process continuous
// uninterrupted.
// Rules are added to the recommended policy by their scope and in the following
// order:
// 1. Egress to domains
// 2. Egress to services
// 3. Namespaces
// 4. NetworkSets or GlobalNetworkSet
// 5. Private network
// 6. Public network
func (ere *recommendationEngine) processEngineRuleFromFlow(apiFlow api.Flow) {
	// Get the flow's type and direction.
	var flowType flowType
	var direction calicores.DirectionType
	if ere.matchesSourceNamespace(apiFlow) {
		if flowType = getFlowType(calicores.EgressTraffic, apiFlow, ere.serviceNameSuffix); flowType == unsupportedFlowType {
			ere.clog.Debug("Unsupported flow type")
			return
		}
		direction = calicores.EgressTraffic
	} else if ere.matchesDestinationNamespace(apiFlow) {
		if flowType = getFlowType(calicores.IngressTraffic, apiFlow, ere.serviceNameSuffix); flowType == unsupportedFlowType {
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
		engRules.addFlowToEgressToDomainRules(direction, apiFlow, ere.clock, ere.serviceNameSuffix)
	case egressToServiceFlowType:
		engRules.addFlowToEgressToServiceRules(direction, apiFlow, ere.passIntraNamespaceTraffic, ere.clock)
	case namespaceFlowType:
		engRules.addFlowToNamespaceRules(direction, apiFlow, ere.passIntraNamespaceTraffic, ere.clock)
	case networkSetFlowType:
		engRules.addFlowToNetworkSetRules(direction, apiFlow, ere.passIntraNamespaceTraffic, ere.clock)
	case privateNetworkFlowType:
		engRules.addFlowToPrivateNetworkRules(direction, apiFlow, ere.clock)
	case publicNetworkFlowType:
		engRules.addFlowToPublicNetworkRules(direction, apiFlow, ere.clock)
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

// matchesDestinationNamespace returns true if the flow logs is reported by the destination
// endpoint, is WEP and the namespace is equal to reference namespace.
func (ere *recommendationEngine) matchesDestinationNamespace(flow api.Flow) bool {
	return flow.Reporter == api.ReporterTypeDestination &&
		flow.Destination.Namespace == ere.namespace &&
		flow.Destination.Type == api.FlowLogEndpointTypeWEP
}

// matchesSourceNamespace returns true if the flow logs is reported by the source
// endpoint, is WEP and the namespace is equal to reference namespace.
func (ere *recommendationEngine) matchesSourceNamespace(flow api.Flow) bool {
	return flow.Reporter == api.ReporterTypeSource &&
		flow.Source.Namespace == ere.namespace &&
		flow.Source.Type == api.FlowLogEndpointTypeWEP
}

// ProcessRecommendation generates v3 rules from incoming flows and ingests them into a staged
// network policy.
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
	egress := ere.getV3Rules(calicores.EgressTraffic)
	ingress := ere.getV3Rules(calicores.IngressTraffic)

	// Update the last updated annotation of the policy if there was a change to the SNPs rules
	if calicores.SetSnpRules(snp, egress, ingress) {
		snp.Annotations[calicores.LastUpdatedKey] = ere.clock.NowRFC3339()
	}
}

// Helpers

// lessPorts compares two slices of ports. It returns 0 if the two lists contains the same elements.
// otherwise it returns -1 if slice 'a' contains an element that is lesser than the minPort, maxPort
// or portName than that of slice 'b' has at the same index.
func lessPorts(a, b []numorstring.Port) int {
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

// lessStringArrays returns true if slice a contains an element that at the same index has a lesser
// alphabetical ordering than that of slice b.
func lessStringArrays(a, b []string) bool {
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

// compEgressToDomain compares two egress to domain rules by the alphabetical order of the protocol
// the order of ports, and the alphabetical order of the rule's domains. It is assumed that the
// rules will differ in at least one field.
func compEgressToDomain(direction calicores.DirectionType, left, right v3.Rule) bool {
	if left.Protocol.StrVal != right.Protocol.StrVal {
		return left.Protocol.StrVal < right.Protocol.StrVal
	}
	cp := lessPorts(left.Destination.Ports, right.Destination.Ports)
	if cp != 0 {
		return cp == -1
	}
	return lessStringArrays(left.Destination.Domains, right.Destination.Domains)
}

// compEgressToService compares two egress to service rules by the alphabetical order of the protocol
// the order of ports, and the alphabetical order of the destination name and namespace. It is
// assumed that the rules will differ in at least one field.
func compEgressToService(direction calicores.DirectionType, left, right v3.Rule) bool {
	if left.Protocol.StrVal != right.Protocol.StrVal {
		return left.Protocol.StrVal < right.Protocol.StrVal
	}
	cp := lessPorts(left.Destination.Ports, right.Destination.Ports)
	if cp != 0 {
		return cp == -1
	}
	if left.Destination.Services.Name != right.Destination.Services.Name {
		return left.Destination.Services.Name < right.Destination.Services.Name
	}
	return left.Destination.Services.Namespace < right.Destination.Services.Namespace
}

// compNamespace compares two namespace rules by the alphabetical order of the protocol
// the order of ports, and the alphabetical order of the destination name and namespace. It is
// assumed that the rules will differ in at least one field.
func compNamespace(direction calicores.DirectionType, left, right v3.Rule) bool {
	if left.Protocol.StrVal != right.Protocol.StrVal {
		return left.Protocol.StrVal < right.Protocol.StrVal
	}
	cp := lessPorts(left.Destination.Ports, right.Destination.Ports)
	if cp != 0 {
		return cp == -1
	}
	if direction == calicores.EgressTraffic {
		return left.Destination.NamespaceSelector < right.Destination.NamespaceSelector
	} else {
		return left.Source.NamespaceSelector < right.Source.NamespaceSelector
	}
}

// compNetworkSet compares two network set rules by the alphabetical order of the protocol
// the order of ports, and the alphabetical order of the destination name and namespace. It is
// assumed that the rules will differ in at least one field.
func compNetworkSet(direction calicores.DirectionType, left, right v3.Rule) bool {
	if left.Protocol.StrVal != right.Protocol.StrVal {
		return left.Protocol.StrVal < right.Protocol.StrVal
	}
	cp := lessPorts(left.Destination.Ports, right.Destination.Ports)
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

// compPrivateNetwork compares two namespace rules by the alphabetical order of the protocol. It is
// assumed that no two rules will not have the same protocol.
func compPrivateNetwork(direction calicores.DirectionType, left, right v3.Rule) bool {
	return left.Protocol.StrVal < right.Protocol.StrVal
}

// compPublicNetwork compares two namespace rules by the alphabetical order of the protocol. It is
// assumed that no two rules will not have the same protocol.
func compPublicNetwork(direction calicores.DirectionType, left, right v3.Rule) bool {
	return left.Protocol.StrVal < right.Protocol.StrVal
}

// getEntityRule returns the destination v3.EntityRule for egress and the source for ingress
// traffic.
func getEntityRule(dir calicores.DirectionType, rule *v3.Rule) *v3.EntityRule {
	var entity *v3.EntityRule
	if dir == calicores.EgressTraffic {
		entity = &rule.Destination
	} else {
		entity = &rule.Source
	}

	return entity
}

// getRuleMetadata returns the v3.Rule's annotation, given its key.
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
	snp v3.StagedNetworkPolicy,
	clock Clock,
	interval, stabilization time.Duration,
	passIntraNamespaceTraffic bool,
	serviceNameSuffix string,
	clog log.Entry,
) recommendationEngine {
	eng := newRecommendationEngine(
		snp.Name,
		snp.Namespace,
		snp.Spec.Tier,
		snp.Spec.Order,
		clock,
		interval,
		stabilization,
		passIntraNamespaceTraffic,
		serviceNameSuffix,
		clog,
	)
	eng.buildRules(calicores.EgressTraffic, snp.Spec.Egress)
	eng.buildRules(calicores.IngressTraffic, snp.Spec.Ingress)

	return *eng
}

// processRules returns engine rules as a sorted list of v3 rules.
func processRules(
	direction calicores.DirectionType,
	engineRules map[engineRuleKey]*types.FlowLogData,
	v3RuleCreator func(types.FlowLogData, calicores.DirectionType) *v3.Rule,
	comparator func(direction calicores.DirectionType, i, j v3.Rule) bool,
) []v3.Rule {
	rules := []v3.Rule{}
	for _, data := range engineRules {
		rule := v3RuleCreator(*data, direction)
		rules = append(rules, *rule)
	}
	sort.Slice(rules, func(i, j int) bool {
		return comparator(direction, rules[i], rules[j])
	})

	return rules
}
