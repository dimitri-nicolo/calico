// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package calico

import (
	"reflect"

	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/etcd"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"

	libcalicoapi "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/watch"

	aapi "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
)

// NewGlobalReportTypeStorage creates a new libcalico-based storage.Interface implementation for GlobalReportTypes
func NewGlobalReportTypeStorage(opts Options) (storage.Interface, factory.DestroyFunc) {
	c := createClientFromConfig()
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

	return &resourceStore{
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
	}, func() {}
}

type GlobalReportTypeConverter struct {
}

func (gc GlobalReportTypeConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiGlobalReportType := aapiObj.(*aapi.GlobalReportType)
	lcgGlobalReportType := &libcalicoapi.GlobalReportType{}
	lcgGlobalReportType.TypeMeta = aapiGlobalReportType.TypeMeta
	lcgGlobalReportType.ObjectMeta = aapiGlobalReportType.ObjectMeta
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
