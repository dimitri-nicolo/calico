// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package api

import (
	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/net"
)

const (
	FlowLogGlobalNamespace        = "-"
	FlowLogEndpointTypeWEP        = "wep"
	FlowLogEndpointTypeHEP        = "hep"
	FlowLogEndpointTypeNetworkSet = "ns"
	FlowLogEndpointTypeNetwork    = "net"
	FlowLogNetworkPublic          = "pub"
	FlowLogNetworkPrivate         = "pvt"
)

// Container type to hold the EndpointsReportFlow and/or an error.
type FlowLogResult struct {
	*apiv3.EndpointsReportFlow
	Err error
}

// TODO (ML): The following are all the struct definitions that are used
// as input when calculating actions for a flow in PIP and in Policy
// Recommendation. The es-proxy-image repo should eventually be
// refactored to use these definitions.
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

// GetUnchangedResponse returns a policy calculation Response based on the original flow data.
func (f Flow) GetUnchangedResponse() *Response {
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
}

// IsCalicoEndpoint returns if the endpoint is managed by Calico.
func (e *FlowEndpointData) IsCalicoEndpoint() bool {
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
	case apiv3.LabelNamespace:
		return e.Namespace, e.Namespace != ""
	case apiv3.LabelOrchestrator:
		return apiv3.OrchestratorKubernetes, e.Namespace != ""
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

// TODO (ML): The following are structs for representing the policy calculation
// response for PIP. These should be moved into the same package when more of the
// PIP structs are moved over.
type Response struct {
	// The calculated response for the source endpoint.
	Source EndpointResponse

	// The calculated response for the destination endpoint.
	Destination EndpointResponse
}

type EndpointResponse struct {
	// Whether to include the result in the final aggregated data set. For Calico->Calico endpoint flows we may need to
	// massage the data a little:
	// - For source-reported flows whose action changes from denied to allowed or unknown, we explicitly add the
	//   equivalent data at the destination, since the associated flow data should be missing from the original set.
	// - For destination-reported flows whose source action changes from allowed->denied, we remove the flow completely
	//   as it should not get reported.
	// This means the calculation response can have 0, 1 or 2 results to include in the aggregated data.
	Include bool

	// The calculated action at the endpoint for the supplied flow.
	Action Action

	// The set of policies applied to this flow. The format of each entry is as follows.
	// For policy matches:
	// -  <tierIdx>|<tierName>|<namespaceName>/<policyName>|<action>
	// -  <tierIdx>|<tierName>|<policyName>|<action>
	//
	// For end of tier implicit drop (where policy is the last matching policy that did not match the rule):
	// -  <tierIdx>|<tierName>|<namespaceName>/<policyName>|deny
	// -  <tierIdx>|<tierName>|<policyName>|deny
	//
	// End of tiers allow for Pods (in Kubernetes):
	// -  <tierIdx>|__PROFILE__|__PROFILE__.kns.<namespaceName>|allow
	//
	// End of tiers drop for HostEndpoints:
	// -  <tierIdx>|__PROFILE__|__PROFILE__.__NO_MACH__|deny
	Policies []string
}
