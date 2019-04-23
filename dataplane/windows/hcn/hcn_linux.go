// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Dummy version of the HCN API for compilation on Linux.
package hcn

import (
	"encoding/json"
)

type API struct{}

type HostComputeNetwork struct {
	Id       string
	Name     string
	Type     NetworkType
	Policies []NetworkPolicy
}

func (network HostComputeNetwork) RemovePolicy(request PolicyNetworkRequest) error {
	return nil
}

func (network HostComputeNetwork) AddPolicy(request PolicyNetworkRequest) error {
	return nil
}

type NetworkType string

type RemoteSubnetRoutePolicySetting struct {
	DestinationPrefix           string
	IsolationId                 uint16
	ProviderAddress             string
	DistributedRouterMacAddress string
}

type PolicyNetworkRequest struct {
	Policies []NetworkPolicy
}

// NetworkPolicy is a collection of Policy settings for a Network.
type NetworkPolicy struct {
	Type     NetworkPolicyType
	Settings json.RawMessage
}

// NetworkPolicyType are the potential Policies that apply to Networks.
type NetworkPolicyType string

const (
	RemoteSubnetRoute NetworkPolicyType = "RemoteSubnetRoute"
)

func (_ API) ListNetworks() ([]HostComputeNetwork, error) {
	return nil, nil
}
