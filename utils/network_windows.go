// Copyright 2018 Tigera Inc
//
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
package utils

import (
	"net"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/projectcalico/cni-plugin/types"
	"github.com/sirupsen/logrus"
)

var (
	// IPv4AllNet represents the IPv4 all-addresses CIDR 0.0.0.0/0.
	IPv4AllNet *net.IPNet
	// IPv6AllNet represents the IPv6 all-addresses CIDR ::/0.
	IPv6AllNet    *net.IPNet
	DefaultRoutes []*net.IPNet
)

func init() {
	var err error
	_, IPv4AllNet, err = net.ParseCIDR("0.0.0.0/0")
	if err != nil {
		panic(err)
	}
	_, IPv6AllNet, err = net.ParseCIDR("::/0")
	if err != nil {
		panic(err)
	}
	DefaultRoutes = []*net.IPNet{
		IPv4AllNet,
		IPv6AllNet, // Only used if we end up adding a v6 address.
	}
}

// DoNetworking performs the networking for the given config and IPAM result
func DoNetworking(
	args *skel.CmdArgs,
	conf types.NetConf,
	result *current.Result,
	logger *logrus.Entry,
	desiredVethName string,
	routes []*net.IPNet,
) (hostVethName, contVethMAC string, err error) {
	// Select the first 11 characters of the containerID for the host veth.
	hostVethName = "cali" + args.ContainerID[:Min(11, len(args.ContainerID))]

	// If a desired veth name was passed in, use that instead.
	if desiredVethName != "" {
		hostVethName = desiredVethName
	}

	logger.Infof("STUB: Networking, veth name would be %s", hostVethName)

	return hostVethName, contVethMAC, err
}

// SetupRoutes sets up the routes for the host side of the veth pair.
func SetupRoutes(hostVeth interface{}, result *current.Result) error {

	// Go through all the IPs and add routes for each IP in the result.
	for _, ipAddr := range result.IPs {
		logrus.WithFields(logrus.Fields{"interface": hostVeth, "IP": ipAddr.Address}).Debugf("STUB: CNI adding route")
	}
	return nil
}

// CleanUpNamespace deletes the devices in the network namespace.
func CleanUpNamespace(args *skel.CmdArgs, logger *logrus.Entry) error {
	logger.Warn("STUB: Clean up namespace")
	return nil
}
