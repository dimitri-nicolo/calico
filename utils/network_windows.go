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
	"context"
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
	"github.com/juju/clock"
	"github.com/juju/mutex"
	"github.com/projectcalico/cni-plugin/types"
	calicoclient "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/rakelkar/gonetsh/netsh"
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

func loadNetConf(bytes []byte) (*hns.NetConf, string, error) {
	n := &hns.NetConf{}
	if err := json.Unmarshal(bytes, n); err != nil {
		return nil, "", fmt.Errorf("failed to load netconf: %v", err)
	}
	return n, n.CNIVersion, nil
}

func acquireLock() (mutex.Releaser, error) {
	spec := mutex.Spec{
		Name:    "TigeraCalicoCNINetworkMutex",
		Clock:   clock.WallClock,
		Delay:   50 * time.Millisecond,
		Timeout: 90000 * time.Millisecond,
	}
	logrus.Infof("Trying to acquire lock %v", spec)
	m, err := mutex.Acquire(spec)
	if err != nil {
		logrus.Errorf("Error acquiring lock %v", spec)
		return nil, err
	}
	logrus.Infof("Acquired lock %v", spec)
	return m, nil
}

// DoNetworking performs the networking for the given config and IPAM result
func DoNetworking(
	ctx context.Context,
	calicoClient calicoclient.Interface,
	args *skel.CmdArgs,
	conf types.NetConf,
	result *current.Result,
	logger *logrus.Entry,
	desiredVethName string,
	routes []*net.IPNet,
) (hostVethName, contVethMAC string, err error) {
	// Not used on Windows.
	_ = conf
	_ = desiredVethName
	if len(routes) > 0 {
		logrus.WithField("routes", routes).Warn("Ignoring in-container routes; not supported on Windows.")
	}

	podIP, subNet, _ := net.ParseCIDR(result.IPs[0].Address.String())

	n, _, err := loadNetConf(args.StdinData)
	if err != nil {
		logger.Errorf("Error loading args")
		return "", "", err
	}

	// We need to know the IPAM pools to program the correct NAT exclusion list.  Look those up
	// before we take the global lock.
	allIPAMPools, natOutgoing, err := lookupIPAMPools(ctx, podIP, calicoClient)
	if err != nil {
		logger.WithError(err).Error("Failed to look up IPAM pools")
		return "", "", err
	}

	// Acquire mutex lock
	m, err := acquireLock()
	if err != nil {
		logger.Errorf("Unable to acquiring lock")
		return "", "", err
	}
	defer m.Release()

	// Create hns network
	networkName := createNetworkName(n.Name, subNet)
	hnsNetwork, err := ensureNetworkExists(networkName, subNet, logger)
	if err != nil {
		logger.Errorf("Unable to create hns network %s", networkName)
		return "", "", err
	}

	// Create host hns endpoint
	epName := networkName + "_ep"
	hnsEndpoint, err := createAndAttachHostEP(epName, hnsNetwork, subNet, logger)
	if err != nil {
		logger.Errorf("Unable to create host hns endpoint %s", epName)
		return "", "", err
	}

	// Check for management ip getting assigned to the network, interface with the management ip
	// and then enable forwarding on management interface as well as endpoint
	err = chkMgmtIPandEnableForwarding(networkName, hnsEndpoint, logger)
	if err != nil {
		logger.Errorf("Failed to enable forwarding : %v", err)
		return "", "", err
	}

	// Create endpoint for container
	hnsEndpointCont, err := createAndAttachContainerEP(args, hnsNetwork, subNet, allIPAMPools, natOutgoing, result, n, logger)
	if err != nil {
		logger.Errorf("Unable to create container hns endpoint %s", epName)
		return "", "", err
	}

	contVethMAC = hnsEndpointCont.MacAddress
	return hostVethName, contVethMAC, err
}

func lookupIPAMPools(
	ctx context.Context, podIP net.IP, calicoClient calicoclient.Interface,
) (
	cidrs []*net.IPNet,
	natOutgoing bool,
	err error,
) {
	pools, err := calicoClient.IPPools().List(ctx, options.ListOptions{})
	if err != nil {
		return
	}
	natOutgoing = true
	for _, p := range pools.Items {
		_, ipNet, err := net.ParseCIDR(p.Spec.CIDR)
		if err != nil {
			logrus.WithError(err).WithField("rawCIDR", p.Spec.CIDR).Warn("IP pool contained bad CIDR, ignoring")
			continue
		}
		cidrs = append(cidrs, ipNet)
		if ipNet.Contains(podIP) {
			logrus.WithField("pool", p.Spec).Debug("Found pool containing pod IP")
			natOutgoing = p.Spec.NATOutgoing
		}
	}
	return
}

func ensureNetworkExists(networkName string, subNet *net.IPNet, logger *logrus.Entry) (*hcsshim.HNSNetwork, error) {
	var err error
	createNetwork := true
	addressPrefix := subNet.String()
	gatewayAddress := getNthIP(subNet, 1)

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

func createAndAttachHostEP(epName string, hnsNetwork *hcsshim.HNSNetwork, subNet *net.IPNet, logger *logrus.Entry) (*hcsshim.HNSEndpoint, error) {
	var err error
	endpointAddress := getNthIP(subNet, 2)
	attachEndpoint := true

	// Checking if HNS Endpoint exists.
	hnsEndpoint, _ := hcsshim.GetHNSEndpointByName(epName)
	if hnsEndpoint != nil {
		if !hnsEndpoint.IPAddress.Equal(endpointAddress) {
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
			IPAddress:      endpointAddress,
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

	if attachEndpoint {
		// Attach endpoint to host
		if err = hnsEndpoint.HostAttach(1); err != nil {
			logger.Errorf("Unable to hot attach bridge endpoint [%v] to host compartment, error: %v", epName, err)
			return nil, err
		}
		logger.Infof("Attached bridge endpoint [%v] to host", epName)
	}
	return hnsEndpoint, err
}

func chkMgmtIPandEnableForwarding(networkName string, hnsEndpoint *hcsshim.HNSEndpoint, logger *logrus.Entry) error {
	netHelper := netsh.New(nil)
	var network *hcsshim.HNSNetwork
	var err error

	// Wait for the network to populate Management IP.
	for {
		network, err = hcsshim.GetHNSNetworkByName(networkName)
		if err != nil {
			logger.Errorf("Unable to get hns network %s after creation, error: %v", networkName, err)
			return err
		}

		if len(network.ManagementIP) > 0 {
			logger.Infof("Got managementIP %s", network.ManagementIP)
			break
		}
		time.Sleep(1 * time.Second)
		logger.Infof("Checking ManagementIP with HnsNetwork[%v]", network)
	}

	// Wait for the interface with the management IP
	for {
		if _, err = netHelper.GetInterfaceByIP(network.ManagementIP); err != nil {
			time.Sleep(1 * time.Second)
			logger.Infof("Checking interface with ip %s, err %v", network.ManagementIP, err)
			continue
		}
		break
	}

	// Enable forwarding on the host interface and endpoint
	for _, interfaceIpAddress := range []string{network.ManagementIP, hnsEndpoint.IPAddress.String()} {
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

func createAndAttachContainerEP(args *skel.CmdArgs,
	hnsNetwork *hcsshim.HNSNetwork,
	affineBlockSubnet *net.IPNet,
	allIPAMPools []*net.IPNet,
	natOutgoing bool,
	result *current.Result,
	n *hns.NetConf,
	logger *logrus.Entry) (*hcsshim.HNSEndpoint, error) {

	gatewayAddress := getNthIP(affineBlockSubnet, 2).String()

	pols, err := calculateEndpointPolicies(n, allIPAMPools, natOutgoing, logger)
	if err != nil {
		return nil, err
	}

	endpointName := hns.ConstructEndpointName(args.ContainerID, args.Netns, n.Name)
	logger.Infof("Attempting to create HNS endpoint name : %s for container", endpointName)
	hnsEndpointCont, err := hns.ProvisionEndpoint(endpointName, hnsNetwork.Id, args.ContainerID, func() (*hcsshim.HNSEndpoint, error) {
		hnsEP := &hcsshim.HNSEndpoint{
			Name:           endpointName,
			VirtualNetwork: hnsNetwork.Id,
			GatewayAddress: gatewayAddress,
			IPAddress:      result.IPs[0].Address.IP,
			Policies:       pols,
		}
		return hnsEP, nil
	})
	logger.Infof("Endpoint to container created! %v", hnsEndpointCont)
	return hnsEndpointCont, err
}

type policyMarshaller interface {
	MarshalPolicies() []json.RawMessage
}

// calculateEndpointPolicies augments the hns.Netconf policies with NAT exceptions for our IPAM blocks.
func calculateEndpointPolicies(
	n policyMarshaller,
	allIPAMPools []*net.IPNet,
	natOutgoing bool,
	logger *logrus.Entry,
) ([]json.RawMessage, error) {
	inputPols := n.MarshalPolicies()
	var outputPols []json.RawMessage
	for _, inPol := range inputPols {
		// Decode the raw policy as a dict so we can inspect it without losing any fields.
		decoded := map[string]interface{}{}
		err := json.Unmarshal(inPol, decoded)
		if err != nil {
			logger.WithError(err).Error("MarshalPolicies() returned bad JSON")
			return nil, err
		}

		// We're looking for an entry like this:
		//
		// {
		//   "Type":  "OutBoundNAT",
		//   "ExceptionList":  [
		//     "10.96.0.0/12"
		//   ]
		// }
		// We'll add the other IPAM pools to the list.
		outPol := inPol
		if strings.EqualFold(decoded["Type"].(string), "OutBoundNAT") {
			if !natOutgoing {
				logger.Info("NAT-outgoing disabled for this IP pool, ignoring OutBoundNAT policy from NetConf.")
				continue
			}

			excList, _ := decoded["ExceptionList"].([]interface{})
			for _, poolCIDR := range allIPAMPools {
				excList = append(excList, poolCIDR.String())
			}
			decoded["ExceptionList"] = excList
			outPol, err = json.Marshal(decoded)
			if err != nil {
				logger.WithError(err).Error("Failed to add outbound NAT exclusion.")
				return nil, err
			}
			logger.WithField("policy", string(outPol)).Debug(
				"Updated OutBoundNAT policy to add Calico IP pools.")
		}
		outputPols = append(outputPols, outPol)
	}
	return outputPols, nil
}

// This func increments the subnet IP address by n depending on
// endpoint IP or gateway IP
func getNthIP(PodCIDR *net.IPNet, n int) net.IP {
	gwaddr := PodCIDR.IP.To4()
	buffer := make([]byte, len(gwaddr))
	copy(buffer, gwaddr)
	buffer[3] += byte(n)
	return buffer
}

func createNetworkName(netName string, subnet *net.IPNet) string {
	str := subnet.IP.String()
	network := strings.Replace(str, ".", "-", -1)
	name := netName + "-" + network
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

	err = hns.DeprovisionEndpoint(epName, args.Netns, args.ContainerID)
	if err != nil && strings.Contains(err.Error(), "not found") {
		logger.WithError(err).Warn("Endpoint not found during delete, assuming it's already been cleaned up")
		return nil
	}
	return err
}
