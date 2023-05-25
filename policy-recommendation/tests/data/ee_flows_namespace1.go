// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package fv

import (
	lapi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
)

// Setp-6 results for most tests. Consists of two returned flows.
var Step6Results = []rest.MockResult{
	{
		Body: lapi.List[lapi.L3Flow]{
			TotalHits: 7,
			Items: []lapi.L3Flow{
				{
					// key-1
					Key: lapi.L3FlowKey{
						Action:   lapi.FlowActionAllow,
						Reporter: lapi.FlowReporterSource,
						Source: lapi.Endpoint{
							Namespace:      "namespace1",
							AggregatedName: "namespace1-cb7967547-*",
							Type:           lapi.WEP,
						},
						Destination: lapi.Endpoint{
							Namespace:      "namespace4",
							AggregatedName: "netshoot-n4-cb7967547-*",
							Type:           lapi.WEP,
							Port:           80,
						},
						Protocol: "tcp",
					},
					Service: &lapi.Service{
						Name: "-",
					},
					LogStats: &lapi.LogStats{FlowLogCount: 4370},
				},
				{
					// key-2
					Key: lapi.L3FlowKey{
						Action:   lapi.FlowActionAllow,
						Reporter: lapi.FlowReporterSource,
						Source: lapi.Endpoint{
							Namespace:      "namespace1",
							AggregatedName: "namespace1-cb7967547-*",
							Type:           lapi.WEP,
						},
						Destination: lapi.Endpoint{
							Namespace:      "namespace2",
							AggregatedName: "netshoot-ns2-cb7967547-*",
							Type:           lapi.WEP,
							Port:           81,
						},
						Protocol: "tcp",
					},
					Service: &lapi.Service{
						Name: "-",
					},
					LogStats: &lapi.LogStats{FlowLogCount: 3698},
				},
				{
					// key-3
					Key: lapi.L3FlowKey{
						Action:   lapi.FlowActionAllow,
						Reporter: lapi.FlowReporterSource,
						Source: lapi.Endpoint{
							Namespace:      "namespace3",
							AggregatedName: "namespace3-cb7967547-*",
							Type:           lapi.WEP,
						},
						Destination: lapi.Endpoint{
							Namespace:      "namespace5",
							AggregatedName: "namespace5-cb7967547-*",
							Type:           lapi.WEP,
							Port:           666,
						},
						Protocol: "tcp",
					},
					Service: &lapi.Service{
						Name: "-",
					},
					LogStats: &lapi.LogStats{FlowLogCount: 3698},
				},
				{
					// key-4
					Key: lapi.L3FlowKey{
						Action:   lapi.FlowActionAllow,
						Reporter: lapi.FlowReporterSource,
						Source: lapi.Endpoint{
							Namespace:      "namespace1",
							AggregatedName: "namespace1-cb7967547-*",
							Type:           lapi.WEP,
						},
						Destination: lapi.Endpoint{
							Namespace:      "namespace3",
							AggregatedName: "namespace3-cb7967547-*",
							Type:           lapi.WEP,
							Port:           81,
						},
						Protocol: "tcp",
					},
					Service: &lapi.Service{
						Name: "-",
					},
					LogStats: &lapi.LogStats{FlowLogCount: 3698},
				},
				{
					// key-5
					Key: lapi.L3FlowKey{
						Action:   lapi.FlowActionAllow,
						Reporter: lapi.FlowReporterSource,
						Source: lapi.Endpoint{
							Namespace:      "namespace3",
							AggregatedName: "namespace3-cb7967547-*",
							Type:           lapi.WEP,
						},
						Destination: lapi.Endpoint{
							Namespace:      "namespace5",
							AggregatedName: "namespace5-cb7967547-*",
							Type:           lapi.WEP,
							Port:           667,
						},
						Protocol: "tcp",
					},
					Service: &lapi.Service{
						Name: "-",
					},
					LogStats: &lapi.LogStats{FlowLogCount: 3698},
				},
				{
					// key-6
					Key: lapi.L3FlowKey{
						Action:   lapi.FlowActionAllow,
						Reporter: lapi.FlowReporterSource,
						Source: lapi.Endpoint{
							Namespace:      "namespace3",
							AggregatedName: "namespace3-cb7967547-*",
							Type:           lapi.WEP,
						},
						Destination: lapi.Endpoint{
							Namespace:      "namespace2",
							AggregatedName: "namespace2-cb7967547-*",
							Type:           lapi.WEP,
							Port:           9090,
						},
						Protocol: "udp",
					},
					Service: &lapi.Service{
						Name: "-",
					},
					LogStats: &lapi.LogStats{FlowLogCount: 3698},
				},
				{
					// key-7
					Key: lapi.L3FlowKey{
						Action:   lapi.FlowActionAllow,
						Reporter: lapi.FlowReporterSource,
						Source: lapi.Endpoint{
							Namespace:      "namespace5",
							AggregatedName: "namespace5-cb7967547-*",
							Type:           lapi.WEP,
						},
						Destination: lapi.Endpoint{
							Namespace:      "namespace1",
							AggregatedName: "namespace1-cb7967547-*",
							Type:           lapi.WEP,
							Port:           1,
						},
						Protocol: "tcp",
					},
					Service: &lapi.Service{
						Name: "-",
					},
					LogStats: &lapi.LogStats{FlowLogCount: 3698},
				},
			},
		},
	},
}
