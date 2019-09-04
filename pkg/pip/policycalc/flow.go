package policycalc

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/net"
)

// ------
// This file contains all of the struct definitions that are used as input when calculating the action for flow.
// ------

const (
	ActionInvalid       = ""
	ActionUnknown       = "unknown"
	ActionAllow         = "allow"
	ActionDeny          = "deny"
	ActionEndOfTierDeny = "eot-deny"
	ActionNextTier      = "pass"
)

type EndpointType string

const (
	EndpointTypeInvalid EndpointType = ""
	EndpointTypeWep     EndpointType = "wep"
	EndpointTypeHep     EndpointType = "hep"
	EndpointTypeNs      EndpointType = "ns"
	EndpointTypeNet     EndpointType = "net"
)

type ReporterType string

const (
	ReporterTypeInvalid     ReporterType = ""
	ReporterTypeSource      ReporterType = "src"
	ReporterTypeDestination ReporterType = "dst"
)

// flowCache contains cached data when calculating the before/after impact of the policies on a flow.
type flowCache struct {
	// Cached source and destination caches.
	source      endpointCache
	destination endpointCache

	// Cached policy actions. Populated by the before flow calculation and used by the after policy calculation to
	// speed up processing and to assist with unknown rule matches.
	policies map[string]ActionFlag
}

type endpointCache struct {
	selectors []MatchType
}

type Flow struct {
	// Reporter
	Reporter ReporterType

	// Source endpoint data for the flow.
	Source FlowEndpointData

	// Destination endpoint data for the flow.
	Destination FlowEndpointData

	// Original action for the flow.
	ActionFlag ActionFlag

	// The protocol of the flow. Nil if unknown.
	Proto *uint8

	// The IP version of the flow. Nil if unknown.
	IPVersion *int

	// Enforced policies that were applied to the endpoint.
	// For policy matches:
	// -  <matchIdx>|<tierName>|<namespaceName>/<policyName>|<action>
	// -  <matchIdx>|<tierName>|<policyName>|<action>
	//
	// For end of tier implicit drop (where policy is the last matching policy that did not match the rule):
	// -  <matchIdx>|<tierName>|<namespaceName>/<policyName>|deny
	// -  <matchIdx>|<tierName>|<policyName>|deny
	//
	// End of tiers allow for Pods (in Kubernetes):
	// -  <matchIdx>|__PROFILE__|__PROFILE__.kns.<namespaceName>|allow
	//
	// End of tiers drop for HostEndpoints:
	// -  <matchIdx>|__PROFILE__|__PROFILE__.__NO_MATCH__|deny
	Policies []PolicyHit
}

// FlowEndpointData can be used to describe the source or destination
// of a flow log.
type FlowEndpointData struct {
	// Endpoint type.
	Type EndpointType

	// Name.
	Name string

	// Namespace - should only be set for namespaces endpoints.
	Namespace string

	// Labels - only relevant for Calico endpoints. If not specified on input, this may be filled in by an endpoint
	// cache lookup.
	Labels map[string]string

	// IP, or nil if unknown.
	IP *net.IP

	// Port, or nil if unknown.
	Port *uint16

	// ServiceAccount, or nil if unknown. If not specified on input (nil), this may be filled in by an endpoint cache
	// lookup.
	ServiceAccount *string

	// NamedPorts is the set of named ports for this endpoint.  If not specified on input (nil), this may be filled in
	// by an endpoint cache lookup.
	NamedPorts []EndpointNamedPort
}

// IsCalicoManagedEndpoint returns if the endpoint is managed by Calico.
func (e *FlowEndpointData) IsCalicoManagedEndpoint() bool {
	switch e.Type {
	// Only HEPs and WEPs are calico-managed endpoints.  NetworkSets are handled by Calico, but are not endpoints in
	// the sense that policy is not applied directly to them.
	case EndpointTypeHep, EndpointTypeWep:
		return true
	default:
		return false
	}
}

// Implement the label Get method for use with the selector processing. This allows us to inject additional labels
// without having to update the dictionary.
func (e *FlowEndpointData) Get(labelName string) (value string, present bool) {
	switch labelName {
	case v3.LabelNamespace:
		return e.Namespace, e.Namespace != ""
	case v3.LabelOrchestrator:
		return v3.OrchestratorKubernetes, e.Namespace != ""
	default:
		if e.Labels != nil {
			val, ok := e.Labels[labelName]
			return val, ok
		}
	}
	return "", false
}

// EndpointNamedPort encapsulates details about a named port on an endpoint.
type EndpointNamedPort struct {
	Name     string
	Protocol uint8
	Port     uint16
}

// PolicyHitKey identifies a policy.
type PolicyHit struct {
	// The tier name (or __PROFILE__ for profile match)
	Tier string

	// The policy name. This will include the "staged:" indicator for staged policies.
	Name string

	// Whether this is a staged policy.
	Staged bool

	// The action flag(s) for this policy hit.
	Action ActionFlag

	// The match index for this hit.
	MatchIndex int
}

// ToEnforcedFlowLogPolicyString converts a PolicyHit to a flow log policy string.
func (p PolicyHit) ToEnforcedFlowLogPolicyStrings() []string {
	// Calculate the set of action strings for this policy.
	actions := p.Action.ToActionStrings()

	// Tweak the action strings to the policy hit string required for the flow log.
	for i := range actions {
		actions[i] = fmt.Sprintf("%d|%s|%s|%s", p.MatchIndex, p.Tier, p.Name, actions[i])
	}
	return actions
}

// PolicyHitFromFlowLogPolicyString creates a PolicyHit from a flow log policy string.
func PolicyHitFromFlowLogPolicyString(n string) (PolicyHit, bool) {
	p := PolicyHit{}

	parts := strings.Split(n, "|")
	if len(parts) != 4 {
		return p, false
	}

	// Extract match index.
	var err error
	p.MatchIndex, err = strconv.Atoi(parts[0])
	if err != nil {
		return p, false
	}

	// Extract tier
	p.Tier = parts[1]

	// Extract name and check if it's staged.
	p.Name = parts[2]
	p.Staged = strings.Contains(p.Name, model.PolicyNamePrefixStaged)

	// Extract action.
	p.Action = ActionFlagFromString(parts[3])
	return p, p.Action != 0
}

// SortablePolicyHits is a sortable slice of PolicyHits.
type SortablePolicyHits []PolicyHit

func (s SortablePolicyHits) Len() int { return len(s) }

func (s SortablePolicyHits) Less(i, j int) bool {
	if s[i].MatchIndex != s[j].MatchIndex {
		return s[i].MatchIndex < s[j].MatchIndex
	}
	if s[i].Name != s[j].Name {
		return s[i].Name < s[j].Name
	}
	if s[i].Action != s[j].Action {
		return s[i].Action < s[j].Action
	}
	return s[i].Staged && !s[j].Staged
}

func (s SortablePolicyHits) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// SortAndRenumber sorts the PolicyHit slice and renumbers to be monotonically increasing.
func (s SortablePolicyHits) SortAndRenumber() {
	sort.Sort(s)
	for i := range s {
		s[i].MatchIndex = i
	}
}

// PolicyHitsEqual compares two sets of PolicyHits to see if both order and values are identical.
func PolicyHitsEqual(p1, p2 []PolicyHit) bool {
	if len(p1) != len(p2) {
		return false
	}

	for i := range p1 {
		if p1[i] != p2[i] {
			return false
		}
	}

	return true
}
