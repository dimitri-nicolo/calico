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

	libcalicoapi "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/watch"

	licClient "github.com/tigera/licensing/client"
	features "github.com/tigera/licensing/client/features"

	aapi "github.com/projectcalico/apiserver/pkg/apis/projectcalico"
)

// NewGlobalReportTypeStorage creates a new libcalico-based storage.Interface implementation for GlobalReportTypes
func NewGlobalReportTypeStorage(opts Options) (registry.DryRunnableStorage, factory.DestroyFunc) {
	c := CreateClientFromConfig()
	createFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.GlobalReportType)
		return c.GlobalReportTypes().Create(ctx, res, oso)
	}
	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.GlobalReportType)
		return c.GlobalReportTypes().Update(ctx, res, oso)
	}
	getFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.GlobalReportTypes().Get(ctx, name, ogo)
	}
	deleteFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		odo := opts.(options.DeleteOptions)
		return c.GlobalReportTypes().Delete(ctx, name, odo)
	}
	listFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (resourceListObject, error) {
		olo := opts.(options.ListOptions)
		return c.GlobalReportTypes().List(ctx, olo)
	}
	watchFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (watch.Interface, error) {
		olo := opts.(options.ListOptions)
		return c.GlobalReportTypes().Watch(ctx, olo)
	}
	hasRestrictionsFn := func(obj resourceObject, claims *licClient.LicenseClaims) bool {
		return !claims.ValidateFeature(features.ComplianceReports)
	}

	dryRunnableStorage := registry.DryRunnableStorage{Storage: &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         etcd.APIObjectVersioner{},
		aapiType:          reflect.TypeOf(aapi.GlobalReportType{}),
		aapiListType:      reflect.TypeOf(aapi.GlobalReportTypeList{}),
		libCalicoType:     reflect.TypeOf(libcalicoapi.GlobalReportType{}),
		libCalicoListType: reflect.TypeOf(libcalicoapi.GlobalReportTypeList{}),
		isNamespaced:      false,
		create:            createFn,
		update:            updateFn,
		get:               getFn,
		delete:            deleteFn,
		list:              listFn,
		watch:             watchFn,
		resourceName:      "GlobalReportType",
		converter:         GlobalReportTypeConverter{},
		licenseCache:      opts.LicenseCache,
		hasRestrictions:   hasRestrictionsFn,
	}, Codec: opts.RESTOptions.StorageConfig.Codec}
	return dryRunnableStorage, func() {}
}

type GlobalReportTypeConverter struct {
}

func (gc GlobalReportTypeConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiGlobalReportType := aapiObj.(*aapi.GlobalReportType)
	lcgGlobalReportType := &libcalicoapi.GlobalReportType{}
	lcgGlobalReportType.TypeMeta = aapiGlobalReportType.TypeMeta
	lcgGlobalReportType.ObjectMeta = aapiGlobalReportType.ObjectMeta
	lcgGlobalReportType.Kind = libcalicoapi.KindGlobalReportList
	lcgGlobalReportType.APIVersion = libcalicoapi.GroupVersionCurrent
	lcgGlobalReportType.Spec = aapiGlobalReportType.Spec
	return lcgGlobalReportType
}

func (gc GlobalReportTypeConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgGlobalReportType := libcalicoObject.(*libcalicoapi.GlobalReportType)
	aapiGlobalReportType := aapiObj.(*aapi.GlobalReportType)
	aapiGlobalReportType.Spec = lcgGlobalReportType.Spec
	aapiGlobalReportType.TypeMeta = lcgGlobalReportType.TypeMeta
	aapiGlobalReportType.ObjectMeta = lcgGlobalReportType.ObjectMeta
}

func (gc GlobalReportTypeConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	lcgGlobalReportTypeList := libcalicoListObject.(*libcalicoapi.GlobalReportTypeList)
	aapiGlobalReportTypeList := aapiListObj.(*aapi.GlobalReportTypeList)
	if libcalicoListObject == nil {
		aapiGlobalReportTypeList.Items = []aapi.GlobalReportType{}
		return
	}
	aapiGlobalReportTypeList.TypeMeta = lcgGlobalReportTypeList.TypeMeta
	aapiGlobalReportTypeList.ListMeta = lcgGlobalReportTypeList.ListMeta
	for _, item := range lcgGlobalReportTypeList.Items {
		aapiGlobalReportType := aapi.GlobalReportType{}
		gc.convertToAAPI(&item, &aapiGlobalReportType)
		if matched, err := pred.Matches(&aapiGlobalReportType); err == nil && matched {
			aapiGlobalReportTypeList.Items = append(aapiGlobalReportTypeList.Items, aapiGlobalReportType)
		}
	}
}
