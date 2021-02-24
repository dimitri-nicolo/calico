// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package worker

type Option func(*worker)

func WithMaxRequeueAttempts(maxAttempts int) Option {
	return func(w *worker) {
		w.maxRequeueAttempts = maxAttempts
	}
}
