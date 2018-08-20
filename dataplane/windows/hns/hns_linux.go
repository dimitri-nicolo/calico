// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package hns

import (
	"encoding/json"
	"net"
)

// Type of Request Support in ModifySystem
type PolicyType string

// RequestType const
const (
	Nat                  PolicyType = "Nat"
	ACL                  PolicyType = "ACL"
	PA                   PolicyType = "PA"
	VLAN                 PolicyType = "VLAN"
	VSID                 PolicyType = "VSID"
	VNet                 PolicyType = "VNet"
	L2Driver             PolicyType = "L2Driver"
	Isolation            PolicyType = "Isolation"
	QOS                  PolicyType = "QOS"
	OutboundNat          PolicyType = "OutboundNat"
	ExternalLoadBalancer PolicyType = "ExternalLoadBalancer"
	Route                PolicyType = "Route"
)

//
//type NatPolicy = hcsshim.NatPolicy
//
//type QosPolicy = hcsshim.QosPolicy
//
//type IsolationPolicy = hcsshim.IsolationPolicy
//
//type VlanPolicy = hcsshim.VlanPolicy
//
//type VsidPolicy = hcsshim.VsidPolicy
//
//type PaPolicy = hcsshim.PaPolicy
//
//type OutboundNatPolicy = hcsshim.OutboundNatPolicy

type ActionType string
type DirectionType string
type RuleType string

const (
	Allow ActionType = "Allow"
	Block ActionType = "Block"

	In  DirectionType = "In"
	Out DirectionType = "Out"

	Host   RuleType = "Host"
	Switch RuleType = "Switch"
)

type ACLPolicy struct {
	Type            PolicyType `json:"Type"`
	Id              string     `json:"Id,omitempty"`
	Protocol        uint16
	Protocols       string `json:"Protocols,omitempty"`
	InternalPort    uint16
	Action          ActionType
	Direction       DirectionType
	LocalAddresses  string
	RemoteAddresses string
	LocalPorts      string `json:"LocalPorts,omitempty"`
	LocalPort       uint16
	RemotePorts     string `json:"RemotePorts,omitempty"`
	RemotePort      uint16
	RuleType        RuleType `json:"RuleType,omitempty"`
	Priority        uint16
	ServiceName     string
}

type Policy struct {
}

// HNSEndpoint represents a network endpoint in HNS
type HNSEndpoint struct {
	Id                 string            `json:"ID,omitempty"`
	Name               string            `json:",omitempty"`
	VirtualNetwork     string            `json:",omitempty"`
	VirtualNetworkName string            `json:",omitempty"`
	Policies           []json.RawMessage `json:",omitempty"`
	MacAddress         string            `json:",omitempty"`
	IPAddress          net.IP            `json:",omitempty"`
	DNSSuffix          string            `json:",omitempty"`
	DNSServerList      string            `json:",omitempty"`
	GatewayAddress     string            `json:",omitempty"`
	EnableInternalDNS  bool              `json:",omitempty"`
	DisableICC         bool              `json:",omitempty"`
	PrefixLength       uint8             `json:",omitempty"`
	IsRemoteEndpoint   bool              `json:",omitempty"`
	// Namespace          *Namespace        `json:",omitempty"`
}

// ApplyACLPolicy applies a set of ACL Policies on the Endpoint
func (endpoint *HNSEndpoint) ApplyACLPolicy(policies ...*ACLPolicy) error {
	return nil
}

type HNSSupportedFeatures struct {
	Acl HNSAclFeatures
}

type HNSAclFeatures struct {
	AclAddressLists       bool
	AclNoHostRulePriority bool
	AclPortRanges         bool
	AclRuleId             bool
}

type API struct{}

func (_ API) GetHNSSupportedFeatures() HNSSupportedFeatures {
	return HNSSupportedFeatures{}
}

func (_ API) HNSListEndpointRequest() ([]HNSEndpoint, error) {
	return nil, nil
}
