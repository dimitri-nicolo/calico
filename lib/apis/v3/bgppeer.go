// Copyright (c) 2017-2020 Tigera, Inc. All rights reserved.

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
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/libcalico-go/lib/numorstring"
)

const (
	KindBGPPeer     = "BGPPeer"
	KindBGPPeerList = "BGPPeerList"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BGPPeer contains information about a BGPPeer resource that is a peer of a Calico
// compute node.
type BGPPeer struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the BGPPeer.
	Spec BGPPeerSpec `json:"spec,omitempty"`
}

// BGPPeerSpec contains the specification for a BGPPeer resource.
type BGPPeerSpec struct {
	// The node name identifying the Calico node instance that is peering with this peer.
	// If this is not set, this represents a global peer, i.e. a peer that peers with
	// every node in the deployment.
	Node string `json:"node,omitempty" validate:"omitempty,name"`
	// The IP address of the peer followed by an optional port number to peer with.
	// If port number is given, format should be `[<IPv6>]:port` or `<IPv4>:<port>` for IPv4.
	// If optional port number is not set, and this peer IP and ASNumber belongs to a calico/node
	// with ListenPort set in BGPConfiguration, then we use that port to peer.
	PeerIP string `json:"peerIP" validate:"omitempty,IP:port"`
	// The AS Number of the peer.
	ASNumber numorstring.ASNumber `json:"asNumber"`
	// Extensions is a mapping of keys to values that can be used in custom BGP templates
	Extensions map[string]string `json:"extensions,omitempty" validate:"omitempty"`
	// Selector for the nodes that should have this peering.  When this is set, the Node
	// field must be empty.
	NodeSelector string `json:"nodeSelector,omitempty" validate:"omitempty,selector"`
	// Selector for the remote nodes to peer with.  When this is set, the PeerIP and
	// ASNumber fields must be empty.  For each peering between the local node and
	// selected remote nodes, we configure an IPv4 peering if both ends have
	// NodeBGPSpec.IPv4Address specified, and an IPv6 peering if both ends have
	// NodeBGPSpec.IPv6Address specified.  The remote AS number comes from the remote
	// node’s NodeBGPSpec.ASNumber, or the global default if that is not set.
	PeerSelector string `json:"peerSelector,omitempty" validate:"omitempty,selector"`
	// Option to keep the original nexthop field when routes are sent to a BGP Peer.
	// Setting "true" configures the selected BGP Peers node to use the "next hop keep;"
	// instead of "next hop self;"(default) in the specific branch of the Node on "bird.cfg".
	KeepOriginalNextHop bool `json:"keepOriginalNextHop,omitempty"`
	// Optional BGP password for the peerings generated by this BGPPeer resource.
	Password *BGPPassword `json:"password,omitempty" validate:"omitempty"`
	// Specifies whether and how to configure a source address for the peerings generated by
	// this BGPPeer resource.  Default value "UseNodeIP" means to configure the node IP as the
	// source address.  "None" means not to configure a source address.
	SourceAddress SourceAddress `json:"sourceAddress,omitempty" validate:"omitempty,sourceAddress"`
	// Specifies whether and how to detect loss of connectivity on the peerings generated by
	// this BGPPeer resource.  Default value "None" means nothing beyond BGP's own (slow) hold
	// timer.  "BFDIfDirectlyConnected" means to use BFD when the peer is directly connected.
	FailureDetectionMode FailureDetectionMode `json:"failureDetectionMode,omitempty" validate:"omitempty,failureDetectionMode"`
	// Specifies restart behaviour to configure on the peerings generated by this BGPPeer
	// resource.  Default value "GracefulRestart" means traditional graceful restart.
	// "LongLivedGracefulRestart" means LLGR according to draft-uttaro-idr-bgp-persistence-05.
	RestartMode RestartMode `json:"restartMode,omitempty" validate:"omitempty,restartMode"`
	// Time to allow for software restart.  When specified, this is configured as the graceful
	// restart timeout when RestartMode is "GracefulRestart", and as the LLGR stale time when
	// RestartMode is "LongLivedGracefulRestart".  When not specified, the BIRD defaults are
	// used, which are 120s for "GracefulRestart" and 3600s for "LongLivedGracefulRestart".
	MaxRestartTime *metav1.Duration `json:"maxRestartTime,omitempty"`
	// Specifies the BIRD "gateway" mode, i.e. method for computing the immediate next hop for
	// each received route, for peerings generated by this BGPPeer resource.  Default value
	// "Recursive" means "gateway recursive".  "DirectIfDirectlyConnected" means to configure
	// "gateway direct" when the peer is directly connected.
	BIRDGatewayMode BIRDGatewayMode `json:"birdGatewayMode,omitempty" validate:"omitempty,birdGatewayMode"`
}

type SourceAddress string

const (
	SourceAddressUseNodeIP SourceAddress = "UseNodeIP"
	SourceAddressNone                    = "None"
)

type FailureDetectionMode string

const (
	FailureDetectionModeNone                   FailureDetectionMode = "None"
	FailureDetectionModeBFDIfDirectlyConnected                      = "BFDIfDirectlyConnected"
)

type RestartMode string

const (
	RestartModeGracefulRestart          RestartMode = "GracefulRestart"
	RestartModeLongLivedGracefulRestart             = "LongLivedGracefulRestart"
)

type BIRDGatewayMode string

const (
	BIRDGatewayModeRecursive                 BIRDGatewayMode = "Recursive"
	BIRDGatewayModeDirectIfDirectlyConnected                 = "DirectIfDirectlyConnected"
)

// BGPPassword contains ways to specify a BGP password.
type BGPPassword struct {
	// Selects a key of a secret in the node pod's namespace.
	SecretKeyRef *k8sv1.SecretKeySelector `json:"secretKeyRef,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BGPPeerList contains a list of BGPPeer resources.
type BGPPeerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []BGPPeer `json:"items"`
}

// NewBGPPeer creates a new (zeroed) BGPPeer struct with the TypeMetadata initialised to the current
// version.
func NewBGPPeer() *BGPPeer {
	return &BGPPeer{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindBGPPeer,
			APIVersion: GroupVersionCurrent,
		},
	}
}

// NewBGPPeerList creates a new (zeroed) BGPPeerList struct with the TypeMetadata initialised to the current
// version.
func NewBGPPeerList() *BGPPeerList {
	return &BGPPeerList{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindBGPPeerList,
			APIVersion: GroupVersionCurrent,
		},
	}
}
