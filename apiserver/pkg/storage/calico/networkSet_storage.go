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
	"github.com/projectcalico/calico/licensing/client/features"
)

// NewNetworkSetStorage creates a new libcalico-based storage.Interface implementation for NetworkSets
func NewNetworkSetStorage(opts Options) (registry.DryRunnableStorage, factory.DestroyFunc) {
	c := CreateClientFromConfig()
	createFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*v3.NetworkSet)
		return c.NetworkSets().Create(ctx, res, oso)
	}
	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*v3.NetworkSet)
		return c.NetworkSets().Update(ctx, res, oso)
	}
	getFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.NetworkSets().Get(ctx, ns, name, ogo)
	}
	deleteFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		odo := opts.(options.DeleteOptions)
		return c.NetworkSets().Delete(ctx, ns, name, odo)
	}
	listFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (resourceListObject, error) {
		olo := opts.(options.ListOptions)
		return c.NetworkSets().List(ctx, olo)
	}
	watchFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (watch.Interface, error) {
		olo := opts.(options.ListOptions)
		return c.NetworkSets().Watch(ctx, olo)
	}
	hasRestrictionsFn := func(obj resourceObject) bool {
		res := obj.(*v3.NetworkSet)
		return !opts.LicenseMonitor.GetFeatureStatus(features.EgressAccessControl) && len(res.Spec.AllowedEgressDomains) > 0
	}
	// TODO(doublek): Inject codec, client for nicer testing.
	dryRunnableStorage := registry.DryRunnableStorage{Storage: &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         APIObjectVersioner{},
		aapiType:          reflect.TypeOf(v3.NetworkSet{}),
		aapiListType:      reflect.TypeOf(v3.NetworkSetList{}),
		libCalicoType:     reflect.TypeOf(v3.NetworkSet{}),
		libCalicoListType: reflect.TypeOf(v3.NetworkSetList{}),
		isNamespaced:      true,
		create:            createFn,
		update:            updateFn,
		get:               getFn,
		delete:            deleteFn,
		list:              listFn,
		watch:             watchFn,
		resourceName:      "NetworkSet",
		converter:         NetworkSetConverter{},
		hasRestrictions:   hasRestrictionsFn,
	}, Codec: opts.RESTOptions.StorageConfig.Codec}
	return dryRunnableStorage, func() {}
}

type NetworkSetConverter struct {
}

func (gc NetworkSetConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiNetworkSet := aapiObj.(*v3.NetworkSet)
	lcgNetworkSet := &v3.NetworkSet{}
	lcgNetworkSet.TypeMeta = aapiNetworkSet.TypeMeta
	lcgNetworkSet.ObjectMeta = aapiNetworkSet.ObjectMeta
	lcgNetworkSet.Kind = v3.KindNetworkSet
	lcgNetworkSet.APIVersion = v3.GroupVersionCurrent
	lcgNetworkSet.Spec = aapiNetworkSet.Spec
	return lcgNetworkSet
}

func (gc NetworkSetConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgNetworkSet := libcalicoObject.(*v3.NetworkSet)
	aapiNetworkSet := aapiObj.(*v3.NetworkSet)
	aapiNetworkSet.Spec = lcgNetworkSet.Spec
	aapiNetworkSet.TypeMeta = lcgNetworkSet.TypeMeta
	aapiNetworkSet.ObjectMeta = lcgNetworkSet.ObjectMeta
}

func (gc NetworkSetConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	lcgNetworkSetList := libcalicoListObject.(*v3.NetworkSetList)
	aapiNetworkSetList := aapiListObj.(*v3.NetworkSetList)
	if libcalicoListObject == nil {
		aapiNetworkSetList.Items = []v3.NetworkSet{}
		return
	}
	aapiNetworkSetList.TypeMeta = lcgNetworkSetList.TypeMeta
	aapiNetworkSetList.ListMeta = lcgNetworkSetList.ListMeta
	for _, item := range lcgNetworkSetList.Items {
		aapiNetworkSet := v3.NetworkSet{}
		gc.convertToAAPI(&item, &aapiNetworkSet)
		if matched, err := pred.Matches(&aapiNetworkSet); err == nil && matched {
			aapiNetworkSetList.Items = append(aapiNetworkSetList.Items, aapiNetworkSet)
		}
	}
}
