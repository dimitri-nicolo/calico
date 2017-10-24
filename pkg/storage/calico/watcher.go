/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package calico

import (
	"context"

	cwatch "github.com/projectcalico/libcalico-go/lib/watch"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
)

// watchChan implements watch.Interface.
type watchChan struct {
	resultChan     chan watch.Event
	internalFilter storage.FilterFunc
	watcher        cwatch.Interface
}

func createWatchChan(ctx context.Context, w cwatch.Interface, pred storage.SelectionPredicate) *watchChan {
	wc := &watchChan{
		resultChan:     make(chan watch.Event),
		internalFilter: storage.SimpleFilter(pred),
		watcher:        w,
	}
	if pred.Empty() {
		// The filter doesn't filter out any object.
		wc.internalFilter = nil
	}

	return wc
}

func (wc *watchChan) convertEvent(ce cwatch.Event) (res *watch.Event) {
	switch ce.Type {
	case cwatch.Added:
		aapiObject := convertToAAPI(ce.Object)
		if !wc.filter(aapiObject) {
			return nil
		}
		res = &watch.Event{
			Type:   watch.Added,
			Object: aapiObject,
		}
	case cwatch.Deleted:
		aapiObject := convertToAAPI(ce.Previous)
		if !wc.filter(aapiObject) {
			return nil
		}
		res = &watch.Event{
			Type:   watch.Deleted,
			Object: aapiObject,
		}
	case cwatch.Modified:
		aapiObject := convertToAAPI(ce.Object)
		if wc.acceptAll() {
			res = &watch.Event{
				Type:   watch.Modified,
				Object: aapiObject,
			}
			return res
		}
		oldAapiObject := convertToAAPI(ce.Previous)
		curObjPasses := wc.filter(aapiObject)
		oldObjPasses := wc.filter(oldAapiObject)
		switch {
		case curObjPasses && oldObjPasses:
			res = &watch.Event{
				Type:   watch.Modified,
				Object: aapiObject,
			}
		case curObjPasses && !oldObjPasses:
			res = &watch.Event{
				Type:   watch.Added,
				Object: aapiObject,
			}
		case !curObjPasses && oldObjPasses:
			res = &watch.Event{
				Type:   watch.Deleted,
				Object: oldAapiObject,
			}
		}
	}
	return res
}

func (wc *watchChan) run() {
	for e := range wc.watcher.ResultChan() {
		we := wc.convertEvent(e)
		if we != nil {
			wc.resultChan <- *we
		}
	}
	close(wc.resultChan)
}

func (wc *watchChan) filter(obj runtime.Object) bool {
	if wc.internalFilter == nil {
		return true
	}
	return wc.internalFilter(obj)
}

func (wc *watchChan) acceptAll() bool {
	return wc.internalFilter == nil
}

func (wc *watchChan) Stop() {
	wc.watcher.Stop()
}

func (wc *watchChan) ResultChan() <-chan watch.Event {
	return wc.resultChan
}
