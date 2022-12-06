// Copyright (c) 2022 Tigera Inc. All rights reserved.

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
)

type Controller interface {
	// Run will set up workers and add watches to k8s resource we are interested in and starts the worker.
	// It will block until parent's Done channel is closed, at which point it will cancel the current context
	// and that will trigger workers to shutdown the workqueue and finish processing their current work items.
	Run(ctx context.Context)

	// Close cancel the context created by the Run function and all the internal goroutines.
	Close()
}

// Reconciler is the interface that is used to react to changes to the resources that the worker is watching. When a change
// to a resource is detected, the Reconcile function of the passed in reconciler is used
type Reconciler interface {
	// Reconcile
	Reconcile(name types.NamespacedName) error

	// Close
	Close()
}

type Watcher interface {
	Run(stop <-chan struct{})
	Close()
}
