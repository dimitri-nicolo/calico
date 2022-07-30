// Copyright 2019 Tigera Inc. All rights reserved.

package controller

// Interface that defines a controller.
// TODO(doublek): Move towards our Controller interface defined in kube-controllers.
type Controller interface {
	Run()
}
