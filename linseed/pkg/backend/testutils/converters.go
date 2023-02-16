// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package testutils

import v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"

func Int64Ptr(val int64) *int64 {
	return &val
}

func StringPtr(val string) *string {
	return &val
}

func ActionPtr(val v1.FlowAction) *v1.FlowAction {
	return &val
}
