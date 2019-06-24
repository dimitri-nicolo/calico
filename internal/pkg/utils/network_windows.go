// Copyright (c) 2018-2019 Tigera, Inc. All rights reserved.

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
	"github.com/Microsoft/hcsshim/hcn"
	"github.com/buger/jsonparser"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/plugins/pkg/hns"
	"github.com/juju/clock"
	"github.com/juju/errors"
	"github.com/juju/mutex"
	"github.com/projectcalico/cni-plugin/internal/pkg/utils/winpol"
	"github.com/projectcalico/cni-plugin/pkg/types"
	calicoclient "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/rakelkar/gonetsh/netsh"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	utilexec "k8s.io/utils/exec"
)

var (
	// IPv4AllNet represents the IPv4 all-addresses CIDR 0.0.0.0/0.
	IPv4AllNet *net.IPNet
	// IPv6AllNet represents the IPv6 all-addresses CIDR ::/0.
	IPv6AllNet    *net.IPNet
	DefaultRoutes []*net.IPNet
)

const (
	DefaultVNI = 4096
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
	_ = desiredVethName
	if len(routes) > 0 {
		logrus.WithField("routes", routes).Debug("Ignoring in-container routes; not supported on Windows.")
	}

	podIP, subNet, _ := net.ParseCIDR(result.IPs[0].Address.String())

	n, _, err := loadNetConf(args.StdinData)
	if err != nil {
		logger.Errorf("Error loading args")
		return "", "", err
	}

	// Assigning DNS details read from RuntimeConfig or cni.conf to result
	// If DNS details is present in the RuntimeConfig, then DNS details of RuntimeConfig will take precedence over cni.conf DNS
	if len(conf.RuntimeConfig.DNS.Nameservers) >= 1 {
		result.DNS.Nameservers = conf.RuntimeConfig.DNS.Nameservers
		result.DNS.Domain = conf.RuntimeConfig.DNS.Domain
		result.DNS.Search = conf.RuntimeConfig.DNS.Search
		result.DNS.Options = conf.RuntimeConfig.DNS.Options
	} else {
		result.DNS = n.DNS
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
		logger.Errorf("Unable to acquire lock")
		return "", "", err
	}
	defer m.Release()

	// Create hns network
	var networkName string
	if conf.WindowsUseSingleNetwork {
		logger.WithField("name", conf.Name).Info(
			"Overriding network name, only a single IPAM block will be supported on this host")
		networkName = conf.Name
	} else {
		networkName = CreateNetworkName(n.Name, subNet)
	}

	var hnsNetwork *hcsshim.HNSNetwork
	if conf.Mode == "vxlan" {
		hnsNetwork, err = ensureVxlanNetworkExists(networkName, subNet, conf, result, logger)
	} else {
		hnsNetwork, err = EnsureNetworkExists(networkName, subNet, result, logger)
	}
	if err != nil {
		logger.Errorf("Unable to create hns network %s", networkName)
		return "", "", err
	}

	// Create host hns endpoint
	epName := networkName + "_ep"
	var hnsEndpoint *hcsshim.HNSEndpoint
	if conf.Mode == "vxlan" {
		hnsEndpoint, err = CreateAndAttachVxlanHostEP(epName, hnsNetwork, subNet, conf, result, logger)
	} else {
		hnsEndpoint, err = CreateAndAttachHostEP(epName, hnsNetwork, subNet, result, logger)
	}
	if err != nil {
		logger.Errorf("Unable to create host hns endpoint %s", epName)
		return "", "", err
	}

	// Check for management ip getting assigned to the network, interface with the management ip
	// and then enable forwarding on management interface as well as endpoint.
	// Update the hnsNetwork variable with management ip
	if conf.Mode != "vxlan" {
		hnsNetwork, err = chkMgmtIPandEnableForwarding(networkName, hnsEndpoint, logger)
		if err != nil {
			logger.Errorf("Failed to enable forwarding : %v", err)
			return "", "", err
		}
	}

	// Create endpoint for container
	hnsEndpointCont, err := createAndAttachContainerEP(args, hnsNetwork, subNet, allIPAMPools, natOutgoing, result, &conf, n, logger)
	if err != nil {
		logger.Errorf("Unable to create container hns endpoint %s", epName)
		return "", "", err
	}

	// The Priority is set to the largest value (65500) with which
	// the ACL policy when applied on windows works.
	// The Priority has to be set to a large enough value for the default
	// policy so that if other policies exist then they take precedence.
	if conf.WindowsDisableDefaultDenyAllPolicy == false {
		defaultDenyAllACL := &hcsshim.ACLPolicy{
			Id:        "CNIDefaultDenyAllPolicy",
			Type:      hcsshim.ACL,
			RuleType:  hcsshim.Switch,
			Action:    hcsshim.Block,
			Direction: hcsshim.In,
			Protocol:  256,
			Priority:  65500,
		}

		err := hnsEndpointCont.ApplyACLPolicy(defaultDenyAllACL)
		if err != nil {
			logger.Errorf("Error applying ACL policy DenyAll")
			return "", "", err
		}
	}
	contVethMAC = hnsEndpointCont.MacAddress
	return hostVethName, contVethMAC, err
}

func EnsureVXLANTunnelAddr(ctx context.Context, calicoClient calicoclient.Interface, nodeName string, ipNet *net.IPNet, conf types.NetConf) error {
	logrus.Debug("Checking the node's VXLAN tunnel address")
	var updateRequired bool
	node, err := calicoClient.Nodes().Get(ctx, nodeName, options.GetOptions{})
	if err != nil {
		return err
	}

	expectedIP := getNthIP(ipNet, 1).String()
	if node.Spec.IPv4VXLANTunnelAddr != expectedIP {
		logrus.WithField("ip", expectedIP).Debug("VXLAN tunnel IP to be updated")
		updateRequired = true
	}

	var networkName string
	if conf.WindowsUseSingleNetwork {
		networkName = conf.Name
	} else {
		networkName = CreateNetworkName(conf.Name, ipNet)
	}

	mac, err := GetDRMACAddr(networkName, ipNet)
	if err != nil {
		return err
	}
	expectedMAC := mac.String()
	if node.Spec.VXLANTunnelMACAddr != expectedMAC {
		logrus.WithField("mac", expectedMAC).Debug("VXLAN tunnel MAC to be updated")
		updateRequired = true
	}

	if updateRequired == false {
		return nil
	}

	node.Spec.IPv4VXLANTunnelAddr = expectedIP
	node.Spec.VXLANTunnelMACAddr = expectedMAC
	_, err = calicoClient.Nodes().Update(ctx, node, options.SetOptions{})
	return err
}

func GetDRMACAddr(networkName string, subNet *net.IPNet) (net.HardwareAddr, error) {
	hnsNetwork, err := hcsshim.GetHNSNetworkByName(networkName)
	if err != nil {
		logrus.Infof("hns network %s not found", networkName)
		return nil, err
	}

	hcnNetwork, err := hcn.GetNetworkByName(networkName)
	if err != nil {
		logrus.Infof("hcn network %s not found", networkName)
		return nil, err
	}

	err = hcn.RemoteSubnetSupported()
	if err != nil {
		logrus.Infof("remote subnet not supported")
		return nil, err
	}

	var remoteDRMAC string
	var providerAddress string
	logrus.Infof("Checking HNS network for DR MAC : [%+v]", hnsNetwork)
	for _, policy := range hcnNetwork.Policies {
		logrus.Infof("inside for loop. policy = [%+v]", policy)
		if policy.Type == hcn.DrMacAddress {
			logrus.Infof("policy type is drmacaddress")
			policySettings := hcn.DrMacAddressNetworkPolicySetting{}
			err = json.Unmarshal(policy.Settings, &policySettings)
			if err != nil {
				return nil, fmt.Errorf("Failed to unmarshal settings")
			}
			remoteDRMAC = policySettings.Address
			logrus.Infof("remote dr mac = %v", remoteDRMAC)
		}
		if policy.Type == hcn.ProviderAddress {
			logrus.Infof("policy type is provideraddress")
			policySettings := hcn.ProviderAddressEndpointPolicySetting{}
			err = json.Unmarshal(policy.Settings, &policySettings)
			if err != nil {
				return nil, fmt.Errorf("Failed to unmarshal settings")
			}
			providerAddress = policySettings.ProviderAddress
			logrus.Infof("providerAddress = %v", providerAddress)
		}
	}
	if providerAddress != hnsNetwork.ManagementIP {
		logrus.Infof("Cannot use DR MAC %v since PA %v does not match %v", remoteDRMAC, providerAddress, hnsNetwork.ManagementIP)
		remoteDRMAC = ""
	}

	if len(providerAddress) == 0 {
		return nil, fmt.Errorf("Cannot find network with Management IP %v", hnsNetwork.ManagementIP)
	}
	if len(remoteDRMAC) == 0 {
		return nil, fmt.Errorf("Could not find remote DR MAC for Management IP %v", hnsNetwork.ManagementIP)
	}
	mac, err := net.ParseMAC(string(remoteDRMAC))
	if err != nil {
		return nil, fmt.Errorf("Cannot parse DR MAC %v: %+v", remoteDRMAC, err)
	}

	logrus.Infof("mac address = %v", mac)
	return mac, nil
}

func lookupIPAMPools(
	ctx context.Context, podIP net.IP, calicoClient calicoclient.Interface,
) (
	cidrs []*net.IPNet,
	natOutgoing bool,
	err error,
) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

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

func ensureVxlanNetworkExists(networkName string, subNet *net.IPNet, conf types.NetConf, result *current.Result, logger *logrus.Entry) (*hcsshim.HNSNetwork, error) {
	var err error
	createNetwork := true
	expectedAddressPrefix := subNet.String()
	expectedGW := getNthIP(subNet, 1)
	var expectedVNI uint64

	expectedNetwork := &hcsshim.HNSNetwork{
		Name:    networkName,
		Type:    "Overlay",
		Subnets: make([]hcsshim.Subnet, 0, 1),
	}

	if conf.VXLANVNI == 0 {
		expectedVNI = DefaultVNI
	} else if conf.VXLANVNI < DefaultVNI {
		return nil, fmt.Errorf("Windows does not support VXLANVNI < 4096")
	} else {
		expectedVNI = conf.VXLANVNI
	}

	// Checking if HNS network exists
	existingNetwork, _ := hcsshim.GetHNSNetworkByName(networkName)
	if existingNetwork != nil {
		if existingNetwork.Type == expectedNetwork.Type {
			for _, subnet := range existingNetwork.Subnets {
				if subnet.AddressPrefix == expectedAddressPrefix && subnet.GatewayAddress == expectedGW.String() {
					createNetwork = false
					logger.Infof("Found existing HNS network [%+v]", existingNetwork)
					break
				}
			}
		}
	}

	if createNetwork {
		// Delete stale network
		if existingNetwork != nil {
			if _, err := existingNetwork.Delete(); err != nil {
				logger.Errorf("Unable to delete existing network [%v], error: %v", existingNetwork.Name, err)
				return nil, err
			}
			logger.Infof("Deleted stale HNS network [%v]")
		}

		// Add a VxLan subnet
		expectedNetwork.Subnets = append(expectedNetwork.Subnets, hcsshim.Subnet{
			AddressPrefix:  expectedAddressPrefix,
			GatewayAddress: expectedGW.String(),
			Policies: []json.RawMessage{
				[]byte(fmt.Sprintf(`{"Type":"VSID","VSID":%d}`, expectedVNI)),
			},
		})

		// Config request params
		jsonRequest, err := json.Marshal(expectedNetwork)
		if err != nil {
			return nil, errors.Annotatef(err, "failed to marshal %+v", expectedNetwork)
		}

		logger.Infof("Attempting to create HNSNetwork %s", string(jsonRequest))
		newNetwork, err := hcsshim.HNSNetworkRequest("POST", "", string(jsonRequest))
		if err != nil {
			return nil, errors.Annotatef(err, "failed to create HNSNetwork %s", networkName)
		}

		var waitErr, lastErr error
		// Wait for the network to populate Management IP
		logger.Infof("Waiting to get ManagementIP from HNSNetwork %s", networkName)
		waitErr = wait.Poll(500*time.Millisecond, 5*time.Second, func() (done bool, err error) {
			newNetwork, lastErr = hcsshim.HNSNetworkRequest("GET", newNetwork.Id, "")
			return newNetwork != nil && len(newNetwork.ManagementIP) != 0, nil
		})
		if waitErr == wait.ErrWaitTimeout {
			return nil, errors.Annotatef(lastErr, "timeout, failed to get management IP from HNSNetwork %s", networkName)
		}

		// Wait for the interface with the management IP
		netshHelper := netsh.New(utilexec.New())
		logger.Infof("Waiting to get net interface for HNSNetwork %s (%s)", networkName, newNetwork.ManagementIP)
		waitErr = wait.Poll(500*time.Millisecond, 5*time.Second, func() (done bool, err error) {
			_, lastErr = netshHelper.GetInterfaceByIP(newNetwork.ManagementIP)
			return lastErr == nil, nil
		})
		if waitErr == wait.ErrWaitTimeout {
			return nil, errors.Annotatef(lastErr, "timeout, failed to get net interface for HNSNetwork %s (%s)", networkName, newNetwork.ManagementIP)
		}

		logger.Infof("Created HNSNetwork %s", networkName)
		existingNetwork = newNetwork
	}

	existingNetworkV2, err := hcn.GetNetworkByID(existingNetwork.Id)
	if err != nil {
		return nil, errors.Annotatef(err, "Could not find vxlan0 in V2")
	}

	addHostRoute := true
	for _, policy := range existingNetworkV2.Policies {
		if policy.Type == hcn.HostRoute {
			addHostRoute = false
		}
	}
	if addHostRoute {
		hostRoutePolicy := hcn.NetworkPolicy{
			Type:     hcn.HostRoute,
			Settings: []byte("{}"),
		}

		networkRequest := hcn.PolicyNetworkRequest{
			Policies: []hcn.NetworkPolicy{hostRoutePolicy},
		}
		err := existingNetworkV2.AddPolicy(networkRequest)
		if err != nil {
			logger.Warnf("Error adding policy to network : %v", err)
		}
	}

	return existingNetwork, nil
}

func EnsureNetworkExists(networkName string, subNet *net.IPNet, result *current.Result, logger *logrus.Entry) (*hcsshim.HNSNetwork, error) {
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
			"DNSServerList": strings.Join(result.DNS.Nameservers, ","),
			"DNSSuffix":     strings.Join(result.DNS.Search, ","),
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

func CreateAndAttachVxlanHostEP(epName string, hnsNetwork *hcsshim.HNSNetwork, subNet *net.IPNet, conf types.NetConf, result *current.Result, logger *logrus.Entry) (*hcsshim.HNSEndpoint, error) {
	var err error
	endpointAddress := getNthIP(subNet, 2)

	// 1. Check if the HNSEndpoint exists and has the expected settings
	existingEndpoint, err := hcsshim.GetHNSEndpointByName(epName)
	if err == nil && existingEndpoint.VirtualNetwork == hnsNetwork.Id {
		// Check policies if there is PA type
		targetType := "PA"
		for _, policy := range existingEndpoint.Policies {
			policyType, _ := jsonparser.GetUnsafeString(policy, "Type")
			if policyType == targetType {
				actualPaIP, _ := jsonparser.GetUnsafeString(policy, targetType)
				if actualPaIP == hnsNetwork.ManagementIP {
					logger.Infof("Found existing remote HNSEndpoint %s", epName)
					return existingEndpoint, nil
				}
			}
		}
	}

	// 2. Create a new HNSNetwork
	if existingEndpoint != nil {
		if _, err := existingEndpoint.Delete(); err != nil {
			return nil, errors.Annotatef(err, "failed to delete existing remote HNSEndpoint %s", epName)
		}
		logger.Infof("Deleted stale HNSEndpoint %s", epName)
	}

	macAddr := GetMacAddr(hnsNetwork.ManagementIP)

	newEndpoint := &hcsshim.HNSEndpoint{
		Name:             epName,
		IPAddress:        endpointAddress,
		MacAddress:       macAddr,
		VirtualNetwork:   hnsNetwork.Id,
		IsRemoteEndpoint: true,
		Policies: []json.RawMessage{
			[]byte(fmt.Sprintf(`{"Type":"PA","PA":"%s"}`, hnsNetwork.ManagementIP)),
		},
	}
	if _, err := newEndpoint.Create(); err != nil {
		return nil, errors.Annotatef(err, "failed to create remote HNSEndpoint %s", epName)
	}
	logger.Infof("Created HNSEndpoint %s", epName)

	return newEndpoint, nil
}

func CreateAndAttachHostEP(epName string, hnsNetwork *hcsshim.HNSNetwork, subNet *net.IPNet, result *current.Result, logger *logrus.Entry) (*hcsshim.HNSEndpoint, error) {
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

func chkMgmtIPandEnableForwarding(networkName string, hnsEndpoint *hcsshim.HNSEndpoint, logger *logrus.Entry) (network *hcsshim.HNSNetwork, err error) {
	netHelper := netsh.New(nil)

	startTime := time.Now()
	logCxt := logger.WithField("network", networkName)

	// Wait for the network to populate Management IP and for it to match one of the host interfaces.
	for {
		// Look up the network afresh each time, in case the management IP changes.
		network, err = hcsshim.GetHNSNetworkByName(networkName)
		if err != nil {
			logger.Errorf("Unable to get hns network %s after creation, error: %v", networkName, err)
			return nil, err
		}

		if time.Since(startTime) > 30*time.Second {
			return nil, fmt.Errorf(
				"timed out waiting for interface matching the management IP (%v) of network %s",
				network.ManagementIP, networkName)
		}

		if len(network.ManagementIP) == 0 {
			logCxt.Info("Waiting for management IP...")
			time.Sleep(1 * time.Second)
			continue
		}

		if mgmtIface, err := netHelper.GetInterfaceByIP(network.ManagementIP); err != nil {
			logCxt.WithField("ip", network.ManagementIP).WithError(err).Warn(
				"Waiting for interface matching management IP...")
			time.Sleep(1 * time.Second)
			continue
		} else {
			err := enableForwarding(netHelper, mgmtIface, logger)
			if err != nil {
				return nil, err
			}
		}

		break
	}

	ourEpAddr := hnsEndpoint.IPAddress.String()
	netInterface, err := netHelper.GetInterfaceByIP(ourEpAddr)
	if err != nil {
		logger.WithError(err).Errorf("Unable to find interface matching our host endpoint [%v]", ourEpAddr)
		return nil, err
	}

	logger.Infof("Found Interface with IP[%s]: %v", ourEpAddr, netInterface)
	err = enableForwarding(netHelper, netInterface, logger)
	if err != nil {
		return nil, err
	}

	return network, nil
}

func enableForwarding(netHelper netsh.Interface, netInterface netsh.Ipv4Interface, logger *logrus.Entry) error {
	interfaceIdx := strconv.Itoa(netInterface.Idx)
	if err := netHelper.EnableForwarding(interfaceIdx); err != nil {
		logger.WithError(err).Errorf("Unable to enable forwarding on [%v] index [%v]",
			netInterface.Name, interfaceIdx)
		return err
	}
	logger.Infof("Enabled forwarding on [%v] index [%v]", netInterface.Name, interfaceIdx)
	return nil
}

func createAndAttachContainerEP(args *skel.CmdArgs,
	hnsNetwork *hcsshim.HNSNetwork,
	affineBlockSubnet *net.IPNet,
	allIPAMPools []*net.IPNet,
	natOutgoing bool,
	result *current.Result,
	ourNetconf *types.NetConf,
	n *hns.NetConf,
	logger *logrus.Entry) (*hcsshim.HNSEndpoint, error) {

	var gatewayAddress string
	if ourNetconf.Mode == "vxlan" {
		gatewayAddress = getNthIP(affineBlockSubnet, 1).String()
	} else {
		gatewayAddress = getNthIP(affineBlockSubnet, 2).String()
	}

	natExclusions := allIPAMPools

	mgmtIP := net.ParseIP(hnsNetwork.ManagementIP)
	if len(mgmtIP) == 0 {
		// We just checked the management IP so we shouldn't lose it again.
		return nil, fmt.Errorf("HNS network lost its management IP")
	}

	if natOutgoing {
		logger.Debug("Looking up management subnet to add outgoing NAT exclusion.")
		mgmtNet, err := lookupManagementAddr(mgmtIP, logger)
		if err != nil {
			return nil, err
		}
		if !ourNetconf.WindowsDisableHostSubnetNATExclusion {
			natExclusions = make([]*net.IPNet, len(allIPAMPools)+1)
			copy(natExclusions, allIPAMPools)
			natExclusions[len(natExclusions)-1] = mgmtNet
		}
	}

	pols, err := winpol.CalculateEndpointPolicies(n, natExclusions, natOutgoing, mgmtIP, logger)
	if err != nil {
		return nil, err
	}

	endpointName := hns.ConstructEndpointName(args.ContainerID, args.Netns, n.Name)
	epIP := result.IPs[0].Address.IP
	epIPBytes := epIP.To4()
	macAddr := ""

	if ourNetconf.Mode == "vxlan" {
		if len(ourNetconf.VXLANMacPrefix) != 0 {
			if len(ourNetconf.VXLANMacPrefix) != 5 || ourNetconf.VXLANMacPrefix[2] != '-' {
				return nil, fmt.Errorf("endpointMacPrefix [%v] is invalid, value must be of the format xx-xx", ourNetconf.VXLANMacPrefix)
			}
		} else {
			ourNetconf.VXLANMacPrefix = "0E-2A"
		}
		// conjure a MAC based on the IP for Overlay
		macAddr = fmt.Sprintf("%v-%02x-%02x-%02x-%02x", ourNetconf.VXLANMacPrefix, epIPBytes[0], epIPBytes[1], epIPBytes[2], epIPBytes[3])

		pols = append(pols, []json.RawMessage{
			[]byte(fmt.Sprintf(`{"Type":"PA","PA":"%s"}`, hnsNetwork.ManagementIP)),
		}...)

	}

	attempts := 3
	for {
		logger.Infof("Attempting to create HNS endpoint name : %s for container", endpointName)
		hnsEndpointCont, err := hns.ProvisionEndpoint(endpointName, hnsNetwork.Id, args.ContainerID, func() (*hcsshim.HNSEndpoint, error) {
			hnsEP := &hcsshim.HNSEndpoint{
				Name:           endpointName,
				VirtualNetwork: hnsNetwork.Id,
				DNSServerList:  strings.Join(result.DNS.Nameservers, ","),
				DNSSuffix:      strings.Join(result.DNS.Search, ","),
				GatewayAddress: gatewayAddress,
				IPAddress:      epIP,
				MacAddress:     macAddr,
				Policies:       pols,
			}
			return hnsEP, nil
		})
		if err != nil {
			logger.WithError(err).Error("Error provisioning endpoint, checking if we need to clean it up.")

			// If a previous call failed here, before cleaning up, it may have left an orphaned endpoint.  Check for that
			// and clean up.
			cleanupErr := cleanUpEndpointByIP(epIP, logger)
			if cleanupErr != nil {
				logger.WithError(err).Error("Failed to clean up (by IP) after failure.")
			} else {
				logger.Info("Cleanup (by IP) succeeded.")
			}

			// If provision endpoint fails at the attach stage, we can be left with an orphaned endpoint.  Check for
			// that and clean it up.
			cleanupErr = cleanUpEndpointByName(endpointName, logger)
			if cleanupErr != nil {
				logger.WithError(err).Error("Failed to clean up (by name) after failure.")
			} else {
				logger.Info("Cleanup (by name) succeeded.")
			}

			if attempts > 0 {
				// Cleanup may have unblocked another attempt, see if we can retry...
				logger.Info("Retrying...")
				attempts--
				time.Sleep(time.Second)
				continue
			}
			return nil, err
		}

		logger.Infof("Endpoint to container created! %v", hnsEndpointCont)
		return hnsEndpointCont, nil
	}
}

func cleanUpEndpointByIP(IP net.IP, logger *logrus.Entry) error {
	endpoints, err := hcsshim.HNSListEndpointRequest()
	if err != nil {
		logger.WithError(err).Error("Failed to list endpoints")
		return err
	}
	for _, ep := range endpoints {
		if ep.IPAddress.Equal(IP) {
			logger.WithField("conflictingEndpoint", ep).Error("Found pre-existing conflicting endpoint.")
			_, err := ep.Delete()
			if err != nil {
				logger.WithError(err).Error("Failed to delete old endpoint")
			}
			return err // Exit early since there can be only one endpoint with the same IP.
		}
	}
	return nil
}

func cleanUpEndpointByName(endpointName string, logger *logrus.Entry) error {
	hnsEndpoint, err := hcsshim.GetHNSEndpointByName(endpointName)
	if hcsshim.IsNotExist(err) {
		logger.Debug("Endpoint already gone.  Nothing to do.")
		return nil
	}
	if err != nil {
		logger.WithError(err).Error("Failed to get endpoint for cleanup.")
		return err
	}

	_, err = hnsEndpoint.Delete()
	return err
}

func lookupManagementAddr(mgmtIP net.IP, logger *logrus.Entry) (*net.IPNet, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		logger.WithError(err).Error("Failed to look up host interfaces")
		return nil, err
	}

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			logger.WithError(err).WithField("iface", iface.Name).Error(
				"Failed to look up host interface addresses")
			return nil, err
		}
		for _, addr := range addrs {
			if ipAddr, ok := addr.(*net.IPNet); ok {
				if ipAddr.Contains(mgmtIP) {
					return ipAddr, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("couldn't find an interface matching management IP %s", mgmtIP.String())
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

func CreateNetworkName(netName string, subnet *net.IPNet) string {
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

// NetworkApplicationContainer tries to attach the application container to the endpoint that is attached to its pause container.
// On failure, it returns the error.
// This is done so that the DNS details are reflected in the container.
func NetworkApplicationContainer(args *skel.CmdArgs) error {

	n, _, err := loadNetConf(args.StdinData)
	hnsEndpointName := hns.ConstructEndpointName(args.ContainerID, args.Netns, n.Name)

	hnsEndpoint, err := hcsshim.GetHNSEndpointByName(hnsEndpointName)
	if err != nil {
		logrus.Errorf("Endpoint does not exist with hns endpoint name: %v\n ", hnsEndpointName)
		return err
	}

	if err = hcsshim.HotAttachEndpoint(args.ContainerID, hnsEndpoint.Id); err != nil {
		if err == hcsshim.ErrComputeSystemDoesNotExist {
			// kubelet Windows uses ADD CmdArgs to get pod status. It is possible for Calico CNI to recieve an ADD after application container has completed and been removed from runtime.
			// In that case, return nil to allow Calico CNI to return good pod status to kubelet.
			return nil
		}
		logrus.Errorf("Failed to attach hns endpoint: %s to container: %v\n ", hnsEndpoint, args.ContainerID)
		return err
	}

	return nil
}

func updateHostLocalIPAMDataForOS(subnet string, ipamData map[string]interface{}) error {
	return UpdateHostLocalIPAMDataForWindows(subnet, ipamData)
}

// GetMacAddr gets the MAC hardware
// address of the host machine
func GetMacAddr(mgmtIp string) (addr string) {
	interfaces, err := net.Interfaces()
	if err == nil {
	outerLoop:
		for _, i := range interfaces {
			addrs, err := i.Addrs()
			if err == nil {
				for _, j := range addrs {
					ip := strings.Split(j.String(), "/")
					if strings.Compare(ip[0], mgmtIp) == 0 {
						addr = i.HardwareAddr.String()
						break outerLoop
					}
				}
			}
		}
	}
	return
}
