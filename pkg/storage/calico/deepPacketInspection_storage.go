// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package calico

import (
	"reflect"

	"golang.org/x/net/context"

	licClient "github.com/tigera/licensing/client"
	features "github.com/tigera/licensing/client/features"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/storage"
	etcd "k8s.io/apiserver/pkg/storage/etcd3"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"

	aapi "github.com/projectcalico/apiserver/pkg/apis/projectcalico"

	libcalicoapi "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/watch"
)

// NewDeepPacketInspectionStorage creates a new libcalico-based storage.Interface implementation for DeepPacketInspections
func NewDeepPacketInspectionStorage(opts Options) (registry.DryRunnableStorage, factory.DestroyFunc) {
	c := CreateClientFromConfig()
	createFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.DeepPacketInspection)
		return c.DeepPacketInspections().Create(ctx, res, oso)
	}
	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.DeepPacketInspection)
		return c.DeepPacketInspections().Update(ctx, res, oso)
	}
	getFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.DeepPacketInspections().Get(ctx, ns, name, ogo)
	}
	deleteFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		odo := opts.(options.DeleteOptions)
		return c.DeepPacketInspections().Delete(ctx, ns, name, odo)
	}
	listFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (resourceListObject, error) {
		olo := opts.(options.ListOptions)
		return c.DeepPacketInspections().List(ctx, olo)
	}
	watchFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (watch.Interface, error) {
		olo := opts.(options.ListOptions)
		return c.DeepPacketInspections().Watch(ctx, olo)
	}
	hasRestrictionsFn := func(obj resourceObject, claims *licClient.LicenseClaims) bool {
		return !claims.ValidateFeature(features.ThreatDefense)
	}
	// TODO(doublek): Inject codec, client for nicer testing.
	dryRunnableStorage := registry.DryRunnableStorage{Storage: &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         etcd.APIObjectVersioner{},
		aapiType:          reflect.TypeOf(aapi.DeepPacketInspection{}),
		aapiListType:      reflect.TypeOf(aapi.DeepPacketInspectionList{}),
		libCalicoType:     reflect.TypeOf(libcalicoapi.DeepPacketInspection{}),
		libCalicoListType: reflect.TypeOf(libcalicoapi.DeepPacketInspectionList{}),
		isNamespaced:      true,
		create:            createFn,
		update:            updateFn,
		get:               getFn,
		delete:            deleteFn,
		list:              listFn,
		watch:             watchFn,
		resourceName:      "DeepPacketInspection",
		converter:         DeepPacketInspectionConverter{},
		licenseCache:      opts.LicenseCache,
		hasRestrictions:   hasRestrictionsFn,
	}, Codec: opts.RESTOptions.StorageConfig.Codec}
	return dryRunnableStorage, func() {}
}

type DeepPacketInspectionConverter struct {
}

func (gc DeepPacketInspectionConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiDeepPacketInspection := aapiObj.(*aapi.DeepPacketInspection)
	lcgDeepPacketInspection := &libcalicoapi.DeepPacketInspection{}
	lcgDeepPacketInspection.TypeMeta = aapiDeepPacketInspection.TypeMeta
	lcgDeepPacketInspection.ObjectMeta = aapiDeepPacketInspection.ObjectMeta
	lcgDeepPacketInspection.Kind = libcalicoapi.KindDeepPacketInspection
	lcgDeepPacketInspection.APIVersion = libcalicoapi.GroupVersionCurrent
	lcgDeepPacketInspection.Spec = aapiDeepPacketInspection.Spec
	lcgDeepPacketInspection.Status = aapiDeepPacketInspection.Status
	return lcgDeepPacketInspection
}

func (gc DeepPacketInspectionConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgDeepPacketInspection := libcalicoObject.(*libcalicoapi.DeepPacketInspection)
	aapiDeepPacketInspection := aapiObj.(*aapi.DeepPacketInspection)
	aapiDeepPacketInspection.Spec = lcgDeepPacketInspection.Spec
	aapiDeepPacketInspection.Status = lcgDeepPacketInspection.Status
	aapiDeepPacketInspection.TypeMeta = lcgDeepPacketInspection.TypeMeta
	aapiDeepPacketInspection.ObjectMeta = lcgDeepPacketInspection.ObjectMeta
}

func (gc DeepPacketInspectionConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	lcgDeepPacketInspectionList := libcalicoListObject.(*libcalicoapi.DeepPacketInspectionList)
	aapiDeepPacketInspectionList := aapiListObj.(*aapi.DeepPacketInspectionList)
	if libcalicoListObject == nil {
		aapiDeepPacketInspectionList.Items = []aapi.DeepPacketInspection{}
		return
	}
	aapiDeepPacketInspectionList.TypeMeta = lcgDeepPacketInspectionList.TypeMeta
	aapiDeepPacketInspectionList.ListMeta = lcgDeepPacketInspectionList.ListMeta
	for _, item := range lcgDeepPacketInspectionList.Items {
		aapiDeepPacketInspection := aapi.DeepPacketInspection{}
		gc.convertToAAPI(&item, &aapiDeepPacketInspection)
		if matched, err := pred.Matches(&aapiDeepPacketInspection); err == nil && matched {
			aapiDeepPacketInspectionList.Items = append(aapiDeepPacketInspectionList.Items, aapiDeepPacketInspection)
		}
	}
}
