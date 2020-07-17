// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package timeshim

import (
	"time"
)

// Time is our shim interface to the time package.
type Interface interface {
	Now() time.Time
	Since(t time.Time) time.Duration
	Until(t time.Time) time.Duration
	After(t time.Duration) <-chan time.Time
}

var singleton realTime

func RealTime() Interface {
	return singleton
}

// realTime is the real implementation of timeIface, which calls through to the real time package.
type realTime struct{}

func (realTime) Until(t time.Time) time.Duration {
	return time.Until(t)
}

func (realTime) After(t time.Duration) <-chan time.Time {
	return time.After(t)
}

func (realTime) Now() time.Time {
	return time.Now()
}

func (realTime) Since(t time.Time) time.Duration {
	return time.Since(t)
}
