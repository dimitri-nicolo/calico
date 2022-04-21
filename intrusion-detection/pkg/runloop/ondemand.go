// Copyright 2019 Tigera Inc. All rights reserved.

package runloop

import (
	"context"
	"sync"
)

type RunFunc func(context.Context, func(context.Context, interface{}))

type EnqueueFunc func(interface{})

func OnDemand() (RunFunc, EnqueueFunc) {
	var done bool
	var next interface{}
	var lock sync.Mutex
	cond := sync.NewCond(&lock)

	run := func(ctx context.Context, f func(context.Context, interface{})) {
		go func() {
			<-ctx.Done()
			lock.Lock()
			done = true
			cond.Signal()
			lock.Unlock()
		}()

		for {
			lock.Lock()
			if done {
				lock.Unlock()
				break
			} else if next != nil {
				cur := next
				next = nil
				lock.Unlock()

				f(ctx, cur)
			} else {
				cond.Wait()
				lock.Unlock()
			}
		}
	}
	enqueue := func(x interface{}) {
		lock.Lock()
		next = x
		cond.Signal()
		lock.Unlock()
	}
	return run, enqueue
}
