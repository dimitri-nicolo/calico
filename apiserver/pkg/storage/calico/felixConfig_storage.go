// Copyright (c) 2019 Tigera, Inc. All rights reserved.

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
)

// NewFelixConfigurationStorage creates a new libcalico-based storage.Interface implementation for FelixConfigurations
func NewFelixConfigurationStorage(opts Options) (registry.DryRunnableStorage, factory.DestroyFunc) {
	c := CreateClientFromConfig()
	createFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*v3.FelixConfiguration)
		return c.FelixConfigurations().Create(ctx, res, oso)
	}
	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*v3.FelixConfiguration)
		return c.FelixConfigurations().Update(ctx, res, oso)
	}
	getFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.FelixConfigurations().Get(ctx, name, ogo)
	}
	deleteFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		odo := opts.(options.DeleteOptions)
		return c.FelixConfigurations().Delete(ctx, name, odo)
	}
	listFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (resourceListObject, error) {
		olo := opts.(options.ListOptions)
		return c.FelixConfigurations().List(ctx, olo)
	}
	watchFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (watch.Interface, error) {
		olo := opts.(options.ListOptions)
		return c.FelixConfigurations().Watch(ctx, olo)
	}
	hasRestrictionsFn := func(obj resourceObject) bool {
		return false
	}

	dryRunnableStorage := registry.DryRunnableStorage{Storage: &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         APIObjectVersioner{},
		aapiType:          reflect.TypeOf(v3.FelixConfiguration{}),
		aapiListType:      reflect.TypeOf(v3.FelixConfigurationList{}),
		libCalicoType:     reflect.TypeOf(v3.FelixConfiguration{}),
		libCalicoListType: reflect.TypeOf(v3.FelixConfigurationList{}),
		isNamespaced:      false,
		create:            createFn,
		update:            updateFn,
		get:               getFn,
		delete:            deleteFn,
		list:              listFn,
		watch:             watchFn,
		resourceName:      "FelixConfiguration",
		converter:         FelixConfigurationConverter{},
		hasRestrictions:   hasRestrictionsFn,
	}, Codec: opts.RESTOptions.StorageConfig.Codec}
	return dryRunnableStorage, func() {}
}

type FelixConfigurationConverter struct {
}

func (gc FelixConfigurationConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiFelixConfig := aapiObj.(*v3.FelixConfiguration)
	lcgFelixConfig := &v3.FelixConfiguration{}
	lcgFelixConfig.TypeMeta = aapiFelixConfig.TypeMeta
	lcgFelixConfig.ObjectMeta = aapiFelixConfig.ObjectMeta
	lcgFelixConfig.Kind = v3.KindFelixConfiguration
	lcgFelixConfig.APIVersion = v3.GroupVersionCurrent
	lcgFelixConfig.Spec = aapiFelixConfig.Spec
	return lcgFelixConfig
}

func (gc FelixConfigurationConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgFelixConfig := libcalicoObject.(*v3.FelixConfiguration)
	aapiFelixConfig := aapiObj.(*v3.FelixConfiguration)
	aapiFelixConfig.Spec = lcgFelixConfig.Spec
	aapiFelixConfig.TypeMeta = lcgFelixConfig.TypeMeta
	aapiFelixConfig.ObjectMeta = lcgFelixConfig.ObjectMeta
}

func (gc FelixConfigurationConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	lcgFelixConfigList := libcalicoListObject.(*v3.FelixConfigurationList)
	aapiFelixConfigList := aapiListObj.(*v3.FelixConfigurationList)
	if libcalicoListObject == nil {
		aapiFelixConfigList.Items = []v3.FelixConfiguration{}
		return
	}
	aapiFelixConfigList.TypeMeta = lcgFelixConfigList.TypeMeta
	aapiFelixConfigList.ListMeta = lcgFelixConfigList.ListMeta
	for _, item := range lcgFelixConfigList.Items {
		aapiFelixConfig := v3.FelixConfiguration{}
		gc.convertToAAPI(&item, &aapiFelixConfig)
		if matched, err := pred.Matches(&aapiFelixConfig); err == nil && matched {
			aapiFelixConfigList.Items = append(aapiFelixConfigList.Items, aapiFelixConfig)
		}
	}
}
