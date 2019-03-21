// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/apis/audit"
)

// ReportData contains the aggregated data available for rendering in report templates. The data available is dependent
// on the selector/report configuration.
//
// The data is stored directly in elastic search. To reduce nesting and simplify indexing, all summary values are
// contained at the top level.
type ReportData struct {
	ReportName string      `json:"reportName"`
	ReportSpec ReportSpec  `json:"reportSpec"`
	StartTime  metav1.Time `json:"startTime"`
	EndTime    metav1.Time `json:"endTime"`

	// The total number of in-scope endpoints.
	//
	// Source: Calculated from pod/wep, hep, namespace and service account labels.
	EndpointsNumTotal int `json:"endpointsNumTotal,omitempty"`

	// The number of in-scope endpoints that were ingress protected during the reporting interval.
	// See below for defn of ingress-protected.
	EndpointsNumIngressProtected int `json:"endpointsNumIngressProtected,omitempty"`

	// The number of in-scope endpoints that were egress protected during the reporting interval.
	// See below for defn of egress-protected.
	EndpointsNumEgressProtected int `json:"endpointsNumEgressProtected,omitempty"`

	// The number of inscope endpoints whose policy would allow ingress traffic from the internet for *any* period within the
	// reporting interval.
	// See below for how this is calculated for an endpoint.
	EndpointsNumIngressFromInternet int `json:"endpointsNumIngressFromInternet,omitempty"`

	// The number of inscope endpoints whose policy would allow egress traffic to the internet for *any* period within the
	// reporting interval.
	// See below for how this is calculated for an endpoint.
	EndpointsNumEgressToInternet int `json:"endpointsNumEgressToInternet,omitempty"`

	// The number of inscope endpoints whose policy would allow ingress traffic from a different namespace for *any* period
	// within the reporting interval.
	// See below for how this is calculated for an endpoint.
	EndpointsNumIngressFromOtherNamespace int `json:"endpointsNumIngressFromOtherNamespace,omitempty"`

	// The number of inscope endpoints whose policy would allow ingress traffic from a different namespace for *any* period
	// within the reporting interval.
	// See below for how this is calculated for an endpoint.
	EndpointsNumEgressToOtherNamespace int `json:"endpointsNumEgressToOtherNamespace,omitempty"`

	// The number of in-scope endpoints that were envoy-enabled within the reporting interval (see below for defn of
	// envoy-enabled)
	EndpointsNumEnvoyEnabled int `json:"endpointsNumEnvoyEnabled,omitempty"`

	// The set of in-scope endpoints.
	Endpoints []EndpointsReportEndpoint `json:"endpoints,omitempty"`

	// The total number of namespaces containing in-scope endpoints.
	//
	// Source: Calculated from pod/wep, hep, namespace and service account labels.
	NamespacesNumTotal int `json:"namespacesNumTotal,omitempty"`

	// The number of namespaces whose in-scope endpoints were ingress protected during the reporting interval.
	NamespacesNumIngressProtected int `json:"namespacesNumIngressProtected,omitempty"`

	// The number of namespaces whose in-scope endpoints were egress protected during the reporting interval.
	NamespacesNumEgressProtected int `json:"namespacesNumEgressProtected,omitempty"`

	// The number of namespaces that contained in-scope endpoints that would allow ingress traffic from the internet for
	// *any* period within the reporting interval.
	NamespacesNumIngressFromInternet int `json:"namespacesNumIngressFromInternet,omitempty"`

	// The number of namespaces that contained in-scope endpoints that would allow egress traffic to the internet for
	// *any* period within the reporting interval.
	NamespacesNumEgressToInternet int `json:"namespacesNumEgressToInternet,omitempty"`

	// The number of namespaces that contained in-scope endpoints that would allow ingress traffic from another
	// namespace for *any* period within the reporting interval.
	NamespacesNumIngressFromOtherNamespace int `json:"namespacesNumIngressFromOtherNamespace,omitempty"`

	// The number of namespaces that contained in-scope endpoints that would allow egress traffic to another
	// namespace for *any* period within the reporting interval.
	NamespacesNumEgressToOtherNamespace int `json:"namespacesNumEgressToOtherNamespace,omitempty"`

	// The number of namespaces whose in-scope endpoints were always Envoy-enabled
	NamespacesNumEnvoyEnabled int `json:"namespacesNumEnvoyEnabled,omitempty"`

	// The set of namespaces containing in-scope endpoints.
	Namespaces []EndpointsReportNamespace `json:"namespaces,omitempty"`

	// The total number of services containing in-scope endpoints.
	//
	// Source: Calculated from pod/wep, hep, service and service account labels.
	ServicesNumTotal int `json:"servicesNumTotal,omitempty"`

	// The number of services whose in-scope endpoints were ingress protected during the reporting interval.
	ServicesNumIngressProtected int `json:"servicesNumIngressProtected,omitempty"`

	// The number of services that contained in-scope endpoints that would allow ingress traffic from the internet for
	// *any* period within the reporting interval.
	ServicesNumIngressFromInternet int `json:"servicesNumIngressFromInternet,omitempty"`

	// The number of services that contained in-scope endpoints that would allow ingress traffic from another
	// namespace for *any* period within the reporting interval.
	ServicesNumIngressFromOtherNamespace int `json:"servicesNumIngressFromOtherNamespace,omitempty"`

	// The number of services whose in-scope endpoints were always Envoy-enabled
	ServicesNumEnvoyEnabled int `json:"servicesNumEnvoyEnabled,omitempty"`

	// The set of services containing in-scope endpoints.
	Services []EndpointsReportService `json:"services,omitempty"`

	// The total number of in-scope audit logs.
	AuditNumTotal int `json:"auditNumTotal,omitempty"`

	// The number of in-scope audit log create events.
	AuditNumCreate int `json:"auditNumCreate,omitempty"`

	// The number of in-scope audit log patch or replace events.
	AuditNumModified int `json:"auditNumModified,omitempty"`

	// The number of in-scope audit log delete events.
	AuditNumDelete int `json:"auditNumDelete,omitempty"`

	// The time-ordered set of in-scope audit events that occurred within the reporting interval.
	AuditEvents []audit.Event `json:"auditEvents,omitempty"`
}

type EndpointsReportEndpoint struct {
	ID ResourceID `json:"id,omitempty"`

	// Whether ingress traffic to this endpoint was always protected during the reporting interval.
	//
	// Ingress protection is defined as denying ingress traffic unless explicitly whitelisted. This is translated as
	// the endpoint having some explicit ingress policy applied to it.
	//
	// Source: Calculated from the set of ingress policies that apply to each endpoint.
	//
	// Set to:
	// - false if there are no ingress policies applied to the endpoint at any point during the reporting interval.
	// - true otherwise.
	//
	// Note: Policy is not inspected for protection bypass: for example match-all-and-allow rules which would effectively
	//       short-circuit the default tier-drop behavior, in this case the match-all-and-allow would be considered to be
	//       an explicit whitelist of all traffic. We could include simplistic all-match rules and check that they
	//       don't result in an allow. To check for more circuitous match-all allows is much trickier (e.g. you have one
	//       rule that allows for src!=1.2.3.0/24 and another rule that allows for src==1.2.3.0/24, which combined
	//       is essentially an allow-all).
	IngressProtected bool `json:"ingressProtected,omitempty"`

	// Whether egress traffic to this endpoint was always protected during the reporting interval.
	//
	// Egress protection is defined as denying egress traffic unless explicitly whitelisted. This is translated as
	// the endpoint having some explicit egress policy applied to it.
	//
	// Source: Calculated from the set of egress policies that apply to each endpoint.
	//
	// Set to:
	// - false if there are no egress policies applied to the endpoint at any point during the reporting interval.
	// - true otherwise.
	//
	// Note: Policy is not inspected for protection bypass: for example match-all-and-allow rules which would effectively
	//       short-circuit the default tier-drop behavior, in this case the match-all-and-allow would be considered to be
	//       an explicit whitelist of all traffic. We could include simplistic all-match rules and check that they
	//       don't result in an allow. To check for more circuitous match-all allows is much trickier (e.g. you have one
	//       rule that allows for src!=1.2.3.0/24 and another rule that allows for src==1.2.3.0/24, which combined
	//       is essentially an allow-all). Similarly, policy that only contains pass rules would still count as being
	//       protected.
	EgressProtected bool `json:"egressProtected,omitempty"`

	// Whether the matching policy has any ingress allow rules from a public IP address (as defined by the complement of
	// the private addresses; private addresses default to those defined in RFC 1918, but may also be configured separately).
	//
	// Source: Calculated from the policies applied to the endpoint. The ingress allow rules in each policy are checked
	//         to determine if any CIDR specified in the rule, either directly or through a matching network set, is an
	//         internet address. Endpoint addresses are not included - therefore ingress from a pod that has a public
	//         IP address will not be considered as “from internet”.
	//
	// Note: This is a simplification since it does not examine the policies to determine if it's actually possible to
	//       hit one of these allow rules (e.g. a previous rule may be a match-all-deny).
	IngressFromInternet bool `json:"ingressFromInternet,omitempty"`

	// Whether the matching policy has any egress allow rules to a public IP address (as defined by the complement of
	// the private addresses; private addresses default to those defined in RFC 1918, but may also be configured separately).
	//
	// Source: Calculated from the policies applied to the endpoint. The egress allow rules in each policy are checked
	//         to determine if any CIDR specified in the rule, either directly or through a matching network set, is an
	//         internet address. Endpoint addresses are not included - therefore egress to a pod that has a public
	//         IP address will not be considered as “to internet”.
	//
	// Note 1: This is a simplification since it does not examine the policies to determine if it's actually possible to
	//         hit one of these allow rules (e.g. a previous rule may be a match-all-deny).
	EgressToInternet bool `json:"egressToInternet,omitempty"`

	// Whether the matching policy has any ingress allow rules from another namespace.
	//
	// Source: Calculated from the policies applied to the endpoint.
	//
	// Set to true if:
	// - this is a pod (i.e. namespaced) with an applied GlobalNetworkPolicy with an ingress allow rule with no CIDR match.
	// - this is a pod with an applied NetworkPolicy with an ingress allow rule with a non-empty NamespaceSelector.
	//
	// Note: This is a simplification since it does not examine the policies to determine if it's actually possible to
	//       hit one of these allow rules (e.g. a previous rule may be a match-all-deny, or endpoint selector may not
	//       match any endpoints within the namespace).
	IngressFromOtherNamespace bool `json:"ingressFromOtherNamespace,omitempty"`

	// Whether the matching policy has any egress allow rules to another namespace.
	//
	// Source: Calculated from the policies applied to the endpoint.
	//
	// Set to true if:
	// - this is a pod endpoint (i.e. namespaced) matches a GlobalNetworkPolicy with an egress allow rule with no CIDR match.
	// - this is a pod endpoint which matches a NetworkPolicy with an egress allow rule with a non-empty NamespaceSelector.
	//
	// Note: This is a simplification since it does not examine the policies to determine if it's actually possible to
	//       hit one of these allow rules (e.g. a previous rule may be a match-all-deny, or endpoint selector may not
	//       match any endpoints within the namespace).
	EgressToOtherNamespace bool `json:"egressToOtherNamespace,omitempty"`

	// Whether this pod is envoy-enabled. This is simply an indicator of whether an Envoy container is running within the pod.
	// Provided Istio is configured appropriately, this can provide a simplistic determination of whether the pod is mTLS
	// enabled.
	//
	// Source: Pod spec.
	//
	// Set to:
	// - true if envoy is running within the pod
	// - false if envoy is not running within the pod
	EnvoyEnabled bool `json:"envoyEnabled,omitempty"`

	// The set of policies that apply to an endpoint may change within the reporting interval, this is the superset of all
	// policies that applied to the endpoint during that interval.
	AppliedPolicies []ResourceID `json:"appliedPolicies,omitempty"`

	// The list of services that exposed this endpoint at any moment during the reporting interval.
	//
	// Source: Determined from the Kubernetes endpoints resource associated with the service.
	Services []ResourceID `json:"services,omitempty"`

	// The list of all endpoints that have been generating traffic to this endpoint. This list includes endpoints that are
	// not necessarily in-scope.
	//
	// Source: Measured from flow flogs.
	EndpointsGeneratingTrafficToThisEndpoint []EndpointsReportEndpointFlow `json:"endpointsGeneratingTrafficToThisEndpoint,omitempty"`

	// The list of endpoints that have been receiving traffic from this endpoint.  This list includes endpoints that are
	// not necessarily in-scope.
	//
	// Source: Measured from flow flogs.
	EndpointsReceivingTrafficFromThisEndpoint []EndpointsReportEndpointFlow `json:"endpointsReceivingTrafficFromThisEndpoint,omitempty"`
}

type EndpointsReportEndpointFlow struct {
	Endpoint ResourceID       `json:"endpoint,omitempty"`
	Allowed  EndpointFlowData `json:"allowed,omitempty"`
	Denied   EndpointFlowData `json:"denied,omitempty"`
}

type EndpointFlowData struct {
	Bytes               int `json:"bytes,omitempty"`
	Packets             int `json:"packets,omitempty"`
	HTTPRequestsAllowed int `json:"httpRequestsAllowed,omitempty"`
	HTTPRequestsDenied  int `json:"httpRequestsDenied,omitempty"`
}

type EndpointsReportNamespace struct {
	Namespace ResourceID `json:"namespace,omitempty"`

	// Whether ingress traffic was protected for all endpoints within this namespace within the reporting interval.
	// This is a summary of information contained in the endpoints data.
	IngressProtected bool `json:"ingressProtected,omitempty"`

	// Whether egress traffic was protected for all endpoints within this namespace within the reporting interval.
	// This is a summary of information contained in the endpoints data.
	EgressProtected bool `json:"egressProtected,omitempty"`

	// Whether ingress traffic was allowed from the internet for any endpoint within this namespace within the reporting
	// interval.
	IngressFromInternet bool `json:"ingressFromInternet,omitempty"`

	// Whether ingress traffic was allowed from the internet for any endpoint within this namespace within the reporting
	// interval.
	EgressToInternet bool `json:"egressToInternet,omitempty"`

	// Whether ingress traffic was allowed from another namespace for any endpoint within this namespace within the
	// reporting interval.
	IngressFromOtherNamespace bool `json:"ingressFromOtherNamespace,omitempty"`

	// Whether ingress traffic was allowed from another namespace for any endpoint within this namespace within the
	// reporting interval.
	EgressToOtherNamespace bool `json:"egressToOtherNamespace,omitempty"`

	// Whether envoy was enabled for all endpoints within this namespace within the reporting interval.
	// This is a summary of information contained in the endpoints data.
	EnvoyEnabled bool `json:"envoyEnabled,omitempty"`
}

type EndpointsReportService struct {
	Service ResourceID `json:"service,omitempty"`

	// Whether ingress traffic was protected for all endpoints within this namespace within the reporting interval.
	// This is a summary of information contained in the endpoints data.
	IngressProtected bool `json:"ingressProtected,omitempty"`

	// Whether ingress traffic was allowed from the internet for any endpoint exposed by this service within the reporting
	// interval.
	IngressFromInternet bool `json:"ingressFromInternet,omitempty"`

	// Whether ingress traffic was allowed from another namespace for any endpoint exposed by this service within the
	// reporting interval.
	IngressFromOtherNamespace bool `json:"ingressFromOtherNamespace,omitempty"`

	// Whether envoy was enabled for all endpoints that were exposed by this service within the reporting interval.
	// This is a summary of information contained in the endpoints data.
	EnvoyEnabled bool `json:"envoyEnabled,omitempty"`
}
