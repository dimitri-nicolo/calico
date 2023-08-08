package calc_test

import (
	"fmt"
	"math"
	"strings"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/felix/calc"
	"github.com/projectcalico/calico/felix/dataplane/mock"
	"github.com/projectcalico/calico/felix/proto"
	libapiv3 "github.com/projectcalico/calico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/encap"
	. "github.com/projectcalico/calico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/calico/libcalico-go/lib/net"
)

// Values used in remote cluster testing.
var local = ""
var remoteA = "remote-a"
var remoteB = "remote-b"

var localClusterHost = "local-host"
var localClusterHost2 = "local-host-2"
var remoteClusterAHost = "remote-a-host"
var remoteClusterBHost = "remote-b-host"

var localClusterHostIPAddr = "192.168.0.1"
var localClusterHost2IPAddr = "192.168.1.1"
var remoteClusterAHostIPAddr = "192.168.0.2"
var remoteClusterBHostIPAddr = "192.168.0.3"

var localClusterHostMAC = "66:05:91:0f:93:57"
var localClusterHost2MAC = "66:67:b3:72:12:71"
var remoteClusterAHostMAC = "66:0b:75:83:64:51"
var remoteClusterBHostMAC = "66:ac:b1:ca:37:70"

// StateWithPool is a convenience function to help compose remote cluster testing states.
func StateWithPool(state State, cluster string, cidr string, flush bool) State {
	var kvp KVPair
	if cluster == "" {
		kvp = KVPair{
			Key: IPPoolKey{CIDR: mustParseNet(cidr)},
			Value: &IPPool{
				CIDR:      mustParseNet(cidr),
				VXLANMode: encap.Always,
			},
		}
	} else {
		kvp = KVPair{
			Key: RemoteClusterResourceKey{
				Cluster:     cluster,
				ResourceKey: ResourceKey{Kind: v3.KindIPPool, Name: cluster + "-ip-pool"},
			},
			Value: &v3.IPPool{
				TypeMeta: metav1.TypeMeta{
					Kind:       v3.KindIPPool,
					APIVersion: v3.GroupVersionCurrent,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: cluster + "-ip-pool",
				},
				Spec: v3.IPPoolSpec{
					CIDR:      cidr,
					VXLANMode: v3.VXLANModeAlways,
				},
			},
		}
	}

	routeUpdate := proto.RouteUpdate{
		Type:       proto.RouteType_CIDR_INFO,
		IpPoolType: proto.IPPoolType_VXLAN,
		Dst:        cidr,
	}

	newState := state.Copy()
	newState.DatastoreState = append(newState.DatastoreState, kvp)
	if flush {
		newState.ExpectedRoutes.Add(routeUpdate)
		if cluster == "" {
			newState.ExpectedEncapsulation.VxlanEnabled = true
		}
	}

	return newState
}

// StateWithBlock is a convenience function to help compose remote cluster testing states.
func StateWithBlock(state State, cluster string, cidr string, flush bool, poolType proto.IPPoolType, host string, hostIP string) State {
	keyName := host
	if cluster != "" {
		keyName = cluster + "/" + keyName
	}
	affinity := "host:" + keyName
	var kvp KVPair
	if cluster == "" {
		kvp = KVPair{
			Key: BlockKey{CIDR: mustParseNet(cidr)},
			Value: &AllocationBlock{
				CIDR:        mustParseNet(cidr),
				Affinity:    &affinity,
				Allocations: createAllocationsArray(cidr),
				Unallocated: createUnallocatedArray(cidr),
			}}
	} else {
		kvp = KVPair{
			Key: RemoteClusterResourceKey{
				Cluster:     cluster,
				ResourceKey: ResourceKey{Kind: libapiv3.KindIPAMBlock, Name: escapeCIDR(cidr)},
			},
			Value: &libapiv3.IPAMBlock{
				TypeMeta: metav1.TypeMeta{
					Kind:       libapiv3.KindIPAMBlock,
					APIVersion: v3.GroupVersionCurrent,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: escapeCIDR(cidr),
				},
				Spec: libapiv3.IPAMBlockSpec{
					CIDR:        cidr,
					Affinity:    &affinity,
					Allocations: createAllocationsArray(cidr),
					Unallocated: createUnallocatedArray(cidr),
				},
			},
		}
	}

	routeUpdate := proto.RouteUpdate{
		Type:        proto.RouteType_REMOTE_WORKLOAD,
		IpPoolType:  poolType,
		Dst:         cidr,
		DstNodeName: keyName,
		DstNodeIp:   hostIP,
	}

	newState := state.Copy()
	newState.DatastoreState = append(newState.DatastoreState, kvp)
	if flush {
		newState.ExpectedRoutes.Add(routeUpdate)
	}

	return newState
}

// StateWithNode is a convenience function to help compose remote cluster testing states.
func StateWithNode(state State, cluster string, host string, hostIP string, vxlanTunnelIP string) State {
	keyName := host
	if cluster != "" {
		keyName = cluster + "/" + host
	}
	kvp := KVPair{
		Key: ResourceKey{
			Kind: libapiv3.KindNode,
			Name: keyName,
		},
		Value: &libapiv3.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: host,
			},
			Spec: libapiv3.NodeSpec{
				BGP: &libapiv3.NodeBGPSpec{
					IPv4Address: hostIP + "/24",
				},
				IPv4VXLANTunnelAddr: vxlanTunnelIP,
			},
		},
	}

	routeType := proto.RouteType_REMOTE_HOST
	if host == localHostname {
		routeType = proto.RouteType_LOCAL_HOST
	}

	routeUpdate := proto.RouteUpdate{
		Type:        routeType,
		IpPoolType:  proto.IPPoolType_NONE,
		Dst:         hostIP + "/32",
		DstNodeName: keyName,
		DstNodeIp:   hostIP,
	}
	metadataUpdate := proto.HostMetadataV4V6Update{
		Hostname: keyName,
		Ipv4Addr: hostIP + "/24",
	}

	newState := state.Copy()
	newState.DatastoreState = append(newState.DatastoreState, kvp)
	newState.ExpectedRoutes.Add(routeUpdate)
	newState.ExpectedHostMetadataV4V6[keyName] = metadataUpdate

	return newState
}

// StateWithWEP is a convenience function to help compose remote cluster testing states.
func StateWithWEP(state State, cluster string, ip string, flush bool, poolType proto.IPPoolType, name string, host string, hostIP string) State {
	hostKeyName := host
	if cluster != "" {
		hostKeyName = cluster + "/" + host
	}
	kvp := KVPair{
		Key: WorkloadEndpointKey{Hostname: hostKeyName, OrchestratorID: "orch", WorkloadID: "wl-" + name, EndpointID: "ep-" + name},
		Value: &WorkloadEndpoint{
			State:      "active",
			Name:       name,
			Mac:        mustParseMac("01:02:03:04:05:06"),
			ProfileIDs: []string{},
			IPv4Nets:   []net.IPNet{mustParseNet(ip + "/32")},
			Labels: map[string]string{
				"id": "ep-" + name,
			},
		},
	}

	epData := &calc.EndpointData{
		Key:      kvp.Key,
		Endpoint: kvp.Value,
	}

	routeUpdate := proto.RouteUpdate{
		Type:        proto.RouteType_REMOTE_WORKLOAD,
		IpPoolType:  poolType,
		Dst:         ip + "/32",
		DstNodeName: hostKeyName,
		DstNodeIp:   hostIP,
	}
	if host == localHostname {
		routeUpdate.Type = proto.RouteType_LOCAL_WORKLOAD
		routeUpdate.LocalWorkload = true
	}

	newState := state.Copy()

	if host == localHostname {
		newState = newState.withEndpoint(fmt.Sprintf("orch/wl-%s/ep-%s", name, name), []mock.TierInfo{})
	} else {
		// WEPs are only received by the FV calc graph for local WEPs, unless in WorkloadIPs mode.
		newState.DatastoreState = append(newState.DatastoreState, KVPair{Key: GlobalConfigKey{Name: "RouteSource"}, Value: &workloadIPs})
		newState.ExpectedCachedRemoteEndpoints = append(newState.ExpectedCachedRemoteEndpoints, epData)
	}

	newState.DatastoreState = append(newState.DatastoreState, kvp)
	if flush {
		newState.ExpectedRoutes.Add(routeUpdate)
	}

	return newState
}

// StateWithVTEP is a convenience function to help compose remote cluster testing states.
func StateWithVTEP(state State, cluster string, ip string, flush bool, mac string, host string, hostIP string) State {
	keyName := host
	if cluster != "" {
		keyName = cluster + "/" + host
	}

	kvp := KVPair{
		Key:   HostConfigKey{Name: "IPv4VXLANTunnelAddr", Hostname: keyName},
		Value: ip,
	}
	vtep := proto.VXLANTunnelEndpointUpdate{
		Node:           keyName,
		Mac:            mac,
		Ipv4Addr:       ip,
		ParentDeviceIp: hostIP,
	}
	tunnelRouteUpdate := proto.RouteUpdate{
		Type:        proto.RouteType_REMOTE_TUNNEL,
		IpPoolType:  proto.IPPoolType_VXLAN,
		Dst:         ip + "/32",
		DstNodeName: keyName,
		DstNodeIp:   hostIP,
		TunnelType:  &proto.TunnelType{Vxlan: true},
	}

	newState := state.Copy()
	newState.DatastoreState = append(newState.DatastoreState, kvp)
	newState.ExpectedVTEPs.Add(vtep)
	if flush {
		newState.ExpectedRoutes.Add(tunnelRouteUpdate)
	}

	return newState
}

// Used for remote cluster testing. Adds complete VXLAN block configuration for "pool 2" to the local cluster.
func StateWithVXLANBlockForLocal(state State, shouldFlush bool) State {
	state = StateWithPool(state, local, "10.0.0.0/16", shouldFlush)
	state = StateWithBlock(state, local, "10.0.1.0/29", shouldFlush, proto.IPPoolType_VXLAN, localClusterHost, localClusterHostIPAddr)
	state = StateWithVTEP(state, local, "10.0.1.1", shouldFlush, localClusterHostMAC, localClusterHost, localClusterHostIPAddr)
	state = StateWithNode(state, local, localClusterHost, localClusterHostIPAddr, "10.0.1.1")
	return state
}

// Used for remote cluster testing. Adds complete VXLAN block configuration for "pool 2" to the remote A cluster.
func StateWithVXLANBlockForRemoteA(state State, shouldFlush bool) State {
	state = StateWithPool(state, remoteA, "10.0.0.0/16", shouldFlush)
	state = StateWithBlock(state, remoteA, "10.0.1.0/29", shouldFlush, proto.IPPoolType_VXLAN, remoteClusterAHost, remoteClusterAHostIPAddr)
	state = StateWithVTEP(state, remoteA, "10.0.1.1", shouldFlush, remoteClusterAHostMAC, remoteClusterAHost, remoteClusterAHostIPAddr)
	state = StateWithNode(state, remoteA, remoteClusterAHost, remoteClusterAHostIPAddr, "10.0.1.1")
	return state
}

// Used for remote cluster testing. Adds complete VXLAN block configuration for "pool 2" to the remote B cluster.
func StateWithVXLANBlockForRemoteB(state State, shouldFlush bool) State {
	state = StateWithPool(state, remoteB, "10.0.0.0/16", shouldFlush)
	state = StateWithBlock(state, remoteB, "10.0.1.0/29", shouldFlush, proto.IPPoolType_VXLAN, remoteClusterBHost, remoteClusterBHostIPAddr)
	state = StateWithVTEP(state, remoteB, "10.0.1.1", shouldFlush, remoteClusterBHostMAC, remoteClusterBHost, remoteClusterBHostIPAddr)
	state = StateWithNode(state, remoteB, remoteClusterBHost, remoteClusterBHostIPAddr, "10.0.1.1")
	return state
}

// Used for remote cluster testing. Adds complete VXLAN WEP configuration for "pool 2" to the local cluster.
func StateWithVXLANWEPForLocal(state State, shouldFlush bool) State {
	state = StateWithPool(state, local, "10.0.0.0/16", shouldFlush)
	state = StateWithWEP(state, local, "10.0.0.5", shouldFlush, proto.IPPoolType_VXLAN, "local-wep", localClusterHost, localClusterHostIPAddr)
	state = StateWithVTEP(state, local, "10.0.1.1", shouldFlush, localClusterHostMAC, localClusterHost, localClusterHostIPAddr)
	state = StateWithNode(state, local, localClusterHost, localClusterHostIPAddr, "10.0.1.1")
	return state
}

// Used for remote cluster testing. Adds complete VXLAN WEP configuration for "pool 2" to the remote A cluster.
func StateWithVXLANWEPForRemoteA(state State, shouldFlush bool) State {
	state = StateWithPool(state, remoteA, "10.0.0.0/16", shouldFlush)
	state = StateWithWEP(state, remoteA, "10.0.0.5", shouldFlush, proto.IPPoolType_VXLAN, "local-wep", remoteClusterAHost, remoteClusterAHostIPAddr)
	state = StateWithVTEP(state, remoteA, "10.0.1.1", shouldFlush, remoteClusterAHostMAC, remoteClusterAHost, remoteClusterAHostIPAddr)
	state = StateWithNode(state, remoteA, remoteClusterAHost, remoteClusterAHostIPAddr, "10.0.1.1")
	return state
}

// Used for remote cluster testing. Adds complete VXLAN WEP configuration for "pool 2" to the remote B cluster.
func StateWithVXLANWEPForRemoteB(state State, shouldFlush bool) State {
	state = StateWithPool(state, remoteB, "10.0.0.0/16", shouldFlush)
	state = StateWithWEP(state, remoteB, "10.0.0.5", shouldFlush, proto.IPPoolType_VXLAN, "local-wep", remoteClusterBHost, remoteClusterBHostIPAddr)
	state = StateWithVTEP(state, remoteB, "10.0.1.1", shouldFlush, remoteClusterBHostMAC, remoteClusterBHost, remoteClusterBHostIPAddr)
	state = StateWithNode(state, remoteB, remoteClusterBHost, remoteClusterBHostIPAddr, "10.0.1.1")
	return state
}

func escapeCIDR(cidr string) string {
	return strings.ReplaceAll(strings.ReplaceAll(cidr, ".", "-"), "/", "-")
}

func createAllocationsArray(cidr string) []*int {
	prefixLength, _ := mustParseNet(cidr).Mask.Size()
	return make([]*int, int(math.Pow(2, float64(32-prefixLength))))
}

func createUnallocatedArray(cidr string) []int {
	prefixLength, _ := mustParseNet(cidr).Mask.Size()
	var unallocated []int
	for i := 0; i < int(math.Pow(2, float64(32-prefixLength))); i++ {
		unallocated = append(unallocated, i)
	}
	return unallocated
}
