//go:build !windows

// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package timeshim

import (
	"time"
)

// Time is our shim interface to the time package.
type interfaceCommon interface {
	Now() time.Time
	Since(t time.Time) time.Duration
	Until(t time.Time) time.Duration
	After(t time.Duration) <-chan time.Time
	NewTimer(d Duration) Timer
	NewTicker(d Duration) Ticker
}

type Time = time.Time
type Duration = time.Duration

type Timer interface {
	Stop() bool
	Reset(clean Duration)
	Chan() <-chan Time
}

type Ticker interface {
	Stop()
	Reset(clean Duration)
	Chan() <-chan Time
}

var singleton realTime

func RealTime() Interface {
	return singleton
}

// realTime is the real implementation of timeIface, which calls through to the real time package.
type realTime struct{}

func (t realTime) NewTimer(d Duration) Timer {
	timer := time.NewTimer(d)
	return (*timerWrapper)(timer)
}

func (t realTime) NewTicker(d Duration) Ticker {
	timer := time.NewTicker(d)
	return (*tickerWrapper)(timer)
}

type timerWrapper time.Timer

func (t *timerWrapper) Stop() bool {
	return (*time.Timer)(t).Stop()
}

func (t *timerWrapper) Reset(duration Duration) {
	(*time.Timer)(t).Reset(duration)
}

func (t *timerWrapper) Chan() <-chan Time {
	return (*time.Timer)(t).C
}

type tickerWrapper time.Ticker

func (t *tickerWrapper) Stop() {
	(*time.Ticker)(t).Stop()
}

func (t *tickerWrapper) Reset(duration Duration) {
	(*time.Ticker)(t).Reset(duration)
}

func (t *tickerWrapper) Chan() <-chan Time {
	return (*time.Ticker)(t).C
}

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
