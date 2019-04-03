// Copyright 2019 Tigera Inc. All rights reserved.

package watcher

import "k8s.io/client-go/tools/cache"

type ping struct{}

// pingKey is a sentinel to represent a ping on the FIFO.  Note that we use characters
// not allowed in Kubernetes names so that we won't conflict with GlobalThreatFeed
// names.
const pingKey = "~ping~"

// NewPingableFifo returns a DeltaFIFO that accepts GlobalThreatFeed objects and
// a special "ping" object that is used for health checking.  The ping object
// exists in the store at start of day, and is never removed, only updated.
func NewPingableFifo() (*cache.DeltaFIFO, cache.Store) {
	// This will hold the client state, as we know it.
	clientState := cache.NewStore(DeletionHandlingPingableKeyFunc)

	// Add a special key to represent Ping objects.
	err := clientState.Add(ping{})
	if err != nil {
		// Local cache never errors unless key func fails.
		panic(err)
	}

	// This will hold incoming changes. Note how we pass clientState in as a
	// KeyLister, that way resync operations will result in the correct set
	// of update/delete deltas.
	fifo := cache.NewDeltaFIFO(PingableKeyFunc, clientState)
	return fifo, clientState
}

func DeletionHandlingPingableKeyFunc(obj interface{}) (string, error) {
	_, ok := obj.(ping)
	if ok {
		return pingKey, nil
	}
	return cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
}

func PingableKeyFunc(obj interface{}) (string, error) {
	_, ok := obj.(ping)
	if ok {
		return pingKey, nil
	}
	return cache.MetaNamespaceKeyFunc(obj)
}
