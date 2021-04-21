// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package earlynetworking

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KindEarlyNetworkConfiguration = "EarlyNetworkConfiguration"
)

type EarlyNetworkConfiguration struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the EarlyNetworkConfiguration.
	Spec EarlyNetworkConfigurationSpec `json:"spec,omitempty"`
}

type EarlyNetworkConfigurationSpec struct {
	Nodes []ConfigNode
}

type ConfigNode struct {
	InterfaceAddresses []string            `yaml:"interfaceAddresses"`
	ASNumber           int                 `yaml:"asNumber"`
	StableAddress      ConfigStableAddress `yaml:"stableAddress"`
	Peerings           []ConfigPeering     `yaml:"peerings"`
	Labels             map[string]string   `yaml:"labels"`
}

type ConfigStableAddress struct {
	Address string
}

type ConfigPeering struct {
	PeerIP       string `yaml:"peerIP"`
	PeerASNumber int    `yaml:"peerASNumber"`
}
