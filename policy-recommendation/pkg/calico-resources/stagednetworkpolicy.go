// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package calicoresources

import (
	"fmt"
	"reflect"
	"sort"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/api/pkg/lib/numorstring"
)

type DirectionType string
type ScopeType string

const (
	// Default SNP Spec values
	StagedNetworkPoliciesDefaultOrder = float64(100)
	// The possible directions a flow can take.
	EgressTraffic  DirectionType = "egress"
	IngressTraffic DirectionType = "ingress"

	// Scope annotations the v3 rules can be tagged with.
	EgressToDomainScope    ScopeType = "Domains"
	EgressToDomainSetScope ScopeType = "DomainSet"
	EgressToServiceScope   ScopeType = "Service"
	NamespaceScope         ScopeType = "Namespace"
	NetworkSetScope        ScopeType = "NetworkSet"
	PrivateNetworkScope    ScopeType = "Private"
	PublicNetworkScope     ScopeType = "Public"

	projectCalicoKeyName = "projectcalico.org"
	PolicyRecKeyName     = "policyrecommendation.tigera.io"

	nonServiceTypeWarning          = "NonServicePortsAndProtocol"
	policyRecommendationTimeFormat = time.RFC3339
	namespaceScope                 = "namespace"

	LastUpdatedKey  = PolicyRecKeyName + "/lastUpdated"
	NameKey         = PolicyRecKeyName + "/name"
	NamespaceKey    = PolicyRecKeyName + "/namespace"
	ScopeKey        = PolicyRecKeyName + "/scope"
	StagedActionKey = projectCalicoKeyName + "/spec.stagedAction"
	StatusKey       = PolicyRecKeyName + "/status"
	TierKey         = projectCalicoKeyName + "/tier"

	LearningStatus    = "Learning"
	NoDataStatus      = "NoData"
	StableStatus      = "Stable"
	StabilizingStatus = "Stabilizing"
	StaleStatus       = "Stale"
)

var (
	// Private RFC 1918 blocks
	// Note: Make sure this list reflects the equivalent list in felix/collector/flowlog_util.go
	privateNetwork24BitBlock = "10.0.0.0/8"
	privateNetwork20BitBlock = "172.16.0.0/12"
	privateNetwork16BitBlock = "192.168.0.0/16"
)

// NewStagedNetworkPolicy returns a pointer to a staged network policy.
func NewStagedNetworkPolicy(name, namespace, tier string, owner metav1.OwnerReference) *v3.StagedNetworkPolicy {
	return &v3.StagedNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				StatusKey: NoDataStatus,
			},
			// TODO(dimitri): Must define a valid RFC1123 policy name.
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"projectcalico.org/tier":                tier,
				"projectcalico.org/ownerReference.kind": owner.Kind,
				"policyrecommendation.tigera.io/scope":  namespaceScope,
				"projectcalico.org/spec.stagedAction":   "Learn",
			},
			OwnerReferences: []metav1.OwnerReference{owner},
		},

		Spec: v3.StagedNetworkPolicySpec{
			StagedAction: v3.StagedActionLearn,
			Tier:         tier,
			Selector:     fmt.Sprintf("%s/namespace == '%s'", projectCalicoKeyName, namespace),
			Egress:       []v3.Rule{},
			Ingress:      []v3.Rule{},
			Types:        []v3.PolicyType{},
		},
	}
}

// UpdateStagedNetworkPolicyRules updates the egress, ingress rules of a staged network policy.
func UpdateStagedNetworkPolicyRules(snp *v3.StagedNetworkPolicy, egress, ingress []v3.Rule) bool {
	updated := false

	types := getPolicyTypes(egress, ingress)
	if !reflect.DeepEqual(types, snp.Spec.Types) {
		snp.Spec.Types = types
		updated = true
	}
	if !reflect.DeepEqual(egress, snp.Spec.Egress) {
		snp.Spec.Egress = egress
		updated = true
	}
	if !reflect.DeepEqual(ingress, snp.Spec.Ingress) {
		snp.Spec.Ingress = ingress
		updated = true
	}

	return updated
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

// GetEgressToDomainV3Rule returns the egress traffic to domain rule. The destination entity rule
// ports are in sorted order and the domains are in alphabetical order.
//
// Metadata.Annotations:
//
//	policyrecommendation.tigera.io/lastUpdated	=	<RFC3339 formatted timestamp>
//	policyrecommendation.tigera.io/scope 				= 'Domains'
//
// EntityRule.Ports:
//
//	set of ports from flows (always destination rule)
//
// EntityRule.Domains:
//
//	set of domains from flows
func GetEgressToDomainV3Rule(
	domains []string, port numorstring.Port, protocol *numorstring.Protocol, rfc3339Time string,
) *v3.Rule {
	rule := &v3.Rule{
		Metadata: &v3.RuleMetadata{
			Annotations: map[string]string{
				fmt.Sprintf("%s/lastUpdated", PolicyRecKeyName): rfc3339Time,
				fmt.Sprintf("%s/scope", PolicyRecKeyName):       string(EgressToDomainScope),
			},
		},
		Action:   v3.Allow,
		Protocol: protocol,
	}

	if protocol.SupportsPorts() {
		rule.Destination.Ports = []numorstring.Port{port}
	}

	// Domains are stored in alphabetical order
	rule.Destination.Domains = sortDomains(domains)

	return rule
}

// GetEgressToDomainSetV3Rule returns an egress traffic to domain set rule. The destination entity
// rule ports are in sorted order.
//
// Metadata.Annotations
//
//	policyrecommendation.tigera.io/lastUpdated	= <RFC3339 formatted timestamp>
//	policyrecommendation.tigera.io/name 				= ‘<namespace>-egress-domains’
//	policyrecommendation.tigera.io/namespace 		= <namespace>
//	policyrecommendation.tigera.io/scope 				= 'DomainsSet'
//
// EntityRule.Ports:
//
//	set of ports from flows (always destination rule)
//
// EntityRule.Selector
//
//	policyrecommendation.tigera.io/scope == 'Domains'
func GetEgressToDomainSetV3Rule(
	namespace string, ports []numorstring.Port, protocol *numorstring.Protocol, rfc3339Time string,
) *v3.Rule {
	rule := &v3.Rule{
		Metadata: &v3.RuleMetadata{
			Annotations: map[string]string{
				fmt.Sprintf("%s/lastUpdated", PolicyRecKeyName): rfc3339Time,
				fmt.Sprintf("%s/name", PolicyRecKeyName):        fmt.Sprintf("%s-egress-domains", namespace),
				fmt.Sprintf("%s/namespace", PolicyRecKeyName):   namespace,
				fmt.Sprintf("%s/scope", PolicyRecKeyName):       string(EgressToDomainSetScope),
			},
		},
		Action:   v3.Allow,
		Protocol: protocol,
	}

	if protocol.SupportsPorts() {
		rule.Destination.Ports = sortPorts(ports)
	}
	rule.Destination.Selector = fmt.Sprintf("%s/scope == '%s'", PolicyRecKeyName, string(EgressToDomainScope))

	return rule
}

// GetEgressToServiceV3Rule returns the egress traffic to service rule. The destination entity
// rule ports are in sorted order.
//
// Metadata.Annotations:
//
//	policyrecommendation.tigera.io/lastUpdated 	= <RFC3339 formatted timestamp>
//	policyrecommendation.tigera.io/name 				= '<service_name>'
//	policyrecommendation.tigera.io/namespace 		= '<namespace>'
//	policyrecommendation.tigera.io/scope 				= 'Service'
//
// EntityRule.Ports:
//
//	set of ports from flows (always destination rule)
//
// EntityRule.Name:
//
//	<service_name>
//
// EntityRule.Namespace:
//
//	<service_namespace>
func GetEgressToServiceV3Rule(
	name, namespace string, ports []numorstring.Port, protocol *numorstring.Protocol, rfc3339Time string,
) *v3.Rule {
	rule := &v3.Rule{
		Metadata: &v3.RuleMetadata{
			Annotations: map[string]string{
				fmt.Sprintf("%s/lastUpdated", PolicyRecKeyName): rfc3339Time,
				fmt.Sprintf("%s/name", PolicyRecKeyName):        name,
				fmt.Sprintf("%s/namespace", PolicyRecKeyName):   namespace,
				fmt.Sprintf("%s/scope", PolicyRecKeyName):       string(EgressToServiceScope),
			},
		},
		Action:   v3.Allow,
		Protocol: protocol,
	}

	if protocol.SupportsPorts() {
		rule.Destination.Ports = sortPorts(ports)
	}
	rule.Destination.Services = &v3.ServiceMatch{
		Name:      name,
		Namespace: namespace,
	}

	return rule
}

// GetNamespaceV3Rule returns the traffic to namespace rule. The entity rule ports are
// in sorted order.
//
// Metadata.Annotations:
//
//	policyrecommendation.tigera.io/lastUpdated=<RFC3339 formatted timestamp>
//	policyrecommendation.tigera.io/namespace = '<namespace>'
//	policyrecommendation.tigera.io/scope = 'Namespace'
//
// EntityRule.Ports:
//
//	set of ports from flows (always destination rule)
//
// EntityRule.Selector:
//
// EntityRule.NamespaceSelector:
//
//	projectcalico.org/name == '<namespace>'
func GetNamespaceV3Rule(
	direction DirectionType, namespace string, ports []numorstring.Port, protocol *numorstring.Protocol, rfc3339Time string,
) *v3.Rule {
	rule := &v3.Rule{
		Metadata: &v3.RuleMetadata{
			Annotations: map[string]string{
				fmt.Sprintf("%s/lastUpdated", PolicyRecKeyName): rfc3339Time,
				fmt.Sprintf("%s/namespace", PolicyRecKeyName):   namespace,
				fmt.Sprintf("%s/scope", PolicyRecKeyName):       string(NamespaceScope),
				fmt.Sprintf("%s/warnings", PolicyRecKeyName):    nonServiceTypeWarning,
			},
		},
		Action:   v3.Allow,
		Protocol: protocol,
	}

	entityRule := getEntityRuleReference(direction, rule)
	entityRule.NamespaceSelector = fmt.Sprintf("%s/name == '%s'", projectCalicoKeyName, namespace)
	if protocol.SupportsPorts() {
		rule.Destination.Ports = sortPorts(ports)
	}

	return rule
}

// GetNetworkSetV3Rule returns the traffic to network set rule. The entity rule ports are in sorted
// order.
//
// Metadata.Annotations
//
//	policyrecommendation.tigera.io/lastUpdated=<RFC3339 formatted timestamp>
//	policyrecommendation.tigera.io/name = <name>
//	policyrecommendation.tigera.io/namespace = <namespace>
//	policyrecommendation.tigera.io/scope = ‘NetworkSet’
//
// EntityRule.Ports:
//
//	set of ports from flows (always destination rule)
//
// EntityRule.Selector:
//
//	projectcalico.org/name == '<name>' && projectcalico.org/kind == 'NetworkSet'
//
// EntityRule.NamespaceSelector:
//
//	projectcalico.org/name == '<namespace>', or global()
func GetNetworkSetV3Rule(
	direction DirectionType, name, namespace string, global bool, ports []numorstring.Port, protocol *numorstring.Protocol, rfc3339Time string,
) *v3.Rule {
	rule := &v3.Rule{
		Metadata: &v3.RuleMetadata{
			Annotations: map[string]string{
				fmt.Sprintf("%s/lastUpdated", PolicyRecKeyName): rfc3339Time,
				fmt.Sprintf("%s/name", PolicyRecKeyName):        name,
				fmt.Sprintf("%s/namespace", PolicyRecKeyName):   namespace,
				fmt.Sprintf("%s/scope", PolicyRecKeyName):       string(NetworkSetScope),
			},
		},
		Action:   v3.Allow,
		Protocol: protocol,
	}

	entityRule := getEntityRuleReference(direction, rule)
	entityRule.Selector = fmt.Sprintf("%s/name == '%s' && %s/kind == '%s'",
		projectCalicoKeyName, name, projectCalicoKeyName, string(NetworkSetScope))
	if global {
		entityRule.NamespaceSelector = "global()"
	} else {
		entityRule.NamespaceSelector = fmt.Sprintf("%s/name == '%s'", projectCalicoKeyName, namespace)
	}

	if protocol.SupportsPorts() {
		rule.Destination.Ports = sortPorts(ports)
	}

	return rule
}

// GetPrivateNetworkV3Rule returns the traffic to private network set rule. The entity rule ports
// are in sorted order.
//
// Metadata.Annotations
//
//	policyrecommendation.tigera.io/lastUpdated=<RFC3339 formatted timestamp>
//	policyrecommendation.tigera.io/scope = ‘Private’
//
// Destination.Ports:
//
//	set of ports from flows (always destination rule)
//
// EntityRule.Selector
//
//	policyrecommendation.tigera.io/scope == ‘Private’
//
// EntityRule.Nets:
//
//   - "10.0.0.0/8"
//   - "172.16.0.0/12"
//   - "192.168.0.0/16"
func GetPrivateNetworkV3Rule(
	direction DirectionType, ports []numorstring.Port, protocol *numorstring.Protocol, rfc3339Time string,
) *v3.Rule {
	rule := &v3.Rule{
		Metadata: &v3.RuleMetadata{
			Annotations: map[string]string{
				fmt.Sprintf("%s/lastUpdated", PolicyRecKeyName): rfc3339Time,
				fmt.Sprintf("%s/scope", PolicyRecKeyName):       string(PrivateNetworkScope),
			},
		},
		Action:   v3.Allow,
		Protocol: protocol,
	}

	entityRule := getEntityRuleReference(direction, rule)
	if protocol.SupportsPorts() {
		rule.Destination.Ports = sortPorts(ports)
	}
	entityRule.Nets = []string{privateNetwork24BitBlock, privateNetwork20BitBlock, privateNetwork16BitBlock}

	return rule
}

// GetPublicV3Rule returns the traffic to public network set rule. The entity rule ports are in
// sorted order.
//
// Metadata.Annotations:
//
//	policyrecommendation.tigera.io/lastUpdated = <RFC3339 formatted timestamp>
//	policyrecommendation.tigera.io/scope = ‘Public’
//
// EntityRule.Ports:
//
//	set of ports from flows (always destination rule)
func GetPublicV3Rule(ports []numorstring.Port, protocol *numorstring.Protocol, rfc3339Time string) *v3.Rule {
	rule := &v3.Rule{
		Metadata: &v3.RuleMetadata{
			Annotations: map[string]string{
				fmt.Sprintf("%s/lastUpdated", PolicyRecKeyName): rfc3339Time,
				fmt.Sprintf("%s/scope", PolicyRecKeyName):       string(PublicNetworkScope),
			},
		},
		Action:   v3.Allow,
		Protocol: protocol,
	}

	if protocol.SupportsPorts() {
		rule.Destination.Ports = sortPorts(ports)
	}

	return rule
}

// getEntityRuleReference returns the entity rule pointer, given the traffic direction.
func getEntityRuleReference(direction DirectionType, rule *v3.Rule) *v3.EntityRule {
	var entityRule *v3.EntityRule
	if direction == EgressTraffic {
		entityRule = &rule.Destination
	} else if direction == IngressTraffic {
		entityRule = &rule.Source
	}

	return entityRule
}

// sortDomains returns a sorted list of ports, sorted by min port.
func sortDomains(domains []string) []string {
	sortedDomains := domains
	sort.SliceStable(sortedDomains, func(i, j int) bool {
		return sortedDomains[i] < sortedDomains[j]
	})

	return sortedDomains
}

// sortPorts returns a sorted list of ports, sorted by min port.
func sortPorts(ports []numorstring.Port) []numorstring.Port {
	sortedPorts := ports
	sort.SliceStable(sortedPorts, func(i, j int) bool {
		if sortedPorts[i].MinPort != sortedPorts[j].MinPort {
			return sortedPorts[i].MinPort < sortedPorts[j].MinPort
		}
		if sortedPorts[i].MaxPort != sortedPorts[j].MaxPort {
			return sortedPorts[i].MaxPort < sortedPorts[j].MaxPort
		}
		return sortedPorts[i].PortName < sortedPorts[j].PortName
	})

	return sortedPorts
}
