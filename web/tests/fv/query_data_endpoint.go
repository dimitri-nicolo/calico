package fv

import (
	"errors"

	"github.com/projectcalico/calicoctl/calicoctl/resourcemgr"
	"github.com/tigera/calicoq/web/pkg/querycache/client"
)

func endpointTestQueryData() []testQueryData {
	return []testQueryData{
		{
			"multiple weps and heps, no policy - query exact wep",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, wep1_n1_ns1, wep5_n3_ns2_unlabelled,
			},
			client.QueryEndpointsReq{
				Endpoint: resourceKey(wep3_n1_ns2),
			},
			&client.QueryEndpointsResp{
				Count: 1,
				Items: []client.Endpoint{qcEndpoint(wep3_n1_ns2, 0, 0)},
			},
		},
		{
			"multiple weps and heps, no policy - query exact hep",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, wep1_n1_ns1, wep5_n3_ns2_unlabelled,
				wep2_n1_ns1_filtered_out,
			},
			client.QueryEndpointsReq{
				Endpoint: resourceKey(hep2_n3),
			},
			&client.QueryEndpointsResp{
				Count: 1,
				Items: []client.Endpoint{qcEndpoint(hep2_n3, 0, 0)},
			},
		},
		{
			"multiple weps and heps, no policy - query all of them",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, wep1_n1_ns1, wep5_n3_ns2_unlabelled,
			},
			client.QueryEndpointsReq{},
			&client.QueryEndpointsResp{
				Count: 8,
				Items: []client.Endpoint{
					qcEndpoint(hep4_n4_unlabelled, 0, 0), qcEndpoint(hep3_n4, 0, 0), qcEndpoint(wep1_n1_ns1, 0, 0),
					qcEndpoint(wep3_n1_ns2, 0, 0), qcEndpoint(wep4_n2_ns1, 0, 0), qcEndpoint(hep1_n2, 0, 0),
					qcEndpoint(wep5_n3_ns2_unlabelled, 0, 0), qcEndpoint(hep2_n3, 0, 0),
				},
			},
		},
		{
			"multiple weps and heps, no policy - query unprotected endpoints",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, wep1_n1_ns1, wep5_n3_ns2_unlabelled,
			},
			client.QueryEndpointsReq{
				Unprotected: true,
			},
			&client.QueryEndpointsResp{
				Count: 8,
				Items: []client.Endpoint{
					qcEndpoint(hep4_n4_unlabelled, 0, 0), qcEndpoint(hep3_n4, 0, 0), qcEndpoint(wep1_n1_ns1, 0, 0),
					qcEndpoint(wep3_n1_ns2, 0, 0), qcEndpoint(wep4_n2_ns1, 0, 0), qcEndpoint(hep1_n2, 0, 0),
					qcEndpoint(wep5_n3_ns2_unlabelled, 0, 0), qcEndpoint(hep2_n3, 0, 0),
				},
			},
		},
		{
			"multiple weps and heps, no policy - query all of them - page 0 of 2",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, wep1_n1_ns1, wep5_n3_ns2_unlabelled,
				wep2_n1_ns1_filtered_out,
			},
			client.QueryEndpointsReq{
				Page: &client.Page{
					PageNum:    0,
					NumPerPage: 5,
				},
			},
			&client.QueryEndpointsResp{
				Count: 8,
				Items: []client.Endpoint{
					qcEndpoint(hep4_n4_unlabelled, 0, 0), qcEndpoint(hep3_n4, 0, 0), qcEndpoint(wep1_n1_ns1, 0, 0),
					qcEndpoint(wep3_n1_ns2, 0, 0), qcEndpoint(wep4_n2_ns1, 0, 0),
				},
			},
		},
		{
			"multiple weps and heps, no policy - query all of them - page 1 of 2",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, wep1_n1_ns1, wep5_n3_ns2_unlabelled,
				wep2_n1_ns1_filtered_out,
			},
			client.QueryEndpointsReq{
				Page: &client.Page{
					PageNum:    1,
					NumPerPage: 5,
				},
			},
			&client.QueryEndpointsResp{
				Count: 8,
				Items: []client.Endpoint{
					qcEndpoint(hep1_n2, 0, 0), qcEndpoint(wep5_n3_ns2_unlabelled, 0, 0), qcEndpoint(hep2_n3, 0, 0),
				},
			},
		},
		{
			"multiple weps and heps, no policy - query all of them - page 2 of 2",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, wep1_n1_ns1, wep5_n3_ns2_unlabelled,
				wep2_n1_ns1_filtered_out,
			},
			client.QueryEndpointsReq{
				Page: &client.Page{
					PageNum:    2,
					NumPerPage: 5,
				},
			},
			&client.QueryEndpointsResp{
				Count: 8,
				Items: []client.Endpoint{},
			},
		},
		{
			"multiple weps and heps, no policy - query all of them, filter on node2",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, wep1_n1_ns1, wep5_n3_ns2_unlabelled,
			},
			client.QueryEndpointsReq{
				Node: node2.Name,
			},
			&client.QueryEndpointsResp{
				Count: 2,
				Items: []client.Endpoint{
					qcEndpoint(wep4_n2_ns1, 0, 0), qcEndpoint(hep1_n2, 0, 0),
				},
			},
		},
		{
			"multiple weps and heps, no policy - query unprotected nodes, filter on node2",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, wep1_n1_ns1, wep5_n3_ns2_unlabelled,
			},
			client.QueryEndpointsReq{
				Unprotected: true,
				Node:        node2.Name,
			},
			&client.QueryEndpointsResp{
				Count: 2,
				Items: []client.Endpoint{
					qcEndpoint(wep4_n2_ns1, 0, 0), qcEndpoint(hep1_n2, 0, 0),
				},
			},
		},
		{
			"multiple weps and heps, selector: all()",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, wep1_n1_ns1, wep5_n3_ns2_unlabelled,
			},
			client.QueryEndpointsReq{
				Selector: "all()",
			},
			&client.QueryEndpointsResp{
				Count: 8,
				Items: []client.Endpoint{
					qcEndpoint(hep4_n4_unlabelled, 0, 0), qcEndpoint(hep3_n4, 0, 0), qcEndpoint(wep1_n1_ns1, 0, 0),
					qcEndpoint(wep3_n1_ns2, 0, 0), qcEndpoint(wep4_n2_ns1, 0, 0), qcEndpoint(hep1_n2, 0, 0),
					qcEndpoint(wep5_n3_ns2_unlabelled, 0, 0), qcEndpoint(hep2_n3, 0, 0),
				},
			},
		},
		{
			"multiple weps and heps, selector: (rack == '001' || rack == '002') && server == '1'",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled,
			},
			client.QueryEndpointsReq{
				Selector: "(rack == '001' || rack == '002') && server == '1'",
			},
			&client.QueryEndpointsResp{
				Count: 3,
				Items: []client.Endpoint{
					qcEndpoint(wep1_n1_ns1, 0, 0), qcEndpoint(wep3_n1_ns2, 0, 0), qcEndpoint(hep2_n3, 0, 0),
				},
			},
		},
		{
			"multiple weps and heps, tier1 policy - query all of them",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
			},
			client.QueryEndpointsReq{},
			&client.QueryEndpointsResp{
				Count: 8,
				Items: []client.Endpoint{
					qcEndpoint(hep4_n4_unlabelled, 1, 0), qcEndpoint(hep3_n4, 1, 0), qcEndpoint(wep1_n1_ns1, 2, 1),
					qcEndpoint(wep3_n1_ns2, 2, 1), qcEndpoint(wep4_n2_ns1, 2, 0), qcEndpoint(hep1_n2, 2, 0),
					qcEndpoint(wep5_n3_ns2_unlabelled, 1, 1), qcEndpoint(hep2_n3, 1, 0),
				},
			},
		},
		{
			"multiple weps and heps, tier1 policy - query unprotected endpoints",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
			},
			client.QueryEndpointsReq{
				Unprotected: true,
			},
			&client.QueryEndpointsResp{
				Count: 0,
				Items: []client.Endpoint{},
			},
		},
		{
			"multiple weps and heps, some tier1 policies (no all() policies) - query unprotected endpoints",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, gnp1_t1_o3,
			},
			client.QueryEndpointsReq{
				Unprotected: true,
			},
			&client.QueryEndpointsResp{
				Count: 4,
				Items: []client.Endpoint{
					qcEndpoint(hep4_n4_unlabelled, 0, 0), qcEndpoint(hep3_n4, 0, 0),
					qcEndpoint(wep5_n3_ns2_unlabelled, 0, 0), qcEndpoint(hep2_n3, 0, 0),
				},
			},
		},
		{
			"multiple weps and heps, tier1 policy - selector: rack == '001' && server == '2'",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp2_t1_o4,
			},
			client.QueryEndpointsReq{
				Selector: "rack == '001' && server == '2'",
			},
			// We've removed gnp1 so counts are lower for GNP than previous run.
			&client.QueryEndpointsResp{
				Count: 2,
				Items: []client.Endpoint{
					qcEndpoint(wep4_n2_ns1, 1, 0), qcEndpoint(hep1_n2, 1, 0),
				},
			},
		},
		{
			"multiple weps and heps, tier1 policy - endpoints matching policy selector gnp1_t1_o3",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
				wep2_n1_ns1_filtered_out,
			},
			client.QueryEndpointsReq{
				Policy: resourceKey(gnp1_t1_o3),
			},
			&client.QueryEndpointsResp{
				Count: 4,
				Items: []client.Endpoint{
					qcEndpoint(wep1_n1_ns1, 2, 1), qcEndpoint(wep3_n1_ns2, 2, 1), qcEndpoint(wep4_n2_ns1, 2, 0),
					qcEndpoint(hep1_n2, 2, 0),
				},
			},
		},
		{
			"multiple weps and heps, tier1 policy - endpoints matching policy selector gnp1_t1_o3, filter in wep2",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
				wep2_n1_ns1_filtered_in,
			},
			client.QueryEndpointsReq{
				Policy: resourceKey(gnp1_t1_o3),
			},
			&client.QueryEndpointsResp{
				Count: 5,
				Items: []client.Endpoint{
					qcEndpoint(wep1_n1_ns1, 2, 1), qcEndpoint(wep2_n1_ns1_filtered_in, 2, 1),
					qcEndpoint(wep3_n1_ns2, 2, 1), qcEndpoint(wep4_n2_ns1, 2, 0), qcEndpoint(hep1_n2, 2, 0),
				},
			},
		},
		{
			"multiple weps and heps, tier1 policy - endpoints matching policy selector gnp1_t1_o3",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
			},
			client.QueryEndpointsReq{
				Policy: resourceKey(gnp1_t1_o3),
			},
			&client.QueryEndpointsResp{
				Count: 4,
				Items: []client.Endpoint{
					qcEndpoint(wep1_n1_ns1, 2, 1), qcEndpoint(wep3_n1_ns2, 2, 1), qcEndpoint(wep4_n2_ns1, 2, 0),
					qcEndpoint(hep1_n2, 2, 0),
				},
			},
		},
		{
			"multiple weps and heps, tier1 policy - endpoints matching gnp2-t1-o4;egress;idx=0;source;notSelector",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
			},
			client.QueryEndpointsReq{
				Policy:              resourceKey(gnp2_t1_o4),
				RuleDirection:       "egress",
				RuleIndex:           0,
				RuleEntity:          "source",
				RuleNegatedSelector: true,
			},
			&client.QueryEndpointsResp{
				Count: 2,
				Items: []client.Endpoint{
					qcEndpoint(wep1_n1_ns1, 2, 1), qcEndpoint(wep3_n1_ns2, 2, 1),
				},
			},
		},
		{
			"multiple weps and heps, tier1 policy - endpoints matching gnp2-t1-o4;egress;idx=1;source;selector",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
			},
			client.QueryEndpointsReq{
				Policy:              resourceKey(gnp2_t1_o4),
				RuleDirection:       "egress",
				RuleIndex:           1,
				RuleEntity:          "source",
				RuleNegatedSelector: false,
			},
			&client.QueryEndpointsResp{
				Count: 2,
				Items: []client.Endpoint{
					qcEndpoint(wep4_n2_ns1, 2, 0), qcEndpoint(hep1_n2, 2, 0),
				},
			},
		},
		{
			"multiple weps and heps, tier1 policy - endpoints matching gnp2-t1-o4;ingress;idx=0;destination;selector",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
			},
			client.QueryEndpointsReq{
				Policy:              resourceKey(gnp2_t1_o4),
				RuleDirection:       "ingress",
				RuleIndex:           0,
				RuleEntity:          "destination",
				RuleNegatedSelector: false,
			},
			&client.QueryEndpointsResp{
				Count: 2,
				Items: []client.Endpoint{
					qcEndpoint(hep3_n4, 1, 0), qcEndpoint(hep2_n3, 1, 0),
				},
			},
		},
		{
			"multiple weps and heps, tier1 policy - endpoints matching gnp2-t1-o4;ingress;idx=1;destination;notSelector",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
			},
			client.QueryEndpointsReq{
				Policy:              resourceKey(gnp2_t1_o4),
				RuleDirection:       "ingress",
				RuleIndex:           1,
				RuleEntity:          "destination",
				RuleNegatedSelector: true,
			},
			&client.QueryEndpointsResp{
				Count: 5,
				Items: []client.Endpoint{
					qcEndpoint(hep3_n4, 1, 0), qcEndpoint(wep1_n1_ns1, 2, 1),
					qcEndpoint(wep3_n1_ns2, 2, 1), qcEndpoint(wep4_n2_ns1, 2, 0), qcEndpoint(hep1_n2, 2, 0),
				},
			},
		},
		{
			"updated GNPs orders and rules - check no change to main policy selectors and counts",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o4_more_rules,
				gnp2_t1_o3_fewer_rules,
			},
			client.QueryEndpointsReq{
				Policy: resourceKey(gnp1_t1_o3),
			},
			&client.QueryEndpointsResp{
				Count: 4,
				Items: []client.Endpoint{
					qcEndpoint(wep1_n1_ns1, 2, 1), qcEndpoint(wep3_n1_ns2, 2, 1), qcEndpoint(wep4_n2_ns1, 2, 0),
					qcEndpoint(hep1_n2, 2, 0),
				},
			},
		},
		{
			"updated GNPs orders and rules - endpoints matching gnp2-t1-o4;egress;idx=0;source;selector (should match previous idx=1)",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o4_more_rules,
				gnp2_t1_o3_fewer_rules,
			},
			client.QueryEndpointsReq{
				Policy:              resourceKey(gnp2_t1_o3_fewer_rules),
				RuleDirection:       "egress",
				RuleIndex:           0,
				RuleEntity:          "source",
				RuleNegatedSelector: false,
			},
			&client.QueryEndpointsResp{
				Count: 2,
				Items: []client.Endpoint{
					qcEndpoint(wep4_n2_ns1, 2, 0), qcEndpoint(hep1_n2, 2, 0),
				},
			},
		},
		{
			"updated GNPs orders and rules - endpoints matching gnp2-t1-o4;ingress;idx=0;destination;notSelector (should match previous idx=1)",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o4_more_rules,
				gnp2_t1_o3_fewer_rules,
			},
			client.QueryEndpointsReq{
				Policy:              resourceKey(gnp2_t1_o3_fewer_rules),
				RuleDirection:       "ingress",
				RuleIndex:           0,
				RuleEntity:          "destination",
				RuleNegatedSelector: true,
			},
			&client.QueryEndpointsResp{
				Count: 5,
				Items: []client.Endpoint{
					qcEndpoint(hep3_n4, 1, 0), qcEndpoint(wep1_n1_ns1, 2, 1),
					qcEndpoint(wep3_n1_ns2, 2, 1), qcEndpoint(wep4_n2_ns1, 2, 0), qcEndpoint(hep1_n2, 2, 0),
				},
			},
		},
		{
			"updated GNPs orders and rules - endpoints matching gnp2-t1-o4;ingress;idx=1;destination;notSelector (should not exist anymore)",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o4_more_rules,
				gnp2_t1_o3_fewer_rules,
			},
			client.QueryEndpointsReq{
				Policy:              resourceKey(gnp2_t1_o3_fewer_rules),
				RuleDirection:       "ingress",
				RuleIndex:           1,
				RuleEntity:          "destination",
				RuleNegatedSelector: true,
			},
			errors.New("rule parameters request is not valid: GlobalNetworkPolicy(ccc-tier1.gnp2-t1-o4)"),
		},
		{
			"updated GNPs orders and rules - endpoints matching gnp1_t1_o4;ingress;idx=0;destination;selector (should match rules from previous gnp2)",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o4_more_rules,
				gnp2_t1_o3_fewer_rules,
			},
			client.QueryEndpointsReq{
				Policy:              resourceKey(gnp1_t1_o4_more_rules),
				RuleDirection:       "ingress",
				RuleIndex:           0,
				RuleEntity:          "destination",
				RuleNegatedSelector: false,
			},
			&client.QueryEndpointsResp{
				Count: 2,
				Items: []client.Endpoint{
					qcEndpoint(hep3_n4, 1, 0), qcEndpoint(hep2_n3, 1, 0),
				},
			},
		},
		{
			"updated GNPs orders and rules - endpoints matching gnp1_t1_o4;egress;idx=0;source;notSelector (should match rules from previous gnp2)",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o4_more_rules,
				gnp2_t1_o3_fewer_rules,
			},
			client.QueryEndpointsReq{
				Policy:              resourceKey(gnp1_t1_o4_more_rules),
				RuleDirection:       "egress",
				RuleIndex:           0,
				RuleEntity:          "source",
				RuleNegatedSelector: true,
			},
			&client.QueryEndpointsResp{
				Count: 2,
				Items: []client.Endpoint{
					qcEndpoint(wep1_n1_ns1, 2, 1), qcEndpoint(wep3_n1_ns2, 2, 1),
				},
			},
		},
		{
			"updated GNPs orders and rules - endpoints matching gnp1_t1_o4;ingress;idx=1;destination;selector (should match all)",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o4_more_rules,
				gnp2_t1_o3_fewer_rules,
			},
			client.QueryEndpointsReq{
				Policy:              resourceKey(gnp1_t1_o4_more_rules),
				RuleDirection:       "ingress",
				RuleIndex:           1,
				RuleEntity:          "destination",
				RuleNegatedSelector: false,
			},
			&client.QueryEndpointsResp{
				Count: 8,
				Items: []client.Endpoint{
					qcEndpoint(hep4_n4_unlabelled, 1, 0), qcEndpoint(hep3_n4, 1, 0), qcEndpoint(wep1_n1_ns1, 2, 1),
					qcEndpoint(wep3_n1_ns2, 2, 1), qcEndpoint(wep4_n2_ns1, 2, 0), qcEndpoint(hep1_n2, 2, 0),
					qcEndpoint(wep5_n3_ns2_unlabelled, 1, 1), qcEndpoint(hep2_n3, 1, 0),
				},
			},
		},
		{
			"updated GNPs orders and rules - endpoints matching gnp1_t1_o4;egress;idx=1;source;notSelector (should match all)",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o4_more_rules,
				gnp2_t1_o3_fewer_rules,
			},
			client.QueryEndpointsReq{
				Policy:              resourceKey(gnp1_t1_o4_more_rules),
				RuleDirection:       "egress",
				RuleIndex:           1,
				RuleEntity:          "source",
				RuleNegatedSelector: true,
			},
			&client.QueryEndpointsResp{
				Count: 8,
				Items: []client.Endpoint{
					qcEndpoint(hep4_n4_unlabelled, 1, 0), qcEndpoint(hep3_n4, 1, 0), qcEndpoint(wep1_n1_ns1, 2, 1),
					qcEndpoint(wep3_n1_ns2, 2, 1), qcEndpoint(wep4_n2_ns1, 2, 0), qcEndpoint(hep1_n2, 2, 0),
					qcEndpoint(wep5_n3_ns2_unlabelled, 1, 1), qcEndpoint(hep2_n3, 1, 0),
				},
			},
		},
		{
			"multiple weps and heps, tier1 policy - query all of them; reverse sort",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
			},
			client.QueryEndpointsReq{
				Sort: &client.Sort{
					Reverse: true,
				},
			},
			&client.QueryEndpointsResp{
				Count: 8,
				Items: []client.Endpoint{
					qcEndpoint(hep2_n3, 1, 0), qcEndpoint(wep5_n3_ns2_unlabelled, 1, 1), qcEndpoint(hep1_n2, 2, 0),
					qcEndpoint(wep4_n2_ns1, 2, 0), qcEndpoint(wep3_n1_ns2, 2, 1), qcEndpoint(wep1_n1_ns1, 2, 1),
					qcEndpoint(hep3_n4, 1, 0), qcEndpoint(hep4_n4_unlabelled, 1, 0),
				},
			},
		},
		{
			"multiple weps and heps, tier1 policy - query all of them; sort by name and namespace",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
			},
			client.QueryEndpointsReq{
				Sort: &client.Sort{
					SortBy: []string{"name", "namespace"},
				},
			},
			&client.QueryEndpointsResp{
				Count: 8,
				Items: []client.Endpoint{
					qcEndpoint(hep4_n4_unlabelled, 1, 0), qcEndpoint(hep3_n4, 1, 0), qcEndpoint(wep1_n1_ns1, 2, 1),
					qcEndpoint(wep3_n1_ns2, 2, 1), qcEndpoint(wep4_n2_ns1, 2, 0), qcEndpoint(hep1_n2, 2, 0),
					qcEndpoint(wep5_n3_ns2_unlabelled, 1, 1), qcEndpoint(hep2_n3, 1, 0),
				},
			},
		},
		{
			"multiple weps and heps, tier1 policy - query all of them; sort by kind",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
			},
			client.QueryEndpointsReq{
				Sort: &client.Sort{
					SortBy: []string{"kind"},
				},
			},
			&client.QueryEndpointsResp{
				Count: 8,
				Items: []client.Endpoint{
					qcEndpoint(hep4_n4_unlabelled, 1, 0), qcEndpoint(hep3_n4, 1, 0), qcEndpoint(hep1_n2, 2, 0),
					qcEndpoint(hep2_n3, 1, 0), qcEndpoint(wep1_n1_ns1, 2, 1), qcEndpoint(wep3_n1_ns2, 2, 1),
					qcEndpoint(wep4_n2_ns1, 2, 0), qcEndpoint(wep5_n3_ns2_unlabelled, 1, 1),
				},
			},
		},
		{
			"multiple weps and heps, tier1 policy - query all of them; sort by namespace",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
			},
			client.QueryEndpointsReq{
				Sort: &client.Sort{
					SortBy: []string{"namespace"},
				},
			},
			&client.QueryEndpointsResp{
				Count: 8,
				Items: []client.Endpoint{
					qcEndpoint(hep4_n4_unlabelled, 1, 0), qcEndpoint(hep3_n4, 1, 0), qcEndpoint(hep1_n2, 2, 0),
					qcEndpoint(hep2_n3, 1, 0), qcEndpoint(wep1_n1_ns1, 2, 1), qcEndpoint(wep4_n2_ns1, 2, 0),
					qcEndpoint(wep3_n1_ns2, 2, 1), qcEndpoint(wep5_n3_ns2_unlabelled, 1, 1),
				},
			},
		},
		{
			"multiple weps and heps, tier1 policy - query all of them; sort by node",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
			},
			client.QueryEndpointsReq{
				Sort: &client.Sort{
					SortBy: []string{"node"},
				},
			},
			&client.QueryEndpointsResp{
				Count: 8,
				Items: []client.Endpoint{
					qcEndpoint(hep4_n4_unlabelled, 1, 0), qcEndpoint(hep3_n4, 1, 0), qcEndpoint(wep1_n1_ns1, 2, 1),
					qcEndpoint(wep3_n1_ns2, 2, 1), qcEndpoint(wep4_n2_ns1, 2, 0), qcEndpoint(hep1_n2, 2, 0),
					qcEndpoint(wep5_n3_ns2_unlabelled, 1, 1), qcEndpoint(hep2_n3, 1, 0),
				},
			},
		},
		{
			"multiple weps and heps, tier1 policy - query all of them; sort by orchestrator",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
			},
			client.QueryEndpointsReq{
				Sort: &client.Sort{
					SortBy: []string{"orchestrator"},
				},
			},
			&client.QueryEndpointsResp{
				Count: 8,
				Items: []client.Endpoint{
					qcEndpoint(hep4_n4_unlabelled, 1, 0), qcEndpoint(hep3_n4, 1, 0), qcEndpoint(hep1_n2, 2, 0),
					qcEndpoint(hep2_n3, 1, 0), qcEndpoint(wep5_n3_ns2_unlabelled, 1, 1),
					qcEndpoint(wep1_n1_ns1, 2, 1), qcEndpoint(wep3_n1_ns2, 2, 1), qcEndpoint(wep4_n2_ns1, 2, 0),
				},
			},
		},
		{
			"multiple weps and heps, tier1 policy - query all of them; sort by pod",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
			},
			client.QueryEndpointsReq{
				Sort: &client.Sort{
					SortBy: []string{"pod"},
				},
			},
			&client.QueryEndpointsResp{
				Count: 8,
				Items: []client.Endpoint{
					qcEndpoint(hep4_n4_unlabelled, 1, 0), qcEndpoint(hep3_n4, 1, 0), qcEndpoint(wep4_n2_ns1, 2, 0),
					qcEndpoint(hep1_n2, 2, 0), qcEndpoint(wep5_n3_ns2_unlabelled, 1, 1), qcEndpoint(hep2_n3, 1, 0),
					qcEndpoint(wep1_n1_ns1, 2, 1), qcEndpoint(wep3_n1_ns2, 2, 1),
				},
			},
		},
		{
			"multiple weps and heps, tier1 policy - query all of them; sort by workload",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
			},
			client.QueryEndpointsReq{
				Sort: &client.Sort{
					SortBy: []string{"workload"},
				},
			},
			&client.QueryEndpointsResp{
				Count: 8,
				Items: []client.Endpoint{
					qcEndpoint(hep4_n4_unlabelled, 1, 0), qcEndpoint(hep3_n4, 1, 0), qcEndpoint(wep1_n1_ns1, 2, 1),
					qcEndpoint(wep3_n1_ns2, 2, 1), qcEndpoint(hep1_n2, 2, 0), qcEndpoint(wep5_n3_ns2_unlabelled, 1, 1),
					qcEndpoint(hep2_n3, 1, 0), qcEndpoint(wep4_n2_ns1, 2, 0),
				},
			},
		},
		{
			"multiple weps and heps, tier1 policy - query all of them; sort by interfaceName",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
			},
			client.QueryEndpointsReq{
				Sort: &client.Sort{
					SortBy: []string{"interfaceName"},
				},
			},
			&client.QueryEndpointsResp{
				Count: 8,
				Items: []client.Endpoint{
					qcEndpoint(wep3_n1_ns2, 2, 1), qcEndpoint(wep1_n1_ns1, 2, 1),
					qcEndpoint(wep5_n3_ns2_unlabelled, 1, 1), qcEndpoint(wep4_n2_ns1, 2, 0),
					qcEndpoint(hep3_n4, 1, 0), qcEndpoint(hep4_n4_unlabelled, 1, 0),
					qcEndpoint(hep1_n2, 2, 0), qcEndpoint(hep2_n3, 1, 0),
				},
			},
		},
		{
			"multiple weps and heps, tier1 policy - query all of them; sort by ipNetworks",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
			},
			client.QueryEndpointsReq{
				Sort: &client.Sort{
					SortBy: []string{"ipNetworks"},
				},
			},
			&client.QueryEndpointsResp{
				Count: 8,
				Items: []client.Endpoint{
					qcEndpoint(hep4_n4_unlabelled, 1, 0), qcEndpoint(hep3_n4, 1, 0), qcEndpoint(hep2_n3, 1, 0),
					qcEndpoint(wep1_n1_ns1, 2, 1), qcEndpoint(wep3_n1_ns2, 2, 1), qcEndpoint(wep4_n2_ns1, 2, 0),
					qcEndpoint(wep5_n3_ns2_unlabelled, 1, 1), qcEndpoint(hep1_n2, 2, 0),
				},
			},
		},
		{
			"multiple weps and heps, tier1 policy - query all of them; sort by numGlobalNetworkPolicies",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
			},
			client.QueryEndpointsReq{
				Sort: &client.Sort{
					SortBy: []string{"numGlobalNetworkPolicies"},
				},
			},
			&client.QueryEndpointsResp{
				Count: 8,
				Items: []client.Endpoint{
					qcEndpoint(hep4_n4_unlabelled, 1, 0), qcEndpoint(hep3_n4, 1, 0),
					qcEndpoint(wep5_n3_ns2_unlabelled, 1, 1), qcEndpoint(hep2_n3, 1, 0),
					qcEndpoint(wep1_n1_ns1, 2, 1), qcEndpoint(wep3_n1_ns2, 2, 1), qcEndpoint(wep4_n2_ns1, 2, 0),
					qcEndpoint(hep1_n2, 2, 0),
				},
			},
		},
		{
			"multiple weps and heps, tier1 policy - query all of them; sort by numNetworkPolicies",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
			},
			client.QueryEndpointsReq{
				Sort: &client.Sort{
					SortBy: []string{"numNetworkPolicies"},
				},
			},
			&client.QueryEndpointsResp{
				Count: 8,
				Items: []client.Endpoint{
					qcEndpoint(hep4_n4_unlabelled, 1, 0), qcEndpoint(hep3_n4, 1, 0), qcEndpoint(wep4_n2_ns1, 2, 0),
					qcEndpoint(hep1_n2, 2, 0), qcEndpoint(hep2_n3, 1, 0), qcEndpoint(wep1_n1_ns1, 2, 1),
					qcEndpoint(wep3_n1_ns2, 2, 1), qcEndpoint(wep5_n3_ns2_unlabelled, 1, 1),
				},
			},
		},
		{
			"multiple weps and heps, tier1 policy - query all of them; sort by numPolicies",
			[]resourcemgr.ResourceObject{
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
			},
			client.QueryEndpointsReq{
				Sort: &client.Sort{
					SortBy: []string{"numPolicies"},
				},
			},
			&client.QueryEndpointsResp{
				Count: 8,
				Items: []client.Endpoint{
					qcEndpoint(hep4_n4_unlabelled, 1, 0), qcEndpoint(hep3_n4, 1, 0), qcEndpoint(hep2_n3, 1, 0),
					qcEndpoint(wep4_n2_ns1, 2, 0), qcEndpoint(hep1_n2, 2, 0), qcEndpoint(wep5_n3_ns2_unlabelled, 1, 1),
					qcEndpoint(wep1_n1_ns1, 2, 1), qcEndpoint(wep3_n1_ns2, 2, 1),
				},
			},
		},
		{
			"reset by removing all endpoints and policy; perform empty query",
			[]resourcemgr.ResourceObject{},
			client.QueryEndpointsReq{},
			&client.QueryEndpointsResp{
				Count: 0,
				Items: []client.Endpoint{},
			},
		},
	}
}
