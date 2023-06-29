// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package fv

import (
	lapi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
)

// Setp-4 results for most tests. Consists of two returned flows.
var Step4DomainWithNamespacesResults = []rest.MockResult{
	{
		Body: lapi.List[lapi.L3Flow]{
			TotalHits: 3,
			Items: []lapi.L3Flow{
				{
					// key-1
					Key: lapi.L3FlowKey{
						Action:   lapi.FlowActionAllow,
						Reporter: lapi.FlowReporterSource,
						Source: lapi.Endpoint{
							Namespace:      "namespace5",
							AggregatedName: "netshoot5-cb7967547-*",
							Type:           lapi.WEP,
						},
						Destination: lapi.Endpoint{
							Name:           "-",
							Namespace:      "-",
							AggregatedName: "pvt",
							Type:           lapi.Network,
							Port:           101,
						},
						Protocol: "udp",
					},
					LogStats:    &lapi.LogStats{FlowLogCount: 1},
					DestDomains: []string{"namespace2service.namespace2.svc.cluster.local"},
				},
				{
					// key-2
					Key: lapi.L3FlowKey{
						Action:   lapi.FlowActionAllow,
						Reporter: lapi.FlowReporterSource,
						Source: lapi.Endpoint{
							Namespace:      "namespace5",
							AggregatedName: "namespace5-cb7967547-*",
							Type:           lapi.WEP,
						},
						Destination: lapi.Endpoint{
							Namespace:      "-",
							AggregatedName: "pub",
							Type:           lapi.Network,
							Port:           99,
						},
						Protocol: "udp",
					},
					LogStats:    &lapi.LogStats{FlowLogCount: 3698},
					DestDomains: []string{"www.website.com"},
				},
				{
					// key-3
					Key: lapi.L3FlowKey{
						Action:   lapi.FlowActionAllow,
						Reporter: lapi.FlowReporterSource,
						Source: lapi.Endpoint{
							Namespace:      "namespace5",
							AggregatedName: "namespace5-cb7967547-*",
							Type:           lapi.WEP,
						},
						Destination: lapi.Endpoint{
							Name:           "-",
							Namespace:      "-",
							AggregatedName: "pvt",
							Type:           lapi.Network,
							Port:           99,
						},
						Protocol: "tcp",
					},
					LogStats:    &lapi.LogStats{FlowLogCount: 3698},
					DestDomains: []string{"namespace3service.namespace3.svc.cluster.local"},
				},
			},
		},
	},
}
