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
	features "github.com/projectcalico/calico/licensing/client/features"
)

// NewGlobalNetworkSetStorage creates a new libcalico-based storage.Interface implementation for GlobalNetworkSets
func NewGlobalNetworkSetStorage(opts Options) (registry.DryRunnableStorage, factory.DestroyFunc) {
	c := CreateClientFromConfig()
	createFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*v3.GlobalNetworkSet)
		return c.GlobalNetworkSets().Create(ctx, res, oso)
	}
	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*v3.GlobalNetworkSet)
		return c.GlobalNetworkSets().Update(ctx, res, oso)
	}
	getFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.GlobalNetworkSets().Get(ctx, name, ogo)
	}
	deleteFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		odo := opts.(options.DeleteOptions)
		return c.GlobalNetworkSets().Delete(ctx, name, odo)
	}
	listFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (resourceListObject, error) {
		olo := opts.(options.ListOptions)
		return c.GlobalNetworkSets().List(ctx, olo)
	}
	watchFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (watch.Interface, error) {
		olo := opts.(options.ListOptions)
		return c.GlobalNetworkSets().Watch(ctx, olo)
	}
	hasRestrictionsFn := func(obj resourceObject) bool {
		res := obj.(*v3.GlobalNetworkSet)
		return !opts.LicenseMonitor.GetFeatureStatus(features.EgressAccessControl) && len(res.Spec.AllowedEgressDomains) > 0
	}
	// TODO(doublek): Inject codec, client for nicer testing.
	dryRunnableStorage := registry.DryRunnableStorage{Storage: &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         APIObjectVersioner{},
		aapiType:          reflect.TypeOf(v3.GlobalNetworkSet{}),
		aapiListType:      reflect.TypeOf(v3.GlobalNetworkSetList{}),
		libCalicoType:     reflect.TypeOf(v3.GlobalNetworkSet{}),
		libCalicoListType: reflect.TypeOf(v3.GlobalNetworkSetList{}),
		isNamespaced:      false,
		create:            createFn,
		update:            updateFn,
		get:               getFn,
		delete:            deleteFn,
		list:              listFn,
		watch:             watchFn,
		resourceName:      "GlobalNetworkSet",
		converter:         GlobalNetworkSetConverter{},
		hasRestrictions:   hasRestrictionsFn,
	}, Codec: opts.RESTOptions.StorageConfig.Codec}
	return dryRunnableStorage, func() {}
}

type GlobalNetworkSetConverter struct {
}

func (gc GlobalNetworkSetConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiGlobalNetworkSet := aapiObj.(*v3.GlobalNetworkSet)
	lcgGlobalNetworkSet := &v3.GlobalNetworkSet{}
	lcgGlobalNetworkSet.TypeMeta = aapiGlobalNetworkSet.TypeMeta
	lcgGlobalNetworkSet.ObjectMeta = aapiGlobalNetworkSet.ObjectMeta
	lcgGlobalNetworkSet.Kind = v3.KindGlobalNetworkSet
	lcgGlobalNetworkSet.APIVersion = v3.GroupVersionCurrent
	lcgGlobalNetworkSet.Spec = aapiGlobalNetworkSet.Spec
	return lcgGlobalNetworkSet
}

func (gc GlobalNetworkSetConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgGlobalNetworkSet := libcalicoObject.(*v3.GlobalNetworkSet)
	aapiGlobalNetworkSet := aapiObj.(*v3.GlobalNetworkSet)
	aapiGlobalNetworkSet.Spec = lcgGlobalNetworkSet.Spec
	aapiGlobalNetworkSet.TypeMeta = lcgGlobalNetworkSet.TypeMeta
	aapiGlobalNetworkSet.ObjectMeta = lcgGlobalNetworkSet.ObjectMeta
}

func (gc GlobalNetworkSetConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	lcgGlobalNetworkSetList := libcalicoListObject.(*v3.GlobalNetworkSetList)
	aapiGlobalNetworkSetList := aapiListObj.(*v3.GlobalNetworkSetList)
	if libcalicoListObject == nil {
		aapiGlobalNetworkSetList.Items = []v3.GlobalNetworkSet{}
		return
	}
	aapiGlobalNetworkSetList.TypeMeta = lcgGlobalNetworkSetList.TypeMeta
	aapiGlobalNetworkSetList.ListMeta = lcgGlobalNetworkSetList.ListMeta
	for _, item := range lcgGlobalNetworkSetList.Items {
		aapiGlobalNetworkSet := v3.GlobalNetworkSet{}
		gc.convertToAAPI(&item, &aapiGlobalNetworkSet)
		if matched, err := pred.Matches(&aapiGlobalNetworkSet); err == nil && matched {
			aapiGlobalNetworkSetList.Items = append(aapiGlobalNetworkSetList.Items, aapiGlobalNetworkSet)
		}
	}
}
