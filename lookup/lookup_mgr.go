// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package lookup

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/felix/rules"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/set"
)

const (
	// String values used in the string representation of the RuleID. These are used
	// in some of the external APIs and therefore should not be modified.
	RuleDirIngressStr  string = "ingress"
	RuleDirEgressStr          = "egress"
	ActionAllowStr            = "allow"
	ActionDenyStr             = "deny"
	ActionNextTierStr         = "pass"
	GlobalNamespaceStr        = "__GLOBAL__"
	ProfileTierStr            = "__PROFILE__"
	NoMatchNameStr            = "__NO_MATCH__"
)

// RuleID contains the complete identifiers for a particular rule. This is a breakdown of the
// Felix v1 representation into the v3 representation used by the API and the collector.
type RuleID struct {
	// The tier name. If this is blank this represents a Profile backed rule.
	Tier string
	// The policy or profile name. This has the tier removed from the name. If this is blank, this represents
	// a "no match" rule. For k8s policies, this will be the full v3 name (knp.default.<k8s name>) - this avoids
	// name conflicts with Calico policies.
	Name string
	// The namespace. This is only non-blank for a NetworkPolicy type. For Tiers, GlobalNetworkPolicies and the
	// no match rules this will be blank.
	Namespace string
	// The rule direction.
	Direction rules.RuleDir
	// The index into the rule slice.
	Index int
	// A stringified version of the above index (stored to avoid frequent conversion)
	IndexStr string
	// The rule action.
	Action rules.RuleAction

	// Optimization so that the hot path doesn't need to create strings.
	dpName string
}

func NewRuleID(tier, policy, namespace string, ruleIndex int, ruleDirection rules.RuleDir, ruleAction rules.RuleAction) *RuleID {
	rid := &RuleID{
		Tier:      tier,
		Name:      policy,
		Namespace: namespace,
		Direction: ruleDirection,
		Index:     ruleIndex,
		IndexStr:  strconv.Itoa(ruleIndex),
		Action:    ruleAction,
	}
	rid.setDeniedPacketRuleName()
	return rid
}

func (r *RuleID) Equals(r2 *RuleID) bool {
	return r.Tier == r2.Tier &&
		r.Name == r2.Name &&
		r.Namespace == r2.Namespace &&
		r.Direction == r2.Direction &&
		r.Index == r2.Index &&
		r.Action == r2.Action
}

func (r *RuleID) String() string {
	return fmt.Sprintf(
		"Rule(Tier=%s,Name=%s,Namespace=%s,Direction=%s,Index=%d,Action=%s)",
		r.TierString(), r.NameString(), r.NamespaceString(), r.DirectionString(), r.IndexStr, r.ActionString(),
	)
}

func (r *RuleID) IsNamespaced() bool {
	return len(r.Namespace) != 0
}

func (r *RuleID) IsProfile() bool {
	return len(r.Tier) == 0
}

func (r *RuleID) IsNoMatchRule() bool {
	return len(r.Name) == 0
}

// TierString returns either the Tier name or the Profile indication string.
func (r *RuleID) TierString() string {
	if len(r.Tier) == 0 {
		return ProfileTierStr
	}
	return r.Tier
}

// NameString returns either the resource name or the No-match indication string.
func (r *RuleID) NameString() string {
	if len(r.Name) == 0 {
		return NoMatchNameStr
	}
	return r.Name
}

// NamespaceString returns either the resource namespace or the Global indication string.
func (r *RuleID) NamespaceString() string {
	if len(r.Namespace) == 0 {
		return GlobalNamespaceStr
	}
	return r.Namespace
}

// ActionString converts the action to a string value.
func (r *RuleID) ActionString() string {
	switch r.Action {
	case rules.RuleActionDeny:
		return ActionDenyStr
	case rules.RuleActionAllow:
		return ActionAllowStr
	case rules.RuleActionNextTier:
		return ActionNextTierStr
	}
	return ""
}

// DirectionString converts the direction to a string value.
func (r *RuleID) DirectionString() string {
	switch r.Direction {
	case rules.RuleDirIngress:
		return RuleDirIngressStr
	case rules.RuleDirEgress:
		return RuleDirEgressStr
	}
	return ""
}

func (r *RuleID) setDeniedPacketRuleName() {
	if r.Action != rules.RuleActionDeny {
		return
	}
	if r.IsNamespaced() {
		r.dpName = fmt.Sprintf(
			"%s|%s|%s|%s",
			r.TierString(),
			r.NameString(),
			r.IndexStr,
			r.ActionString(),
		)
	}
	r.dpName = fmt.Sprintf(
		"%s|%s/%s|%s|%s",
		r.TierString(),
		r.Namespace,
		r.NameString(),
		r.IndexStr,
		r.ActionString(),
	)
}

func (r *RuleID) GetDeniedPacketRuleName() string {
	return r.dpName
}

// Endpoint is the minimum cached information for each endpoint.
type Endpoint struct {
	// The endpoint name (usable as a unique key for an endpoint)
	Name string
	// An ordered set of tiers that may apply to this endpoint.
	OrderedTiers []string
}

// TODO (Matt): WorkloadEndpoints are only local; so we can't get nice information for the remote ends.
// TODO (rlb): The reverse mapping doesn't work if there are multiple addresses (e.g. IPv4 and IPv6) per
//             endpoint.
type LookupManager struct {
	// `string`s are IP.String().
	epMutex          sync.RWMutex
	endpoints        map[[16]byte]Endpoint
	endpointsReverse map[string]*[16]byte

	nflogPrefixesPolicy  map[model.PolicyKey]set.Set
	nflogPrefixesProfile map[model.ProfileKey]set.Set
	nflogPrefixHash      map[[64]byte]*RuleID
	nflogMutex           sync.RWMutex

	tierRefs map[string]int
}

func NewLookupManager() *LookupManager {
	lm := &LookupManager{
		endpoints:            map[[16]byte]Endpoint{},
		endpointsReverse:     map[string]*[16]byte{},
		epMutex:              sync.RWMutex{},
		nflogPrefixesPolicy:  map[model.PolicyKey]set.Set{},
		nflogPrefixesProfile: map[model.ProfileKey]set.Set{},
		nflogPrefixHash:      map[[64]byte]*RuleID{},
		nflogMutex:           sync.RWMutex{},
		tierRefs:             map[string]int{},
	}
	// Add NFLog mappings for the no-profile match.
	lm.addNFLogPrefixEntry(
		rules.CalculateNoMatchProfileNFLOGPrefixStr(rules.RuleDirIngress),
		NewRuleID("", "", "", 0, rules.RuleDirIngress, rules.RuleActionDeny),
	)

	lm.addNFLogPrefixEntry(
		rules.CalculateNoMatchProfileNFLOGPrefixStr(rules.RuleDirEgress),
		NewRuleID("", "", "", 0, rules.RuleDirEgress, rules.RuleActionDeny),
	)
	return lm
}

func (m *LookupManager) OnUpdate(protoBufMsg interface{}) {
	switch msg := protoBufMsg.(type) {
	case *proto.WorkloadEndpointUpdate:
		name := workloadEndpointName(msg.Id)
		ep := Endpoint{
			Name:         name,
			OrderedTiers: make([]string, len(msg.Endpoint.Tiers)),
		}
		for i := range msg.Endpoint.Tiers {
			ep.OrderedTiers[i] = msg.Endpoint.Tiers[i].Name
		}

		// Store the endpoint keyed off the IP addresses, and the reverse maps.
		for _, ipv4 := range msg.Endpoint.Ipv4Nets {
			addr, _, err := net.ParseCIDR(ipv4)
			if err != nil {
				log.Warnf("Error parsing CIDR %v", ipv4)
				continue
			}
			var addrB [16]byte
			copy(addrB[:], addr.To16()[:16])
			m.epMutex.Lock()
			m.endpoints[addrB] = ep
			m.endpointsReverse[name] = &addrB
			m.epMutex.Unlock()
		}
		for _, ipv6 := range msg.Endpoint.Ipv6Nets {
			addr, _, err := net.ParseCIDR(ipv6)
			if err != nil {
				log.Warnf("Error parsing CIDR %v", ipv6)
				continue
			}
			var addrB [16]byte
			copy(addrB[:], addr.To16()[:16])
			m.epMutex.Lock()
			m.endpoints[addrB] = ep
			m.endpointsReverse[name] = &addrB
			m.epMutex.Unlock()
		}
	case *proto.WorkloadEndpointRemove:
		name := workloadEndpointName(msg.Id)
		m.epMutex.Lock()
		epIp := m.endpointsReverse[name]
		if epIp != nil {
			delete(m.endpoints, *epIp)
			delete(m.endpointsReverse, name)
		}
		m.epMutex.Unlock()

	case *proto.ActivePolicyUpdate:
		m.updatePolicyRulesNFLOGPrefixes(msg)

	case *proto.ActivePolicyRemove:
		m.removePolicyRulesNFLOGPrefixes(msg)

	case *proto.ActiveProfileUpdate:
		m.updateProfileRulesNFLOGPrefixes(msg)

	case *proto.ActiveProfileRemove:
		m.removeProfileRulesNFLOGPrefixes(msg)

	case *proto.HostEndpointUpdate:
		name := hostEndpointName(msg.Id)
		numUntracked := len(msg.Endpoint.UntrackedTiers)
		ep := Endpoint{
			Name:         name,
			OrderedTiers: make([]string, numUntracked+len(msg.Endpoint.Tiers)),
		}
		for i := range msg.Endpoint.UntrackedTiers {
			ep.OrderedTiers[i] = msg.Endpoint.UntrackedTiers[i].Name
		}
		for i := range msg.Endpoint.Tiers {
			ep.OrderedTiers[numUntracked+i] = msg.Endpoint.Tiers[i].Name
		}

		// Store the endpoint keyed off the IP addresses, and the reverse maps.
		for _, ipv4 := range msg.Endpoint.ExpectedIpv4Addrs {
			addr := net.ParseIP(ipv4)
			if addr == nil {
				log.Warn("Error parsing IP ", ipv4)
				continue
			}
			var addrB [16]byte
			copy(addrB[:], addr.To16()[:16])
			m.epMutex.Lock()
			m.endpoints[addrB] = ep
			m.endpointsReverse[name] = &addrB
			m.epMutex.Unlock()
		}
		for _, ipv6 := range msg.Endpoint.ExpectedIpv6Addrs {
			addr := net.ParseIP(ipv6)
			if addr == nil {
				log.Warn("Error parsing IP ", ipv6)
				continue
			}
			var addrB [16]byte
			copy(addrB[:], addr.To16()[:16])
			m.epMutex.Lock()
			m.endpoints[addrB] = ep
			m.endpointsReverse[name] = &addrB
			m.epMutex.Unlock()
		}
	case *proto.HostEndpointRemove:
		// We omit setting Hostname since at this point it is implied that the
		// endpoint belongs to this host.
		name := hostEndpointName(msg.Id)
		m.epMutex.Lock()
		epIp := m.endpointsReverse[name]
		if epIp != nil {
			delete(m.endpoints, *epIp)
			delete(m.endpointsReverse, name)
		}
		m.epMutex.Unlock()
	}
}

func (m *LookupManager) CompleteDeferredWork() error {
	return nil
}

// workloadEndpointName returns a single string rep of the workload endpoint that can also
// be used as a lookup key in our maps.
func workloadEndpointName(wep *proto.WorkloadEndpointID) string {
	// We are only interested in local endpoints, so host does not need to be included.
	return "WEP(" + wep.OrchestratorId + "/" + wep.WorkloadId + "/" + wep.EndpointId + ")"
}

// hostEndpointName returns a single string rep of the host endpoint that can also
// be used as a lookup key in our maps.
func hostEndpointName(hep *proto.HostEndpointID) string {
	// We are only interested in local endpoints, so host does not need to be included.
	return "HEP(" + hep.EndpointId + ")"
}

// addNFLogPrefixEntry adds a single NFLOG prefix entry to our internal cache.
func (m *LookupManager) addNFLogPrefixEntry(prefix string, ruleIDs *RuleID) {
	var bph [64]byte
	copy(bph[:], []byte(prefix[:]))
	m.nflogMutex.Lock()
	defer m.nflogMutex.Unlock()
	m.nflogPrefixHash[bph] = ruleIDs
}

// deleteNFLogPrefixEntry deletes a single NFLOG prefix entry to our internal cache.
func (m *LookupManager) deleteNFLogPrefixEntry(prefix string) {
	var bph [64]byte
	copy(bph[:], []byte(prefix[:]))
	m.nflogMutex.Lock()
	defer m.nflogMutex.Unlock()
	delete(m.nflogPrefixHash, bph)
}

// updatePolicyRulesNFLOGPrefixes stores the required prefix to RuleID maps for a policy, deleting any
// stale entries if the number of rules or action types have changed.
func (m *LookupManager) updatePolicyRulesNFLOGPrefixes(msg *proto.ActivePolicyUpdate) {
	// If this is the first time we have seen this tier, add the default deny entries for the tier.
	count, ok := m.tierRefs[msg.Id.Tier]
	if !ok {
		m.addNFLogPrefixEntry(
			rules.CalculateNoMatchPolicyNFLOGPrefixStr(rules.RuleDirIngress, msg.Id.Tier),
			NewRuleID(msg.Id.Tier, "", "", 0, rules.RuleDirIngress, rules.RuleActionDeny),
		)
		m.addNFLogPrefixEntry(
			rules.CalculateNoMatchPolicyNFLOGPrefixStr(rules.RuleDirEgress, msg.Id.Tier),
			NewRuleID(msg.Id.Tier, "", "", 0, rules.RuleDirEgress, rules.RuleActionDeny),
		)
	}
	m.tierRefs[msg.Id.Tier] = count + 1

	namespace, tier, name, err := deconstructPolicyName(msg.Id.Name)
	if err != nil {
		log.WithError(err).Error("Unable to parse policy name")
		return
	}

	key := model.PolicyKey{
		Tier: msg.Id.Tier,
		Name: msg.Id.Name,
	}
	oldPrefixes := m.nflogPrefixesPolicy[key]
	m.nflogPrefixesPolicy[key] = m.updateRulesNFLOGPrefixes(
		msg.Id.Name,
		namespace,
		tier,
		name,
		oldPrefixes,
		msg.Policy.InboundRules,
		msg.Policy.OutboundRules,
	)
}

// removePolicyRulesNFLOGPrefixes removes the prefix to RuleID maps for a policy.
func (m *LookupManager) removePolicyRulesNFLOGPrefixes(msg *proto.ActivePolicyRemove) {
	// If this is the last entry for the tier, remove the default deny entries for the tier.
	// Increment the reference count so that we don't keep adding tiers.
	count := m.tierRefs[msg.Id.Tier]
	if count == 1 {
		delete(m.tierRefs, msg.Id.Tier)
		m.deleteNFLogPrefixEntry(
			rules.CalculateNoMatchPolicyNFLOGPrefixStr(rules.RuleDirIngress, msg.Id.Tier),
		)
		m.deleteNFLogPrefixEntry(
			rules.CalculateNoMatchPolicyNFLOGPrefixStr(rules.RuleDirEgress, msg.Id.Tier),
		)
	} else {
		m.tierRefs[msg.Id.Tier] = count - 1
	}

	key := model.PolicyKey{
		Tier: msg.Id.Tier,
		Name: msg.Id.Name,
	}
	oldPrefixes := m.nflogPrefixesPolicy[key]
	m.deleteRulesNFLOGPrefixes(oldPrefixes)
	delete(m.nflogPrefixesPolicy, key)
}

// updateProfileRulesNFLOGPrefixes stores the required prefix to RuleID maps for a profile, deleting any
// stale entries if the number of rules or action types have changed.
func (m *LookupManager) updateProfileRulesNFLOGPrefixes(msg *proto.ActiveProfileUpdate) {
	key := model.ProfileKey{
		Name: msg.Id.Name,
	}
	oldPrefixes := m.nflogPrefixesProfile[key]
	m.nflogPrefixesProfile[key] = m.updateRulesNFLOGPrefixes(
		msg.Id.Name,
		"",
		"",
		msg.Id.Name,
		oldPrefixes,
		msg.Profile.InboundRules,
		msg.Profile.OutboundRules,
	)
}

// removeProfileRulesNFLOGPrefixes removes the prefix to RuleID maps for a profile.
func (m *LookupManager) removeProfileRulesNFLOGPrefixes(msg *proto.ActiveProfileRemove) {
	key := model.ProfileKey{
		Name: msg.Id.Name,
	}
	oldPrefixes := m.nflogPrefixesProfile[key]
	m.deleteRulesNFLOGPrefixes(oldPrefixes)
	delete(m.nflogPrefixesProfile, key)
}

// updateRulesNFLOGPrefixes updates our NFLOG prefix to RuleID map based on the supplied set of
// ingress and egress rules, and the old set of prefixes associated with the previous resource
// settings. This method adds any new rules and removes any obsolete rules.
// TODO (rlb): Maybe we should do a lazy clean up of rules?
func (m *LookupManager) updateRulesNFLOGPrefixes(
	v1Name, namespace, tier, name string, oldPrefixes set.Set, ingress []*proto.Rule, egress []*proto.Rule,
) set.Set {
	newPrefixes := set.New()

	convertAction := func(a string) rules.RuleAction {
		switch a {
		case "allow":
			return rules.RuleActionAllow
		case "deny":
			return rules.RuleActionDeny
		case "pass", "next-tier":
			return rules.RuleActionNextTier
		}
		return rules.RuleActionDeny
	}
	owner := rules.RuleOwnerTypePolicy
	if tier == "" {
		owner = rules.RuleOwnerTypeProfile
	}
	for ii, rule := range ingress {
		action := convertAction(rule.Action)
		prefix := rules.CalculateNFLOGPrefixStr(action, owner, rules.RuleDirIngress, ii, v1Name)
		m.addNFLogPrefixEntry(
			prefix,
			NewRuleID(tier, name, namespace, ii, rules.RuleDirIngress, action),
		)
		newPrefixes.Add(prefix)
	}
	for ii, rule := range egress {
		action := convertAction(rule.Action)
		prefix := rules.CalculateNFLOGPrefixStr(action, owner, rules.RuleDirEgress, ii, v1Name)
		m.addNFLogPrefixEntry(
			prefix,
			NewRuleID(tier, name, namespace, ii, rules.RuleDirEgress, action),
		)
		newPrefixes.Add(prefix)
	}

	// Delete the stale prefixes.
	if oldPrefixes != nil {
		oldPrefixes.Iter(func(item interface{}) error {
			if !newPrefixes.Contains(item) {
				m.deleteNFLogPrefixEntry(item.(string))
			}
			return nil
		})
	}

	return newPrefixes
}

// deleteRulesNFLOGPrefixes deletes the supplied set of prefixes.
func (m *LookupManager) deleteRulesNFLOGPrefixes(prefixes set.Set) {
	if prefixes != nil {
		prefixes.Iter(func(item interface{}) error {
			m.deleteNFLogPrefixEntry(item.(string))
			return nil
		})
	}
}

// IsEndpoint returns true if the supplied address is a local endpoint, otherwise returns false.
func (m *LookupManager) IsEndpoint(addr [16]byte) bool {
	m.epMutex.RLock()
	defer m.epMutex.RUnlock()
	_, ok := m.endpoints[addr]
	return ok
}

// GetEndpoint returns the ordered list of tiers for a particular endpoint.
func (m *LookupManager) GetEndpoint(addr [16]byte) (Endpoint, bool) {
	m.epMutex.RLock()
	defer m.epMutex.RUnlock()
	ep, ok := m.endpoints[addr]
	return ep, ok
}

// GetRuleIDFromNFLOGPrefix returns the RuleID associated with the supplied NFLOG prefix.
func (m *LookupManager) GetRuleIDFromNFLOGPrefix(prefix [64]byte) *RuleID {
	m.nflogMutex.RLock()
	defer m.nflogMutex.RUnlock()
	return m.nflogPrefixHash[prefix]
}

// deconstructPolicyName deconstructs the v1 policy name that is constructed by the SyncerUpdateProcessors in
// libcalico-go and extracts the v3 fields: namespace, tier, name.
//
// The v1 policy name is of the format:
// -  <namespace>/<tier>.<name> for a namespaced NetworkPolicies
// -  <tier>.<name> for GlobalNetworkPolicies.
// -  <namespace>/knp.default.k8spolicy for a k8s NetworkPolicies
//
// The namespace is returned blank for GlobalNetworkPolicies.
// For k8s network policies, the tier is always "default" and the name will be returned including the
// knp.default prefix.
func deconstructPolicyName(name string) (string, string, string, error) {
	parts := strings.Split(name, "/")
	switch len(parts) {
	case 1: // GlobalNetworkPolicy
		nameParts := strings.Split(parts[0], ".")
		if len(nameParts) != 2 {
			return "", "", "", fmt.Errorf("Could not parse policy %s", name)
		}
		return "", nameParts[0], nameParts[1], nil
	case 2: // NetworkPolicy
		nameParts := strings.Split(parts[1], ".")
		switch len(nameParts) {
		case 2: // Non-k8s
			return parts[0], nameParts[0], nameParts[1], nil
		case 3: // K8s
			if nameParts[0] == "knp" && nameParts[1] == "default" {
				return parts[0], nameParts[1], parts[1], nil
			}
		}
	}
	return "", "", "", fmt.Errorf("Could not parse policy %s", name)
}

// Dump returns the contents of important structures in the LookupManager used for
// logging purposes in the test code. This should not be used in any mainline code.
func (m *LookupManager) Dump() string {
	lines := []string{}
	for p, r := range m.nflogPrefixHash {
		lines = append(lines, string(p[:])+": "+r.String())
	}
	return strings.Join(lines, "\n")
}

// SetMockData fills in some of the data structures for use in the test code. This should not
// be called from any mainline code.
func (m *LookupManager) SetMockData(
	em map[[16]byte]*model.WorkloadEndpointKey,
	nm map[[64]byte]*RuleID,
) {
	m.nflogPrefixHash = nm
	for ip, wep := range em {
		ep := Endpoint{
			Name: workloadEndpointName(&proto.WorkloadEndpointID{
				OrchestratorId: wep.OrchestratorID,
				WorkloadId:     wep.WorkloadID,
				EndpointId:     wep.EndpointID,
			}),
			OrderedTiers: []string{"default"},
		}
		m.endpoints[ip] = ep
	}
}
