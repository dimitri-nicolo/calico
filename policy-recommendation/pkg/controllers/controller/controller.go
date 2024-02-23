// Copyright (c) 2024 Tigera Inc. All rights reserved.
package controller

import (
	"k8s.io/apimachinery/pkg/types"
)

type Controller interface {
	// Run will set up workers and add watches to k8s resource we are interested in and starts the
	// worker. It will block until parent's Done channel is closed, at which point it will cancel the
	// current context and that will trigger workers to shutdown the workqueue and finish processing
	// their current work items.
	Run(stopChan chan struct{})
}

// Reconciler is the interface that is used to react to changes to the resources that the worker is
// watching. When a change to a resource is detected, the Reconcile function of the passed in
// reconciler is used
type Reconciler interface {
	Reconcile(name types.NamespacedName) error
}
