package fv

import (
	"github.com/projectcalico/calicoctl/calicoctl/resourcemgr"
	"github.com/tigera/calicoq/web/pkg/querycache/client"
)

func policyTestQueryData() []testQueryData {
	// Create the query Policy resources for the tier 1 policies that have some selectors in the rules.  We create them
	// and tweak the rule counts to adjust for the selectors that are not all().
	qcPolicy_gnp2_t1_all_res := qcPolicy(gnp2_t1_o4, 4, 4, 4, 4)
	qcPolicy_gnp2_t1_all_res.Egress[0].Source.NotSelector.NumHostEndpoints = 0
	qcPolicy_gnp2_t1_all_res.Egress[0].Source.NotSelector.NumWorkloadEndpoints = 2
	qcPolicy_gnp2_t1_all_res.Egress[1].Source.Selector.NumHostEndpoints = 1
	qcPolicy_gnp2_t1_all_res.Egress[1].Source.Selector.NumWorkloadEndpoints = 1
	qcPolicy_gnp2_t1_all_res.Ingress[0].Destination.Selector.NumHostEndpoints = 2
	qcPolicy_gnp2_t1_all_res.Ingress[0].Destination.Selector.NumWorkloadEndpoints = 0
	qcPolicy_gnp2_t1_all_res.Ingress[1].Destination.NotSelector.NumHostEndpoints = 2
	qcPolicy_gnp2_t1_all_res.Ingress[1].Destination.NotSelector.NumWorkloadEndpoints = 3

	qcPolicy_gnp1_t1_all_res_more := qcPolicy(gnp1_t1_o4_more_rules, 1, 3	, 4, 4)
	qcPolicy_gnp1_t1_all_res_more.Egress[0].Source.NotSelector.NumHostEndpoints = 0
	qcPolicy_gnp1_t1_all_res_more.Egress[0].Source.NotSelector.NumWorkloadEndpoints = 2
	qcPolicy_gnp1_t1_all_res_more.Ingress[0].Destination.Selector.NumHostEndpoints = 2
	qcPolicy_gnp1_t1_all_res_more.Ingress[0].Destination.Selector.NumWorkloadEndpoints = 0

	qcPolicy_gnp2_t1_all_res_fewer := qcPolicy(gnp2_t1_o3_fewer_rules, 4, 4, 4, 4)
	qcPolicy_gnp2_t1_all_res_fewer.Egress[0].Source.Selector.NumHostEndpoints = 1
	qcPolicy_gnp2_t1_all_res_fewer.Egress[0].Source.Selector.NumWorkloadEndpoints = 1
	qcPolicy_gnp2_t1_all_res_fewer.Ingress[0].Destination.NotSelector.NumHostEndpoints = 2
	qcPolicy_gnp2_t1_all_res_fewer.Ingress[0].Destination.NotSelector.NumWorkloadEndpoints = 3

	qcPolicy_gnp1_t1_all_res_more_updated_wep1 := qcPolicy(gnp1_t1_o4_more_rules, 1, 2, 4, 4)
	qcPolicy_gnp1_t1_all_res_more_updated_wep1.Egress[0].Source.NotSelector.NumHostEndpoints = 0
	qcPolicy_gnp1_t1_all_res_more_updated_wep1.Egress[0].Source.NotSelector.NumWorkloadEndpoints = 1
	qcPolicy_gnp1_t1_all_res_more_updated_wep1.Ingress[0].Destination.Selector.NumHostEndpoints = 2
	qcPolicy_gnp1_t1_all_res_more_updated_wep1.Ingress[0].Destination.Selector.NumWorkloadEndpoints = 1

	qcPolicy_gnp2_t1_all_res_fewer_updated_wep1 := qcPolicy(gnp2_t1_o3_fewer_rules, 4, 4, 4, 4)
	qcPolicy_gnp2_t1_all_res_fewer_updated_wep1.Egress[0].Source.Selector.NumHostEndpoints = 1
	qcPolicy_gnp2_t1_all_res_fewer_updated_wep1.Egress[0].Source.Selector.NumWorkloadEndpoints = 1
	qcPolicy_gnp2_t1_all_res_fewer_updated_wep1.Ingress[0].Destination.NotSelector.NumHostEndpoints = 2
	qcPolicy_gnp2_t1_all_res_fewer_updated_wep1.Ingress[0].Destination.NotSelector.NumWorkloadEndpoints = 3

	qcPolicy_gnp2_t1_some_unmatched := qcPolicy(gnp2_t1_o4, 3, 2, 3, 2)
	qcPolicy_gnp2_t1_some_unmatched.Egress[0].Source.NotSelector.NumHostEndpoints = 0
	qcPolicy_gnp2_t1_some_unmatched.Egress[0].Source.NotSelector.NumWorkloadEndpoints = 1
	qcPolicy_gnp2_t1_some_unmatched.Egress[1].Source.Selector.NumHostEndpoints = 1
	qcPolicy_gnp2_t1_some_unmatched.Egress[1].Source.Selector.NumWorkloadEndpoints = 1
	qcPolicy_gnp2_t1_some_unmatched.Ingress[0].Destination.Selector.NumHostEndpoints = 2
	qcPolicy_gnp2_t1_some_unmatched.Ingress[0].Destination.Selector.NumWorkloadEndpoints = 0
	qcPolicy_gnp2_t1_some_unmatched.Ingress[1].Destination.NotSelector.NumHostEndpoints = 2
	qcPolicy_gnp2_t1_some_unmatched.Ingress[1].Destination.NotSelector.NumWorkloadEndpoints = 2

	return []testQueryData{
		{
			"multiple gnps and nps, no endpoints - query exact np",
			[]resourcemgr.ResourceObject{
				tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
				tier2, np1_t2_o1_ns1, np2_t2_o2_ns2, gnp1_t2_o3, gnp2_t2_o4,
			},
			client.QueryPoliciesReq{
				Policy: resourceKey(np1_t2_o1_ns1),
			},
			&client.QueryPoliciesResp{
				Count: 1,
				Items: []client.Policy{qcPolicy(np1_t2_o1_ns1, 0, 0, 0, 0)},
			},
		},
		{
			"multiple gnps and nps, no endpoints - query exact gnp",
			[]resourcemgr.ResourceObject{
				tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
				tier2, np1_t2_o1_ns1, np2_t2_o2_ns2, gnp1_t2_o3, gnp2_t2_o4,
			},
			client.QueryPoliciesReq{
				Policy: resourceKey(gnp1_t1_o3),
			},
			&client.QueryPoliciesResp{
				Count: 1,
				Items: []client.Policy{qcPolicy(gnp1_t1_o3, 0, 0, 0, 0)},
			},
		},
		{
			"multiple gnps and nps, no endpoints - query all of them",
			[]resourcemgr.ResourceObject{
				tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
				tier2, np1_t2_o1_ns1, np2_t2_o2_ns2, gnp1_t2_o3, gnp2_t2_o4,
			},
			client.QueryPoliciesReq{},
			&client.QueryPoliciesResp{
				Count: 8,
				Items: []client.Policy{
					qcPolicy(np1_t1_o1_ns1, 0, 0, 0, 0), qcPolicy(np2_t1_o2_ns2, 0, 0, 0, 0),
					qcPolicy(gnp1_t1_o3, 0, 0, 0, 0), qcPolicy(gnp2_t1_o4, 0, 0, 0, 0),
					qcPolicy(np1_t2_o1_ns1, 0, 0, 0, 0), qcPolicy(np2_t2_o2_ns2, 0, 0, 0, 0),
					qcPolicy(gnp1_t2_o3, 0, 0, 0, 0), qcPolicy(gnp2_t2_o4, 0, 0, 0, 0),
				},
			},
		},
		{
			"multiple gnps and nps, no endpoints - query all of them - page 0 of 2",
			[]resourcemgr.ResourceObject{
				tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
				tier2, np1_t2_o1_ns1, np2_t2_o2_ns2, gnp1_t2_o3, gnp2_t2_o4,
			},
			client.QueryPoliciesReq{
				Page: &client.Page{
					PageNum:    0,
					NumPerPage: 5,
				},
			},
			&client.QueryPoliciesResp{
				Count: 8,
				Items: []client.Policy{
					qcPolicy(np1_t1_o1_ns1, 0, 0, 0, 0), qcPolicy(np2_t1_o2_ns2, 0, 0, 0, 0),
					qcPolicy(gnp1_t1_o3, 0, 0, 0, 0), qcPolicy(gnp2_t1_o4, 0, 0, 0, 0),
					qcPolicy(np1_t2_o1_ns1, 0, 0, 0, 0),
				},
			},
		},
		{
			"multiple gnps and nps, no endpoints - query all of them - page 1 of 2",
			[]resourcemgr.ResourceObject{
				tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
				tier2, np1_t2_o1_ns1, np2_t2_o2_ns2, gnp1_t2_o3, gnp2_t2_o4,
			},
			client.QueryPoliciesReq{
				Page: &client.Page{
					PageNum:    1,
					NumPerPage: 5,
				},
			},
			&client.QueryPoliciesResp{
				Count: 8,
				Items: []client.Policy{
					qcPolicy(np2_t2_o2_ns2, 0, 0, 0, 0),
					qcPolicy(gnp1_t2_o3, 0, 0, 0, 0), qcPolicy(gnp2_t2_o4, 0, 0, 0, 0),
				},
			},
		},
		{
			"multiple gnps and nps, no endpoints - query all of them - page 0 of 2",
			[]resourcemgr.ResourceObject{
				tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
				tier2, np1_t2_o1_ns1, np2_t2_o2_ns2, gnp1_t2_o3, gnp2_t2_o4,
			},
			client.QueryPoliciesReq{
				Page: &client.Page{
					PageNum:    2,
					NumPerPage: 5,
				},
			},
			&client.QueryPoliciesResp{
				Count: 8,
				Items: []client.Policy{},
			},
		},
		{
			"multiple gnps and nps, no endpoints, reordered tiers - query all of them",
			[]resourcemgr.ResourceObject{
				tier1_o2, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
				tier2_o1, np1_t2_o1_ns1, np2_t2_o2_ns2, gnp1_t2_o3, gnp2_t2_o4,
			},
			client.QueryPoliciesReq{},
			&client.QueryPoliciesResp{
				Count: 8,
				Items: []client.Policy{
					qcPolicy(np1_t2_o1_ns1, 0, 0, 0, 0), qcPolicy(np2_t2_o2_ns2, 0, 0, 0, 0),
					qcPolicy(gnp1_t2_o3, 0, 0, 0, 0), qcPolicy(gnp2_t2_o4, 0, 0, 0, 0),
					qcPolicy(np1_t1_o1_ns1, 0, 0, 0, 0), qcPolicy(np2_t1_o2_ns2, 0, 0, 0, 0),
					qcPolicy(gnp1_t1_o3, 0, 0, 0, 0), qcPolicy(gnp2_t1_o4, 0, 0, 0, 0),
				},
			},
		},
		{
			"multiple gnps and nps, no policies, reordered policies - query all of them",
			[]resourcemgr.ResourceObject{
				tier1_o2, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o4_more_rules, gnp2_t1_o3_fewer_rules,
				tier2_o1, np1_t2_o1_ns1, np2_t2_o2_ns2, gnp1_t2_o3, gnp2_t2_o4,
			},
			client.QueryPoliciesReq{},
			&client.QueryPoliciesResp{
				Count: 8,
				Items: []client.Policy{
					qcPolicy(np1_t2_o1_ns1, 0, 0, 0, 0), qcPolicy(np2_t2_o2_ns2, 0, 0, 0, 0),
					qcPolicy(gnp1_t2_o3, 0, 0, 0, 0), qcPolicy(gnp2_t2_o4, 0, 0, 0, 0),
					qcPolicy(np1_t1_o1_ns1, 0, 0, 0, 0), qcPolicy(np2_t1_o2_ns2, 0, 0, 0, 0),
					qcPolicy(gnp2_t1_o3_fewer_rules, 0, 0, 0, 0), qcPolicy(gnp1_t1_o4_more_rules, 0, 0, 0, 0),
				},
			},
		},
		{
			"bunch of endpoints and tier2 policies (rules selectors are all()) - query all of them",
			[]resourcemgr.ResourceObject{
				tier2, np1_t2_o1_ns1, np2_t2_o2_ns2, gnp1_t2_o3, gnp2_t2_o4,
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, wep2_n1_ns1_filtered_out,
			},
			client.QueryPoliciesReq{},
			&client.QueryPoliciesResp{
				Count: 4,
				Items: []client.Policy{
					qcPolicy(np1_t2_o1_ns1, 0, 1, 4, 4), qcPolicy(np2_t2_o2_ns2, 0, 2, 4, 4),
					qcPolicy(gnp1_t2_o3, 1, 1, 4, 4), qcPolicy(gnp2_t2_o4, 4, 4, 4, 4),
				},
			},
		},
		{
			"bunch of endpoints and tier2 policies - filter in wep2",
			[]resourcemgr.ResourceObject{
				tier2, np1_t2_o1_ns1, np2_t2_o2_ns2, gnp1_t2_o3, gnp2_t2_o4,
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, wep2_n1_ns1_filtered_in,
			},
			client.QueryPoliciesReq{},
			&client.QueryPoliciesResp{
				Count: 4,
				Items: []client.Policy{
					qcPolicy(np1_t2_o1_ns1, 0, 1, 4, 5), qcPolicy(np2_t2_o2_ns2, 0, 2, 4, 5),
					qcPolicy(gnp1_t2_o3, 1, 1, 4, 5), qcPolicy(gnp2_t2_o4, 4, 5, 4, 5),
				},
			},
		},
		{
			"bunch of endpoints and tier 1 and tier2 policies - query all of them (gnp2 has rule selectors)",
			[]resourcemgr.ResourceObject{
				tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
				tier2, np1_t2_o1_ns1, np2_t2_o2_ns2, gnp1_t2_o3, gnp2_t2_o4,
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, wep2_n1_ns1_filtered_out,
			},
			client.QueryPoliciesReq{},
			&client.QueryPoliciesResp{
				Count: 8,
				Items: []client.Policy{
					qcPolicy(np1_t1_o1_ns1, 0, 1, 4, 4), qcPolicy(np2_t1_o2_ns2, 0, 2, 4, 4),
					qcPolicy(gnp1_t1_o3, 1, 3, 4, 4), qcPolicy_gnp2_t1_all_res,
					qcPolicy(np1_t2_o1_ns1, 0, 1, 4, 4), qcPolicy(np2_t2_o2_ns2, 0, 2, 4, 4),
					qcPolicy(gnp1_t2_o3, 1, 1, 4, 4), qcPolicy(gnp2_t2_o4, 4, 4, 4, 4),
				},
			},
		},
		{
			"bunch of endpoints and tier 1 and tier2 policies - query tier 2",
			[]resourcemgr.ResourceObject{
				tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
				tier2, np1_t2_o1_ns1, np2_t2_o2_ns2, gnp1_t2_o3, gnp2_t2_o4,
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, wep2_n1_ns1_filtered_out,
			},
			client.QueryPoliciesReq{
				Tier: tier2.Name,
			},
			&client.QueryPoliciesResp{
				Count: 4,
				Items: []client.Policy{
					qcPolicy(np1_t2_o1_ns1, 0, 1, 4, 4), qcPolicy(np2_t2_o2_ns2, 0, 2, 4, 4),
					qcPolicy(gnp1_t2_o3, 1, 1, 4, 4), qcPolicy(gnp2_t2_o4, 4, 4, 4, 4),
				},
			},
		},
		{
			"bunch of endpoints and tier 1 and tier2 policies - query tier 1",
			[]resourcemgr.ResourceObject{
				tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
				tier2, np1_t2_o1_ns1, np2_t2_o2_ns2, gnp1_t2_o3, gnp2_t2_o4,
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, wep2_n1_ns1_filtered_out,
			},
			client.QueryPoliciesReq{
				Tier: tier1.Name,
			},
			&client.QueryPoliciesResp{
				Count: 4,
				Items: []client.Policy{
					qcPolicy(np1_t1_o1_ns1, 0, 1, 4, 4), qcPolicy(np2_t1_o2_ns2, 0, 2, 4, 4),
					qcPolicy(gnp1_t1_o3, 1, 3, 4, 4), qcPolicy_gnp2_t1_all_res,
				},
			},
		},
		{
			"bunch of endpoints and tier 1 and tier2 policies - query policies matching labels",
			[]resourcemgr.ResourceObject{
				tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
				tier2, np1_t2_o1_ns1, np2_t2_o2_ns2, gnp1_t2_o3, gnp2_t2_o4,
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, wep2_n1_ns1_filtered_out,
			},
			client.QueryPoliciesReq{
				Labels: map[string]string {
					"projectcalico.org/namespace": "namespace-1",
					"rack": "001",
					"server": "1",
				},
			},
			&client.QueryPoliciesResp{
				Count: 4,
				Items: []client.Policy{
					qcPolicy(np1_t1_o1_ns1, 0, 1, 4, 4), qcPolicy(gnp1_t1_o3, 1, 3, 4, 4), qcPolicy_gnp2_t1_all_res,
					qcPolicy(gnp2_t2_o4, 4, 4, 4, 4),
				},
			},
		},
		{
			"bunch of endpoints and tier 1 and tier2 policies - query policies matching endpoint",
			[]resourcemgr.ResourceObject{
				tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
				tier2, np1_t2_o1_ns1, np2_t2_o2_ns2, gnp1_t2_o3, gnp2_t2_o4,
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, wep2_n1_ns1_filtered_out,
			},
			client.QueryPoliciesReq{
				Endpoint: resourceKey(wep4_n2_ns1),
			},
			&client.QueryPoliciesResp{
				Count: 4,
				Items: []client.Policy{
					qcPolicy(gnp1_t1_o3, 1, 3, 4, 4), qcPolicy_gnp2_t1_all_res,
					qcPolicy(np1_t2_o1_ns1, 0, 1, 4, 4), qcPolicy(gnp2_t2_o4, 4, 4, 4, 4),
				},
			},
		},
		{
			"bunch of endpoints and tier 1 and tier2 policies - query policies matching labels and endpoint",
			[]resourcemgr.ResourceObject{
				tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
				tier2, np1_t2_o1_ns1, np2_t2_o2_ns2, gnp1_t2_o3, gnp2_t2_o4,
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, wep2_n1_ns1_filtered_out,
			},
			client.QueryPoliciesReq{
				Labels: map[string]string {
					"projectcalico.org/namespace": "namespace-1",
					"rack": "001",
					"server": "1",
				},
				Endpoint: resourceKey(wep4_n2_ns1),
			},
			&client.QueryPoliciesResp{
				Count: 3,
				Items: []client.Policy{
					qcPolicy(gnp1_t1_o3, 1, 3, 4, 4), qcPolicy_gnp2_t1_all_res,
					qcPolicy(gnp2_t2_o4, 4, 4, 4, 4),
				},
			},
		},
		{
			"bunch of endpoints and tier 1 and tier2 policies - query policies matching labels, endpoint and tier",
			[]resourcemgr.ResourceObject{
				tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
				tier2, np1_t2_o1_ns1, np2_t2_o2_ns2, gnp1_t2_o3, gnp2_t2_o4,
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, wep2_n1_ns1_filtered_out,
			},
			client.QueryPoliciesReq{
				Labels: map[string]string {
					"projectcalico.org/namespace": "namespace-1",
					"rack": "001",
					"server": "1",
				},
				Endpoint: resourceKey(wep4_n2_ns1),
				Tier: tier2.Name,
			},
			&client.QueryPoliciesResp{
				Count: 1,
				Items: []client.Policy{
					qcPolicy(gnp2_t2_o4, 4, 4, 4, 4),
				},
			},
		},
		{
			"gnp1 and gnp2 #rules and order updated - query all of them",
			[]resourcemgr.ResourceObject{
				tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o4_more_rules, gnp2_t1_o3_fewer_rules,
				tier2, np1_t2_o1_ns1, np2_t2_o2_ns2, gnp1_t2_o3, gnp2_t2_o4,
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_001, wep1_n1_ns1,
				wep5_n3_ns2_unlabelled, wep2_n1_ns1_filtered_out,
			},
			client.QueryPoliciesReq{},
			&client.QueryPoliciesResp{
				Count: 8,
				Items: []client.Policy{
					qcPolicy(np1_t1_o1_ns1, 0, 1, 4, 4), qcPolicy(np2_t1_o2_ns2, 0, 2, 4, 4),
					qcPolicy_gnp2_t1_all_res_fewer, qcPolicy_gnp1_t1_all_res_more,
					qcPolicy(np1_t2_o1_ns1, 0, 1, 4, 4), qcPolicy(np2_t2_o2_ns2, 0, 2, 4, 4),
					qcPolicy(gnp1_t2_o3, 1, 1, 4, 4), qcPolicy(gnp2_t2_o4, 4, 4, 4, 4),
				},
			},
		},
		{
			"gnp1 and gnp2 #rules and order updated, updated WEP1 profile - query all of them",
			[]resourcemgr.ResourceObject{
				tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o4_more_rules, gnp2_t1_o3_fewer_rules,
				tier2, np1_t2_o1_ns1, np2_t2_o2_ns2, gnp1_t2_o3, gnp2_t2_o4,
				hep2_n3, hep3_n4, hep1_n2, hep4_n4_unlabelled, wep4_n2_ns1, wep3_n1_ns2, profile_rack_099,
				wep1_n1_ns1_updated_profile, wep5_n3_ns2_unlabelled, wep2_n1_ns1_filtered_out,
			},
			client.QueryPoliciesReq{},
			&client.QueryPoliciesResp{
				Count: 8,
				Items: []client.Policy{
					qcPolicy(np1_t1_o1_ns1, 0, 0, 4, 4), qcPolicy(np2_t1_o2_ns2, 0, 2, 4, 4),
					qcPolicy_gnp2_t1_all_res_fewer_updated_wep1, qcPolicy_gnp1_t1_all_res_more_updated_wep1,
					qcPolicy(np1_t2_o1_ns1, 0, 1, 4, 4), qcPolicy(np2_t2_o2_ns2, 0, 2, 4, 4),
					qcPolicy(gnp1_t2_o3, 1, 1, 4, 4), qcPolicy(gnp2_t2_o4, 4, 4, 4, 4),
				},
			},
		},
		{
			"tier1 and tier2 policies, but no namespace-2 endpoints and no rackless endpoints; some policies unmatched",
			[]resourcemgr.ResourceObject{
				tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
				tier2, np1_t2_o1_ns1, np2_t2_o2_ns2, gnp1_t2_o3, gnp2_t2_o4,
				hep2_n3, hep3_n4, hep1_n2, wep4_n2_ns1, profile_rack_001, wep1_n1_ns1,
				wep2_n1_ns1_filtered_out,
			},
			client.QueryPoliciesReq{},
			&client.QueryPoliciesResp{
				Count: 8,
				Items: []client.Policy{
					qcPolicy(np1_t1_o1_ns1, 0, 1, 3, 2), qcPolicy(np2_t1_o2_ns2, 0, 0, 3, 2),
					qcPolicy(gnp1_t1_o3, 1, 2, 3, 2), qcPolicy_gnp2_t1_some_unmatched,
					qcPolicy(np1_t2_o1_ns1, 0, 1, 3, 2), qcPolicy(np2_t2_o2_ns2, 0, 0, 3, 2),
					qcPolicy(gnp1_t2_o3, 0, 0, 3, 2), qcPolicy(gnp2_t2_o4, 3, 2, 3, 2),
				},
			},
		},
		{
			"tier1 and tier2 policies, but no namespace-2 endpoints and no rackless endpoints; some policies unmatched; filter on unmatched",
			[]resourcemgr.ResourceObject{
				tier1, np1_t1_o1_ns1, np2_t1_o2_ns2, gnp1_t1_o3, gnp2_t1_o4,
				tier2, np1_t2_o1_ns1, np2_t2_o2_ns2, gnp1_t2_o3, gnp2_t2_o4,
				hep2_n3, hep3_n4, hep1_n2, wep4_n2_ns1, profile_rack_001, wep1_n1_ns1,
				wep2_n1_ns1_filtered_out,
			},
			client.QueryPoliciesReq{
				Unmatched: true,
			},
			&client.QueryPoliciesResp{
				Count: 3,
				Items: []client.Policy{
					qcPolicy(np2_t1_o2_ns2, 0, 0, 3, 2), qcPolicy(np2_t2_o2_ns2, 0, 0, 3, 2),
					qcPolicy(gnp1_t2_o3, 0, 0, 3, 2),
				},
			},
		},
		{
			"reset by removing all endpoints and policy; perform empty query",
			[]resourcemgr.ResourceObject{},
			client.QueryPoliciesReq{},
			&client.QueryPoliciesResp{
				Count: 0,
				Items: []client.Policy{},
			},
		},
	}
}
