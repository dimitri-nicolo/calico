// Copyright (c) 2018-2021 Tigera, Inc. All rights reserved.

package utils

import (
	"context"
	"net"

	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/sirupsen/logrus"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/projectcalico/cni-plugin/pkg/types"
	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	calicoclient "github.com/projectcalico/libcalico-go/lib/clientv3"
)

func updateHostLocalIPAMDataForOS(subnet string, ipamData map[string]interface{}) error {
	return nil
}

func EnsureVXLANTunnelAddr(ctx context.Context, calicoClient calicoclient.Interface, nodeName string, ipNet *net.IPNet, conf types.NetConf) error {
	return nil
}

func RegisterDeletedWep(containerID string) error {
	return nil
}

func CheckForSpuriousDockerAdd(args *skel.CmdArgs,
	conf types.NetConf,
	epIDs WEPIdentifiers,
	endpoint *api.WorkloadEndpoint,
	logger *logrus.Entry) (*current.Result, error) {
	return nil, nil
}
