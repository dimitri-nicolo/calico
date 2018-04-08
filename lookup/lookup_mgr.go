// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package lookup

import (
	"errors"
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
}

func (r1 *RuleID) Equals(r2 *RuleID) bool {
	return r1.Tier == r2.Tier &&
		r1.Name == r2.Name &&
		r1.Namespace == r2.Namespace &&
		r1.Direction == r2.Direction &&
		r1.Index == r2.Index &&
		r1.Action == r2.Action
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

var (
	UnknownEndpointError = errors.New("Unknown endpoint")
)

type QueryInterface interface {
	GetEndpointKey(addr [16]byte) (interface{}, error)
	GetTierIndex(epKey interface{}, tierName string) int
	GetRuleIDsFromNFLOGPrefix(prefix [64]byte) *RuleID
}

// TODO (Matt): WorkloadEndpoints are only local; so we can't get nice information for the remote ends.
type LookupManager struct {
	// `string`s are IP.String().
	workloadEndpoints        map[[16]byte]*model.WorkloadEndpointKey
	workloadEndpointsReverse map[model.WorkloadEndpointKey]*[16]byte
	workloadEndpointTiers    map[model.WorkloadEndpointKey][]*proto.TierInfo
	epMutex                  sync.RWMutex

	hostEndpoints              map[[16]byte]*model.HostEndpointKey
	hostEndpointsReverse       map[model.HostEndpointKey]*[16]byte
	hostEndpointTiers          map[model.HostEndpointKey][]*proto.TierInfo
	hostEndpointUntrackedTiers map[model.HostEndpointKey][]*proto.TierInfo
	hostEpMutex                sync.RWMutex

	nflogPrefixesPolicy  map[model.PolicyKey]set.Set
	nflogPrefixesProfile map[model.ProfileKey]set.Set
	nflogPrefixHash      map[[64]byte]*RuleID
	nflogMutex           sync.RWMutex

	tierRefs map[string]int
}

func NewLookupManager() *LookupManager {
	lm := &LookupManager{
		workloadEndpoints:          map[[16]byte]*model.WorkloadEndpointKey{},
		workloadEndpointsReverse:   map[model.WorkloadEndpointKey]*[16]byte{},
		workloadEndpointTiers:      map[model.WorkloadEndpointKey][]*proto.TierInfo{},
		hostEndpoints:              map[[16]byte]*model.HostEndpointKey{},
		hostEndpointsReverse:       map[model.HostEndpointKey]*[16]byte{},
		hostEndpointTiers:          map[model.HostEndpointKey][]*proto.TierInfo{},
		hostEndpointUntrackedTiers: map[model.HostEndpointKey][]*proto.TierInfo{},
		epMutex:                    sync.RWMutex{},
		hostEpMutex:                sync.RWMutex{},
		nflogPrefixesPolicy:        map[model.PolicyKey]set.Set{},
		nflogPrefixesProfile:       map[model.ProfileKey]set.Set{},
		nflogPrefixHash:            map[[64]byte]*RuleID{},
		nflogMutex:                 sync.RWMutex{},
		tierRefs:                   map[string]int{},
	}
	// Add NFLog mappings for the no-profile match.
	lm.addNFLogPrefixEntry(
		rules.CalculateNoMatchProfileNFLOGPrefixStr(rules.RuleDirIngress),
		&RuleID{
			Tier:      "",
			Name:      "",
			Namespace: "",
			Direction: rules.RuleDirIngress,
			Index:     0,
			IndexStr:  "0",
			Action:    rules.RuleActionDeny,
		},
	)
	lm.addNFLogPrefixEntry(
		rules.CalculateNoMatchProfileNFLOGPrefixStr(rules.RuleDirEgress),
		&RuleID{
			Tier:      "",
			Name:      "",
			Namespace: "",
			Direction: rules.RuleDirEgress,
			Index:     0,
			IndexStr:  "0",
			Action:    rules.RuleActionDeny,
		},
	)
	return lm
}

func (m *LookupManager) OnUpdate(protoBufMsg interface{}) {
	switch msg := protoBufMsg.(type) {
	case *proto.WorkloadEndpointUpdate:
		// We omit setting Hostname since at this point it is implied that the
		// endpoint belongs to this host.
		wlEpKey := model.WorkloadEndpointKey{
			OrchestratorID: msg.Id.OrchestratorId,
			WorkloadID:     msg.Id.WorkloadId,
			EndpointID:     msg.Id.EndpointId,
		}
		m.epMutex.Lock()
		// Store tiers and policies
		m.workloadEndpointTiers[wlEpKey] = msg.Endpoint.Tiers
		// Store IP addresses
		for _, ipv4 := range msg.Endpoint.Ipv4Nets {
			addr, _, err := net.ParseCIDR(ipv4)
			if err != nil {
				log.Warnf("Error parsing CIDR %v", ipv4)
				continue
			}
			var addrB [16]byte
			copy(addrB[:], addr.To16()[:16])
			m.workloadEndpoints[addrB] = &wlEpKey
			m.workloadEndpointsReverse[wlEpKey] = &addrB
		}
		for _, ipv6 := range msg.Endpoint.Ipv6Nets {
			addr, _, err := net.ParseCIDR(ipv6)
			if err != nil {
				log.Warnf("Error parsing CIDR %v", ipv6)
				continue
			}
			var addrB [16]byte
			copy(addrB[:], addr.To16()[:16])
			m.workloadEndpoints[addrB] = &wlEpKey
			m.workloadEndpointsReverse[wlEpKey] = &addrB
		}
		m.epMutex.Unlock()
	case *proto.WorkloadEndpointRemove:
		// We omit setting Hostname since at this point it is implied that the
		// endpoint belongs to this host.
		wlEpKey := model.WorkloadEndpointKey{
			OrchestratorID: msg.Id.OrchestratorId,
			WorkloadID:     msg.Id.WorkloadId,
			EndpointID:     msg.Id.EndpointId,
		}
		m.epMutex.Lock()
		epIp := m.workloadEndpointsReverse[wlEpKey]
		if epIp != nil {
			delete(m.workloadEndpoints, *epIp)
			delete(m.workloadEndpointsReverse, wlEpKey)
			delete(m.workloadEndpointTiers, wlEpKey)
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
		// We omit setting Hostname since at this point it is implied that the
		// endpoint belongs to this host.
		hostEpKey := model.HostEndpointKey{
			EndpointID: msg.Id.EndpointId,
		}
		m.hostEpMutex.Lock()
		// Store tiers and policies
		m.hostEndpointTiers[hostEpKey] = msg.Endpoint.Tiers
		m.hostEndpointUntrackedTiers[hostEpKey] = msg.Endpoint.UntrackedTiers
		// Store IP addresses
		for _, ipv4 := range msg.Endpoint.ExpectedIpv4Addrs {
			addr := net.ParseIP(ipv4)
			if addr == nil {
				log.Warn("Error parsing IP ", ipv4)
				continue
			}
			var addrB [16]byte
			copy(addrB[:], addr.To16()[:16])
			m.hostEndpoints[addrB] = &hostEpKey
			m.hostEndpointsReverse[hostEpKey] = &addrB
		}
		for _, ipv6 := range msg.Endpoint.ExpectedIpv6Addrs {
			addr := net.ParseIP(ipv6)
			if addr == nil {
				log.Warn("Error parsing IP ", ipv6)
				continue
			}
			var addrB [16]byte
			copy(addrB[:], addr.To16()[:16])
			m.hostEndpoints[addrB] = &hostEpKey
			m.hostEndpointsReverse[hostEpKey] = &addrB
		}
		m.hostEpMutex.Unlock()
	case *proto.HostEndpointRemove:
		// We omit setting Hostname since at this point it is implied that the
		// endpoint belongs to this host.
		hostEpKey := model.HostEndpointKey{
			EndpointID: msg.Id.EndpointId,
		}
		m.epMutex.Lock()
		epIp := m.hostEndpointsReverse[hostEpKey]
		if epIp != nil {
			delete(m.hostEndpoints, *epIp)
			delete(m.hostEndpointsReverse, hostEpKey)
			delete(m.hostEndpointTiers, hostEpKey)
			delete(m.hostEndpointUntrackedTiers, hostEpKey)
		}
		m.epMutex.Unlock()
	}
}

func (m *LookupManager) CompleteDeferredWork() error {
	return nil
}

func (m *LookupManager) addNFLogPrefixEntry(prefix string, ruleIDs *RuleID) {
	var bph [64]byte
	copy(bph[:], []byte(prefix[:]))
	m.nflogMutex.Lock()
	defer m.nflogMutex.Unlock()
	m.nflogPrefixHash[bph] = ruleIDs
}

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
			&RuleID{
				Tier:      msg.Id.Tier,
				Name:      "",
				Namespace: "",
				Direction: rules.RuleDirIngress,
				Index:     0,
				IndexStr:  "0",
				Action:    rules.RuleActionDeny,
			},
		)
		m.addNFLogPrefixEntry(
			rules.CalculateNoMatchPolicyNFLOGPrefixStr(rules.RuleDirEgress, msg.Id.Tier),
			&RuleID{
				Tier:      msg.Id.Tier,
				Name:      "",
				Namespace: "",
				Direction: rules.RuleDirEgress,
				Index:     0,
				IndexStr:  "0",
				Action:    rules.RuleActionDeny,
			},
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
			&RuleID{
				Tier:      tier,
				Name:      name,
				Namespace: namespace,
				Direction: rules.RuleDirIngress,
				Index:     ii,
				IndexStr:  strconv.Itoa(ii),
				Action:    action,
			},
		)
		newPrefixes.Add(prefix)
	}
	for ii, rule := range egress {
		action := convertAction(rule.Action)
		prefix := rules.CalculateNFLOGPrefixStr(action, owner, rules.RuleDirEgress, ii, v1Name)
		m.addNFLogPrefixEntry(
			prefix,
			&RuleID{
				Tier:      tier,
				Name:      name,
				Namespace: namespace,
				Direction: rules.RuleDirEgress,
				Index:     ii,
				IndexStr:  strconv.Itoa(ii),
				Action:    action,
			},
		)
		newPrefixes.Add(prefix)
	}

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

func (m *LookupManager) deleteRulesNFLOGPrefixes(oldPrefixes set.Set) {
	if oldPrefixes != nil {
		oldPrefixes.Iter(func(item interface{}) error {
			m.deleteNFLogPrefixEntry(item.(string))
			return nil
		})
	}
}

// GetEndpointKey returns either a *model.WorkloadEndpointKey or *model.HostEndpointKey
// or nil if addr is a Workload Endpoint or a HostEndpoint or if we don't have any
// idea about it.
func (m *LookupManager) GetEndpointKey(addr [16]byte) (interface{}, error) {
	m.epMutex.RLock()
	// There's no need to copy the result because we never modify fields,
	// only delete or replace.
	epKey := m.workloadEndpoints[addr]
	m.epMutex.RUnlock()
	if epKey != nil {
		return epKey, nil
	}
	m.hostEpMutex.RLock()
	hostEpKey := m.hostEndpoints[addr]
	m.hostEpMutex.RUnlock()
	if hostEpKey != nil {
		return hostEpKey, nil
	}
	return nil, UnknownEndpointError
}

// GetTierIndex returns the number of tiers that have been traversed before reaching a given Tier.
// For a profile, this means it returns the total number of tiers that apply.
// epKey is either a *model.WorkloadEndpointKey or *model.HostEndpointKey
//TODO: RLB: Do we really need to keep track of EP vs. Tier indexes?  Seems an overkill - we only need
// to know the overall tier order to determine the order of the NFLOGs in a set of traces.
//
// Returns -1 if unable to determine the tier index.
func (m *LookupManager) GetTierIndex(epKey interface{}, tierName string) int {
	switch epKey.(type) {
	case *model.WorkloadEndpointKey:
		ek := epKey.(*model.WorkloadEndpointKey)
		m.epMutex.RLock()
		tiers := m.workloadEndpointTiers[*ek]
		m.epMutex.RUnlock()
		if tierName == "" {
			// Finesse the profile case (tier is blank).
			return len(tiers)
		}
		for i, tier := range tiers {
			if tier.Name == tierName {
				return i
			}
		}
	case *model.HostEndpointKey:
		ek := epKey.(*model.HostEndpointKey)
		m.hostEpMutex.RLock()
		untrackedTiers := m.hostEndpointUntrackedTiers[*ek]
		tiers := m.hostEndpointTiers[*ek]
		m.hostEpMutex.RUnlock()
		if tierName == "" {
			// Finesse the profile case (tier is blank).
			return len(untrackedTiers) + len(tiers)
		}
		for i, tier := range untrackedTiers {
			if tier.Name == tierName {
				return i
			}
		}
		for i, tier := range tiers {
			if tier.Name == tierName {
				return len(untrackedTiers) + i
			}
		}
	}
	return -1
}

func (m *LookupManager) GetRuleIDsFromNFLOGPrefix(prefix [64]byte) *RuleID {
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

func (m *LookupManager) Dump() string {
	lines := []string{}
	for p, r := range m.nflogPrefixHash {
		lines = append(lines, string(p[:])+": "+r.String())
	}
	return strings.Join(lines, "\n")
}
