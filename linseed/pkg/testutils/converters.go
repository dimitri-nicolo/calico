// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package testutils

func Int64Ptr(val int64) *int64 {
	return &val
}

func IntPtr(val int) *int {
	return &val
}

func StringPtr(val string) *string {
	return &val
}
