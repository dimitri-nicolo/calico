// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package aws

import (
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/clock"
)

// ResettableBackoff manages a clock.Timer and its channel to implement a simple exponential backoff with
// the ability to reset to the initial delay.  It uses k8s Clock interface to allow time to be shimmed.
type ResettableBackoff struct {
	clock           clock.Clock
	timer           clock.Timer
	initialInterval time.Duration
	maxInterval     time.Duration
	currentInterval time.Duration
	active          bool
	jitter          float64
}

func NewResettableBackoff(clock clock.Clock, initialInterval time.Duration, maxInterval time.Duration, jitter float64) *ResettableBackoff {
	return &ResettableBackoff{
		clock:           clock,
		initialInterval: initialInterval,
		currentInterval: initialInterval,
		maxInterval:     maxInterval,
		jitter:          jitter,
	}
}

func (r *ResettableBackoff) C() <-chan time.Time {
	if r.active {
		return r.timer.C()
	}
	return nil
}

// ResetInterval resets the interval that will be used for the next call to Reschedule().  Does not schedule or
// reschedule the timer.
func (r *ResettableBackoff) ResetInterval() {
	r.currentInterval = r.initialInterval
}

// Reschedule stops the timer if it is active and reschedules it for the next interval.  After scheduling the
// timer it doubles the interval.  The alreadyDrained parameter should be set to true if the caller has already
// read from the channel since the last call to Reschedule() or Stop().
func (r *ResettableBackoff) Reschedule(alreadyDrained bool) {
	r.reschedule(alreadyDrained)
	r.currentInterval *= 2
	if r.currentInterval > r.maxInterval {
		r.currentInterval = r.maxInterval
	}
}

func (r *ResettableBackoff) reschedule(alreadyDrained bool) {
	r.Stop(alreadyDrained)
	jitteredInterval := wait.Jitter(r.currentInterval, r.jitter)
	if r.timer == nil {
		r.timer = r.clock.NewTimer(jitteredInterval)
	} else {
		r.timer.Reset(jitteredInterval)
	}
	r.active = true
	return
}

// Stop stops the timer and drains its channel if required.  The alreadyDrained parameter should be set to true if
// the caller has already read from the channel since the last call to Reschedule() or Stop().
func (r *ResettableBackoff) Stop(alreadyDrained bool) {
	if !r.active {
		return
	}
	if !r.timer.Stop() {
		// Note: I tried to avoid an explicit bool here by doing select{} with a default clause but there is a
		// known issue in the Go runtime that makes that unsafe (https://github.com/golang/go/issues/37196).
		if !alreadyDrained {
			<-r.timer.C()
		}
	}
	r.active = false
}
