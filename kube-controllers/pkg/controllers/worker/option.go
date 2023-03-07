// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package worker

import "time"

type Option func(*worker)

func WithMaxRequeueAttempts(maxAttempts int) Option {
	return func(w *worker) {
		w.maxRequeueAttempts = maxAttempts
	}
}

func WithResyncPeriod(resyncPeriod time.Duration) Option {
	return func(w *worker) {
		w.resyncPeriod = resyncPeriod
	}
}
