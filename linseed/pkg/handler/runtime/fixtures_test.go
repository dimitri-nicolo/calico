// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package runtime

import (
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

const dummyURL = "anyURL"

var (
	noReports        []v1.Report
	noRuntimeReports []v1.RuntimeReport
	runtimeReports   = []v1.RuntimeReport{
		{},
		{},
	}

	reports = []v1.Report{
		{},
		{},
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
