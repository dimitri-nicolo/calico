// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package waf

import (
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

const dummyURL = "anyURL"

var (
	noWAFLogs []v1.WAFLog
	wafLogs   = []v1.WAFLog{
		{},
		{},
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
