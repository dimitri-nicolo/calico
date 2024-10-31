// Copyright (c) 2017-2019 Tigera, Inc. All rights reserved.

package calico

import (
	"context"
	"reflect"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"

	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
	"github.com/projectcalico/calico/libcalico-go/lib/watch"
)

// NewIPPoolStorage creates a new libcalico-based storage.Interface implementation for IPPools
func NewIPPoolStorage(opts Options) (registry.DryRunnableStorage, factory.DestroyFunc) {
	c := CreateClientFromConfig()
	createFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*v3.IPPool)
		return c.IPPools().Create(ctx, res, oso)
	}
	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*v3.IPPool)
		return c.IPPools().Update(ctx, res, oso)
	}
	getFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.IPPools().Get(ctx, name, ogo)
	}
	deleteFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		odo := opts.(options.DeleteOptions)
		return c.IPPools().Delete(ctx, name, odo)
	}
	listFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (resourceListObject, error) {
		olo := opts.(options.ListOptions)
		return c.IPPools().List(ctx, olo)
	}
	watchFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (watch.Interface, error) {
		olo := opts.(options.ListOptions)
		return c.IPPools().Watch(ctx, olo)
	}
	hasRestrictionsFn := func(obj resourceObject) bool {
		return false
	}

	dryRunnableStorage := registry.DryRunnableStorage{Storage: &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         APIObjectVersioner{},
		aapiType:          reflect.TypeOf(v3.IPPool{}),
		aapiListType:      reflect.TypeOf(v3.IPPoolList{}),
		libCalicoType:     reflect.TypeOf(v3.IPPool{}),
		libCalicoListType: reflect.TypeOf(v3.IPPoolList{}),
		isNamespaced:      false,
		create:            createFn,
		update:            updateFn,
		get:               getFn,
		delete:            deleteFn,
		list:              listFn,
		watch:             watchFn,
		resourceName:      "IPPool",
		converter:         IPPoolConverter{},
		hasRestrictions:   hasRestrictionsFn,
	}, Codec: opts.RESTOptions.StorageConfig.Codec}
	return dryRunnableStorage, func() {}
}

type IPPoolConverter struct {
}

func (gc IPPoolConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiIPPool := aapiObj.(*v3.IPPool)
	lcgIPPool := &v3.IPPool{}
	lcgIPPool.TypeMeta = aapiIPPool.TypeMeta
	lcgIPPool.ObjectMeta = aapiIPPool.ObjectMeta
	lcgIPPool.Kind = v3.KindIPPool
	lcgIPPool.APIVersion = v3.GroupVersionCurrent
	lcgIPPool.Spec = aapiIPPool.Spec
	return lcgIPPool
}

func (gc IPPoolConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgIPPool := libcalicoObject.(*v3.IPPool)
	aapiIPPool := aapiObj.(*v3.IPPool)
	aapiIPPool.Spec = lcgIPPool.Spec
	aapiIPPool.TypeMeta = lcgIPPool.TypeMeta
	aapiIPPool.ObjectMeta = lcgIPPool.ObjectMeta
}

func (gc IPPoolConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	lcgIPPoolList := libcalicoListObject.(*v3.IPPoolList)
	aapiIPPoolList := aapiListObj.(*v3.IPPoolList)
	if libcalicoListObject == nil {
		aapiIPPoolList.Items = []v3.IPPool{}
		return
	}
	aapiIPPoolList.TypeMeta = lcgIPPoolList.TypeMeta
	aapiIPPoolList.ListMeta = lcgIPPoolList.ListMeta
	for _, item := range lcgIPPoolList.Items {
		aapiIPPool := v3.IPPool{}
		gc.convertToAAPI(&item, &aapiIPPool)
		if matched, err := pred.Matches(&aapiIPPool); err == nil && matched {
			aapiIPPoolList.Items = append(aapiIPPoolList.Items, aapiIPPool)
		}
	}
}
