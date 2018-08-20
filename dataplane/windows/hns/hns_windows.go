// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package hns

import "github.com/Microsoft/hcsshim"

type HNSSupportedFeatures = hcsshim.HNSSupportedFeatures

// Type of Request Support in ModifySystem
type PolicyType = hcsshim.PolicyType

// RequestType const
const (
	Nat                  = hcsshim.Nat
	ACL                  = hcsshim.ACL
	PA                   = hcsshim.PA
	VLAN                 = hcsshim.VLAN
	VSID                 = hcsshim.VSID
	VNet                 = hcsshim.VNet
	L2Driver             = hcsshim.L2Driver
	Isolation            = hcsshim.Isolation
	QOS                  = hcsshim.QOS
	OutboundNat          = hcsshim.OutboundNat
	ExternalLoadBalancer = hcsshim.ExternalLoadBalancer
	Route                = hcsshim.Route
)

type NatPolicy = hcsshim.NatPolicy

type QosPolicy = hcsshim.QosPolicy

type IsolationPolicy = hcsshim.IsolationPolicy

type VlanPolicy = hcsshim.VlanPolicy

type VsidPolicy = hcsshim.VsidPolicy

type PaPolicy = hcsshim.PaPolicy

type OutboundNatPolicy = hcsshim.OutboundNatPolicy

type ActionType = hcsshim.ActionType
type DirectionType = hcsshim.DirectionType
type RuleType = hcsshim.RuleType

const (
	Allow = hcsshim.Allow
	Block = hcsshim.Block

	In  = hcsshim.In
	Out = hcsshim.Out

	Host   = hcsshim.Host
	Switch = hcsshim.Switch
)

type ACLPolicy = hcsshim.ACLPolicy

type Policy = hcsshim.Policy

type HNSEndpoint = hcsshim.HNSEndpoint

type API struct{}

func (_ API) GetHNSSupportedFeatures() HNSSupportedFeatures {
	return hcsshim.GetHNSSupportedFeatures()
}

func (_ API) HNSListEndpointRequest() ([]HNSEndpoint, error) {
	return hcsshim.HNSListEndpointRequest()
}
