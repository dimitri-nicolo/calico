// Copyright 2019 Tigera Inc. All rights reserved.

package runloop

import (
	"context"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestOnDemand(t *testing.T) {
	g := NewGomegaWithT(t)

	var done bool
	var lock sync.Mutex
	wake := sync.NewCond(&lock)

	ctx, cancel := context.WithCancel(context.TODO())
	defer func() {
		cancel()
		lock.Lock()
		wake.Broadcast()
		lock.Unlock()
		g.Eventually(func() bool { return done }, 10*time.Second, 1*time.Second).Should(BeTrue(), "run terminates on context cancellation")
	}()

	run, enqueue := OnDemand()

	var last int
	go func() {
		run(ctx, func(ctx context.Context, i interface{}) {
			last = i.(int)
			g.Expect(last).ShouldNot(Equal(2))
			lock.Lock()
			wake.Wait()
			lock.Unlock()
		})
		done = true
	}()

	enqueue(1)
	g.Eventually(func() int { return last }, 10*time.Second, 1*time.Second).Should(Equal(1))

	lock.Lock()
	enqueue(2)
	enqueue(3)
	wake.Signal()
	lock.Unlock()

	g.Eventually(func() int { return last }, 10*time.Second, 1*time.Second).Should(Equal(3))
}
