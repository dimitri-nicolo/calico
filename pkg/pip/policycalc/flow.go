package policycalc

import (
	"github.com/projectcalico/libcalico-go/lib/net"
)

// ------
// This file contains all of the struct definitions that are used as input when calculating the action for flow.
// ------

type Action byte

const (
	ActionInvalid Action = iota
	ActionUnknown
	ActionAllow
	ActionDeny
)

func (a Action) String() string {
	switch a {
	case ActionUnknown:
		return "unknown"
	case ActionAllow:
		return "allow"
	case ActionDeny:
		return "deny"
	default:
		return "-"
	}
}

type EndpointType byte

const (
	EndpointTypeInvalid EndpointType = iota
	EndpointTypeWep
	EndpointTypeHep
	EndpointTypeNs
	EndpointTypeNet
)

func (e EndpointType) String() string {
	switch e {
	case EndpointTypeWep:
		return "wep"
	case EndpointTypeHep:
		return "hep"
	case EndpointTypeNs:
		return "ns"
	case EndpointTypeNet:
		return "net"
	default:
		return "-"
	}
}

type ReporterType byte

const (
	ReporterTypeInvalid ReporterType = iota
	ReporterTypeSource
	ReporterTypeDestination
)

func (r ReporterType) String() string {
	switch r {
	case ReporterTypeSource:
		return "src"
	case ReporterTypeDestination:
		return "dst"
	default:
		return "-"
	}
}

type Flow struct {
	// Reporter
	Reporter ReporterType

	// Source endpoint data for the flow.
	Source FlowEndpointData

	// Destination endpoint data for the flow.
	Destination FlowEndpointData

	// Original action for the flow.
	Action Action

	// The protocol of the flow. Nil if unknown.
	Proto *uint8

	// The IP version of the flow. Nil if unknown.
	IPVersion *int

	// The set of policies applied to this endpoint.
	Policies []string
}

// getUnchangedResponse returns a policy calculation Response based on the original flow data.
func (f Flow) getUnchangedResponse() *Response {
	r := &Response{}
	if f.Reporter == ReporterTypeSource {
		r.Source.Action = f.Action
		r.Source.Policies = f.Policies
	} else {
		r.Source.Action = ActionAllow
		r.Destination.Action = f.Action
		r.Destination.Policies = f.Policies
	}
	return r
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

	// Labels - only relevant for Calico endpoints.
	Labels map[string]string

	// IP, or nil if unknown.
	IP *net.IP

	// Port, or nil if unknown.
	Port *uint16

	// ServiceAccount, or nil if unknown.
	ServiceAccount *string

	// Private cache data.
	cachedSelectorResults []MatchType
}

// isCalicoEndpoint returns if the endpoint is managed by Calico.
func (e *FlowEndpointData) isCalicoEndpoint() bool {
	switch e.Type {
	// Only HEPs and WEPs are calico-managed endpoints.  NetworkSets are handled by Calico, but are not endpoints in
	// the sense that policy is not applied directly to them.
	case EndpointTypeHep, EndpointTypeWep:
		return true
	default:
		return false
	}
}
