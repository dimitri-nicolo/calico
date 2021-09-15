// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package calico

import (
	"reflect"

	"golang.org/x/net/context"

	licClient "github.com/tigera/licensing/client"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/storage"
	etcd "k8s.io/apiserver/pkg/storage/etcd3"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"

	aapi "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/watch"
)

// NewUISettingsGroupStorage creates a new storage. Interface implementation for UISettingsGroups.
func NewUISettingsGroupStorage(opts Options) (registry.DryRunnableStorage, factory.DestroyFunc) {
	c := CreateClientFromConfig()
	createFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*aapi.UISettingsGroup)
		return c.UISettingsGroups().Create(ctx, res, oso)
	}
	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*aapi.UISettingsGroup)
		return c.UISettingsGroups().Update(ctx, res, oso)
	}
	getFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.UISettingsGroups().Get(ctx, name, ogo)
	}
	deleteFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		odo := opts.(options.DeleteOptions)
		return c.UISettingsGroups().Delete(ctx, name, odo)
	}
	listFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (resourceListObject, error) {
		olo := opts.(options.ListOptions)
		return c.UISettingsGroups().List(ctx, olo)
	}
	watchFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (watch.Interface, error) {
		olo := opts.(options.ListOptions)
		return c.UISettingsGroups().Watch(ctx, olo)
	}
	hasRestrictionsFn := func(obj resourceObject, claims *licClient.LicenseClaims) bool {
		return false
	}

	// TODO(doublek): Inject codec, client for nicer testing.
	dryRunnableStorage := registry.DryRunnableStorage{Storage: &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         etcd.APIObjectVersioner{},
		aapiType:          reflect.TypeOf(aapi.UISettingsGroup{}),
		aapiListType:      reflect.TypeOf(aapi.UISettingsGroupList{}),
		libCalicoType:     reflect.TypeOf(aapi.UISettingsGroup{}),
		libCalicoListType: reflect.TypeOf(aapi.UISettingsGroupList{}),
		isNamespaced:      false,
		create:            createFn,
		update:            updateFn,
		get:               getFn,
		delete:            deleteFn,
		list:              listFn,
		watch:             watchFn,
		resourceName:      "UISettingsGroup",
		converter:         UISettingsGroupConverter{},
		licenseCache:      opts.LicenseCache,
		hasRestrictions:   hasRestrictionsFn,
	}, Codec: opts.RESTOptions.StorageConfig.Codec}
	return dryRunnableStorage, func() {}
}

type UISettingsGroupConverter struct {
}

func (gc UISettingsGroupConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiUISettingsGroup := aapiObj.(*aapi.UISettingsGroup)
	lcgUISettingsGroup := &aapi.UISettingsGroup{}
	lcgUISettingsGroup.TypeMeta = aapiUISettingsGroup.TypeMeta
	lcgUISettingsGroup.ObjectMeta = aapiUISettingsGroup.ObjectMeta
	lcgUISettingsGroup.Kind = aapi.KindGlobalReport
	lcgUISettingsGroup.APIVersion = aapi.GroupVersionCurrent
	lcgUISettingsGroup.Spec = aapiUISettingsGroup.Spec
	return lcgUISettingsGroup
}

func (gc UISettingsGroupConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgUISettingsGroup := libcalicoObject.(*aapi.UISettingsGroup)
	aapiUISettingsGroup := aapiObj.(*aapi.UISettingsGroup)
	aapiUISettingsGroup.Spec = lcgUISettingsGroup.Spec
	aapiUISettingsGroup.TypeMeta = lcgUISettingsGroup.TypeMeta
	aapiUISettingsGroup.ObjectMeta = lcgUISettingsGroup.ObjectMeta
}

func (gc UISettingsGroupConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	lcgUISettingsGroupList := libcalicoListObject.(*aapi.UISettingsGroupList)
	aapiUISettingsGroupList := aapiListObj.(*aapi.UISettingsGroupList)
	if libcalicoListObject == nil {
		aapiUISettingsGroupList.Items = []aapi.UISettingsGroup{}
		return
	}
	aapiUISettingsGroupList.TypeMeta = lcgUISettingsGroupList.TypeMeta
	aapiUISettingsGroupList.ListMeta = lcgUISettingsGroupList.ListMeta
	for _, item := range lcgUISettingsGroupList.Items {
		aapiUISettingsGroup := aapi.UISettingsGroup{}
		gc.convertToAAPI(&item, &aapiUISettingsGroup)
		if matched, err := pred.Matches(&aapiUISettingsGroup); err == nil && matched {
			aapiUISettingsGroupList.Items = append(aapiUISettingsGroupList.Items, aapiUISettingsGroup)
		}
	}
}
