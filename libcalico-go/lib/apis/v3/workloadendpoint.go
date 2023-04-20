// Copyright (c) 2017-2023 Tigera, Inc. All rights reserved.

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

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/api/pkg/lib/numorstring"
)

const (
	KindWorkloadEndpoint     = "WorkloadEndpoint"
	KindWorkloadEndpointList = "WorkloadEndpointList"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkloadEndpoint contains information about a WorkloadEndpoint resource that is a peer of a Calico
// compute node.
type WorkloadEndpoint struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the WorkloadEndpoint.
	Spec WorkloadEndpointSpec `json:"spec,omitempty"`
	// Status of the WorkloadEndpoint.
	Status WorkloadEndpointStatus `json:"status,omitempty"`
}

// WorkloadEndpointSpec contains the specification for a WorkloadEndpoint resource.
type WorkloadEndpointSpec struct {
	// The name of the orchestrator.
	Orchestrator string `json:"orchestrator,omitempty" validate:"omitempty,name"`
	// The name of the workload.
	Workload string `json:"workload,omitempty" validate:"omitempty,name"`
	// The node name identifying the Calico node instance.
	Node string `json:"node,omitempty" validate:"omitempty,name"`
	// The container ID.
	ContainerID string `json:"containerID,omitempty" validate:"omitempty,containerID"`
	// The Pod name.
	Pod string `json:"pod,omitempty" validate:"omitempty,name"`
	// The Endpoint name.
	Endpoint string `json:"endpoint,omitempty" validate:"omitempty,name"`
	// ServiceAccountName, if specified, is the name of the k8s ServiceAccount  for this pod.
	ServiceAccountName string `json:"serviceAccountName,omitempty" validate:"omitempty,name"`
	// IPNetworks is a list of subnets allocated to this endpoint. IP packets will only be
	// allowed to leave this interface if they come from an address in one of these subnets.
	// Currently only /32 for IPv4 and /128 for IPv6 networks are supported.
	IPNetworks []string `json:"ipNetworks,omitempty" validate:"omitempty,dive,net"`
	// IPNATs is a list of 1:1 NAT mappings to apply to the endpoint. Inbound connections
	// to the external IP will be forwarded to the internal IP. Connections initiated from the
	// internal IP will not have their source address changed, except when an endpoint attempts
	// to connect one of its own external IPs. Each internal IP must be associated with the same
	// endpoint via the configured IPNetworks.
	IPNATs []IPNAT `json:"ipNATs,omitempty" validate:"omitempty,dive"`
	// For workloads from AWS-backed IP pools, a list of candidate elastic IPs to bind to the
	// workload's private IP.  Felix will choose an available elastic IP from the list.
	AWSElasticIPs []string `json:"awsElasticIPs,omitempty" validate:"omitempty,dive,ipv4"`
	// IPv4Gateway is the gateway IPv4 address for traffic from the workload.
	IPv4Gateway string `json:"ipv4Gateway,omitempty" validate:"omitempty,ipv4"`
	// IPv6Gateway is the gateway IPv6 address for traffic from the workload.
	IPv6Gateway string `json:"ipv6Gateway,omitempty" validate:"omitempty,ipv6"`
	// A list of security Profile resources that apply to this endpoint. Each profile is
	// applied in the order that they appear in this list.  Profile rules are applied
	// after the selector-based security policy.
	Profiles []string `json:"profiles,omitempty" validate:"omitempty,dive,name"`
	// InterfaceName the name of the Linux interface on the host: for example, tap80.
	InterfaceName string `json:"interfaceName,omitempty" validate:"interface"`
	// MAC is the MAC address of the endpoint interface.
	MAC string `json:"mac,omitempty" validate:"omitempty,mac"`
	// Ports contains the endpoint's named ports, which may be referenced in security policy rules.
	Ports []WorkloadEndpointPort `json:"ports,omitempty" validate:"dive,omitempty"`
	// AllowSpoofedSourcePrefixes is a list of CIDRs that the endpoint should be able to send traffic from,
	// bypassing the RPF check.
	AllowSpoofedSourcePrefixes []string `json:"allowSpoofedSourcePrefixes,omitempty" validate:"omitempty,dive,cidr"`
	// Egress control.
	EgressGateway *apiv3.EgressGatewaySpec `json:"egressGateway,omitempty" validate:"omitempty"`
	// A list of names of the external networks who are associated with the endpoint for egress traffic.
	ExternalNetworkNames []string `json:"externalNetworkNames,omitempty" validate:"omitempty,dive,name"`
}

// WorkloadEndpointStatus contains the status for a WorkloadEndpoint resource.
type WorkloadEndpointStatus struct {
	// Phase is a simple, high-level summary of where the workload is in its lifecycle.
	Phase string `json:"phase,omitempty" validate:"omitempty"`
	// EgressGateway contains the status of Egress Gateways used by this WorkloadEndpoint.
	EgressGateway *EgressGatewayStatus `json:"egressGateway,omitempty" validate:"omitempty"`
}

type EgressGatewayStatus struct {
	// MaintenanceGatewayIP contains the IP address of the latest Egress Gateway pod used by
	// this workload to be starting a manintenance window.
	MaintenanceGatewayIP string `json:"maintenanceGatewayIP,omitempty" validate:"omitempty,ipv4"`
	// MaintenanceStarted contains a timestamp indicating when the latest Egress Gateway pod used by
	// this workload is starting a manintenance window.
	MaintenanceStarted *metav1.Time `json:"maintenanceStarted,omitempty" validate:"omitempty"`
	// MaintenanceFinished contains a timestamp indicating when the latest Egress Gateway pod used by
	// this workload is finishing a manintenance window.
	MaintenanceFinished *metav1.Time `json:"maintenanceFinished,omitempty" validate:"omitempty"`
}

// WorkloadEndpointPort represents one endpoint's named or mapped port
type WorkloadEndpointPort struct {
	Name     string               `json:"name" validate:"omitempty,portName"`
	Protocol numorstring.Protocol `json:"protocol"`
	Port     uint16               `json:"port" validate:"gt=0"`
	HostPort uint16               `json:"hostPort"`
	HostIP   string               `json:"hostIP" validate:"omitempty,net"`
}

// IPNat contains a single NAT mapping for a WorkloadEndpoint resource.
type IPNAT struct {
	// The internal IP address which must be associated with the owning endpoint via the
	// configured IPNetworks for the endpoint.
	InternalIP string `json:"internalIP" validate:"omitempty,ip"`
	// The external IP address.
	ExternalIP string `json:"externalIP" validate:"omitempty,ip"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkloadEndpointList contains a list of WorkloadEndpoint resources.
type WorkloadEndpointList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []WorkloadEndpoint `json:"items"`
}

// NewWorkloadEndpoint creates a new (zeroed) WorkloadEndpoint struct with the TypeMetadata initialised to the current
// version.
func NewWorkloadEndpoint() *WorkloadEndpoint {
	return &WorkloadEndpoint{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindWorkloadEndpoint,
			APIVersion: apiv3.GroupVersionCurrent,
		},
	}
}

// NewWorkloadEndpointList creates a new (zeroed) WorkloadEndpointList struct with the TypeMetadata initialised to the current
// version.
func NewWorkloadEndpointList() *WorkloadEndpointList {
	return &WorkloadEndpointList{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindWorkloadEndpointList,
			APIVersion: apiv3.GroupVersionCurrent,
		},
	}
}
