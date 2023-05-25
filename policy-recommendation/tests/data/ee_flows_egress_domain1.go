// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package fv

import (
	lapi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
)

// Setp-1 results for most tests. Consists of two returned flows.
var Step1Results = []rest.MockResult{
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
							AggregatedName: "pub",
							Type:           lapi.Network,
							Port:           80,
						},
						Protocol: "tcp",
					},
					LogStats:    &lapi.LogStats{FlowLogCount: 4370},
					DestDomains: []string{"www.google.com"},
				},
				{
					// key-2
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
							AggregatedName: "pub",
							Type:           lapi.Network,
							Port:           81,
						},
						Protocol: "tcp",
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
							Namespace:      "namespace3",
							AggregatedName: "netshoot3-cb7967547-*",
							Type:           lapi.WEP,
						},
						Destination: lapi.Endpoint{
							Namespace:      "-",
							AggregatedName: "pub",
							Type:           lapi.Network,
							Port:           666,
						},
						Protocol: "tcp",
					},
					LogStats:    &lapi.LogStats{FlowLogCount: 3698},
					DestDomains: []string{"www.tigera.io"},
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
							AggregatedName: "pub",
							Type:           lapi.Network,
							Port:           81,
						},
						Protocol: "tcp",
					},
					LogStats:    &lapi.LogStats{FlowLogCount: 3698},
					DestDomains: []string{"www.google.com"},
				},
				{
					// key-5
					Key: lapi.L3FlowKey{
						Action:   lapi.FlowActionAllow,
						Reporter: lapi.FlowReporterSource,
						Source: lapi.Endpoint{
							Namespace:      "namespace3",
							AggregatedName: "netshoot3-cb7967547-*",
							Type:           lapi.WEP,
						},
						Destination: lapi.Endpoint{
							Namespace:      "-",
							AggregatedName: "pub",
							Type:           lapi.Network,
							Port:           667,
						},
						Protocol: "tcp",
					},
					LogStats:    &lapi.LogStats{FlowLogCount: 3698},
					DestDomains: []string{"www.tigera.io"},
				},
				{
					// key-6
					Key: lapi.L3FlowKey{
						Action:   lapi.FlowActionAllow,
						Reporter: lapi.FlowReporterSource,
						Source: lapi.Endpoint{
							Namespace:      "namespace3",
							AggregatedName: "netshoot3-cb7967547-*",
							Type:           lapi.WEP,
						},
						Destination: lapi.Endpoint{
							Namespace:      "-",
							AggregatedName: "pub",
							Type:           lapi.Network,
							Port:           9090,
						},
						Protocol: "udp",
					},
					LogStats:    &lapi.LogStats{FlowLogCount: 3698},
					DestDomains: []string{"www.calico.org"},
				},
				{
					// key-7
					Key: lapi.L3FlowKey{
						Action:   lapi.FlowActionAllow,
						Reporter: lapi.FlowReporterSource,
						Source: lapi.Endpoint{
							Namespace:      "namespace3",
							AggregatedName: "netshoot3-cb7967547-*",
							Type:           lapi.WEP,
						},
						Destination: lapi.Endpoint{
							Namespace:      "-",
							AggregatedName: "pub",
							Type:           lapi.Network,
							Port:           1,
						},
						Protocol: "tcp",
					},
					LogStats:    &lapi.LogStats{FlowLogCount: 3698},
					DestDomains: []string{"www.calico.org"},
				},
			},
		},
	},
}
