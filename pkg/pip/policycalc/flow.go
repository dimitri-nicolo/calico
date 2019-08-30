package policycalc

import (
	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/net"
)

// ------
// This file contains all of the struct definitions that are used as input when calculating the action for flow.
// ------

type Action string

const (
	ActionInvalid  Action = ""
	ActionUnknown  Action = "unknown"
	ActionAllow    Action = "allow"
	ActionDeny     Action = "deny"
	ActionNextTier Action = "pass"
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

	// Policies is the set of policies applied to the flow. This is used assist with uncertain calculations.
	Policies []string
}

// getUnchangedResponse returns a policy calculation Response based on the original flow data.
func (f Flow) getUnchangedResponse() *Response {
	r := &Response{}
	if f.Reporter == ReporterTypeSource {
		r.Source.Include = true
		r.Source.Action = f.Action
		r.Source.Policies = nil
	} else {
		r.Source.Action = ActionAllow
		r.Destination.Include = true
		r.Destination.Action = f.Action
		r.Destination.Policies = nil
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

	// ---- Private data ----

	// Selector cache results
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
