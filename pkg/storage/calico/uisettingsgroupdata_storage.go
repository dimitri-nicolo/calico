// Copyright (c) 2017-2019 Tigera, Inc. All rights reserved.

package calico

import (
	"reflect"

	"golang.org/x/net/context"

	copier "github.com/jinzhu/copier"
	log "github.com/sirupsen/logrus"
	licClient "github.com/tigera/licensing/client"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/storage"
	etcd "k8s.io/apiserver/pkg/storage/etcd3"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"

	aapi "github.com/tigera/api/pkg/apis/projectcalico/v3"

	api "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
)

// NewUISettingsGroupDataStorage creates a new libcalico-based storage.Interface implementation for UISettingsGroupData
func NewUISettingsGroupDataStorage(opts Options) (registry.DryRunnableStorage, factory.DestroyFunc) {
	c := CreateClientFromConfig()
	createFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*api.UISettingsGroup)
		return c.UISettingsGroups().Create(ctx, res, oso)
	}
	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*api.UISettingsGroup)
		return c.UISettingsGroups().Update(ctx, res, oso)
	}
	deleteFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		odo := opts.(options.DeleteOptions)
		return c.UISettingsGroups().Delete(ctx, name, odo)
	}
	listFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (resourceListObject, error) {
		olo := opts.(options.ListOptions)
		return c.UISettingsGroups().List(ctx, olo)
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
		libCalicoType:     reflect.TypeOf(api.UISettingsGroup{}),
		libCalicoListType: reflect.TypeOf(api.UISettingsGroupList{}),
		isNamespaced:      false,
		create:            createFn,
		update:            updateFn,
		delete:            deleteFn,
		list:              listFn,
		resourceName:      "UISettingsGroupData",
		converter:         UISettingsGroupDataConverter{},
		licenseCache:      opts.LicenseCache,
		hasRestrictions:   hasRestrictionsFn,
	}, Codec: opts.RESTOptions.StorageConfig.Codec}
	return dryRunnableStorage, func() {}
}

// TODO(dimitrin): Deprecated functionality meant to convert to/from libcacilo and apiserver
// resource structures. Remove converter once resourceStore has been refactored.
type UISettingsGroupDataConverter struct {
}

// TODO(dimitrin): Deprecated functionality meant to convert to/from libcacilo and apiserver
// resource structures. Remove converter once resourceStore has been refactored.
func (gc UISettingsGroupDataConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	uiSettingsGroup := aapiObj.(*api.UISettingsGroup)
	return uiSettingsGroup
}

// TODO(dimitrin): Deprecated functionality meant to convert to/from libcacilo and apiserver
// resource structures. Remove converter once resourceStore has been refactored.
func (gc UISettingsGroupDataConverter) convertToAAPI(apiObject resourceObject, aapiObj runtime.Object) {
	inUISettingsGroup := apiObject.(*api.UISettingsGroup)
	outUISettingsGroup := aapiObj.(*api.UISettingsGroup)
	err := copier.Copy(outUISettingsGroup, inUISettingsGroup)
	if err != nil {
		log.WithError(err).Errorf("failed to copy type %v.", reflect.TypeOf(inUISettingsGroup))
	}
}

// TODO(dimitrin): Deprecated functionality meant to convert to/from libcacilo and apiserver
// resource structures. Remove converter once resourceStore has been refactored.
func (gc UISettingsGroupDataConverter) convertToAAPIList(apiListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	aapiUISettingsGroupList := aapiListObj.(*api.UISettingsGroupList)
	if apiListObject == nil {
		aapiUISettingsGroupList.Items = []api.UISettingsGroup{}
		return
	}
	for _, item := range aapiUISettingsGroupList.Items {
		aapiUISettingsGroup := api.UISettingsGroup{}
		gc.convertToAAPI(&item, &aapiUISettingsGroup)
		if matched, err := pred.Matches(&aapiUISettingsGroup); err == nil && matched {
			aapiUISettingsGroupList.Items = append(aapiUISettingsGroupList.Items, aapiUISettingsGroup)
		}
	}
}
