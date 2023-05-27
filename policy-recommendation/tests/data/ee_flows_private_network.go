// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package fv

import (
	lapi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
)

// Private network linseed results for most tests.
var PrivateNetworkLinseedResults = []rest.MockResult{
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
							AggregatedName: "netshoot-cb7967547-*",
							Type:           lapi.WEP,
						},
						Destination: lapi.Endpoint{
							Namespace:      "-",
							AggregatedName: "pvt",
							Type:           lapi.Network,
							Port:           8080,
						},
						Protocol: "tcp",
					},
					LogStats: &lapi.LogStats{FlowLogCount: 4370},
				},
				{
					// key-2
					Key: lapi.L3FlowKey{
						Action:   lapi.FlowActionAllow,
						Reporter: lapi.FlowReporterDest,
						Source: lapi.Endpoint{
							Namespace:      "-",
							AggregatedName: "pvt",
							Type:           lapi.Network,
						},
						Destination: lapi.Endpoint{
							Namespace:      "namespace1",
							AggregatedName: "netshoot-cb7967547-*",
							Type:           lapi.WEP,
							Port:           5,
						},
						Protocol: "tcp",
					},
					LogStats: &lapi.LogStats{FlowLogCount: 4370},
				},
				{
					// key-3
					Key: lapi.L3FlowKey{
						Action:   lapi.FlowActionAllow,
						Reporter: lapi.FlowReporterSource,
						Source: lapi.Endpoint{
							Namespace:      "namespace1",
							AggregatedName: "netshoot-cb7967547-*",
							Type:           lapi.WEP,
						},
						Destination: lapi.Endpoint{
							Namespace:      "-",
							AggregatedName: "pvt",
							Type:           lapi.Network,
							Port:           8081,
						},
						Protocol: "tcp",
					},
					LogStats: &lapi.LogStats{FlowLogCount: 4370},
				},
				{
					// key-4
					Key: lapi.L3FlowKey{
						Action:   lapi.FlowActionAllow,
						Reporter: lapi.FlowReporterSource,
						Source: lapi.Endpoint{
							Namespace:      "namespace1",
							AggregatedName: "netshoot-cb7967547-*",
							Type:           lapi.WEP,
						},
						Destination: lapi.Endpoint{
							Namespace:      "-",
							AggregatedName: "pvt",
							Type:           lapi.Network,
							Port:           90,
						},
						Protocol: "tcp",
					},
					LogStats: &lapi.LogStats{FlowLogCount: 4370},
				},
				{
					// key-5
					Key: lapi.L3FlowKey{
						Action:   lapi.FlowActionAllow,
						Reporter: lapi.FlowReporterSource,
						Source: lapi.Endpoint{
							Namespace:      "namespace1",
							AggregatedName: "netshoot-cb7967547-*",
							Type:           lapi.WEP,
						},
						Destination: lapi.Endpoint{
							Namespace:      "-",
							AggregatedName: "pvt",
							Type:           lapi.Network,
							Port:           33,
						},
						Protocol: "tcp",
					},
					LogStats: &lapi.LogStats{FlowLogCount: 4370},
				},
				{
					// key-6
					Key: lapi.L3FlowKey{
						Action:   lapi.FlowActionAllow,
						Reporter: lapi.FlowReporterDest,
						Source: lapi.Endpoint{
							Namespace:      "-",
							AggregatedName: "pvt",
							Type:           lapi.Network,
						},
						Destination: lapi.Endpoint{
							Namespace:      "namespace1",
							AggregatedName: "netshoot-cb7967547-*",
							Type:           lapi.WEP,
							Port:           80,
						},
						Protocol: "tcp",
					},
					LogStats: &lapi.LogStats{FlowLogCount: 4370},
				},
				{
					// key-7
					Key: lapi.L3FlowKey{
						Action:   lapi.FlowActionAllow,
						Reporter: lapi.FlowReporterSource,
						Source: lapi.Endpoint{
							Namespace:      "namespace1",
							AggregatedName: "netshoot-cb7967547-*",
							Type:           lapi.WEP,
						},
						Destination: lapi.Endpoint{
							Namespace:      "-",
							AggregatedName: "pvt",
							Type:           lapi.Network,
							Port:           80,
						},
						Protocol: "tcp",
					},
					LogStats: &lapi.LogStats{FlowLogCount: 4370},
				},
			},
		},
	},
}
