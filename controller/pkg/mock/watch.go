// Copyright 2019 Tigera Inc. All rights reserved.

package mock

import (
	"k8s.io/apimachinery/pkg/watch"
)

type Watch struct {
	C chan watch.Event
}

func (w *Watch) ResultChan() <-chan watch.Event {
	return w.C
}

func (w *Watch) Stop() {
	close(w.C)
}
