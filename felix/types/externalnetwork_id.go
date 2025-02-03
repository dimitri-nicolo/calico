// Copyright (c) 2025 Tigera, Inc. All rights reserved.

package types

import "github.com/projectcalico/calico/felix/proto"

type ExternalNetworkID struct {
	Name string
}

func ProtoToExternalNetworkID(n *proto.ExternalNetworkID) ExternalNetworkID {
	return ExternalNetworkID{
		Name: n.GetName(),
	}
}

func ExternalNetworkIDToProto(n ExternalNetworkID) *proto.ExternalNetworkID {
	return &proto.ExternalNetworkID{
		Name: n.Name,
	}
}
