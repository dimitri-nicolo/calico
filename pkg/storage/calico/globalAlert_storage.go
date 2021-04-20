// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package calico

import (
	"reflect"

	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/storage"
	etcd "k8s.io/apiserver/pkg/storage/etcd3"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"

	libcalicoapi "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/watch"

	licClient "github.com/tigera/licensing/client"
	features "github.com/tigera/licensing/client/features"

	aapi "github.com/projectcalico/apiserver/pkg/apis/projectcalico"
)

// NewGlobalAlertStorage creates a new libcalico-based storage.Interface implementation for GlobalAlerts
func NewGlobalAlertStorage(opts Options) (registry.DryRunnableStorage, factory.DestroyFunc) {
	c := CreateClientFromConfig()
	createFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.GlobalAlert)

		return c.GlobalAlerts().Create(ctx, res, oso)
	}
	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.GlobalAlert)
		return c.GlobalAlerts().Update(ctx, res, oso)
	}
	getFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.GlobalAlerts().Get(ctx, name, ogo)
	}
	deleteFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		odo := opts.(options.DeleteOptions)
		return c.GlobalAlerts().Delete(ctx, name, odo)
	}
	listFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (resourceListObject, error) {
		olo := opts.(options.ListOptions)
		return c.GlobalAlerts().List(ctx, olo)
	}
	watchFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (watch.Interface, error) {
		olo := opts.(options.ListOptions)
		return c.GlobalAlerts().Watch(ctx, olo)
	}
	hasRestrictionsFn := func(obj resourceObject, claims *licClient.LicenseClaims) bool {
		return !claims.ValidateFeature(features.AlertManagement)
	}
	// TODO(doublek): Inject codec, client for nicer testing.
	dryRunnableStorage := registry.DryRunnableStorage{Storage: &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         etcd.APIObjectVersioner{},
		aapiType:          reflect.TypeOf(aapi.GlobalAlert{}),
		aapiListType:      reflect.TypeOf(aapi.GlobalAlertList{}),
		libCalicoType:     reflect.TypeOf(libcalicoapi.GlobalAlert{}),
		libCalicoListType: reflect.TypeOf(libcalicoapi.GlobalAlertList{}),
		isNamespaced:      false,
		create:            createFn,
		update:            updateFn,
		get:               getFn,
		delete:            deleteFn,
		list:              listFn,
		watch:             watchFn,
		resourceName:      "GlobalAlert",
		converter:         GlobalAlertConverter{},
		licenseCache:      opts.LicenseCache,
		hasRestrictions:   hasRestrictionsFn,
	}, Codec: opts.RESTOptions.StorageConfig.Codec}
	return dryRunnableStorage, func() {}
}

type GlobalAlertConverter struct {
}

func (gc GlobalAlertConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiGlobalAlert := aapiObj.(*aapi.GlobalAlert)
	lcgGlobalAlert := &libcalicoapi.GlobalAlert{}
	lcgGlobalAlert.TypeMeta = aapiGlobalAlert.TypeMeta
	lcgGlobalAlert.Kind = libcalicoapi.KindGlobalAlert
	lcgGlobalAlert.APIVersion = libcalicoapi.GroupVersionCurrent
	lcgGlobalAlert.ObjectMeta = aapiGlobalAlert.ObjectMeta
	lcgGlobalAlert.Spec = aapiGlobalAlert.Spec
	lcgGlobalAlert.Status = aapiGlobalAlert.Status
	return lcgGlobalAlert
}

func (gc GlobalAlertConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgGlobalAlert := libcalicoObject.(*libcalicoapi.GlobalAlert)
	aapiGlobalAlert := aapiObj.(*aapi.GlobalAlert)
	aapiGlobalAlert.Spec = lcgGlobalAlert.Spec
	aapiGlobalAlert.Status = lcgGlobalAlert.Status
	aapiGlobalAlert.TypeMeta = lcgGlobalAlert.TypeMeta
	aapiGlobalAlert.ObjectMeta = lcgGlobalAlert.ObjectMeta
}

func (gc GlobalAlertConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	lcgGlobalAlertList := libcalicoListObject.(*libcalicoapi.GlobalAlertList)
	aapiGlobalAlertList := aapiListObj.(*aapi.GlobalAlertList)
	if libcalicoListObject == nil {
		aapiGlobalAlertList.Items = []aapi.GlobalAlert{}
		return
	}
	aapiGlobalAlertList.TypeMeta = lcgGlobalAlertList.TypeMeta
	aapiGlobalAlertList.ListMeta = lcgGlobalAlertList.ListMeta
	for _, item := range lcgGlobalAlertList.Items {
		aapiGlobalAlert := aapi.GlobalAlert{}
		gc.convertToAAPI(&item, &aapiGlobalAlert)
		if matched, err := pred.Matches(&aapiGlobalAlert); err == nil && matched {
			aapiGlobalAlertList.Items = append(aapiGlobalAlertList.Items, aapiGlobalAlert)
		}
	}
}
