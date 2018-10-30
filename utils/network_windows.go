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
	"strings"
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
	hostVethName = ""

	// If a desired veth name was passed in, use that instead.
	//if desiredVethName != "" {
	//	hostVethName = desiredVethName
	//}
	_, subNet, _ := net.ParseCIDR(result.IPs[0].Address.String())

	n, _, err := loadNetConf(args.StdinData)
	if err != nil {
		logger.Errorf("Error loading args")
		return "", "", err
	}

	// Create hns network
	//networkName := n.Name
	networkName := CreateNetworkName(subNet)
	hnsNetwork, err := EnsureNetworkExists(networkName, subNet, logger)
	if err != nil {
		logger.Errorf("Unable to create hns network %s", networkName)
		return "", "", err
	}

	// Create host hns endpoint
	epName := networkName + "_ep"
	hnsEndpoint, err := CreateAndAttachHostEP(epName, hnsNetwork, subNet, logger)
	if err != nil {
		logger.Errorf("Unable to create host hns endpoint %s", epName)
		return "", "", err
	}

	// Check for management ip getting assigned to the network, interface with the management ip
	// and then enable forwarding on management interface as well as endpoint
	err = ChkMgmtIPandEnableForwarding(networkName, hnsEndpoint, logger)
	if err != nil {
		logger.Errorf("Failed to enable forwarding : %v", err)
		return "", "", err
	}

	// Create endpoint for container
	hnsEndpointCont, err := CreateAndAttachContainerEP(args, hnsNetwork, subNet, result, n.Name, logger)
	if err != nil {
		logger.Errorf("Unable to create container hns endpoint %s", epName)
		return "", "", err
	}

	contVethMAC = hnsEndpointCont.MacAddress
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
				logger.Errorf("Unable to delete existing network [%v], error: %v", hnsNetwork.Name, err)
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
			logger.Errorf("Error in converting to json format")
			return nil, err
		}

		logger.Infof("Attempting to create HNS network, request: %v", string(reqStr))
		if hnsNetwork, err = hcsshim.HNSNetworkRequest("POST", "", string(reqStr)); err != nil {
			logger.Errorf("unable to create network [%v], error: %v", networkName, err)
			return nil, err
		}
		logger.Infof("Created HNS network [%v] as %+v", networkName, hnsNetwork)
	}
	return hnsNetwork, err
}

func CreateAndAttachHostEP(epName string, hnsNetwork *hcsshim.HNSNetwork, subNet *net.IPNet, logger *logrus.Entry) (*hcsshim.HNSEndpoint, error) {
	var err error
	podGatewayAddress := GetGW(subNet)
	attachEndpoint := true

	// Checking if HNS Endpoint exists.
	hnsEndpoint, _ := hcsshim.GetHNSEndpointByName(epName)
	if hnsEndpoint != nil {
		if !hnsEndpoint.IPAddress.Equal(podGatewayAddress) {
			// IPAddress does not match. Delete stale endpoint
			if _, err = hnsEndpoint.Delete(); err != nil {
				logger.Errorf("Unable to delete existing bridge endpoint [%v], error: %v", epName, err)
				return nil, err
			}
			logger.Infof("Deleted stale bridge endpoint [%v]")
			hnsEndpoint = nil
		} else if hnsEndpoint.VirtualNetwork == hnsNetwork.Id {
			// Endpoint exists for correct network. No processing required
			attachEndpoint = false
		}
	}

	if hnsEndpoint == nil {
		// Create new endpoint
		hnsEndpoint = &hcsshim.HNSEndpoint{
			Name:           epName,
			IPAddress:      podGatewayAddress,
			VirtualNetwork: hnsNetwork.Id,
		}

		logger.Infof("Attempting to create bridge endpoint [%+v]", hnsEndpoint)
		hnsEndpoint, err = hnsEndpoint.Create()
		if err != nil {
			logger.Errorf("Unable to create bridge endpoint [%v], error: %v", epName, err)
			return nil, err
		}
		logger.Infof("Created bridge endpoint [%v] as %+v", epName, hnsEndpoint)
	}

	if attachEndpoint == true {
		// Attach endpoint to host
		if err = hnsEndpoint.HostAttach(1); err != nil {
			logger.Errorf("Unable to hot attach bridge endpoint [%v] to host compartment, error: %v", epName, err)
			return nil, err
		}
		logger.Infof("Attached bridge endpoint [%v] to host", epName)
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
			logger.Errorf("Unable to get hns network %s after creation, error: %v", networkName, err)
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
			logger.Infof("Unable to find interface for IP Address [%v], error: %v", interfaceIpAddress, err)
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
	logger.Infof("Attempting to create HNS endpoint name : %s for container", endpointName)
	hnsEndpointCont, err := hns.ProvisionEndpoint(endpointName, hnsNetwork.Id, args.ContainerID, func() (*hcsshim.HNSEndpoint, error) {
		hnsEP := &hcsshim.HNSEndpoint{
			Name:           endpointName,
			VirtualNetwork: hnsNetwork.Id,
			GatewayAddress: GetGW(subNet).String(),
			IPAddress:      result.IPs[0].Address.IP,
		}
		return hnsEP, nil
	})
	logger.Infof("Endpoint to container created! %v", hnsEndpointCont)
	return hnsEndpointCont, err
}

func GetGW(PodCIDR *net.IPNet) net.IP {
	gwaddr := PodCIDR.IP.To4()
	gwaddr[3]++
	return gwaddr
}

func CreateNetworkName(subnet *net.IPNet) string {
	str := subnet.IP.String()
	network := strings.Replace(str, ".", "-", -1)
	name := "Calico-" + network
	return name
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
	logger.Infof("Cleaning up endpoint")

	n, _, err := loadNetConf(args.StdinData)
	if err != nil {
		return err
	}

	epName := hns.ConstructEndpointName(args.ContainerID, args.Netns, n.Name)
	logger.Infof("Attempting to delete HNS endpoint name : %s for container", epName)

	return hns.DeprovisionEndpoint(epName, args.Netns, args.ContainerID)
}
