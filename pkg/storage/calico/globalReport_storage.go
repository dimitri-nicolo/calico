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

// NewGlobalReportStorage creates a new libcalico-based storage.Interface implementation for GlobalReports
func NewGlobalReportStorage(opts Options) (storage.Interface, factory.DestroyFunc) {
	c := createClientFromConfig()
	createFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.GlobalReport)
		return c.GlobalReports().Create(ctx, res, oso)
	}
	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.GlobalReport)
		return c.GlobalReports().Update(ctx, res, oso)
	}
	getFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.GlobalReports().Get(ctx, name, ogo)
	}
	deleteFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		odo := opts.(options.DeleteOptions)
		return c.GlobalReports().Delete(ctx, name, odo)
	}
	listFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (resourceListObject, error) {
		olo := opts.(options.ListOptions)
		return c.GlobalReports().List(ctx, olo)
	}
	watchFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (watch.Interface, error) {
		olo := opts.(options.ListOptions)
		return c.GlobalReports().Watch(ctx, olo)
	}

	return &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         etcd.APIObjectVersioner{},
		aapiType:          reflect.TypeOf(aapi.GlobalReport{}),
		aapiListType:      reflect.TypeOf(aapi.GlobalReportList{}),
		libCalicoType:     reflect.TypeOf(libcalicoapi.GlobalReport{}),
		libCalicoListType: reflect.TypeOf(libcalicoapi.GlobalReportList{}),
		isNamespaced:      false,
		create:            createFn,
		update:            updateFn,
		get:               getFn,
		delete:            deleteFn,
		list:              listFn,
		watch:             watchFn,
		resourceName:      "GlobalReport",
		converter:         GlobalReportConverter{},
	}, func() {}
}

type GlobalReportConverter struct {
}

func (gc GlobalReportConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiGlobalReport := aapiObj.(*aapi.GlobalReport)
	lcgGlobalReport := &libcalicoapi.GlobalReport{}
	lcgGlobalReport.TypeMeta = aapiGlobalReport.TypeMeta
	lcgGlobalReport.ObjectMeta = aapiGlobalReport.ObjectMeta
	lcgGlobalReport.Spec = aapiGlobalReport.Spec
	lcgGlobalReport.Status = aapiGlobalReport.Status
	return lcgGlobalReport
}

func (gc GlobalReportConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgGlobalReport := libcalicoObject.(*libcalicoapi.GlobalReport)
	aapiGlobalReport := aapiObj.(*aapi.GlobalReport)
	aapiGlobalReport.Spec = lcgGlobalReport.Spec
	aapiGlobalReport.Status = lcgGlobalReport.Status
	aapiGlobalReport.TypeMeta = lcgGlobalReport.TypeMeta
	aapiGlobalReport.ObjectMeta = lcgGlobalReport.ObjectMeta
}

func (gc GlobalReportConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	lcgGlobalReportList := libcalicoListObject.(*libcalicoapi.GlobalReportList)
	aapiGlobalReportList := aapiListObj.(*aapi.GlobalReportList)
	if libcalicoListObject == nil {
		aapiGlobalReportList.Items = []aapi.GlobalReport{}
		return
	}
	aapiGlobalReportList.TypeMeta = lcgGlobalReportList.TypeMeta
	aapiGlobalReportList.ListMeta = lcgGlobalReportList.ListMeta
	for _, item := range lcgGlobalReportList.Items {
		aapiGlobalReport := aapi.GlobalReport{}
		gc.convertToAAPI(&item, &aapiGlobalReport)
		if matched, err := pred.Matches(&aapiGlobalReport); err == nil && matched {
			aapiGlobalReportList.Items = append(aapiGlobalReportList.Items, aapiGlobalReport)
		}
	}
}
