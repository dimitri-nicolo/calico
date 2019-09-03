// Copyright (c) 2017-2019 Tigera, Inc. All rights reserved.

package calico

import (
	"reflect"

	"golang.org/x/net/context"

	aapi "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/etcd"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"

	libcalicoapi "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/watch"
)

// NewLicenseKeyStorage creates a new libcalico-based storage.Interface implementation for LicenseKeys
func NewLicenseKeyStorage(opts Options) (storage.Interface, factory.DestroyFunc) {
	c := createClientFromConfig()
	createFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.LicenseKey)
		return c.LicenseKey().Create(ctx, res, oso)
	}
	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.LicenseKey)
		return c.LicenseKey().Update(ctx, res, oso)
	}
	getFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.LicenseKey().Get(ctx, name, ogo)
	}
	deleteFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		odo := opts.(options.DeleteOptions)
		return c.LicenseKey().Delete(ctx, name, odo)
	}
	listFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (resourceListObject, error) {
		olo := opts.(options.ListOptions)
		return c.LicenseKey().List(ctx, olo)
	}
	watchFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (watch.Interface, error) {
		olo := opts.(options.ListOptions)
		return c.LicenseKey().Watch(ctx, olo)
	}
	return &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         etcd.APIObjectVersioner{},
		aapiType:          reflect.TypeOf(aapi.LicenseKey{}),
		aapiListType:      reflect.TypeOf(aapi.LicenseKeyList{}),
		libCalicoType:     reflect.TypeOf(libcalicoapi.LicenseKey{}),
		libCalicoListType: reflect.TypeOf(libcalicoapi.LicenseKeyList{}),
		isNamespaced:      false,
		create:            createFn,
		update:            updateFn,
		get:               getFn,
		delete:            deleteFn,
		list:              listFn,
		watch:             watchFn,
		resourceName:      "LicenseKey",
		converter:         LicenseKeyConverter{},
	}, func() {}
}

type LicenseKeyConverter struct {
}

func (gc LicenseKeyConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiLicenseKey := aapiObj.(*aapi.LicenseKey)
	lcgLicenseKey := &libcalicoapi.LicenseKey{}
	lcgLicenseKey.TypeMeta = aapiLicenseKey.TypeMeta
	lcgLicenseKey.ObjectMeta = aapiLicenseKey.ObjectMeta
	lcgLicenseKey.Spec = aapiLicenseKey.Spec
	return lcgLicenseKey
}

func (gc LicenseKeyConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgLicenseKey := libcalicoObject.(*libcalicoapi.LicenseKey)
	aapiLicenseKey := aapiObj.(*aapi.LicenseKey)
	aapiLicenseKey.Spec = lcgLicenseKey.Spec
	aapiLicenseKey.TypeMeta = lcgLicenseKey.TypeMeta
	aapiLicenseKey.ObjectMeta = lcgLicenseKey.ObjectMeta
}

func (gc LicenseKeyConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	lcgLicenseKeyList := libcalicoListObject.(*libcalicoapi.LicenseKeyList)
	aapiLicenseKeyList := aapiListObj.(*aapi.LicenseKeyList)
	if libcalicoListObject == nil {
		aapiLicenseKeyList.Items = []aapi.LicenseKey{}
		return
	}
	aapiLicenseKeyList.TypeMeta = lcgLicenseKeyList.TypeMeta
	aapiLicenseKeyList.ListMeta = lcgLicenseKeyList.ListMeta
	for _, item := range lcgLicenseKeyList.Items {
		aapiLicenseKey := aapi.LicenseKey{}
		gc.convertToAAPI(&item, &aapiLicenseKey)
		if matched, err := pred.Matches(&aapiLicenseKey); err == nil && matched {
			aapiLicenseKeyList.Items = append(aapiLicenseKeyList.Items, aapiLicenseKey)
		}
	}
}
