// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// This package re-exports the HCN API as a struct sot that it can be shimmed and UTs can run on Linux.
package hcn

import realhcn "github.com/Microsoft/hcsshim/hcn"

type API struct{}

type HostComputeNetwork = realhcn.HostComputeNetwork
type RemoteSubnetRoutePolicySetting = realhcn.RemoteSubnetRoutePolicySetting
type PolicyNetworkRequest = realhcn.PolicyNetworkRequest
type NetworkPolicy = realhcn.NetworkPolicy

const (
	RemoteSubnetRoute = realhcn.RemoteSubnetRoute
)

func (_ API) ListNetworks() ([]HostComputeNetwork, error) {
	return realhcn.ListNetworks()
}
