package policycalc

import (
	"github.com/projectcalico/libcalico-go/lib/net"
)

// ------
// This file contains all of the struct definitions that are used as input when calculating the action for flow.
// ------

type Action string

const (
	ActionIndeterminate Action = "unknown"
	ActionAllow         Action = "allow"
	ActionDeny          Action = "deny"
)

type EndpointType string

const (
	EndpointTypeWep EndpointType = "wep"
	EndpointTypeHep EndpointType = "hep"
	EndpointTypeNs  EndpointType = "ns"
	EndpointTypeNet EndpointType = "net"
)

type ReporterType string

const (
	ReporterSrc ReporterType = "src"
	ReporterDst ReporterType = "dst"
)

type Flow struct {
	Source      FlowEndpointData
	Destination FlowEndpointData
	Reporter    ReporterType
	Action      Action
	Policies    []FlowPolicy
	Proto       *uint8
	IPVersion   *int
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
