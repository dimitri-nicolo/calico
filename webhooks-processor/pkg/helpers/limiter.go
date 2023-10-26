// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package helpers

import (
	"fmt"
	"time"
)

type RateLimiter struct {
	events   chan bool
	duration time.Duration
}

func NewRateLimiter(duration time.Duration, times uint) (rateLimiter *RateLimiter) {
	rateLimiter = &RateLimiter{
		events:   make(chan bool, times),
		duration: duration,
	}
	return
}

func (r *RateLimiter) Event() (err error) {
	select {
	case r.events <- true:
		go func() {
			<-time.NewTimer(r.duration).C
			<-r.events
		}()
	default:
		err = fmt.Errorf("rate limit of %d events in %s exceeded", cap(r.events), r.duration)
	}
	return
}
