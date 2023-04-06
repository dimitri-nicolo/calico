// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package engine

import (
	"fmt"
	"reflect"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/api/pkg/lib/numorstring"

	calico "github.com/projectcalico/calico/continuous-policy-recommendation/pkg/calico-resources"
	calicores "github.com/projectcalico/calico/continuous-policy-recommendation/pkg/calico-resources"
	"github.com/projectcalico/calico/lma/pkg/api"
)

type flowType string

const (
	egressToDomainFlowType flowType = "egressToDomainFlowType"
	//nolint
	egressToDomainSetFlowType flowType = "egressToDomainSetFlowType"
	egressToServiceFlowType   flowType = "egressToServiceFlowType"
	namespaceFlowType         flowType = "namespaceFlowType"
	networkSetFlowType        flowType = "networkSetFlowType"
	privateNetworkFlowType    flowType = "privateNetworkFlowType"
	publicNetworkFlowType     flowType = "publicNetworkFlowType"
	unsupportedFlowType       flowType = "unsupportedFlowType"
)

type EngineRules interface {
	addFlowToEgressToDomainRules(direction calico.DirectionType, flow api.Flow)
	addFlowToEgressToDomainSetRules(direction calico.DirectionType, flow api.Flow)
	addFlowToEgressToServiceRules(direction calico.DirectionType, flow api.Flow)
	addFlowToNamespaceRules(direction calico.DirectionType, flow api.Flow)
	addFlowToNetworkSetRules(direction calico.DirectionType, flow api.Flow)
	addFlowToPrivateNetworkRules(direction calico.DirectionType, flow api.Flow)
	addFlowToPublicNetworkRules(direction calico.DirectionType, flow api.Flow)

	updateEngineRules(key, value any)
}

// engineRules implements the EngineRules interface. It defines the policy recommendation engine
// rules. The following types of rules are recommended by the PRE, and apply to the following flows:
// 1. Egress to domain flows.
// 2. Egress to service flows.
// 3. Namespace flows.
// 4. NetworkSet flows.
// 5. Private network flows.
// 6. Public network flows.
type engineRules struct {
	egressToDomainRules  map[egressToDomainRuleKey]*egressToDomainRule
	egressToServiceRules map[egressToServiceRuleKey]*egressToServiceRule
	namespaceRules       map[namespaceRuleKey]*namespaceRule
	networkSetRules      map[networkSetRuleKey]*networkSetRule
	privateNetworkRules  map[privateNetworkRuleKey]*privateNetworkRule
	publicNetworkRules   map[publicNetworkRuleKey]*publicNetworkRule

	size int
}

type egressToDomainRuleKey struct {
	protocol numorstring.Protocol
	port     numorstring.Port
}

type egressToDomainRule struct {
	domains   []string
	protocol  numorstring.Protocol
	port      numorstring.Port
	timestamp string
}

type egressToServiceRuleKey struct {
	name      string
	namespace string
	protocol  numorstring.Protocol
}

type egressToServiceRule struct {
	name      string
	namespace string
	ports     []numorstring.Port
	protocol  numorstring.Protocol
	timestamp string
}

type namespaceRuleKey struct {
	namespace string
	protocol  numorstring.Protocol
}

type namespaceRule struct {
	namespace string
	ports     []numorstring.Port
	protocol  numorstring.Protocol
	timestamp string
}

type networkSetRuleKey struct {
	global    bool
	name      string
	namespace string
	protocol  numorstring.Protocol
}

type networkSetRule struct {
	global    bool
	name      string
	namespace string
	ports     []numorstring.Port
	protocol  numorstring.Protocol
	timestamp string
}

type privateNetworkRuleKey struct {
	protocol numorstring.Protocol
}

type privateNetworkRule struct {
	name      string
	protocol  numorstring.Protocol
	ports     []numorstring.Port
	timestamp string
}

type publicNetworkRuleKey struct {
	protocol numorstring.Protocol
}

type publicNetworkRule struct {
	protocol  numorstring.Protocol
	ports     []numorstring.Port
	timestamp string
}

// addFlowToEgressToDomainRules updates the ports if a rule already exists, otherwise defines a
// new egressToServiceRule for egress flows, where the remote is to a service (but not a pod).
// If there are > max number of rules in the policy, all egress domains will be included in an
// egress-domains network set.
//
// Only required if the number of rules exceeded its maximum. Domain rules will be contracted to use
// an egress domain NetworkSet specific to the namespace. If a flow is egress to a domain, rule is
// an egress rule to a namespace specific network set that will contain all of the domains.
//
// Namespaced network sets will be created for egress domains:
//
//	Name: "<namespace>-egress-domains"
//	Labels:
//			policyrecommendation.tigera.io/scope = 'Domains'
//	Domains from flow logs
//	OwnerReference will be the PolicyRecommendationScope resource.
//
// Policy rules will be added as follows for each protocol that we need to support:
// Note: The lastUpdated timestamp will be updated if the corresponding NetworkSet is updated to
// include an additional domain, even if the rule itself is unchanged.
func (er *engineRules) addFlowToEgressToDomainRules(direction calico.DirectionType, flow *api.Flow, clock Clock) {
	if direction != calico.EgressTraffic {
		log.WithField("flow", flow).Warn("flow cannot be processed, unsupported traffic direction")
		return
	}

	endpoint := &flow.Destination

	port, err := numorstring.PortFromRange(*endpoint.Port, *endpoint.Port)
	if err != nil {
		log.WithError(err).WithField("flow", flow).Warn("flow cannot be processed, port could not be converted")
		return
	}
	domains := parseDomains(endpoint.Domains)
	protocol := api.GetProtocol(*flow.Proto)

	key := egressToDomainRuleKey{
		protocol: protocol,
		port:     port,
	}

	// Update the ports and return if the value already exists
	if v, ok := er.egressToDomainRules[key]; ok {
		for _, domain := range domains {
			// If no new domains are added then no update to the timestamp will occur
			if !containsDomain(v.domains, domain) {
				// Timestamp recorded will be that of last update.
				v.domains = append(v.domains, domain)
				v.timestamp = clock.NowRFC3339()
			}
		}

		return
	}

	// The key does not exist, define a new value and add the key-value to the egressToDomain rules
	val := egressToDomainRule{
		protocol:  protocol,
		port:      port,
		timestamp: clock.NowRFC3339(),
	}
	val.domains = domains

	// Add the value to the map of egress to domain rules and increment the total engine rules count
	er.egressToDomainRules[key] = &val
	er.size++
}

// addFlowToEgressToServiceRules updates the ports if a rule already exists, otherwise defines a
// new egressToDomainRule for egress flows, where the remote is to a domain.
func (er *engineRules) addFlowToEgressToServiceRules(direction calico.DirectionType, flow *api.Flow, clock Clock) {
	if direction != calico.EgressTraffic {
		log.WithField("flow", flow).Warn("flow cannot be processed, unsupported traffic direction")
		return
	}

	endpoint := &flow.Destination

	port, err := numorstring.PortFromRange(*endpoint.Port, *endpoint.Port)
	if err != nil {
		log.WithError(err).WithField("flow", flow).Warn("flow cannot be processed, port could not be converted")
		return
	}
	name := endpoint.ServiceName
	namespace := endpoint.Namespace
	protocol := api.GetProtocol(*flow.Proto)

	key := egressToServiceRuleKey{
		name:      name,
		namespace: namespace,
		protocol:  protocol,
	}

	// Update the ports and return if the value already exists.
	if v, ok := er.egressToServiceRules[key]; ok {
		if containsPort(v.ports, port) {
			// no update necessary
			return
		}
		v.ports = append(v.ports, port)
		v.timestamp = clock.NowRFC3339()

		return
	}

	// The key does not exist, define a new value and add the key-value to the egress to service rules.

	val := egressToServiceRule{
		name:      name,
		namespace: namespace,
		protocol:  protocol,
		timestamp: clock.NowRFC3339(),
	}
	val.ports = []numorstring.Port{port}

	// Add the value to the map of egress to service rules and increment the total engine rules count.
	er.egressToServiceRules[key] = &val
	er.size++
}

// addFlowToNamespaceRules updates the ports if a rule already exists, otherwise defines a
// new namespaceRule for flows where the remote is a pod. The rule simply selects the pod's
// namespace.
func (er *engineRules) addFlowToNamespaceRules(direction calico.DirectionType, flow *api.Flow, clock Clock) {
	var endpoint *api.FlowEndpointData
	if direction == calico.EgressTraffic {
		endpoint = &flow.Destination
	} else if direction == calico.IngressTraffic {
		endpoint = &flow.Source
	} else {
		log.WithField("flow", flow).Warn("flow cannot be processed, unsupported traffic direction")
		return
	}

	port, err := numorstring.PortFromRange(*flow.Destination.Port, *flow.Destination.Port)
	if err != nil {
		log.WithError(err).WithField("flow", flow).Warn("flow cannot be processed, port could not be converted")
		return
	}
	namespace := endpoint.Namespace
	protocol := api.GetProtocol(*flow.Proto)

	key := namespaceRuleKey{
		namespace: namespace,
		protocol:  protocol,
	}

	// Update the ports and return if the value already exists.
	if v, ok := er.namespaceRules[key]; ok {
		if containsPort(v.ports, port) {
			// no update necessary
			return
		}
		v.ports = append(v.ports, port)
		v.timestamp = clock.NowRFC3339()

		return
	}

	// The key does not exist, define a new value and add the key-value to the egress to service rules.

	val := namespaceRule{
		namespace: namespace,
		protocol:  protocol,
		timestamp: clock.NowRFC3339(),
	}
	val.ports = []numorstring.Port{}
	val.ports = append(val.ports, port)

	// Add the value to the map of egress to service rules and increment the total engine rules count.
	er.namespaceRules[key] = &val
	er.size++
}

// addFlowToNetworkSetRules updates the ports if a rule already exists, otherwise defines a
// new networkSetRule for flows where the remote is an existing NetworkSet or GlobalNetworkSet.
// A rule will be added to select the NetworkSet by name - this ensures we don’t require label
// schema knowledge.
func (er *engineRules) addFlowToNetworkSetRules(direction calico.DirectionType, flow *api.Flow, clock Clock) {
	var endpoint *api.FlowEndpointData
	if direction == calico.EgressTraffic {
		endpoint = &flow.Destination
	} else if direction == calico.IngressTraffic {
		endpoint = &flow.Source
	} else {
		log.WithField("flow", flow).Warn("flow cannot be processed, unsupported traffic direction")
		return
	}

	port, err := numorstring.PortFromRange(*endpoint.Port, *endpoint.Port)
	if err != nil {
		log.WithError(err).WithField("flow", flow).Warn("flow cannot be processed, port could not be converted")
		return
	}
	name := endpoint.Name
	namespace := endpoint.Namespace
	protocol := api.GetProtocol(*flow.Proto)

	gl := false
	if namespace == "-" || namespace == "" {
		gl = true
	}

	key := networkSetRuleKey{
		global:    gl,
		name:      name,
		namespace: namespace,
		protocol:  protocol,
	}

	// Update the ports
	if v, ok := er.networkSetRules[key]; ok {
		if containsPort(v.ports, port) {
			// port present, no update necessary
			return
		}
		v.ports = append(v.ports, port)
		v.timestamp = clock.NowRFC3339()

		return
	}

	// The key does not exist, add the new key-value rule

	val := networkSetRule{
		global:    gl,
		name:      name,
		namespace: namespace,
		protocol:  protocol,
		timestamp: clock.NowRFC3339(),
	}
	val.ports = []numorstring.Port{port}

	// Add the value to the map of egress to service rules and increment the total engine rules count.
	er.networkSetRules[key] = &val
	er.size++
}

// addFlowToPrivateNetworkRules updates the ports if a rule already exists, otherwise defines a
// new privateNetworkRule all ingress and egress flows from/to private network CIDRs that are not
// covered by all other categories, except for the public network rules. A global network set will
// be created by the PRE containing private CIDRs.
//
// A global network set will be created by the PRE containing private CIDRs.
//
//	Name: 'private-network'
//	Label:
//			policyrecommendation.tigera.io/scope = 'Private'
//	CIDRs will be defaults to those defined in RFC 1918
//	OwnerReference will be the PolicyRecommendationScope resource.
//
// The set of private CIDRs may/should be updated by the customer to only contain private CIDRs
// specific to the customer network, and should exclude the CIDR ranges used by the cluster for
// nodes and pods (**). The PRE will not update the CIDRs once the network set is created.
func (er *engineRules) addFlowToPrivateNetworkRules(direction calico.DirectionType, flow *api.Flow, clock Clock) {
	port, err := numorstring.PortFromRange(*flow.Destination.Port, *flow.Destination.Port)
	if err != nil {
		log.WithError(err).WithField("flow", flow).Warn("flow cannot be processed, port could not be converted")
		return
	}
	protocol := api.GetProtocol(*flow.Proto)

	key := privateNetworkRuleKey{
		protocol: protocol,
	}

	// Update the nets, if the value already exists
	if v, ok := er.privateNetworkRules[key]; ok {
		if containsPort(v.ports, port) {
			// no update necessary
			return
		}
		v.ports = append(v.ports, port)
		v.timestamp = clock.NowRFC3339()

		return
	}

	// The key does not exist, define a new value and add the key-value to the egress to service rules

	val := &privateNetworkRule{
		name:      calicores.PrivateNetworkSetName,
		protocol:  protocol,
		timestamp: clock.NowRFC3339(),
	}
	val.ports = []numorstring.Port{port}

	// Add the value to the map of private network rules and increment the total engine rules count
	er.privateNetworkRules[key] = val
	er.size++
}

// addFlowToPublicNetworkRules updates the ports if a rule already exists, otherwise defines a
// new publicNetworkRule covering all ingress and egress flows from/to other CIDRs that are not all
// other categories. It is a mop-up rule that is broad in scope. It covers all remaining flow
// endpoints limited only by port and protocol.
func (er *engineRules) addFlowToPublicNetworkRules(direction calico.DirectionType, flow *api.Flow, clock Clock) {
	if flow == nil {
		err := fmt.Errorf("empty flow")
		log.WithError(err).Debug("Failed to process public network flow")
		return
	}

	endpoint := flow.Destination
	port, err := numorstring.PortFromRange(*endpoint.Port, *endpoint.Port)
	if err != nil {
		log.WithError(err).WithField("flow", flow).Warn("flow cannot be processed, port could not be converted")
		return
	}
	protocol := api.GetProtocol(*flow.Proto)

	key := publicNetworkRuleKey{
		protocol: protocol,
	}

	// Update the ports and return if the value already exists
	if v, ok := er.publicNetworkRules[key]; ok {
		if containsPort(v.ports, port) {
			return
		}
		v.ports = append(v.ports, port)
		v.timestamp = clock.NowRFC3339()

		return
	}

	// The key does not exist, define a new value and add the key-value to the egress to service rules
	val := publicNetworkRule{
		protocol:  protocol,
		timestamp: clock.NowRFC3339(),
	}
	val.ports = []numorstring.Port{port}

	// Add the value to the map of egress to service rules and increment the total engine rules count
	er.publicNetworkRules[key] = &val
	er.size++
}

// containsDomain returns true if the array contains the domain.
func containsDomain(arr []string, val string) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}

	return false
}

// containsPort returns true if the array contains the port.
func containsPort(arr []numorstring.Port, val numorstring.Port) bool {
	for _, v := range arr {
		if reflect.DeepEqual(v, val) {
			// The value is already present
			return true
		}
	}

	return false
}

// getFlowType returns the engine flow type, or an error if the flow defines unsupported traffic.
func getFlowType(direction calico.DirectionType, flow *api.Flow) flowType {
	var endpoint *api.FlowEndpointData
	dest := false
	if direction == calicores.EgressTraffic {
		dest = true
		endpoint = &flow.Destination
	} else {
		endpoint = &flow.Source
	}

	if endpoint.Type == api.FlowLogEndpointTypeNetworkSet {
		// Private
		if endpoint.Name == calicores.PrivateNetworkSetName {
			return privateNetworkFlowType
		}
		// NetworkSet
		return networkSetFlowType
	}

	if endpoint.Type == api.FlowLogEndpointTypeWEP {
		if dest && (endpoint.Name == "-" || endpoint.Name == "") && endpoint.ServiceName != "-" && endpoint.ServiceName != "" {
			// EgressToService. Non-pod service
			return egressToServiceFlowType
		}
		if endpoint.Namespace != "-" && endpoint.Namespace != "" {
			// Namespace
			return namespaceFlowType
		}
	}

	if endpoint.Type == api.EndpointTypeNet {
		// EgressToDomain
		if dest && endpoint.Domains != "" && endpoint.Domains != "-" {
			return egressToDomainFlowType
		}
		// An api.Flow name is defined by the source/dest_name of a flow log, or by the
		// source/dest_name_aggr, if the name is a "-".
		name := endpoint.Name
		switch name {
		case api.FlowLogNetworkPrivate:
			// Private
			return privateNetworkFlowType
		default:
			// Public
			return publicNetworkFlowType
		}
	}

	log.Warnf("Unsupported flow type: %s", endpoint.Type)
	return unsupportedFlowType
}

// parseDomains separates a comma delimited string into a slice of strings and returns the slice.
func parseDomains(domainsAsStr string) []string {
	domains := strings.Split(domainsAsStr, ",")

	return domains
}
