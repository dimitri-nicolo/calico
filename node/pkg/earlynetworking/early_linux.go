// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package earlynetworking

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

const (
	BIRD_CONFIG_FILE = "/etc/calico/confd/config/bird.cfg"
	BIRD_CONFIG_MAIN = `
router id %v;
listen bgp port 8179;

protocol direct {
    interface "*";
}

protocol kernel {
    learn;            # Learn all alien routes from the kernel
    scan time 5;      # Scan kernel routing table every 5 seconds
    import all;       # Default is import all
    export all;       # Default is export none
    merge paths on;
}

# This pseudo-protocol watches all interface up/down events.
protocol device {
    scan time 5;      # Scan interfaces every 5 seconds
}

filter stable_address_only {
  if ( net = %v/32 ) then { accept; }
  reject;
}

template bgp tors {
  description "Connection to ToR";
  local as %v;
  direct;
  gateway recursive;
  import all;
  export filter stable_address_only;
  add paths on;
  connect delay time 2;
  connect retry time 5;
  error wait time 5,30;
  next hop self;
}
`
	BIRD_CONFIG_PER_PEER = `
protocol bgp tor%v from tors {
  neighbor %v as %v;
}
`
)

// Do setup for a dual ToR node, then run as the "early BGP" daemon
// until calico-node's BIRD can take over.
func Run() {
	logrus.Info("Beginning dual ToR setup for this node")

	// There must be a YAML file mapped in at $CALICO_EARLY_NETWORKING that defines addresses
	// and AS numbers for the nodes in this cluster.  Read that file.
	cfg, err := GetEarlyNetworkConfig(os.Getenv("CALICO_EARLY_NETWORKING"))
	if err != nil {
		logrus.WithError(err).Fatal("Failed to read EarlyNetworkConfiguration")
	}

	// Find the source address that the default route will use, which must be one of the
	// per-interface addresses.  We will use this to identify this node in the overall YAML
	// config, and as the router ID in BIRD config.
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to list routes")
	}
	var routerID net.IP
	for _, route := range routes {
		if isDefaultCIDR(route.Dst) {
			logrus.Infof("Got default route %+v", route)
			if route.Src != nil {
				logrus.Infof("Default route has source address %v", route.Src)
				routerID = route.Src
				goto gotRouterID
			} else if route.Gw != nil {
				logrus.Infof("Default route has gateway address %v", route.Gw)
				// Look up routes to the gateway address.
				routes, err = netlink.RouteGet(route.Gw)
				if err != nil {
					logrus.WithError(err).Fatal("Failed to get routes to gateway address")
				}
				for _, route = range routes {
					logrus.Infof("Got gateway address route %+v", route)
					if route.Src != nil {
						routerID = route.Src
						goto gotRouterID
					}
				}
			}
		}
		logrus.Infof("Skip other route %+v", route)
	}
	if routerID == nil {
		logrus.Fatal("Failed to find default route with source address")
	}
gotRouterID:
	logrus.Infof("Router ID is %v", routerID)

	// Find the entry from the YAML config for this node.
	var thisNode *ConfigNode
nodeLoop:
	for _, nodeCfg := range cfg.Spec.Nodes {
		for _, addr := range nodeCfg.InterfaceAddresses {
			if addr == routerID.String() {
				thisNode = &nodeCfg
				break nodeLoop
			}
		}
	}
	if thisNode == nil {
		logrus.WithField("routerID", routerID).Fatal("Failed to find config for this node")
		return
	}
	logrus.WithField("cfg", *thisNode).Info("Found config for this node")

	// Configure the stable address.
	loopback, err := netlink.LinkByName("lo")
	if err != nil {
		logrus.WithError(err).Fatal("Failed to get loopback interface")
	}
	_, cidr, err := net.ParseCIDR(thisNode.StableAddress.Address + "/32")
	if err != nil {
		logrus.WithError(err).Fatalf("Failed to parse stable address CIDR %v/32", thisNode.StableAddress.Address)
	}
	err = netlink.AddrAdd(loopback, &netlink.Addr{IPNet: cidr})
	if err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "exists") {
			logrus.WithError(err).Fatalf("Failed to add stable address %v/32 to loopback device", thisNode.StableAddress.Address)
		}
	}

	var bootstrapIPs []string
	if strings.ToLower(cfg.Spec.Platform) == PlatformOpenShift {
		// Look up the IP of the bootstrap node.  On nodes that are directly connected to
		// the bootstrap node, we want to create a specific route to ensure that we will use
		// our stable address as the source.
		bootstrapIPs, err = net.LookupHost("bootstrap")
		logrus.WithError(err).Infof("DNS lookup for bootstrap node returned %v", bootstrapIPs)
	}

	// Change interface-specific addresses to be scope link, and create specific routes where
	// directly connected to a bootstrap IP.
	ensureNodeAddressesAndRoutes(thisNode, bootstrapIPs)

	// Use multiple ECMP paths based on hashing 5-tuple.  These are not necessarily fatal, if
	// setting fails.
	err = writeProcSys("/proc/sys/net/ipv4/fib_multipath_hash_policy", "1")
	if err != nil {
		logrus.WithError(err).Warning("Failed to set fib_multipath_hash_policy")
	}
	err = writeProcSys("/proc/sys/net/ipv4/fib_multipath_use_neigh", "1")
	if err != nil {
		logrus.WithError(err).Warning("Failed to set fib_multipath_use_neigh")
	}

	// Generate BIRD config.
	birdConfig := fmt.Sprintf(BIRD_CONFIG_MAIN, routerID, thisNode.StableAddress.Address, thisNode.ASNumber)
	for index, peering := range thisNode.Peerings {
		peerAS := peering.PeerASNumber
		if peerAS == 0 {
			// Default to same AS number as this node.
			peerAS = thisNode.ASNumber
		}
		birdConfig = birdConfig + fmt.Sprintf(BIRD_CONFIG_PER_PEER, index+1, peering.PeerIP, peerAS)
	}
	err = os.WriteFile(BIRD_CONFIG_FILE, []byte(birdConfig), 0644)
	if err != nil {
		logrus.WithError(err).Fatalf("Failed to write BIRD config at %v", BIRD_CONFIG_FILE)
	}

	// Start BIRD and check its status - e.g. in case we've generated invalid config.
	out, err := exec.Command("sv", "-w", "2", "start", "bird").CombinedOutput()
	if err != nil {
		logrus.WithError(err).Fatalf("Failed sv start bird:\n%v", string(out))
	}
	logrus.Infof("sv start bird:\n%v", string(out))

	// Loop deciding whether to run early BIRD or not.
	logrus.Info("Early networking set up; now monitoring BIRD")
	monitorOngoing(thisNode)
}

func isDefaultCIDR(dst *net.IPNet) bool {
	if dst == nil {
		return true
	}
	if ones, _ := dst.Mask.Size(); ones == 0 {
		return true
	}
	return false
}

func monitorOngoing(thisNode *ConfigNode) {
	// Channel used to signal when early BIRD is wanted, based on the state of normal BIRD.
	earlyBirdWantedC := make(chan bool)
	go monitorNormalBird(earlyBirdWantedC)

	periodicCheckC := time.NewTicker(10 * time.Second).C
	earlyBirdRunning := true
	var (
		earlyBirdCheckTicker  *time.Ticker
		earlyBirdCheckC       <-chan time.Time
		earlyBirdCheckRetries int
	)
	startCheckingEarlyBird := func() {
		earlyBirdCheckTicker = time.NewTicker(300 * time.Millisecond)
		earlyBirdCheckC = earlyBirdCheckTicker.C
		earlyBirdCheckRetries = 10
	}
	stopCheckingEarlyBird := func() {
		earlyBirdCheckTicker.Stop()
		earlyBirdCheckC = nil
	}
	startCheckingEarlyBird()
	for {
		select {
		case earlyBirdWanted := <-earlyBirdWantedC:
			if earlyBirdWanted && !earlyBirdRunning {
				logrus.Info("Restart early BGP")
				err := exec.Command("sv", "up", "bird").Run()
				if err != nil {
					logrus.WithError(err).Fatal("Failed sv up bird")
				}
				earlyBirdRunning = true
				startCheckingEarlyBird()
			} else if earlyBirdRunning && !earlyBirdWanted {
				logrus.Info("Stop early BGP")
				err := exec.Command("sv", "down", "bird").Run()
				if err != nil {
					logrus.WithError(err).Fatal("Failed sv down bird")
				}
				earlyBirdRunning = false
				stopCheckingEarlyBird()
			}
		case <-earlyBirdCheckC:
			if earlyBirdRunning {
				// Early BIRD should be running.  Check that it really is.
				if earlyBGPRunning() {
					logrus.Info("Early BGP is really running")
					stopCheckingEarlyBird()
					// We're good, and don't need to keep checking until earlyBirdWanted changes.
				} else {
					earlyBirdCheckRetries -= 1
					if earlyBirdCheckRetries > 0 {
						logrus.Infof("Early BGP not really running yet (retries=%v)", earlyBirdCheckRetries)
						// We'll check again when earlyBirdCheckC fires again.
					} else {
						logrus.Fatal("Early BGP failed to start running")
						// Bail out, then the calico-node early container will retry.
					}
				}
			} else {
				logrus.Info("Early BIRD shouldn't be running, so stop checking for it")
				stopCheckingEarlyBird()
				// We're good, and don't need to keep checking until earlyBirdWanted changes.
			}
		case <-periodicCheckC:
			// Recheck interface addresses and routes.
			ensureNodeAddressesAndRoutes(thisNode, nil)
		}
	}
}

func monitorNormalBird(earlyBirdWantedC chan<- bool) {
	periodicCheckC := time.NewTicker(10 * time.Second).C
	var gracefulTimeoutC <-chan time.Time
	normalBirdRunningRecorded := false
	for {
		select {
		case <-periodicCheckC:
			nowRunning := normalBGPRunning()
			if nowRunning {
				// Normal BIRD is up.
				if normalBirdRunningRecorded {
					// Was running, and still is: no change.
				} else if gracefulTimeoutC != nil {
					logrus.Info("Normal BGP restarted within graceful restart period")
					gracefulTimeoutC = nil
				} else {
					logrus.Info("Normal BGP has (re)started")
					earlyBirdWantedC <- false
				}
				normalBirdRunningRecorded = true
			} else {
				// Normal BIRD is not running.
				if normalBirdRunningRecorded {
					logrus.Info("Normal BGP stopped; wait for graceful restart period")
					gracefulTimeoutC = time.NewTimer(120 * time.Second).C
				}
				// Otherwise we already detected and handled that normal BIRD had
				// stopped.  Either we're now in the graceful restart period - in
				// which case the next event will be the timer firing when that
				// expires - or we're past that and normal BIRD has been stopped for
				// a long time.  Either way, there's no output event that we need to
				// generate right now.
				normalBirdRunningRecorded = false
			}
		case <-gracefulTimeoutC:
			logrus.Info("End of graceful restart period for normal BGP")
			earlyBirdWantedC <- true
			gracefulTimeoutC = nil
		}
	}
}

func ensureNodeAddressesAndRoutes(thisNode *ConfigNode, bootstrapIPs []string) {
	links, err := netlink.LinkList()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to list all links")
	}
	for _, link := range links {
		addrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
		if err != nil {
			logrus.WithError(err).Fatalf("Failed to list addresses for link %+v", link)
		}
		for _, addr := range addrs {
			for _, peering := range thisNode.Peerings {
				if sameSubnet(addr, peering.PeerIP) {
					ensureLinkAddressAndRoutes(link, addr, peering.PeerIP)
					break
				}
			}
			for _, bootstrapIP := range bootstrapIPs {
				if sameSubnet(addr, bootstrapIP) {
					_, ipNet, err := net.ParseCIDR(bootstrapIP + "/32")
					if err == nil {
						ensureRoute(&netlink.Route{
							Dst:       ipNet,
							LinkIndex: link.Attrs().Index,
							Type:      syscall.RTN_UNICAST,
							Table:     syscall.RT_TABLE_MAIN,
							Src:       net.ParseIP(thisNode.StableAddress.Address),
						})
					} else {
						logrus.WithError(err).Warningf("Failed to parse OpenShift bootstrap IP (%v)", bootstrapIP)
					}

				}
			}
		}
	}
}

func sameSubnet(addr netlink.Addr, peerIP string) bool {
	maskedAddr := addr.IP.Mask(addr.Mask)
	logrus.Debugf("Masked interface address %v -> %v", addr.IPNet, maskedAddr)
	maskedPeer := net.ParseIP(peerIP).Mask(addr.Mask)
	logrus.Debugf("Masked peer address %v -> %v", peerIP, maskedPeer)
	return maskedAddr.Equal(maskedPeer)
}

// Given an address and interface in the same subnet as a ToR
// address/prefix, update the address in the ways that we need for
// dual ToR operation, and ensure that we still have the routes that
// we'd expect through that interface.
func ensureLinkAddressAndRoutes(link netlink.Link, addr netlink.Addr, peerIP string) {
	if addr.Scope != int(netlink.SCOPE_LINK) {
		// Delete the given address and re-add it with scope link.
		err := netlink.AddrDel(link, &addr)
		if err != nil {
			logrus.WithError(err).Fatalf("Failed to delete address %+v", addr)
		}
		addr.Scope = int(netlink.SCOPE_LINK)
		err = netlink.AddrAdd(link, &addr)
		if err != nil {
			logrus.WithError(err).Fatalf("Failed to add address %+v", addr)
		}
	}

	// Ensure that the subnet route is present.
	prefix := *addr.IPNet
	prefix.IP = prefix.IP.Mask(prefix.Mask)
	ensureRoute(&netlink.Route{
		Dst:       &prefix,
		LinkIndex: link.Attrs().Index,
		Type:      syscall.RTN_UNICAST,
		Scope:     netlink.SCOPE_LINK,
		Table:     syscall.RT_TABLE_MAIN,
	})

	// Try to add a default route via the ToR.
	ensureRoute(&netlink.Route{
		Gw:        net.ParseIP(peerIP),
		LinkIndex: link.Attrs().Index,
		Type:      syscall.RTN_UNICAST,
		Table:     syscall.RT_TABLE_MAIN,
	})
}

func ensureRoute(route *netlink.Route) {
	err := netlink.RouteAdd(route)
	if err == nil {
		logrus.Infof("Added route: %+v", *route)
	} else if strings.Contains(strings.ToLower(err.Error()), "exists") {
		logrus.Debugf("Route already exists: %+v", *route)
	} else {
		logrus.Fatalf("Failed to add route %+v", *route)
	}
}

func writeProcSys(path, value string) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	_, err = f.Write([]byte(value))
	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}

func earlyBGPRunning() bool {
	// 00000000:1FF3, if present, indicates a process listening on port 8179.  (8179 = 0x1FF3)
	return tcpListenOn("00000000:1FF3", "Early BGP")
}

func normalBGPRunning() bool {
	// 00000000:00B3, if present, indicates a process listening on port 179.  (179 = 0xB3)
	return tcpListenOn("00000000:00B3", "Normal BGP")
}

func tcpListenOn(addrPort, description string) bool {
	// /proc/net/tcp shows TCP listens and connections.
	connFile, err := os.Open("/proc/net/tcp")
	if err != nil {
		logrus.WithError(err).Fatal("Failed to open /proc/net/tcp")
	}
	defer connFile.Close()

	scanner := bufio.NewScanner(connFile)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), addrPort) {
			logrus.Debugf("%v is running", description)
			return true
		}
	}
	err = scanner.Err()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to read /proc/net/tcp")
	}

	logrus.Debugf("%v is not running", description)
	return false
}
