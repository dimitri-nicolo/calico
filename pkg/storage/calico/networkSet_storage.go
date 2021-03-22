// Copyright (c) 2017-2019 Tigera, Inc. All rights reserved.

package calico

import (
	"reflect"

	"golang.org/x/net/context"

	aapi "github.com/tigera/apiserver/pkg/apis/projectcalico"
	licClient "github.com/tigera/licensing/client"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/storage"
	etcd "k8s.io/apiserver/pkg/storage/etcd3"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"

	libcalicoapi "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/watch"
)

// NewNetworkSetStorage creates a new libcalico-based storage.Interface implementation for NetworkSets
func NewNetworkSetStorage(opts Options) (registry.DryRunnableStorage, factory.DestroyFunc) {
	c := CreateClientFromConfig()
	createFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.NetworkSet)
		return c.NetworkSets().Create(ctx, res, oso)
	}
	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.NetworkSet)
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
	hasRestrictionsFn := func(obj resourceObject, claims *licClient.LicenseClaims) bool {
		return false
	}
	// TODO(doublek): Inject codec, client for nicer testing.
	dryRunnableStorage := registry.DryRunnableStorage{Storage: &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         etcd.APIObjectVersioner{},
		aapiType:          reflect.TypeOf(aapi.NetworkSet{}),
		aapiListType:      reflect.TypeOf(aapi.NetworkSetList{}),
		libCalicoType:     reflect.TypeOf(libcalicoapi.NetworkSet{}),
		libCalicoListType: reflect.TypeOf(libcalicoapi.NetworkSetList{}),
		isNamespaced:      true,
		create:            createFn,
		update:            updateFn,
		get:               getFn,
		delete:            deleteFn,
		list:              listFn,
		watch:             watchFn,
		resourceName:      "NetworkSet",
		converter:         NetworkSetConverter{},
		licenseCache:      opts.LicenseCache,
		hasRestrictions:   hasRestrictionsFn,
	}, Codec: opts.RESTOptions.StorageConfig.Codec}
	return dryRunnableStorage, func() {}
}

type NetworkSetConverter struct {
}

func (gc NetworkSetConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiNetworkSet := aapiObj.(*aapi.NetworkSet)
	lcgNetworkSet := &libcalicoapi.NetworkSet{}
	lcgNetworkSet.TypeMeta = aapiNetworkSet.TypeMeta
	lcgNetworkSet.ObjectMeta = aapiNetworkSet.ObjectMeta
	lcgNetworkSet.Kind = libcalicoapi.KindNetworkSet
	lcgNetworkSet.APIVersion = libcalicoapi.GroupVersionCurrent
	lcgNetworkSet.Spec = aapiNetworkSet.Spec
	return lcgNetworkSet
}

func (gc NetworkSetConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgNetworkSet := libcalicoObject.(*libcalicoapi.NetworkSet)
	aapiNetworkSet := aapiObj.(*aapi.NetworkSet)
	aapiNetworkSet.Spec = lcgNetworkSet.Spec
	aapiNetworkSet.TypeMeta = lcgNetworkSet.TypeMeta
	aapiNetworkSet.ObjectMeta = lcgNetworkSet.ObjectMeta
}

func (gc NetworkSetConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	lcgNetworkSetList := libcalicoListObject.(*libcalicoapi.NetworkSetList)
	aapiNetworkSetList := aapiListObj.(*aapi.NetworkSetList)
	if libcalicoListObject == nil {
		aapiNetworkSetList.Items = []aapi.NetworkSet{}
		return
	}
	aapiNetworkSetList.TypeMeta = lcgNetworkSetList.TypeMeta
	aapiNetworkSetList.ListMeta = lcgNetworkSetList.ListMeta
	for _, item := range lcgNetworkSetList.Items {
		aapiNetworkSet := aapi.NetworkSet{}
		gc.convertToAAPI(&item, &aapiNetworkSet)
		if matched, err := pred.Matches(&aapiNetworkSet); err == nil && matched {
			aapiNetworkSetList.Items = append(aapiNetworkSetList.Items, aapiNetworkSet)
		}
	}
}
