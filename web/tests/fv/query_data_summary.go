package fv

import (
	"github.com/projectcalico/calicoctl/calicoctl/resourcemgr"
	"github.com/tigera/calicoq/web/pkg/querycache/client"
)

func summaryTestQueryData() []testQueryData {
	return []testQueryData{
		{
			"single node",
			[]resourcemgr.ResourceObject{node1},
			client.QueryClusterReq{},
			&client.QueryClusterResp{
				NumGlobalNetworkPolicies:        0,
				NumNetworkPolicies:              0,
				NumHostEndpoints:                0,
				NumWorkloadEndpoints:            0,
				NumUnlabelledWorkloadEndpoints:  0,
				NumUnlabelledHostEndpoints:      0,
				NumNodes:                        1,
				NumNodesWithNoEndpoints:         1,
				NumNodesWithNoWorkloadEndpoints: 1,
				NumNodesWithNoHostEndpoints:     1,
			},
		},
		{
			"single wep",
			[]resourcemgr.ResourceObject{wep4_n2_ns1},
			client.QueryClusterReq{},
			&client.QueryClusterResp{
				NumGlobalNetworkPolicies:        0,
				NumNetworkPolicies:              0,
				NumHostEndpoints:                0,
				NumWorkloadEndpoints:            1,
				NumUnlabelledWorkloadEndpoints:  0,
				NumUnlabelledHostEndpoints:      0,
				NumNodes:                        1,
				NumNodesWithNoEndpoints:         0,
				NumNodesWithNoWorkloadEndpoints: 0,
				NumNodesWithNoHostEndpoints:     1,
			},
		},
		{
			"single hep",
			[]resourcemgr.ResourceObject{hep3_n4},
			client.QueryClusterReq{},
			&client.QueryClusterResp{
				NumGlobalNetworkPolicies:        0,
				NumNetworkPolicies:              0,
				NumHostEndpoints:                1,
				NumWorkloadEndpoints:            0,
				NumUnlabelledWorkloadEndpoints:  0,
				NumUnlabelledHostEndpoints:      0,
				NumNodes:                        1,
				NumNodesWithNoEndpoints:         0,
				NumNodesWithNoWorkloadEndpoints: 1,
				NumNodesWithNoHostEndpoints:     0,
			},
		},
		{
			"single tier + np",
			[]resourcemgr.ResourceObject{tier1, np2_t1_o2_ns2},
			client.QueryClusterReq{},
			&client.QueryClusterResp{
				NumGlobalNetworkPolicies:        0,
				NumNetworkPolicies:              1,
				NumHostEndpoints:                0,
				NumWorkloadEndpoints:            0,
				NumUnlabelledWorkloadEndpoints:  0,
				NumUnlabelledHostEndpoints:      0,
				NumNodes:                        0,
				NumNodesWithNoEndpoints:         0,
				NumNodesWithNoWorkloadEndpoints: 0,
				NumNodesWithNoHostEndpoints:     0,
			},
		},
		{
			"single tier + gnp",
			[]resourcemgr.ResourceObject{tier1, gnp1_t1_o3},
			client.QueryClusterReq{},
			&client.QueryClusterResp{
				NumGlobalNetworkPolicies:        1,
				NumNetworkPolicies:              0,
				NumHostEndpoints:                0,
				NumWorkloadEndpoints:            0,
				NumUnlabelledWorkloadEndpoints:  0,
				NumUnlabelledHostEndpoints:      0,
				NumNodes:                        0,
				NumNodesWithNoEndpoints:         0,
				NumNodesWithNoWorkloadEndpoints: 0,
				NumNodesWithNoHostEndpoints:     0,
			},
		},
		{
			"multiple nodes, weps, heps, tier1 1 nps and 1 gnps - summary stats, endpoints on all nodes",
			[]resourcemgr.ResourceObject{
				node1, node2, hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, gnp2_t1_o4, np2_t1_o2_ns2,
			},
			client.QueryClusterReq{},
			&client.QueryClusterResp{
				NumGlobalNetworkPolicies:        1,
				NumNetworkPolicies:              1,
				NumHostEndpoints:                4,
				NumWorkloadEndpoints:            4,
				NumUnlabelledWorkloadEndpoints:  1,
				NumUnlabelledHostEndpoints:      1,
				NumNodes:                        4,
				NumNodesWithNoEndpoints:         0,
				NumNodesWithNoWorkloadEndpoints: 1,
				NumNodesWithNoHostEndpoints:     1,
			},
		},
		{
			"multiple nodes, weps, no heps, tier1 2nps - summary stats",
			[]resourcemgr.ResourceObject{
				node1, node2, node3, node4, wep4_n2_ns1, wep3_n1_ns2, wep1_n1_ns1, wep5_n3_ns2_unlabelled,
				tier1, np2_t1_o2_ns2, np1_t1_o1_ns1,
			},
			client.QueryClusterReq{},
			&client.QueryClusterResp{
				NumGlobalNetworkPolicies:        0,
				NumNetworkPolicies:              2,
				NumHostEndpoints:                0,
				NumWorkloadEndpoints:            4,
				NumUnlabelledWorkloadEndpoints:  1,
				NumUnlabelledHostEndpoints:      0,
				NumNodes:                        4,
				NumNodesWithNoEndpoints:         1,
				NumNodesWithNoWorkloadEndpoints: 1,
				NumNodesWithNoHostEndpoints:     4,
			},
		},
		{
			"multiple nodes, heps, no weps, tier1 2 gnps - summary stats",
			[]resourcemgr.ResourceObject{
				node1, node2, node3, node4, hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled,
				tier1, gnp1_t1_o3, gnp2_t1_o4,
			},
			client.QueryClusterReq{},
			&client.QueryClusterResp{
				NumGlobalNetworkPolicies:        2,
				NumNetworkPolicies:              0,
				NumHostEndpoints:                4,
				NumWorkloadEndpoints:            0,
				NumUnlabelledWorkloadEndpoints:  0,
				NumUnlabelledHostEndpoints:      1,
				NumNodes:                        4,
				NumNodesWithNoEndpoints:         1,
				NumNodesWithNoWorkloadEndpoints: 4,
				NumNodesWithNoHostEndpoints:     1,
			},
		},
		{
			"reset by removing all resources",
			[]resourcemgr.ResourceObject{},
			client.QueryClusterReq{},
			&client.QueryClusterResp{},
		},
	}
}
