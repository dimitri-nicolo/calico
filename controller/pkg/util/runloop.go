package util

import (
	"context"
	"errors"
	"sync"
	"time"
)

func RunLoop(ctx context.Context, f func(), period time.Duration) error {
	return runLoop(ctx, func() {}, f, period, make(chan struct{}), func() {}, 0)
}

type RunFuncWithReschedule func(context.Context, func(), time.Duration, func(), time.Duration) error
type RescheduleFunc func() error

func RunLoopWithReschedule() (RunFuncWithReschedule, RescheduleFunc) {
	ch := make(chan struct{})
	var started bool

	initFunc := func() {
		started = true
	}
	runFunc := func(ctx context.Context, f func(), period time.Duration, rescheduleFunc func(), reschedulePeriod time.Duration) error {
		return runLoop(ctx, initFunc, f, period, ch, rescheduleFunc, reschedulePeriod)
	}
	rescheduleFunc := func() (err error) {
		if !started {
			return errors.New("RunFunc has not yet started")
		}
		defer func() {
			if r := recover(); r != nil {
				err = errors.New("RunFunc has terminated")
			}
		}()
		select {
		case ch <- struct{}{}:
			return nil
		default:
			return errors.New("RunFunc is not currently reschedulable")
		}
	}

	return runFunc, rescheduleFunc
}

func runLoop(ctx context.Context, initFunc func(), f func(), period time.Duration, rescheduleCh chan struct{}, rescheduleFunc func(), reschedulePeriod time.Duration) error {
	t := time.NewTicker(period)
	defer t.Stop()
	defer close(rescheduleCh)

	initFunc()
	wg := sync.WaitGroup{}
	for {
		wg.Wait()
		wg.Add(1)
		go func() {
			defer wg.Done()
			f()
		}()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-rescheduleCh:
			rescheduleFunc()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(reschedulePeriod):
				// drain t.C so that we don't run again immediately
				for done := false; !done; {
					select {
					case <-t.C:
						// nothing
					default:
						done = true
					}
				}
				// continue
			}
		case <-t.C:
			// continue
		}
	}
}
