// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package controller

type Controller interface {
	Run(stopCh <-chan struct{}) error
}
