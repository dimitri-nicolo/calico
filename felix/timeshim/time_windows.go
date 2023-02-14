//go:build windows

// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package timeshim

// Time is our shim interface to the time package.
type Interface interface {
	interfaceCommon
}
