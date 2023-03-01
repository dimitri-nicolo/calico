// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package l7

import (
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

const dummyURL = "anyURL"

var (
	noL7Logs []v1.L7Log
	l7Logs   = []v1.L7Log{
		{
			SourceNamespace: "ns-source",
			SourceNameAggr:  "source-*",
			SourceType:      "wep",
			SourcePortNum:   996,

			DestNamespace:   "ns-dest",
			DestNameAggr:    "dest-*",
			DestType:        "wep",
			DestServicePort: 997,

			ResponseCode: "200",
		},
		{
			SourceNamespace: "ns-source",
			SourceNameAggr:  "source-*",
			SourceType:      "wep",
			SourcePortNum:   996,

			DestNamespace:   "ns-dest",
			DestNameAggr:    "dest-*",
			DestType:        "wep",
			DestServicePort: 997,

			ResponseCode: "200",
		},
	}
)

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
			Resource: "res",
			Type:     "index error",
			Reason:   "I couldn't do it",
		},
	},
}
