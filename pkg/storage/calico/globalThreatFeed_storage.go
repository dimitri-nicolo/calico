// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package calico

import (
	"reflect"

	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/etcd"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"

	libcalicoapi "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/watch"

	aapi "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
)

// NewGlobalThreatFeedStorage creates a new libcalico-based storage.Interface implementation for GlobalThreatFeeds
func NewGlobalThreatFeedStorage(opts Options) (storage.Interface, factory.DestroyFunc) {
	c := createClientFromConfig()
	createFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.GlobalThreatFeed)
		return c.GlobalThreatFeeds().Create(ctx, res, oso)
	}
	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.GlobalThreatFeed)
		return c.GlobalThreatFeeds().Update(ctx, res, oso)
	}
	getFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.GlobalThreatFeeds().Get(ctx, name, ogo)
	}
	deleteFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		odo := opts.(options.DeleteOptions)
		return c.GlobalThreatFeeds().Delete(ctx, name, odo)
	}
	listFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (resourceListObject, error) {
		olo := opts.(options.ListOptions)
		return c.GlobalThreatFeeds().List(ctx, olo)
	}
	watchFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (watch.Interface, error) {
		olo := opts.(options.ListOptions)
		return c.GlobalThreatFeeds().Watch(ctx, olo)
	}
	// TODO(doublek): Inject codec, client for nicer testing.
	return &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         etcd.APIObjectVersioner{},
		aapiType:          reflect.TypeOf(aapi.GlobalThreatFeed{}),
		aapiListType:      reflect.TypeOf(aapi.GlobalThreatFeedList{}),
		libCalicoType:     reflect.TypeOf(libcalicoapi.GlobalThreatFeed{}),
		libCalicoListType: reflect.TypeOf(libcalicoapi.GlobalThreatFeedList{}),
		isNamespaced:      false,
		create:            createFn,
		update:            updateFn,
		get:               getFn,
		delete:            deleteFn,
		list:              listFn,
		watch:             watchFn,
		resourceName:      "GlobalThreatFeed",
		converter:         GlobalThreatFeedConverter{},
	}, func() {}
}

type GlobalThreatFeedConverter struct {
}

func (gc GlobalThreatFeedConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiGlobalThreatFeed := aapiObj.(*aapi.GlobalThreatFeed)
	lcgGlobalThreatFeed := &libcalicoapi.GlobalThreatFeed{}
	lcgGlobalThreatFeed.TypeMeta = aapiGlobalThreatFeed.TypeMeta
	lcgGlobalThreatFeed.ObjectMeta = aapiGlobalThreatFeed.ObjectMeta
	lcgGlobalThreatFeed.Spec = aapiGlobalThreatFeed.Spec
	lcgGlobalThreatFeed.Status = aapiGlobalThreatFeed.Status
	return lcgGlobalThreatFeed
}

func (gc GlobalThreatFeedConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgGlobalThreatFeed := libcalicoObject.(*libcalicoapi.GlobalThreatFeed)
	aapiGlobalThreatFeed := aapiObj.(*aapi.GlobalThreatFeed)
	aapiGlobalThreatFeed.Spec = lcgGlobalThreatFeed.Spec
	aapiGlobalThreatFeed.Status = lcgGlobalThreatFeed.Status
	aapiGlobalThreatFeed.TypeMeta = lcgGlobalThreatFeed.TypeMeta
	aapiGlobalThreatFeed.ObjectMeta = lcgGlobalThreatFeed.ObjectMeta
}

func (gc GlobalThreatFeedConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	lcgGlobalThreatFeedList := libcalicoListObject.(*libcalicoapi.GlobalThreatFeedList)
	aapiGlobalThreatFeedList := aapiListObj.(*aapi.GlobalThreatFeedList)
	if libcalicoListObject == nil {
		aapiGlobalThreatFeedList.Items = []aapi.GlobalThreatFeed{}
		return
	}
	aapiGlobalThreatFeedList.TypeMeta = lcgGlobalThreatFeedList.TypeMeta
	aapiGlobalThreatFeedList.ListMeta = lcgGlobalThreatFeedList.ListMeta
	for _, item := range lcgGlobalThreatFeedList.Items {
		aapiGlobalThreatFeed := aapi.GlobalThreatFeed{}
		gc.convertToAAPI(&item, &aapiGlobalThreatFeed)
		if matched, err := pred.Matches(&aapiGlobalThreatFeed); err == nil && matched {
			aapiGlobalThreatFeedList.Items = append(aapiGlobalThreatFeedList.Items, aapiGlobalThreatFeed)
		}
	}
}
