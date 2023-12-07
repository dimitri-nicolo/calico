package statscache

import (
	"context"
	"time"
)

// LazyTicker is a ticker that only starts the next interval on successful send
// cancellation of the context will stop the ticker
type LazyTicker interface {
	Start(context.Context)
	C() <-chan struct{}
}

type lazyTicker struct {
	interval time.Duration
	ch       chan struct{}
}

func (t *lazyTicker) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case t.ch <- struct{}{}:
		}
		select {
		case <-time.After(t.interval):
		case <-ctx.Done():
			return
		}
	}
}

func (t *lazyTicker) C() <-chan struct{} {
	return t.ch
}

func NewLazyTicker(interval time.Duration) LazyTicker {
	return &lazyTicker{
		interval: interval,
		ch:       make(chan struct{}),
	}
}
