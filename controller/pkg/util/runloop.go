package util

import (
	"context"
	"time"
)

func RunLoop(ctx context.Context, f func(), period time.Duration) {
	t := time.NewTicker(period)
	defer t.Stop()

	for {
		f()
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			// continue
		}
	}
}
