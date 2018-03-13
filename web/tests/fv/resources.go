package fv

import (
	"context"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calicoctl/calicoctl/resourcemgr"
	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/errors"
	"github.com/tigera/calicoq/web/pkg/querycache/api"
	"github.com/tigera/calicoq/web/pkg/querycache/client"
)

/*
This file defines a number of WEP, HEP, GNP, NP resources that can be used to test the EP<->Policy mappings.
Summary of configuration below.  Rules not explicitly specified have an all() or empty selector.

Endpoint                     rack  server  ns
------------------------------------------------------------------------------
hep4_n4_unlabelled
hep3_n4                      099
wep1_n1_ns1                  001   1       1                          (using profile-rack-001)
wep1_n1_ns1_updated_profile  099   1       1                          (using profile-rack-099)
wep2_n1_ns1_filtered_out     001   1       1
wep2_n1_ns1_filtered_in      001   1       1
wep3_n1_ns2                  001   1       2
wep4_n2_ns1                  001   2       1
hep1_n2                      001   2
wep5_n3_ns2_unlabelled                     2
hep2_n3                      002   1

Policy  Rule                 rack  server  ns  numEgress numIngress tier
------------------------------------------------------------------------------
np1_t1_o1_ns1                001   1       1   1         0          1
np2_t1_o2_ns1                              2   1         1          1
gnp1_t1_o3                   001               1         1          1
gnp2_t1_o3_fewer_rules                                              1
        egress;0;src;sel     001   2
        ingress;0;dest;!sel  !=002
gnp1_t1_o4_more_rules        001               2         2          1
        egress;0;src!sel     001   1
        ingress;0;dest;sel   !=001
gnp2_t1_o4                                     2         2          1
        egress;0;src;!sel    001   1
        egess;1;src;sel      001   2
        ingress;0;dest;sel   !=001
        ingress;1;dest;!sel  !=002
np1_t2_o1_ns1                001   2       1   1         1          2
np2_t2_o2_ns2                              2   1         1          2
gnp1_t2_o3                   !has              1         1          2
gnp2_t2_o4                                     1         1          2

Profile                      rack  server
------------------------------------------------------------------------------
profile-rack-001             1
profile-rack-099             99
*/

var (
	order1 = 1.0
	order2 = 2.0
	order3 = 3.0
	order4 = 4.0

	node1 = &apiv3.Node{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindNode,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "rack.1-server.1",
			Labels: map[string]string{
				"rack":   "001",
				"server": "1",
			},
		},
		Spec: apiv3.NodeSpec{
			BGP: &apiv3.NodeBGPSpec{
				IPv4Address: "1.2.3.1/24",
				IPv6Address: "aabb:ccdd:ee11:2233:3344:4455:6677:8891/120",
			},
		},
	}

	node2 = &apiv3.Node{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindNode,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "rack.1-server.2",
			Labels: map[string]string{
				"rack":   "001",
				"server": "1",
			},
		},
		Spec: apiv3.NodeSpec{
			BGP: &apiv3.NodeBGPSpec{
				IPv4Address: "1.2.3.2/24",
				IPv6Address: "aabb:ccdd:ee11:2233:3344:4455:6677:8892/120",
			},
		},
	}

	node3 = &apiv3.Node{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindNode,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "rack.2-server.1",
			Labels: map[string]string{
				"rack":   "002",
				"server": "1",
			},
		},
		Spec: apiv3.NodeSpec{
			BGP: &apiv3.NodeBGPSpec{
				IPv4Address: "1.2.4.1/24",
				IPv6Address: "aabb:ccdd::88a1/120",
			},
		},
	}

	node4 = &apiv3.Node{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindNode,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "master-node.0001",
			Labels: map[string]string{
				"rack": "099",
			},
		},
		Spec: apiv3.NodeSpec{},
	}

	wep1_n1_ns1 = &apiv3.WorkloadEndpoint{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindWorkloadEndpoint,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rack.1--server.1-k8s-pod1--aaa-eth0",
			Namespace: "namespace-1",
			Labels: map[string]string{
				"server": "1",
				"name":   "wep1_n1_ns1",
			},
		},
		Spec: apiv3.WorkloadEndpointSpec{
			Node:          "rack.1-server.1",
			Profiles:      []string{"profile-rack-001"},
			Workload:      "",
			Orchestrator:  "k8s",
			Pod:           "pod1-aaa",
			ContainerID:   "abcdefg",
			Endpoint:      "eth0",
			InterfaceName: "cali987654",
			IPNetworks:    []string{"1.2.3.4/32"},
		},
	}

	wep1_n1_ns1_updated_profile = &apiv3.WorkloadEndpoint{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindWorkloadEndpoint,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rack.1--server.1-k8s-pod1--aaa-eth0",
			Namespace: "namespace-1",
			Labels: map[string]string{
				"server": "1",
				"name":   "wep1_n1_ns1",
			},
		},
		Spec: apiv3.WorkloadEndpointSpec{
			Node:          "rack.1-server.1",
			Profiles:      []string{"profile-rack-099"},
			Workload:      "",
			Orchestrator:  "k8s",
			Pod:           "pod1-aaa",
			ContainerID:   "abcdefg",
			Endpoint:      "eth0",
			InterfaceName: "cali987654",
			IPNetworks:    []string{"1.2.3.4/32"},
		},
	}

	wep2_n1_ns1_filtered_out = &apiv3.WorkloadEndpoint{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindWorkloadEndpoint,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rack.1--server.1-k8s-pod1--abc-eth0",
			Namespace: "namespace-1",
			Labels: map[string]string{
				"rack":   "001",
				"server": "1",
				"name":   "wep2_n1_ns1_filtered_out",
			},
		},
		Spec: apiv3.WorkloadEndpointSpec{
			Node:          "rack.1-server.1",
			Workload:      "",
			Orchestrator:  "k8s",
			Pod:           "pod1-abc",
			ContainerID:   "abcdefg",
			Endpoint:      "eth0",
			InterfaceName: "cali987654",
			// No IPNetworks, so WEP will be filtered out.
			IPNetworks: []string{},
		},
	}

	wep2_n1_ns1_filtered_in = &apiv3.WorkloadEndpoint{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindWorkloadEndpoint,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rack.1--server.1-k8s-pod1--abc-eth0",
			Namespace: "namespace-1",
			Labels: map[string]string{
				"rack":   "001",
				"server": "1",
				"name":   "wep2_n1_ns1_filtered_out",
			},
		},
		Spec: apiv3.WorkloadEndpointSpec{
			Node:          "rack.1-server.1",
			Workload:      "",
			Orchestrator:  "k8s",
			Pod:           "pod1-abc",
			ContainerID:   "abcdefg",
			Endpoint:      "eth0",
			InterfaceName: "cali987654",
			// Thie one has an IP address and will therefore be filtered in.
			IPNetworks: []string{"10.20.30.40/32"},
		},
	}

	wep3_n1_ns2 = &apiv3.WorkloadEndpoint{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindWorkloadEndpoint,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rack.1--server.1-k8s-pod2--acd-eth0",
			Namespace: "namespace-2",
			Labels: map[string]string{
				"rack":   "001",
				"server": "1",
				"name":   "wep3_n1_ns2",
			},
		},
		Spec: apiv3.WorkloadEndpointSpec{
			Node:          "rack.1-server.1",
			Workload:      "",
			Orchestrator:  "k8s",
			Pod:           "pod2-acd",
			ContainerID:   "abcde00",
			Endpoint:      "eth0",
			InterfaceName: "cali123456",
			IPNetworks:    []string{"1.2.3.6/32"},
		},
	}

	wep4_n2_ns1 = &apiv3.WorkloadEndpoint{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindWorkloadEndpoint,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rack.1--server.2-openstack-aabbccdd-eth01234",
			Namespace: "namespace-1",
			Labels: map[string]string{
				"rack":   "001",
				"server": "2",
				"name":   "wep4_n2_ns1",
			},
		},
		Spec: apiv3.WorkloadEndpointSpec{
			Node:          "rack.1-server.2",
			Workload:      "aabbccdd",
			Orchestrator:  "openstack",
			Pod:           "",
			ContainerID:   "",
			Endpoint:      "eth01234",
			InterfaceName: "caliabcdef",
			IPNetworks:    []string{"1.2.3.7/32"},
		},
	}

	wep5_n3_ns2_unlabelled = &apiv3.WorkloadEndpoint{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindWorkloadEndpoint,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rack.2--server.1-cni-badcafe1-foobarbaz",
			Namespace: "namespace-2",
		},
		Spec: apiv3.WorkloadEndpointSpec{
			Node:          "rack.2-server.1",
			Workload:      "",
			Orchestrator:  "cni",
			Pod:           "",
			ContainerID:   "badcafe1",
			Endpoint:      "foobarbaz",
			InterfaceName: "calia1b2c3",
			IPNetworks:    []string{"1.2.3.8/32"},
		},
	}

	hep1_n2 = &apiv3.HostEndpoint{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindHostEndpoint,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "rack.1-server.2---eth1",
			Labels: map[string]string{
				"rack":   "001",
				"server": "2",
				"name":   "hep1_n2",
			},
		},
		Spec: apiv3.HostEndpointSpec{
			Node:          "rack.1-server.2",
			InterfaceName: "eth1",
			ExpectedIPs:   []string{"10.11.12.13"},
		},
	}

	hep2_n3 = &apiv3.HostEndpoint{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindHostEndpoint,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "rack.2-server.1---eth1",
			Labels: map[string]string{
				"rack":   "002",
				"server": "1",
				"name":   "hep2_n3",
			},
		},
		Spec: apiv3.HostEndpointSpec{
			Node:          "rack.2-server.1",
			InterfaceName: "eth1",
		},
	}

	hep3_n4 = &apiv3.HostEndpoint{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindHostEndpoint,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "master-main-interface",
			Labels: map[string]string{
				"rack": "099",
				"name": "hep3_n4",
			},
		},
		Spec: apiv3.HostEndpointSpec{
			Node:          "master-node.0001",
			InterfaceName: "eth0",
		},
	}

	hep4_n4_unlabelled = &apiv3.HostEndpoint{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindHostEndpoint,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "master-backup-interface",
		},
		Spec: apiv3.HostEndpointSpec{
			Node:          "master-node.0001",
			InterfaceName: "eth1",
		},
	}

	tier1 = &apiv3.Tier{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindTier,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "ccc-tier1",
		},
		Spec: apiv3.TierSpec{
			Order: &order1,
		},
	}

	tier2 = &apiv3.Tier{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindTier,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "aaa-tier2",
		},
		Spec: apiv3.TierSpec{
			Order: &order2,
		},
	}

	tier3 = &apiv3.Tier{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindTier,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "bbb-tier3",
		},
		Spec: apiv3.TierSpec{
			Order: &order3,
		},
	}

	// Create a couple of re-ordered tiers
	tier1_o2 = &apiv3.Tier{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindTier,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "ccc-tier1",
		},
		Spec: apiv3.TierSpec{
			Order: &order2,
		},
	}

	tier2_o1 = &apiv3.Tier{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindTier,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "aaa-tier2",
		},
		Spec: apiv3.TierSpec{
			Order: &order1,
		},
	}

	np1_t1_o1_ns1 = &apiv3.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindNetworkPolicy,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ccc-tier1.np1-t1-o1-ns1",
			Namespace: "namespace-1",
		},
		Spec: apiv3.NetworkPolicySpec{
			Tier:     "ccc-tier1",
			Selector: "rack == '001' && server == '1'",
			Order:    &order1,
			Egress: []apiv3.Rule{
				{
					Action: "Allow",
					Source: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
					Destination: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
				},
			},
		},
	}

	np2_t1_o2_ns2 = &apiv3.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindNetworkPolicy,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ccc-tier1.np2-t1-o2-ns1",
			Namespace: "namespace-2",
		},
		Spec: apiv3.NetworkPolicySpec{
			Tier:     "ccc-tier1",
			Selector: "all()",
			Order:    &order2,
			Egress: []apiv3.Rule{
				{
					Action: "Pass",
					Source: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
					Destination: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
				},
			},
			Ingress: []apiv3.Rule{
				{
					Action: "Pass",
					Source: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
					Destination: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
				},
			},
		},
	}

	gnp1_t1_o3 = &apiv3.GlobalNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindGlobalNetworkPolicy,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "ccc-tier1.gnp1-t1-o3",
		},
		Spec: apiv3.GlobalNetworkPolicySpec{
			Tier:     "ccc-tier1",
			Selector: "rack == '001'",
			Order:    &order3,
			Egress: []apiv3.Rule{
				{
					Action: "Pass",
					Source: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
					Destination: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
				},
			},
			Ingress: []apiv3.Rule{
				{
					Action: "Pass",
					Source: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
					Destination: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
				},
			},
		},
	}

	gnp2_t1_o4 = &apiv3.GlobalNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindGlobalNetworkPolicy,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "ccc-tier1.gnp2-t1-o4",
		},
		Spec: apiv3.GlobalNetworkPolicySpec{
			Tier:     "ccc-tier1",
			Selector: "",
			Order:    &order4,
			Egress: []apiv3.Rule{
				{
					Action: "Allow",
					Source: apiv3.EntityRule{
						Selector:    "all()",
						NotSelector: "rack == '001' && server == '1'",
					},
					Destination: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
				},
				{
					Action: "Allow",
					Source: apiv3.EntityRule{
						Selector:    "rack == '001' && server == '2'",
						NotSelector: "",
					},
					Destination: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
				},
			},
			Ingress: []apiv3.Rule{
				{
					Action: "Allow",
					Source: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
					Destination: apiv3.EntityRule{
						Selector:    "has(rack) && rack != '001'",
						NotSelector: "",
					},
				},
				{
					Action: "Deny",
					Source: apiv3.EntityRule{
						Selector:    "all()",
						NotSelector: "",
					},
					Destination: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "has(rack) && rack != '002'",
					},
				},
			},
		},
	}

	np1_t2_o1_ns1 = &apiv3.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindNetworkPolicy,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aaa-tier2.np1-t2-o1-ns1",
			Namespace: "namespace-1",
		},
		Spec: apiv3.NetworkPolicySpec{
			Tier:     "aaa-tier2",
			Selector: "rack == '001' && server == '2'",
			Order:    &order1,
			Egress: []apiv3.Rule{
				{
					Action: "Allow",
					Source: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
					Destination: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
				},
			},
			Ingress: []apiv3.Rule{
				{
					Action: "Allow",
					Source: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
					Destination: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
				},
			},
		},
	}

	np2_t2_o2_ns2 = &apiv3.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindNetworkPolicy,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aaa-tier2.np2-t2-o2-ns1",
			Namespace: "namespace-2",
		},
		Spec: apiv3.NetworkPolicySpec{
			Tier:     "aaa-tier2",
			Selector: "all()",
			Order:    &order2,
			Egress: []apiv3.Rule{
				{
					Action: "Pass",
					Source: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
					Destination: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
				},
			},
			Ingress: []apiv3.Rule{
				{
					Action: "Pass",
					Source: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
					Destination: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
				},
			},
		},
	}

	gnp1_t2_o3 = &apiv3.GlobalNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindGlobalNetworkPolicy,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "aaa-tier2.gnp1-t2-o3",
		},
		Spec: apiv3.GlobalNetworkPolicySpec{
			Tier:     "aaa-tier2",
			Selector: "!has(rack)",
			Order:    &order3,
			Egress: []apiv3.Rule{
				{
					Action: "Pass",
					Source: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
					Destination: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
				},
			},
			Ingress: []apiv3.Rule{
				{
					Action: "Pass",
					Source: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
					Destination: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
				},
			},
		},
	}

	gnp2_t2_o4 = &apiv3.GlobalNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindGlobalNetworkPolicy,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "aaa-tier2.gnp2-t2-o4",
		},
		Spec: apiv3.GlobalNetworkPolicySpec{
			Tier:     "aaa-tier2",
			Selector: "",
			Order:    &order4,
			Egress: []apiv3.Rule{
				{
					Action: "Deny",
					Source: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
					Destination: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
				},
			},
			Ingress: []apiv3.Rule{
				{
					Action: "Deny",
					Source: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
					Destination: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
				},
			},
		},
	}

	// Create a couple of adjusted policies that have different orders and different numbers of rules.
	gnp1_t1_o4_more_rules = &apiv3.GlobalNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindGlobalNetworkPolicy,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "ccc-tier1.gnp1-t1-o3",
		},
		Spec: apiv3.GlobalNetworkPolicySpec{
			Tier:     "ccc-tier1",
			Selector: "rack == '001'",
			Order:    &order4,
			Egress: []apiv3.Rule{
				{
					Action: "Allow",
					Source: apiv3.EntityRule{
						Selector:    "all()",
						NotSelector: "rack == '001' && server == '1'",
					},
					Destination: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
				},
				{
					Action: "Pass",
					Source: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
					Destination: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
				},
			},
			Ingress: []apiv3.Rule{
				{
					Action: "Allow",
					Source: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
					Destination: apiv3.EntityRule{
						Selector:    "has(rack) && rack != '001'",
						NotSelector: "",
					},
				},
				{
					Action: "Pass",
					Source: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
					Destination: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
				},
			},
		},
	}

	gnp2_t1_o3_fewer_rules = &apiv3.GlobalNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindGlobalNetworkPolicy,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "ccc-tier1.gnp2-t1-o4",
		},
		Spec: apiv3.GlobalNetworkPolicySpec{
			Tier:     "ccc-tier1",
			Selector: "",
			Order:    &order3,
			Egress: []apiv3.Rule{
				{
					Action: "Allow",
					Source: apiv3.EntityRule{
						Selector:    "rack == '001' && server == '2'",
						NotSelector: "",
					},
					Destination: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "",
					},
				},
			},
			Ingress: []apiv3.Rule{
				{
					Action: "Deny",
					Source: apiv3.EntityRule{
						Selector:    "all()",
						NotSelector: "",
					},
					Destination: apiv3.EntityRule{
						Selector:    "",
						NotSelector: "has(rack) && rack != '002'",
					},
				},
			},
		},
	}

	profile_rack_001 = &apiv3.Profile{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindProfile,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "profile-rack-001",
		},
		Spec: apiv3.ProfileSpec{
			LabelsToApply: map[string]string{
				"rack": "001",
			},
		},
	}

	profile_rack_099 = &apiv3.Profile{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv3.GroupVersionCurrent,
			Kind:       apiv3.KindProfile,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "profile-rack-099",
		},
		Spec: apiv3.ProfileSpec{
			LabelsToApply: map[string]string{
				"rack": "099",
			},
		},
	}
)

// qcNode returns a client.Node from an v3.Node, v3.WorkloadEndpoint or v3.HostEndpoint.
func qcNode(r api.Resource, numWEP, numHEP int) client.Node {
	n := client.Node{
		NumWorkloadEndpoints: numWEP,
		NumHostEndpoints:     numHEP,
	}

	switch nr := r.(type) {
	case *apiv3.Node:
		n.Name = nr.Name
		if nr.Spec.BGP != nil {
			if nr.Spec.BGP.IPv4Address != "" {
				n.BGPIPAddresses = append(n.BGPIPAddresses, nr.Spec.BGP.IPv4Address)
			}
			if nr.Spec.BGP.IPv6Address != "" {
				n.BGPIPAddresses = append(n.BGPIPAddresses, nr.Spec.BGP.IPv6Address)
			}
		}
	case *apiv3.WorkloadEndpoint:
		n.Name = nr.Spec.Node
	case *apiv3.HostEndpoint:
		n.Name = nr.Spec.Node
	}
	return n
}

// qcNode returns a client.Node from an v3.WorkloadEndpoint or v3.HostEndpoint.
func qcEndpoint(r api.Resource, numGNP, numNP int) client.Endpoint {
	e := client.Endpoint{
		Kind:                     r.GetObjectKind().GroupVersionKind().Kind,
		Name:                     r.GetObjectMeta().GetName(),
		Namespace:                r.GetObjectMeta().GetNamespace(),
		NumGlobalNetworkPolicies: numGNP,
		NumNetworkPolicies:       numNP,
	}

	switch er := r.(type) {
	case *apiv3.WorkloadEndpoint:
		// Copy labels to add implicit labels.
		labels := map[string]string{}
		for k, v := range r.GetObjectMeta().GetLabels() {
			labels[k] = v
		}
		labels["projectcalico.org/namespace"] = er.Namespace
		labels["projectcalico.org/orchestrator"] = er.Spec.Orchestrator
		e.Labels = labels
		e.Node = er.Spec.Node
		e.Workload = er.Spec.Workload
		e.Orchestrator = er.Spec.Orchestrator
		e.Pod = er.Spec.Pod
		e.InterfaceName = er.Spec.InterfaceName
		e.IPNetworks = er.Spec.IPNetworks
	case *apiv3.HostEndpoint:
		e.Labels = r.GetObjectMeta().GetLabels()
		e.Node = er.Spec.Node
		e.InterfaceName = er.Spec.InterfaceName
		e.IPNetworks = er.Spec.ExpectedIPs
	}
	return e
}

// qcPolicy returns a client.Policy from an v3.NetworkPolicy or v3.GlobalNetworkPolicy.
// To keep the interface simple, it assigns the totWEP and totHEP values to all of the
// rule selectors and not selectors (i.e. it assumes they simply match all).
func qcPolicy(r api.Resource, numHEP, numWEP, totHEP, totWEP int) client.Policy {
	p := client.Policy{
		Kind:                 r.GetObjectKind().GroupVersionKind().Kind,
		Name:                 r.GetObjectMeta().GetName(),
		Namespace:            r.GetObjectMeta().GetNamespace(),
		NumWorkloadEndpoints: numWEP,
		NumHostEndpoints:     numHEP,
	}

	createRulesFn := func(num int) []client.RuleDirection {
		rules := make([]client.RuleDirection, num)
		for i := 0; i < num; i++ {
			rules[i] = client.RuleDirection{
				Source: client.RuleEntity{
					Selector: client.RuleEntityEndpoints{
						NumWorkloadEndpoints: totWEP,
						NumHostEndpoints:     totHEP,
					},
					NotSelector: client.RuleEntityEndpoints{
						NumWorkloadEndpoints: totWEP,
						NumHostEndpoints:     totHEP,
					},
				},
				Destination: client.RuleEntity{
					Selector: client.RuleEntityEndpoints{
						NumWorkloadEndpoints: totWEP,
						NumHostEndpoints:     totHEP,
					},
					NotSelector: client.RuleEntityEndpoints{
						NumWorkloadEndpoints: totWEP,
						NumHostEndpoints:     totHEP,
					},
				},
			}
		}
		return rules
	}

	switch er := r.(type) {
	case *apiv3.NetworkPolicy:
		p.Tier = er.Spec.Tier
		p.Ingress = createRulesFn(len(er.Spec.Ingress))
		p.Egress = createRulesFn(len(er.Spec.Egress))
	case *apiv3.GlobalNetworkPolicy:
		p.Tier = er.Spec.Tier
		p.Ingress = createRulesFn(len(er.Spec.Ingress))
		p.Egress = createRulesFn(len(er.Spec.Egress))
	}
	return p
}

// createResources ensures the supplied set of `configure` resources is configured by either
// creating or updating as necessary and deleting any old resources from the configured map that
// are no longer required.  This allows test to churn configuration rather than simply deleting
// the entire contents of etcd before each run.
func createResources(
	client clientv3.Interface, configure []resourcemgr.ResourceObject,
	configured map[model.ResourceKey]resourcemgr.ResourceObject,
) map[model.ResourceKey]resourcemgr.ResourceObject {
	if configured == nil {
		configured = make(map[model.ResourceKey]resourcemgr.ResourceObject, 0)
	}
	unhandled := make(map[model.ResourceKey]resourcemgr.ResourceObject)
	ctx := context.Background()

	// Construct the map of unhandled resources. We use this to easily look up which resources
	// we don't want to delete when tidying up the currently configured resources.
	for _, res := range configure {
		unhandled[resourceKey(res)] = res
	}

	// First delete any resources that are not in the unhandled set of resource.  We need to do this first
	// because deleting some resources (e.g. nodes) may delete other associated resources that we were not
	// intending to delete.  We require two iterations, the first to delete non-tiers, the second to delete
	// tiers (because they can only be deleted once the associated policies are also deleted).
	for i := 0; i < 2; i++ {
		for key, res := range configured {
			if _, ok := unhandled[key]; ok {
				// This resource is in our unhandled map so we'll be creating or updating it
				// later - no need to delete.
				continue
			}
			if i == 0 && key.Kind == apiv3.KindTier {
				// Skip tiers on the first iteration.
				continue
			}
			delete(configured, key)
			rm := resourcemgr.GetResourceManager(res)
			Expect(rm).NotTo(BeNil())
			_, err := rm.Delete(ctx, client, res)
			if _, ok := err.(errors.ErrorResourceDoesNotExist); !ok {
				// Only check for no error if the error is does not indicate a missing resource.
				Expect(err).NotTo(HaveOccurred())
			}
		}
	}

	// Apply the resources in the specified order.
	for _, res := range configure {
		res := res.DeepCopyObject().(resourcemgr.ResourceObject)
		rm := resourcemgr.GetResourceManager(res)
		Expect(rm).NotTo(BeNil())
		configured[resourceKey(res)] = res
		_, err := rm.Apply(ctx, client, res)
		Expect(err).NotTo(HaveOccurred())
	}

	return configured
}

func resourceKey(res resourcemgr.ResourceObject) model.ResourceKey {
	return model.ResourceKey{
		Kind:      res.GetObjectKind().GroupVersionKind().Kind,
		Name:      res.GetObjectMeta().GetName(),
		Namespace: res.GetObjectMeta().GetNamespace(),
	}
}
