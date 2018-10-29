// Copyright (c) 2018 Tigera, Inc. All rights reserved.
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
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/Microsoft/hcsshim"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/plugins/pkg/hns"
	"github.com/projectcalico/cni-plugin/types"
	netsh "github.com/rakelkar/gonetsh/netsh"
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

type NetConf struct {
	hns.NetConf

	IPMasqNetwork string `json:"ipMasqNetwork,omitempty"`
}

func loadNetConf(bytes []byte) (*NetConf, string, error) {
	n := &NetConf{}
	if err := json.Unmarshal(bytes, n); err != nil {
		return nil, "", fmt.Errorf("failed to load netconf: %v", err)
	}
	return n, n.CNIVersion, nil
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
	_, subNet, _ := net.ParseCIDR(result.IPs[0].Address.String())

	n, _, err := loadNetConf(args.StdinData)
	if err != nil {
		logger.Infof("Error loading args")
		return hostVethName, contVethMAC, err
	}

	// Create hns network
	networkName := n.Name
	hnsNetwork, err := EnsureNetworkExists(networkName, subNet, logger)
	if hnsNetwork == nil && err != nil {
		logger.Infof("Unable to create hns network %s", networkName)
		return hostVethName, contVethMAC, err
	}

	// Create host hns endpoint
	epName := networkName + "_ep"
	hnsEndpoint, err := CreateAndAttachHostEP(epName, hnsNetwork, subNet, logger)
	if hnsEndpoint == nil && err != nil {
		logger.Infof("Unable to create host hns endpoint %s", epName)
		return hostVethName, contVethMAC, err
	}

	// Check for management ip getting assigned to the network, interface with the management ip
	// and then enable forwarding on management interface as well as endpoint
	err = ChkMgmtIPandEnableForwarding(networkName, hnsEndpoint, logger)
	if err != nil {
		logger.Infof("Failed to enable forwarding : %v", err)
		return hostVethName, contVethMAC, err
	}

	// Create endpoint for container
	hnsEndpoint_cont, err := CreateAndAttachContainerEP(args, hnsNetwork, subNet, result, n.Name, logger)
	if err != nil {
		logger.Infof("Unable to create container hns endpoint %s", epName)
		return hostVethName, contVethMAC, err
	}

	contVethMAC = hnsEndpoint_cont.MacAddress
	return hostVethName, contVethMAC, err
}

func EnsureNetworkExists(networkName string, subNet *net.IPNet, logger *logrus.Entry) (*hcsshim.HNSNetwork, error) {
	var err error
	createNetwork := true
	addressPrefix := subNet.String()
	gatewayAddress := GetGW(subNet)

	// Checking if HNS network exists
	hnsNetwork, _ := hcsshim.GetHNSNetworkByName(networkName)
	if hnsNetwork != nil {
		for _, subnet := range hnsNetwork.Subnets {
			if subnet.AddressPrefix == addressPrefix && subnet.GatewayAddress == gatewayAddress.String() {
				createNetwork = false
				logger.Infof("Found existing HNS network [%+v]", hnsNetwork)
				break
			}

		}
	}

	if createNetwork {
		// Delete stale network
		if hnsNetwork != nil {
			if _, err := hnsNetwork.Delete(); err != nil {
				logger.Infof("unable to delete existing network [%v], error: %v", hnsNetwork.Name, err)
				return nil, err
			}
			logger.Infof("Deleted stale HNS network [%v]")
		}

		// Create new hnsNetwork
		req := map[string]interface{}{
			"Name": networkName,
			"Type": "L2Bridge",
			"Subnets": []interface{}{
				map[string]interface{}{
					"AddressPrefix":  addressPrefix,
					"GatewayAddress": gatewayAddress,
				},
			},
		}
		reqStr, err := json.Marshal(req)
		if err != nil {
			logger.Infof("error in converting to json format")
			return nil, err
		}

		logger.Infof("DEBUG: Attempting to create HNS network, request: %v", string(reqStr))
		if hnsNetwork, err = hcsshim.HNSNetworkRequest("POST", "", string(reqStr)); err != nil {
			logger.Infof("unable to create network [%v], error: %v", networkName, err)
			return nil, err
		}
		logger.Infof("Created HNS network [%v] as %+v", networkName, hnsNetwork)
	}
	return hnsNetwork, err
}

func CreateAndAttachHostEP(epName string, hnsNetwork *hcsshim.HNSNetwork, subNet *net.IPNet, logger *logrus.Entry) (*hcsshim.HNSEndpoint, error) {
	var endpointToAttach *hcsshim.HNSEndpoint
	var err error
	createEndpoint := true
	podGatewayAddress := GetGW(subNet)

	// Checking if HNS Endpoint exists.
	hnsEndpoint, _ := hcsshim.GetHNSEndpointByName(epName)
	if hnsEndpoint != nil && hnsEndpoint.IPAddress.String() == podGatewayAddress.String() {
		endpointToAttach = hnsEndpoint
		createEndpoint = false
		logger.Infof("Endpoint exists %v ", hnsEndpoint)
		if endpointToAttach.VirtualNetwork != hnsNetwork.Id {
			if err = endpointToAttach.HostAttach(1); err != nil {
				logger.Infof("unable to hot attach bridge endpoint [%v] to host compartment, error: %v", epName, err)
				return nil, err
			}
			logger.Infof("Attached bridge endpoint [%v] to host", epName)
		}
	}

	if createEndpoint {
		// Delete stale endpoint
		if hnsEndpoint != nil {
			if _, err = hnsEndpoint.Delete(); err != nil {
				logger.Infof("unable to delete existing bridge endpoint [%v], error: %v", epName, err)
				return nil, err
			}
			logger.Infof("Deleted stale bridge endpoint [%v]")
		}

		// Create new endpoint
		hnsEndpoint = &hcsshim.HNSEndpoint{
			Name:           epName,
			IPAddress:      podGatewayAddress,
			VirtualNetwork: hnsNetwork.Id,
		}

		logger.Infof("DEBUG: Attempting to create bridge endpoint [%+v]", hnsEndpoint)
		hnsEndpoint, err = hnsEndpoint.Create()
		if err != nil {
			logger.Infof("unable to create bridge endpoint [%v], error: %v", epName, err)
			return nil, err
		}
		logger.Infof("DEBUG: Created bridge endpoint [%v] as %+v", epName, hnsEndpoint)

		// Attach endpoint to host
		endpointToAttach = hnsEndpoint
		if err = endpointToAttach.HostAttach(1); err != nil {
			logger.Infof("unable to hot attach bridge endpoint [%v] to host compartment, error: %v", epName, err)
			return nil, err
		}
		logger.Infof("DEBUG:Attached bridge endpoint [%v] to host", epName)
	}
	return hnsEndpoint, err
}

func ChkMgmtIPandEnableForwarding(networkName string, hnsEndpoint *hcsshim.HNSEndpoint, logger *logrus.Entry) error {
	netHelper := netsh.New(nil)
	var Network *hcsshim.HNSNetwork
	var err error

	// Wait for the network to populate Management IP.
	for {
		Network, err = hcsshim.GetHNSNetworkByName(networkName)
		if err != nil {
			logger.Infof("Unable to get hns network %s after creation, error: %v", networkName, err)
			return err
		}

		if len(Network.ManagementIP) > 0 {
			logger.Infof("Got managementIP %s", Network.ManagementIP)
			break
		}
		time.Sleep(1 * time.Second)
		logger.Infof("Checking ManagementIP with HnsNetwork[%v]", Network)
	}

	//Wait for the interface with the management IP
	for {
		if _, err = netHelper.GetInterfaceByIP(Network.ManagementIP); err != nil {
			time.Sleep(1 * time.Second)
			logger.Infof("Checking interface with ip %s, err %v", Network.ManagementIP, err)
			continue
		}
		break
	}

	// Enable forwarding on the host interface and endpoint
	for _, interfaceIpAddress := range []string{Network.ManagementIP, hnsEndpoint.IPAddress.String()} {
		netInterface, err := netHelper.GetInterfaceByIP(interfaceIpAddress)
		if err != nil {
			logger.Infof("Unable to find interface for IP Addess [%v], error: %v", interfaceIpAddress, err)
			return err
		}

		logger.Infof("Found Interface with IP[%s]: %v", interfaceIpAddress, netInterface)
		interfaceIdx := strconv.Itoa(netInterface.Idx)
		if err = netHelper.EnableForwarding(interfaceIdx); err != nil {
			logger.Infof("Unable to enable forwarding on [%v] index [%v], error: %v", netInterface.Name, interfaceIdx, err)
			return err
		}
		logger.Infof("Enabled forwarding on [%v] index [%v]", netInterface.Name, interfaceIdx)
	}
	return nil
}

func CreateAndAttachContainerEP(args *skel.CmdArgs,
	hnsNetwork *hcsshim.HNSNetwork,
	subNet *net.IPNet,
	result *current.Result,
	Name string,
	logger *logrus.Entry) (*hcsshim.HNSEndpoint, error) {

	endpointName := hns.ConstructEndpointName(args.ContainerID, args.Netns, Name)
	logger.Infof("DEBUG:Attempting to create HNS endpoint name : %s for container", endpointName)
	hnsEndpoint_cont, err := hns.ProvisionEndpoint(endpointName, hnsNetwork.Id, args.ContainerID, func() (*hcsshim.HNSEndpoint, error) {
		hnsEP := &hcsshim.HNSEndpoint{
			Name:           endpointName,
			VirtualNetwork: hnsNetwork.Id,
			GatewayAddress: GetGW(subNet).String(),
			IPAddress:      result.IPs[0].Address.IP,
		}
		return hnsEP, nil
	})
	logger.Infof("Endpoint to container created! %v", hnsEndpoint_cont)
	return hnsEndpoint_cont, err
}

func GetGW(PodCIDR *net.IPNet) net.IP {
	gwaddr := PodCIDR.IP.To4()
	gwaddr[3]++
	return gwaddr
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
	logger.Warn("DEBUG: Cleaning up endpoint")

	n, _, err := loadNetConf(args.StdinData)
	if err != nil {
		return err
	}

	epName := hns.ConstructEndpointName(args.ContainerID, args.Netns, n.Name)
	logger.Infof("DEBUG:Attempting to delete HNS endpoint name : %s for container", epName)

	return hns.DeprovisionEndpoint(epName, args.Netns, args.ContainerID)
}
