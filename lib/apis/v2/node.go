// Copyright (c) 2017 Tigera, Inc. All rights reserved.

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

package v2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/libcalico-go/lib/numorstring"
)

const (
	KindNode     = "Node"
	KindNodeList = "NodeList"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Node contains information about a Node resource.
type Node struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the Node.
	Spec NodeSpec `json:"spec,omitempty"`
}

// NodeSpec contains the specification for a Node resource.
type NodeSpec struct {
	// BGP configuration for this node.
	BGP *NodeBGPSpec `json:"bgp,omitempty" validate:"omitempty"`
}

// NodeBGPSpec contains the specification for the Node BGP configuration.
type NodeBGPSpec struct {
	// The AS Number of the node.  If this is not specified, the global
	// default value will be used.
	ASNumber *numorstring.ASNumber `json:"asNumber,omitempty"`
	// IPv4Address is the IPv4 address and network of this node.  At least
	// one of the IPv4 and IPv6 addresses should be specified.
	IPv4Address string `json:"ipv4Address,omitempty" validate:"omitempty,ipv4"`
	// IPv6Address is the IPv6 address and network of this node.  At least
	// one of the IPv4 and IPv6 addresses should be specified.
	IPv6Address string `json:"ipv6Address,omitempty" validate:"omitempty,ipv6"`
	// IPv4IPIPTunnelAddr is the IPv4 address of the IP in IP tunnel.
	IPv4IPIPTunnelAddr string `json:"ipv4IPIPTunnelAddr,omitempty" validate:"omitempty,ipv4"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeList contains a list of Node resources.
type NodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Node `json:"items"`
}

// NewNode creates a new (zeroed) Node struct with the TypeMetadata initialised to the current
// version.
func NewNode() *Node {
	return &Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindNode,
			APIVersion: GroupVersionCurrent,
		},
	}
}

// NewNodeList creates a new (zeroed) NodeList struct with the TypeMetadata initialised to the current
// version.
func NewNodeList() *NodeList {
	return &NodeList{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindNodeList,
			APIVersion: GroupVersionCurrent,
		},
	}
}
