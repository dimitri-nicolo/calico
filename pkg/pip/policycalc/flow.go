package policycalc

import (
	"github.com/projectcalico/libcalico-go/lib/net"
)

// ------
// This file contains all of the struct definitions that are used as input when calculating the action for flow.
// ------

type Action byte

const (
	ActionUnknown Action = iota
	ActionIndeterminate
	ActionAllow
	ActionDeny
)

func (a Action) String() string {
	switch a {
	case ActionIndeterminate:
		return "Indeterminate"
	case ActionAllow:
		return "Allow"
	case ActionDeny:
		return "Deny"
	default:
		return "-"
	}
}

type EndpointType byte

const (
	EndpointTypeUnknown EndpointType = iota
	EndpointTypeWep
	EndpointTypeHep
	EndpointTypeNs
	EndpointTypeNet
)

func (e EndpointType) String() string {
	switch e {
	case EndpointTypeWep:
		return "WorkloadEndpoint/Pod"
	case EndpointTypeHep:
		return "HostEndpoint"
	case EndpointTypeNs:
		return "NetworkSet"
	case EndpointTypeNet:
		return "Network"
	default:
		return "-"
	}
}

type Flow struct {
	// Source endpoint data for the flow.
	Source FlowEndpointData

	// Destination endpoint data for the flow.
	Destination FlowEndpointData

	// Original action for the flow.
	Action Action

	// The policies originally applied to the flow.
	Policies []FlowPolicy

	// The protocol of the flow. Nil if unknown.
	Proto *uint8

	// The IP version of the flow. Nil if unknown.
	IPVersion *int
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
	case EndpointTypeHep, EndpointTypeWep, EndpointTypeNs:
		return true
	default:
		return false
	}
}

type FlowPolicy struct {
	Order  int64
	Tier   string
	Name   string
	Action string
}
