// Copyright (c) 2024 Tigera Inc. All rights reserved.
package engine

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/google/go-cmp/cmp"
	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/api/pkg/lib/numorstring"

	"github.com/projectcalico/calico/lma/pkg/api"
	calicores "github.com/projectcalico/calico/policy-recommendation/pkg/calico-resources"
	"github.com/projectcalico/calico/policy-recommendation/pkg/types"
	"github.com/projectcalico/calico/policy-recommendation/utils"
)

type recommendation struct {
	// Namespace of the recommended policy.
	namespace string

	// Egress recommendation incoming rules.
	egress *engineRules

	// Ingress recommendation incoming rules.
	ingress *engineRules

	// interval is the recommendation interval. This tracks the time to the next processing cycle. It
	// is also used as a metric to define the look-back period for the flow logs queries.
	interval time.Duration

	// stabilization period is the time period after which the recommendation becomes stable.
	stabilization time.Duration

	// passIntraNamespaceTraffic passes intra-namespace traffic to the next policy.
	passIntraNamespaceTraffic bool

	// serviceNameSuffix is the server name suffix of the local domain.
	serviceNameSuffix string

	// Clock used for formatting and testing purposes.
	clock Clock

	// log entry
	clog *log.Entry
}

func newRecommendation(
	clusterID string,
	namespace string,
	interval time.Duration,
	stabilization time.Duration,
	passIntraNamespaceTraffic bool,
	serviceNameSuffix string,
	snp *v3.StagedNetworkPolicy,
	clock Clock,
) *recommendation {
	logEntry := log.WithField("cluster", clusterID)
	if clusterID == "cluster" {
		logEntry = log.WithField("cluster", "management")
	}

	rec := &recommendation{
		namespace:                 namespace,
		clock:                     clock,
		interval:                  interval,
		stabilization:             stabilization,
		passIntraNamespaceTraffic: passIntraNamespaceTraffic,
		serviceNameSuffix:         serviceNameSuffix,
		clog:                      logEntry,
	}
	// Sets the egress recommendation rules from the staged network policy.
	rec.buildRules(calicores.EgressTraffic, snp.Spec.Egress)
	// Sets the ingress recommendation rules from the staged network policy.
	rec.buildRules(calicores.IngressTraffic, snp.Spec.Ingress)

	return rec
}

// buildEgressToDomain creates a new EgressToDomain recommendation rule key. It is assumed that each new
// rule will generate a new key.
func (rec *recommendation) buildEgressToDomain(rule v3.Rule) {
	if len(rule.Destination.Ports) == 0 {
		err := fmt.Errorf("no ports in this rule")
		rec.clog.WithError(err)
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
	rec.egress.egressToDomainRules[key] = val
	rec.egress.size++
}

// buildEgressToService creates a new EgressToService recommendation rule key. It is assumed that each new
// rule will generate a new key.
func (rec *recommendation) buildEgressToService(rule v3.Rule) {
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
	rec.egress.egressToServiceRules[key] = val
	rec.egress.size++
}

// buildNamespace creates a new Namespace recommendation rule key. It is assumed that each new
// rule will generate a new key.
func (rec *recommendation) buildNamespace(dir calicores.DirectionType, rule v3.Rule) {
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
		erules = rec.egress
	} else {
		erules = rec.ingress
	}
	erules.namespaceRules[key] = val
	erules.size++
}

// buildNetworkSet creates a new NetworkSet recommendation rule key. It is assumed that each new rule will
// generate a new key.
func (rec *recommendation) buildNetworkSet(dir calicores.DirectionType, rule v3.Rule) {
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
		erules = rec.egress
	} else {
		erules = rec.ingress
	}
	erules.networkSetRules[key] = val
	erules.size++
}

// buildPrivate creates a new PrivateNetwork recommendation rule key. It is assumed that each new
// rule will generate a new key.
func (rec *recommendation) buildPrivate(dir calicores.DirectionType, rule v3.Rule) {
	if len(rule.Destination.Ports) == 0 {
		err := errors.New("no ports in private rule")
		rec.clog.WithError(err)
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
		erules = rec.egress
	} else {
		erules = rec.ingress
	}
	erules.privateNetworkRules[key] = val
	erules.size++
}

// buildPublic creates a new PublicNetwork recommendation rule key. It is assumed that each new
// rule will generate a new key.
func (rec *recommendation) buildPublic(dir calicores.DirectionType, rule v3.Rule) {
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
		engRules = rec.egress
	} else {
		engRules = rec.ingress
	}
	engRules.publicNetworkRules[key] = val

	engRules.size++
}

// buildRules builds the recommendation rules from a list of v3 rules.
func (rec *recommendation) buildRules(dir calicores.DirectionType, rules []v3.Rule) {
	if rec.egress == nil {
		rec.egress = NewEngineRules()
	}
	if rec.ingress == nil {
		rec.ingress = NewEngineRules()
	}

	var scope string
	var ok bool
	for _, rule := range rules {
		if rule.Metadata == nil {
			rec.clog.Warn("recommended rule metadata is empty")
			continue
		}
		scope, ok = rule.Metadata.Annotations[calicores.ScopeKey]
		if !ok {
			rec.clog.Warn("recommended rule does not contain a scope")
			continue
		}
		switch scope {
		case string(calicores.EgressToDomainScope):
			rec.buildEgressToDomain(rule)
		case string(calicores.EgressToDomainSetScope):
			// TODO(dimitrin): Create buildEgressToDomainSet
		case string(calicores.EgressToServiceScope):
			rec.buildEgressToService(rule)
		case string(calicores.NamespaceScope):
			rec.buildNamespace(dir, rule)
		case string(calicores.NetworkSetScope):
			rec.buildNetworkSet(dir, rule)
		case string(calicores.PrivateNetworkScope):
			rec.buildPrivate(dir, rule)
		case string(calicores.PublicNetworkScope):
			rec.buildPublic(dir, rule)
		default:
			rec.clog.Warnf("Invalid scope: %s", scope)
		}
	}
}

// getIncomingV3Rules returns the recommendation rules as a sorted list v3 rules.
func (rec *recommendation) getIncomingV3Rules(direction calicores.DirectionType) []v3.Rule {
	var engRules *engineRules
	if direction == calicores.EgressTraffic {
		engRules = rec.egress
	} else {
		engRules = rec.ingress
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

	if len(rules) > 0 {
		return rules
	}
	return nil
}

// processEngineRuleFromFlow converts a flow into an recommendation rule. In case of an unsupported flow type, the
// flow is not added to the recommendation rules, is logged as a warning, and the process continuous
// uninterrupted.
// Rules are added to the recommended policy by their scope and in the following
// order:
// 1. Egress to domains
// 2. Egress to services
// 3. Namespaces
// 4. NetworkSets or GlobalNetworkSet
// 5. Private network
// 6. Public network
func (rec *recommendation) processEngineRuleFromFlow(apiFlow api.Flow) {
	// Get the flow's type and direction.
	var flowType flowType
	var direction calicores.DirectionType
	if rec.matchesSourceNamespace(apiFlow) {
		if flowType = getFlowType(calicores.EgressTraffic, apiFlow, rec.serviceNameSuffix); flowType == unsupportedFlowType {
			rec.clog.Debug("Unsupported flow type")
			return
		}
		direction = calicores.EgressTraffic
	} else if rec.matchesDestinationNamespace(apiFlow) {
		if flowType = getFlowType(calicores.IngressTraffic, apiFlow, rec.serviceNameSuffix); flowType == unsupportedFlowType {
			rec.clog.Debug("Unsupported flow type")
			return
		}
		direction = calicores.IngressTraffic
	} else {
		rec.clog.Warnf("Staged network policy namespace does not match flow. Cannot process flow: %+v",
			apiFlow)
		return
	}

	// Add flow to Ingress or Egress rules
	var engRules *engineRules
	if direction == calicores.EgressTraffic {
		engRules = rec.egress
	} else {
		engRules = rec.ingress
	}

	// Add the flow to the existing set of recommendation rules, or log a warning if unsupported
	switch flowType {
	case egressToDomainFlowType:
		engRules.addFlowToEgressToDomainRules(direction, apiFlow, rec.clock, rec.serviceNameSuffix)
	case egressToServiceFlowType:
		engRules.addFlowToEgressToServiceRules(direction, apiFlow, rec.passIntraNamespaceTraffic, rec.clock)
	case namespaceFlowType:
		engRules.addFlowToNamespaceRules(direction, apiFlow, rec.passIntraNamespaceTraffic, rec.clock)
	case networkSetFlowType:
		engRules.addFlowToNetworkSetRules(direction, apiFlow, rec.passIntraNamespaceTraffic, rec.clock)
	case privateNetworkFlowType:
		engRules.addFlowToPrivateNetworkRules(direction, apiFlow, rec.clock)
	case publicNetworkFlowType:
		engRules.addFlowToPublicNetworkRules(direction, apiFlow, rec.clock)
	}
}

// ProcessFlow adds the flow the recommendation recommendation rules. Actions other than allow, namespace
// mismatches and non WEP destination flows are skipped.
func (rec *recommendation) processFlow(flow *api.Flow) {
	if flow == nil {
		return
	}
	rec.clog.WithField("flow", flow).Debug("Processing")

	// Only allowed flows are used to recommend policy
	if flow.ActionFlag&api.ActionFlagAllow == 0 {
		rec.clog.Debug("Skipping flow, only allow action processed")
		return
	}
	// Make sure we only process flows that have either source or destination in the expected
	// namespace
	if !rec.matchesSourceNamespace(*flow) && !rec.matchesDestinationNamespace(*flow) {
		// Skip this flow, as it doesn't match the namespace, or is not WEP
		rec.clog.Debug("Skipping flow, namespace mismatch or destination isn't WEP")
		return
	}
	// Construct rule
	rec.processEngineRuleFromFlow(*flow)
}

// matchesDestinationNamespace returns true if the flow logs is reported by the destination
// endpoint, is WEP and the namespace is equal to reference namespace.
func (rec *recommendation) matchesDestinationNamespace(flow api.Flow) bool {
	return flow.Reporter == api.ReporterTypeDestination &&
		flow.Destination.Namespace == rec.namespace &&
		flow.Destination.Type == api.FlowLogEndpointTypeWEP
}

// matchesSourceNamespace returns true if the flow logs is reported by the source
// endpoint, is WEP and the namespace is equal to reference namespace.
func (rec *recommendation) matchesSourceNamespace(flow api.Flow) bool {
	return flow.Reporter == api.ReporterTypeSource &&
		flow.Source.Namespace == rec.namespace &&
		flow.Source.Type == api.FlowLogEndpointTypeWEP
}

// update generates v3 rules from incoming flows and ingests them into a staged
// network policy.
func (rec *recommendation) update(flows []*api.Flow, snp *v3.StagedNetworkPolicy) bool {
	if snp == nil {
		rec.clog.Warn("Empty staged network policy")
		return false
	}

	// Process flows into egress/ingress engine rules.
	for _, flow := range flows {
		rec.processFlow(flow)
	}

	// Return true if the recommendation (SNP) is at all altered. The recommendation is altered if
	// the rules are updated or the status is updated. Otherwise, return false.
	return rec.updateRules(snp) || rec.updateStatus(snp)
}

// updateRules returns true if an update to the staged network policy rules occurred.
// Replaces the egress/ingress rules in their entirety if the incoming rules differ from the
// existing ones, as they are a super-set of the existing rules.
func (rec *recommendation) updateRules(snp *v3.StagedNetworkPolicy) bool {
	updated := false
	ingress := rec.getIncomingV3Rules(calicores.IngressTraffic)
	if !reflect.DeepEqual(ingress, snp.Spec.Ingress) {
		log.Debugf("Diff ingress: %s", cmp.Diff(ingress, snp.Spec.Ingress))
		snp.Spec.Ingress = ingress
		updated = true
	}
	egress := rec.getIncomingV3Rules(calicores.EgressTraffic)
	if !reflect.DeepEqual(egress, snp.Spec.Egress) {
		log.Debugf("Diff egress: %s", cmp.Diff(egress, snp.Spec.Egress))
		snp.Spec.Egress = egress
		updated = true
	}
	snp.Spec.Types = getPolicyTypes(snp.Spec.Egress, snp.Spec.Ingress)

	if updated {
		// Reset to learning status, and update the last updated timestamp.
		snp.Annotations[calicores.StatusKey] = calicores.LearningStatus
		snp.Annotations[calicores.LastUpdatedKey] = rec.clock.NowRFC3339()
	}

	return updated
}

// updateStatus updates the recommendation status to 'Stabilizing' or 'Stable' if the last update.
// Returns false, if no update occurred.
func (rec *recommendation) updateStatus(snp *v3.StagedNetworkPolicy) bool {
	lastUpdateStr := snp.Annotations[calicores.LastUpdatedKey]
	if lastUpdateStr == "" {
		// New recommendation, with no rules. It will not be added to the cache for syncing.
		// No need to update status.
		return false
	}

	// Update recommendation status (from 'Learning' to 'Stabilizing' or 'Stabilizing' to 'Stable').
	// A recommendation is stabilizing when the last update is older than twice the interval, and
	// stable when the last update is older than the stabilization period.
	lastUpdate, err := time.Parse(time.RFC3339, lastUpdateStr)
	if err != nil {
		log.WithError(err).Errorf("failed to parse Unix timestamp %s", lastUpdateStr)
		return false
	}
	diff := time.Since(lastUpdate)
	if snp.Annotations[calicores.StatusKey] != calicores.StabilizingStatus && diff > 2*rec.interval && diff <= rec.stabilization {
		log.WithField("name", snp.Name).Infof("Recommendation is stabilizing")
		// Update the status to stabilizing and replace the name suffix.
		snp.Annotations[calicores.StatusKey] = calicores.StabilizingStatus
		snp.Name = utils.GenerateRecommendationName(types.PolicyRecommendationTierName, snp.Namespace, utils.SuffixGenerator)

		return true
	} else if snp.Annotations[calicores.StatusKey] != calicores.StableStatus && diff > rec.stabilization {
		log.WithField("name", snp.Name).Infof("Recommendation is stable")
		// Update the status to stable and replace the name suffix.
		snp.Annotations[calicores.StatusKey] = calicores.StableStatus
		snp.Name = utils.GenerateRecommendationName(types.PolicyRecommendationTierName, snp.Namespace, utils.SuffixGenerator)

		return true
	}

	return false
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

// getPolicyTypes returns the policy rule types (Egress/Ingress). An empty list is returned if the
// rules are empty.
func getPolicyTypes(egress, ingress []v3.Rule) []v3.PolicyType {
	pt := []v3.PolicyType{}
	if len(egress) > 0 {
		pt = append(pt, v3.PolicyTypeEgress)
	}
	if len(ingress) > 0 {
		pt = append(pt, v3.PolicyTypeIngress)
	}

	return pt
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
