// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package nfqueue

type Option func(nfqueue *nfQueue)

func WithDebugLogFDEnabled() Option {
	return func(nfqueue *nfQueue) {
		nfqueue.debugEnableLogFD = true
	}
}
