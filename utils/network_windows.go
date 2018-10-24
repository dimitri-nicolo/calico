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
	"encoding/json"
	"strconv"
	"time"
	"fmt"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/projectcalico/cni-plugin/types"
	"github.com/sirupsen/logrus"
	"github.com/Microsoft/hcsshim"
	"github.com/containernetworking/plugins/pkg/hns"
	netsh "github.com/rakelkar/gonetsh/netsh"
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

	contVethMAC = "00:15:5D:01:02:03"
	conf.IPAM.Subnet = "10.244.194.192/26"

	n, _, err := loadNetConf(args.StdinData)
	if err != nil {
		fmt.Errorf("error loading args")
		return hostVethName, contVethMAC, err
	}

	networkName := n.Name

	/* checking if HNS network exists.*/
	createNetwork := true
	addressPrefix  := conf.IPAM.Subnet
	gatewayAddress := GetPodGW(conf.IPAM.Subnet)
	hnsNetwork,err := hcsshim.GetHNSNetworkByName(networkName)
	if hnsNetwork != nil {
		for _, subnet := range hnsNetwork.Subnets {
			if subnet.AddressPrefix == addressPrefix && subnet.GatewayAddress == gatewayAddress {
                                createNetwork = false
                                logger.Infof("Found existing HNS network [%+v]", hnsNetwork)
                                break
                        }

		}
	}

	if createNetwork {
		/* delete stale network */
		if hnsNetwork != nil {
			if _, err := hnsNetwork.Delete(); err != nil {
				fmt.Errorf("unable to delete existing network [%v], error: %v", hnsNetwork.Name, err)
				return hostVethName, contVethMAC, err
			}
			logger.Infof("Deleted stale HNS network [%v]")
		}

		/* create new hnsNetwork */
		req := map[string]interface{} {
			"Name": networkName,
			"Type": "L2Bridge",
			"Subnets": []interface{}{
				map[string]interface{}{
					"AddressPrefix": addressPrefix,
					"GatewayAddress": gatewayAddress,
				},
			},
		}
		reqStr,err := json.Marshal(req)
		if err != nil {
			fmt.Errorf("error in converting to json format")
			return hostVethName, contVethMAC, err
		}

		logger.Infof("Attempting to create HNS network, request: %v", string(reqStr))
		if hnsNetwork, err = hcsshim.HNSNetworkRequest("POST", "", string(reqStr)); err != nil {
			fmt.Errorf("unable to create network [%v], error: %v", networkName, err)
			return hostVethName, contVethMAC, err
		}
		logger.Infof("Created HNS network [%v] as %+v", networkName, hnsNetwork)
	}

	epName := networkName + "_ep"
	var endpointToAttach *hcsshim.HNSEndpoint
	createEndpoint := true
	podGatewayAddress := "10.244.194.194"
	/* checking if HNS Endpoint exists.*/
	hnsEndpoint,err := hcsshim.GetHNSEndpointByName(epName)
	if hnsEndpoint != nil && hnsEndpoint.IPAddress.String() == podGatewayAddress {
		endpointToAttach = hnsEndpoint
		createEndpoint = false
		logger.Infof("Endpoint exists %v ", hnsEndpoint)
		if endpointToAttach.VirtualNetwork != hnsNetwork.Id {
			if err = endpointToAttach.HostAttach(1); err != nil {
				fmt.Errorf("unable to hot attach bridge endpoint [%v] to host compartment, error: %v", epName, err)
				return hostVethName, contVethMAC, err
			}
			logger.Infof("Attached bridge endpoint [%v] to host", epName)
		}
	}

	if createEndpoint {
		/* delete stale endpoint */
		if hnsEndpoint != nil {
			if _, err = hnsEndpoint.Delete(); err != nil {
				fmt.Errorf("unable to delete existing bridge endpoint [%v], error: %v", epName, err)
				return hostVethName, contVethMAC, err
			}
			logger.Infof("Deleted stale bridge endpoint [%v]")
		}

		/* create new endpoint */
		hnsEndpoint = &hcsshim.HNSEndpoint{
			Name:           epName,
			IPAddress:      net.ParseIP("10.244.194.194"),
			VirtualNetwork: hnsNetwork.Id,
		}

		logger.Infof("Attempting to create bridge endpoint [%+v]", hnsEndpoint)
		hnsEndpoint, err = hnsEndpoint.Create()
		if err != nil {
			fmt.Errorf("unable to create bridge endpoint [%v], error: %v", epName, err)
			return hostVethName, contVethMAC, err
		}
		logger.Infof("Created bridge endpoint [%v] as %+v",epName, hnsEndpoint)

		/* Attach endpoint to host */
		endpointToAttach = hnsEndpoint
                if err = endpointToAttach.HostAttach(1); err != nil {
                        fmt.Errorf("unable to hot attach bridge endpoint [%v] to host compartment, error: %v", epName, err)
                        return hostVethName, contVethMAC, err
                }
                logger.Infof("Attached bridge endpoint [%v] to host", epName)
	}

	netHelper := netsh.New(nil)
	var Network *hcsshim.HNSNetwork
	// Wait for the network to populate Management IP.
	for {
		Network, err = hcsshim.GetHNSNetworkByName(networkName)
		if err != nil {
			fmt.Errorf("unable to get hns network %s after creation, error: %v", networkName, err)
			return hostVethName, contVethMAC, err
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

	// enable forwarding on the host interface and endpoint
	for _, interfaceIpAddress := range []string{Network.ManagementIP, hnsEndpoint.IPAddress.String()} {
		netInterface, err := netHelper.GetInterfaceByIP(interfaceIpAddress)
		if err != nil {
			fmt.Errorf("unable to find interface for IP Addess [%v], error: %v", interfaceIpAddress, err)
			return hostVethName, contVethMAC, err
		}

		logger.Infof("Found Interface with IP[%s]: %v", interfaceIpAddress, netInterface)
		interfaceIdx := strconv.Itoa(netInterface.Idx)
		if err := netHelper.EnableForwarding(interfaceIdx); err != nil {
			fmt.Errorf("unable to enable forwarding on [%v] index [%v], error: %v", netInterface.Name, interfaceIdx, err)
			return hostVethName, contVethMAC, err
		}
		logger.Infof("Enabled forwarding on [%v] index [%v]", netInterface.Name, interfaceIdx)
	}

	/* Create endpoint for container */
	//endpointName := hns.ConstructEndpointName(args.ContainerID, args.Netns, n.Name)
	endpointName := n.Name + "_container"
	logger.Infof("Attempting to create HNS endpoint name : %s for container", endpointName)
	hnsEndpoint_cont, err := hns.ProvisionEndpoint(endpointName, hnsNetwork.Id, args.ContainerID, func() (*hcsshim.HNSEndpoint, error) {
		if len(n.IPMasqNetwork) != 0 {
			n.ApplyOutboundNatPolicy(n.IPMasqNetwork)
		}

		hnsEP := &hcsshim.HNSEndpoint{
			Name:           endpointName,
			VirtualNetwork: hnsNetwork.Id,
			GatewayAddress: "10.244.194.194",
			IPAddress:      net.ParseIP("10.244.194.200"),
		}
		return hnsEP, nil
	})
	logger.Infof("Endpoint to container created! %v", hnsEndpoint_cont)
	return hostVethName, contVethMAC, err
}

func GetPodGW(PodCIDR string) string {
	return "10.244.194.193"
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
