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

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
	"github.com/projectcalico/calico/libcalico-go/lib/watch"
)

// NewBGPConfigurationStorage creates a new libcalico-based storage.Interface implementation for BGPConfigurations
func NewBGPConfigurationStorage(opts Options) (registry.DryRunnableStorage, factory.DestroyFunc) {
	c := CreateClientFromConfig()
	createFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*v3.BGPConfiguration)
		return c.BGPConfigurations().Create(ctx, res, oso)
	}
	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*v3.BGPConfiguration)
		return c.BGPConfigurations().Update(ctx, res, oso)
	}
	getFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.BGPConfigurations().Get(ctx, name, ogo)
	}
	deleteFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		odo := opts.(options.DeleteOptions)
		return c.BGPConfigurations().Delete(ctx, name, odo)
	}
	listFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (resourceListObject, error) {
		olo := opts.(options.ListOptions)
		return c.BGPConfigurations().List(ctx, olo)
	}
	watchFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (watch.Interface, error) {
		olo := opts.(options.ListOptions)
		return c.BGPConfigurations().Watch(ctx, olo)
	}
	hasRestrictionsFn := func(obj resourceObject) bool {
		return false
	}

	dryRunnableStorage := registry.DryRunnableStorage{Storage: &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         etcd.APIObjectVersioner{},
		aapiType:          reflect.TypeOf(v3.BGPConfiguration{}),
		aapiListType:      reflect.TypeOf(v3.BGPConfigurationList{}),
		libCalicoType:     reflect.TypeOf(v3.BGPConfiguration{}),
		libCalicoListType: reflect.TypeOf(v3.BGPConfigurationList{}),
		isNamespaced:      false,
		create:            createFn,
		update:            updateFn,
		get:               getFn,
		delete:            deleteFn,
		list:              listFn,
		watch:             watchFn,
		resourceName:      "BGPConfiguration",
		converter:         BGPConfigurationConverter{},
		hasRestrictions:   hasRestrictionsFn,
	}, Codec: opts.RESTOptions.StorageConfig.Codec}
	return dryRunnableStorage, func() {}
}

type BGPConfigurationConverter struct {
}

func (gc BGPConfigurationConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiBGPConfiguration := aapiObj.(*v3.BGPConfiguration)
	lcgBGPConfiguration := &v3.BGPConfiguration{}
	lcgBGPConfiguration.TypeMeta = aapiBGPConfiguration.TypeMeta
	lcgBGPConfiguration.ObjectMeta = aapiBGPConfiguration.ObjectMeta
	lcgBGPConfiguration.Kind = v3.KindBGPConfiguration
	lcgBGPConfiguration.APIVersion = v3.GroupVersionCurrent
	lcgBGPConfiguration.Spec = aapiBGPConfiguration.Spec
	return lcgBGPConfiguration
}

func (gc BGPConfigurationConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgBGPConfiguration := libcalicoObject.(*v3.BGPConfiguration)
	aapiBGPConfiguration := aapiObj.(*v3.BGPConfiguration)
	aapiBGPConfiguration.Spec = lcgBGPConfiguration.Spec
	aapiBGPConfiguration.TypeMeta = lcgBGPConfiguration.TypeMeta
	aapiBGPConfiguration.ObjectMeta = lcgBGPConfiguration.ObjectMeta
}

func (gc BGPConfigurationConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	lcgBGPConfigurationList := libcalicoListObject.(*v3.BGPConfigurationList)
	aapiBGPConfigurationList := aapiListObj.(*v3.BGPConfigurationList)
	if libcalicoListObject == nil {
		aapiBGPConfigurationList.Items = []v3.BGPConfiguration{}
		return
	}
	aapiBGPConfigurationList.TypeMeta = lcgBGPConfigurationList.TypeMeta
	aapiBGPConfigurationList.ListMeta = lcgBGPConfigurationList.ListMeta
	for _, item := range lcgBGPConfigurationList.Items {
		aapiBGPConfiguration := v3.BGPConfiguration{}
		gc.convertToAAPI(&item, &aapiBGPConfiguration)
		if matched, err := pred.Matches(&aapiBGPConfiguration); err == nil && matched {
			aapiBGPConfigurationList.Items = append(aapiBGPConfigurationList.Items, aapiBGPConfiguration)
		}
	}
}
