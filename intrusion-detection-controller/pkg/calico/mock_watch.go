// Copyright 2019 Tigera Inc. All rights reserved.

package calico

import (
	"k8s.io/apimachinery/pkg/watch"
)

type MockWatch struct {
	C chan watch.Event
}

func (w *MockWatch) ResultChan() <-chan watch.Event {
	return w.C
}

func (w *MockWatch) Stop() {
	close(w.C)
}
