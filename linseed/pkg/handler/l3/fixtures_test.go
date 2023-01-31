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

var noFlowLogs []v1.FlowLog
var flowLogs = []v1.FlowLog{
	{
		SourceNameAggr:  "source-*",
		SourceNamespace: "source-ns",
		SourceType:      "wep",
		SourceLabels:    v1.FlowLogLabels{Labels: []string{"k8s-app=source-app", "projectcalico.org/namespace=source-ns"}},

		DestNameAggr:  "dest-*",
		DestNamespace: "dest-ns",
		DestPort:      443,
		DestType:      "ns",
		DestLabels:    v1.FlowLogLabels{Labels: []string{"k8s-app=dest-app", "projectcalico.org/namespace=dest-ns"}},

		DestServiceNamespace: "dest-ns",
		DestServiceName:      "svc",
		DestServicePortNum:   443,

		Protocol: "tcp",
		Action:   "allow",
		Reporter: "src",
		Policies: []v1.FlowLogPolicy{{AllPolicies: "0|allow-tigera|dest-ns/allow-svc.dest-access|allow|1"}},

		NumFlows:          1,
		NumFlowsCompleted: 0,
		NumFlowsStarted:   0,

		ProcessName:     "./server",
		NumProcessNames: 1,
		NumProcessIDs:   1,
		ProcessArgs:     "-",
		NumProcessArgs:  0,
		//"process_id":"4067",
		//"process_args":[
		//"-"
		//],
	},
	{
		SourceNameAggr:  "source-*",
		SourceNamespace: "source-ns",
		SourceType:      "wep",
		SourceLabels:    v1.FlowLogLabels{Labels: []string{"k8s-app=source-app", "projectcalico.org/namespace=source-ns"}},

		DestNameAggr:  "dest-*",
		DestNamespace: "dest-ns",
		DestPort:      443,
		DestType:      "ns",
		DestLabels:    v1.FlowLogLabels{Labels: []string{"k8s-app=dest-app", "projectcalico.org/namespace=dest-ns"}},

		DestServiceNamespace: "dest-ns",
		DestServiceName:      "svc",
		DestServicePortNum:   443,

		Protocol: "tcp",
		Action:   "allow",
		Reporter: "src",
		Policies: []v1.FlowLogPolicy{{AllPolicies: "0|allow-tigera|dest-ns/allow-svc.dest-access|allow|1"}},

		NumFlows:          1,
		NumFlowsCompleted: 0,
		NumFlowsStarted:   0,

		ProcessName:     "./server",
		NumProcessNames: 1,
		NumProcessIDs:   1,
		ProcessArgs:     "-",
		NumProcessArgs:  0,

		//"process_id":"4067",
		//"process_args":[
		//"-"
		//],
	},
}

var bulkResponseSuccess = &v1.BulkResponse{
	Total:     2,
	Succeeded: 2,
	Failed:    0,
}

var bulkResponsePartialSuccess = &v1.BulkResponse{
	Total:     2,
	Succeeded: 1,
	Failed:    1,
	Errors: []v1.BulkError{
		{
			Message: "fail to index a record",
		},
	},
}
