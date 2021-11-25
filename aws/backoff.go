// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package aws

import (
	"time"

	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/wait"
)

// ResettableBackoff manages a clock.Timer and its channel to implement a simple exponential backoff with
// the ability to reset to the initial delay.  It uses k8s Clock interface to allow time to be shimmed.
//
// Important: users must call MarkAsDrained() whenever they read a value from the channel. The backoff tracks
// whether the timer is active, but it relies on the user to notify it whenever the channel is drained.  The C()
// method returns nil if the backoff is not active; this works well with select{} since a nil channel never
// "fires" in a select block.
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

func (r *ResettableBackoff) Reset() {
	r.currentInterval = r.initialInterval
}

func (r *ResettableBackoff) Reschedule() {
	r.reschedule()
	r.currentInterval *= 2
	if r.currentInterval > r.maxInterval {
		r.currentInterval = r.maxInterval
	}
}

func (r *ResettableBackoff) reschedule() {
	r.Stop()
	jitteredInterval := wait.Jitter(r.currentInterval, r.jitter)
	if r.timer == nil {
		r.timer = r.clock.NewTimer(jitteredInterval)
	} else {
		r.timer.Reset(jitteredInterval)
	}
	r.active = true
	return
}

func (r *ResettableBackoff) Stop() {
	if !r.active {
		return
	}
	if !r.timer.Stop() {
		<-r.timer.C()
	}
	r.active = false
}

func (r *ResettableBackoff) MarkAsDrained() {
	r.active = false
}
