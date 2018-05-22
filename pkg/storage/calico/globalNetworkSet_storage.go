// Copyright (c) 2017 Tigera, Inc. All rights reserved.

package calico

import (
	"reflect"

	"golang.org/x/net/context"

	libcalicoapi "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/watch"
	aapi "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/etcd"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"
)

// NewGlobalNetworkSetStorage creates a new libcalico-based storage.Interface implementation for GlobalNetworkSets
func NewGlobalNetworkSetStorage(opts Options) (storage.Interface, factory.DestroyFunc) {
	c := createClientFromConfig()
	create := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.GlobalNetworkSet)
		return c.GlobalNetworkSets().Create(ctx, res, oso)
	}
	update := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.GlobalNetworkSet)
		return c.GlobalNetworkSets().Update(ctx, res, oso)
	}
	get := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.GlobalNetworkSets().Get(ctx, name, ogo)
	}
	delete := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		odo := opts.(options.DeleteOptions)
		return c.GlobalNetworkSets().Delete(ctx, name, odo)
	}
	list := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (resourceListObject, error) {
		olo := opts.(options.ListOptions)
		return c.GlobalNetworkSets().List(ctx, olo)
	}
	watch := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (watch.Interface, error) {
		olo := opts.(options.ListOptions)
		return c.GlobalNetworkSets().Watch(ctx, olo)
	}
	// TODO(doublek): Inject codec, client for nicer testing.
	return &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         etcd.APIObjectVersioner{},
		aapiType:          reflect.TypeOf(aapi.GlobalNetworkSet{}),
		aapiListType:      reflect.TypeOf(aapi.GlobalNetworkSetList{}),
		libCalicoType:     reflect.TypeOf(libcalicoapi.GlobalNetworkSet{}),
		libCalicoListType: reflect.TypeOf(libcalicoapi.GlobalNetworkSetList{}),
		isNamespaced:      false,
		create:            create,
		update:            update,
		get:               get,
		delete:            delete,
		list:              list,
		watch:             watch,
		resourceName:      "GlobalNetworkSet",
		converter:         GlobalNetworkSetConverter{},
	}, func() {}
}

type GlobalNetworkSetConverter struct {
}

func (gc GlobalNetworkSetConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiGlobalNetworkSet := aapiObj.(*aapi.GlobalNetworkSet)
	lcgGlobalNetworkSet := &libcalicoapi.GlobalNetworkSet{}
	lcgGlobalNetworkSet.TypeMeta = aapiGlobalNetworkSet.TypeMeta
	lcgGlobalNetworkSet.ObjectMeta = aapiGlobalNetworkSet.ObjectMeta
	lcgGlobalNetworkSet.Spec = aapiGlobalNetworkSet.Spec
	return lcgGlobalNetworkSet
}

func (gc GlobalNetworkSetConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgGlobalNetworkSet := libcalicoObject.(*libcalicoapi.GlobalNetworkSet)
	aapiGlobalNetworkSet := aapiObj.(*aapi.GlobalNetworkSet)
	aapiGlobalNetworkSet.Spec = lcgGlobalNetworkSet.Spec
	aapiGlobalNetworkSet.TypeMeta = lcgGlobalNetworkSet.TypeMeta
	aapiGlobalNetworkSet.ObjectMeta = lcgGlobalNetworkSet.ObjectMeta
}

func (gc GlobalNetworkSetConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, filterFunc storage.FilterFunc) {
	lcgGlobalNetworkSetList := libcalicoListObject.(*libcalicoapi.GlobalNetworkSetList)
	aapiGlobalNetworkSetList := aapiListObj.(*aapi.GlobalNetworkSetList)
	if libcalicoListObject == nil {
		aapiGlobalNetworkSetList.Items = []aapi.GlobalNetworkSet{}
		return
	}
	aapiGlobalNetworkSetList.TypeMeta = lcgGlobalNetworkSetList.TypeMeta
	aapiGlobalNetworkSetList.ListMeta = lcgGlobalNetworkSetList.ListMeta
	for _, item := range lcgGlobalNetworkSetList.Items {
		aapiGlobalNetworkSet := aapi.GlobalNetworkSet{}
		gc.convertToAAPI(&item, &aapiGlobalNetworkSet)
		if filterFunc != nil && filterFunc(&aapiGlobalNetworkSet) {
			aapiGlobalNetworkSetList.Items = append(aapiGlobalNetworkSetList.Items, aapiGlobalNetworkSet)
		}
	}
}
