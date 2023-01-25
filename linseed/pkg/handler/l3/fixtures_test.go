// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package l3_test

import v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"

var noFlows []v1.L3Flow
var flows = []v1.L3Flow{
	{
		Key: v1.L3FlowKey{
			Action:   "pass",
			Reporter: "source",
			Protocol: "tcp",
			Source: v1.Endpoint{
				Type:           "wep",
				Name:           "",
				AggregatedName: "source-*",
			},
			Destination: v1.Endpoint{
				Type:           "wep",
				Name:           "",
				AggregatedName: "dest-*",
			},
		},
	},
	{
		Key: v1.L3FlowKey{
			Action:   "pass",
			Reporter: "source",
			Protocol: "udp",
			Source: v1.Endpoint{
				Type:           "wep",
				Name:           "",
				AggregatedName: "source-*",
			},
			Destination: v1.Endpoint{
				Type:           "wep",
				Name:           "",
				AggregatedName: "dns-*",
			},
		},
	},
}
