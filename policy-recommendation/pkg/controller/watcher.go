// Copyright (c) 2022 Tigera Inc. All rights reserved.

package controller

import (
	"reflect"

	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	uruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	log "github.com/sirupsen/logrus"
)

const (
	DefaultMaxRequeueAttempts = 5
	resyncPeriod              = 0
)

type watcher struct {
	workqueue.RateLimitingInterface
	reconciler         Reconciler
	resource           watchedObj
	maxRequeueAttempts int
}

type watchedObj struct {
	listWatcher cache.ListerWatcher
	obj         runtime.Object
}

func NewWatcher(reconciler Reconciler, listWatcher cache.ListerWatcher, obj runtime.Object) Watcher {
	return &watcher{
		RateLimitingInterface: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		reconciler:            reconciler,
		resource: watchedObj{
			listWatcher: listWatcher,
			obj:         obj,
		},
		maxRequeueAttempts: DefaultMaxRequeueAttempts,
	}
}

func (w *watcher) Run(stop <-chan struct{}) {
	defer uruntime.HandleCrash()
	defer w.ShutDown()

	_, ctrl := cache.NewIndexerInformer(w.resource.listWatcher, w.resource.obj, resyncPeriod, w.resourceEventHandlerFuncs(),
		cache.Indexers{})

	go ctrl.Run(stop)

	if !cache.WaitForNamedCacheSync(reflect.TypeOf(w.resource.obj).String(), stop, ctrl.HasSynced) {
		log.Errorf("failed to sync resource %T", w.resource.obj)
		return
	}

	go wait.Until(w.startWatch, time.Second, stop)

	<-stop
}

func (w *watcher) resourceEventHandlerFuncs() cache.ResourceEventHandlerFuncs {
	r := cache.ResourceEventHandlerFuncs{}

	r.AddFunc = func(obj any) {
		key, err := cache.MetaNamespaceKeyFunc(obj)
		if err == nil {
			w.Add(key)
		}
		log.Debugf("Create event received for resource %s", key)
	}

	r.UpdateFunc = func(oldObj any, newObj any) {
		key, err := cache.MetaNamespaceKeyFunc(newObj)
		if err == nil {
			w.Add(key)
		}
		log.Debugf("Update event received for resource %s", key)
	}

	r.DeleteFunc = func(obj any) {
		key, err := cache.MetaNamespaceKeyFunc(obj)
		if err == nil {
			w.Add(key)
		}
		log.Debugf("Delete event received for resource %s", key)
	}

	return r
}

func (w *watcher) startWatch() {
	for w.process() {
	}
}

func (w *watcher) process() bool {
	key, shutdown := w.Get()
	if shutdown {
		return false
	}

	defer w.Done(key)

	log.Debugf("Received %v, and type: %s", key, reflect.TypeOf(w.resource.obj).String())
	reqLogger := log.WithField("key", key)
	reqLogger.Debug("Processing next item")

	keyStr, ok := key.(string)

	if !ok {
		log.Errorf("incorrect key type %+v", key)
		return false
	}

	var namespacedName types.NamespacedName
	var err error
	namespacedName.Namespace, namespacedName.Name, err = cache.SplitMetaNamespaceKey(keyStr)

	if err != nil {
		log.WithError(err).Errorf("unable to process key: %s", keyStr)
		return false
	}

	// call reconciler
	err = w.reconciler.Reconcile(namespacedName)

	if err != nil {
		log.WithError(err).Errorf("error while processing %s", keyStr)
		if w.NumRequeues(key) > w.maxRequeueAttempts {
			reqLogger.Debug("Max number or retries for key reached, forgetting key")
			w.Forget(key)
			uruntime.HandleError(err)
		} else {
			w.AddRateLimited(key)
		}
	}

	return true
}

func (w *watcher) Close() {
	w.reconciler.Close()
}
