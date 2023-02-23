// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package bgp

import (
	"time"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

const dummyURL = "anyURL"

var (
	noBGPLogs []v1.BGPLog
	bgpLogs   = []v1.BGPLog{
		{
			LogTime: time.Now().Format(v1.BGPLogTimeFormat),
			Message: "any",
		},
		{
			LogTime: time.Now().Format(v1.BGPLogTimeFormat),
			Message: "any",
		},
	}

	bulkResponseSuccess = &v1.BulkResponse{
		Total:     2,
		Succeeded: 2,
		Failed:    0,
	}

	bulkResponsePartialSuccess = &v1.BulkResponse{
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
)
