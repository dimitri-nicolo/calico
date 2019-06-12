// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package calico

import (
	"reflect"

	"golang.org/x/net/context"

	libcalicoapi "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/watch"
	aapi "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/etcd"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"
)

// NewBGPConfigurationStorage creates a new libcalico-based storage.Interface implementation for BGPConfigurations
func NewBGPConfigurationStorage(opts Options) (storage.Interface, factory.DestroyFunc) {
	c := createClientFromConfig()
	createFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.BGPConfiguration)
		return c.BGPConfigurations().Create(ctx, res, oso)
	}
	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.BGPConfiguration)
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
	return &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         etcd.APIObjectVersioner{},
		aapiType:          reflect.TypeOf(aapi.BGPConfiguration{}),
		aapiListType:      reflect.TypeOf(aapi.BGPConfigurationList{}),
		libCalicoType:     reflect.TypeOf(libcalicoapi.BGPConfiguration{}),
		libCalicoListType: reflect.TypeOf(libcalicoapi.BGPConfigurationList{}),
		isNamespaced:      false,
		create:            createFn,
		update:            updateFn,
		get:               getFn,
		delete:            deleteFn,
		list:              listFn,
		watch:             watchFn,
		resourceName:      "BGPConfiguration",
		converter:         BGPConfigurationConverter{},
	}, func() {}
}

type BGPConfigurationConverter struct {
}

func (gc BGPConfigurationConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiBGPConfiguration := aapiObj.(*aapi.BGPConfiguration)
	lcgBGPConfiguration := &libcalicoapi.BGPConfiguration{}
	lcgBGPConfiguration.TypeMeta = aapiBGPConfiguration.TypeMeta
	lcgBGPConfiguration.ObjectMeta = aapiBGPConfiguration.ObjectMeta
	lcgBGPConfiguration.Spec = aapiBGPConfiguration.Spec
	return lcgBGPConfiguration
}

func (gc BGPConfigurationConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgBGPConfiguration := libcalicoObject.(*libcalicoapi.BGPConfiguration)
	aapiBGPConfiguration := aapiObj.(*aapi.BGPConfiguration)
	aapiBGPConfiguration.Spec = lcgBGPConfiguration.Spec
	aapiBGPConfiguration.TypeMeta = lcgBGPConfiguration.TypeMeta
	aapiBGPConfiguration.ObjectMeta = lcgBGPConfiguration.ObjectMeta
}

func (gc BGPConfigurationConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	lcgBGPConfigurationList := libcalicoListObject.(*libcalicoapi.BGPConfigurationList)
	aapiBGPConfigurationList := aapiListObj.(*aapi.BGPConfigurationList)
	if libcalicoListObject == nil {
		aapiBGPConfigurationList.Items = []aapi.BGPConfiguration{}
		return
	}
	aapiBGPConfigurationList.TypeMeta = lcgBGPConfigurationList.TypeMeta
	aapiBGPConfigurationList.ListMeta = lcgBGPConfigurationList.ListMeta
	for _, item := range lcgBGPConfigurationList.Items {
		aapiBGPConfiguration := aapi.BGPConfiguration{}
		gc.convertToAAPI(&item, &aapiBGPConfiguration)
		if matched, err := pred.Matches(&aapiBGPConfiguration); err == nil && matched {
			aapiBGPConfigurationList.Items = append(aapiBGPConfigurationList.Items, aapiBGPConfiguration)
		}
	}
}
