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
	"github.com/tigera/licensing/client/features"

	aapi "github.com/projectcalico/apiserver/pkg/apis/projectcalico"
)

// NewRemoteClusterConfigurationStorage creates a new libcalico-based storage.Interface implementation for RemoteClusterConfigurations
func NewRemoteClusterConfigurationStorage(opts Options) (registry.DryRunnableStorage, factory.DestroyFunc) {
	c := CreateClientFromConfig()
	createFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.RemoteClusterConfiguration)
		return c.RemoteClusterConfigurations().Create(ctx, res, oso)
	}
	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.RemoteClusterConfiguration)
		return c.RemoteClusterConfigurations().Update(ctx, res, oso)
	}
	getFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.RemoteClusterConfigurations().Get(ctx, name, ogo)
	}
	deleteFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		odo := opts.(options.DeleteOptions)
		return c.RemoteClusterConfigurations().Delete(ctx, name, odo)
	}
	listFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (resourceListObject, error) {
		olo := opts.(options.ListOptions)
		return c.RemoteClusterConfigurations().List(ctx, olo)
	}
	watchFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (watch.Interface, error) {
		olo := opts.(options.ListOptions)
		return c.RemoteClusterConfigurations().Watch(ctx, olo)
	}
	hasRestrictionsFn := func(obj resourceObject, claims *licClient.LicenseClaims) bool {
		return !claims.ValidateFeature(features.FederatedServices)
	}

	dryRunnableStorage := registry.DryRunnableStorage{Storage: &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         etcd.APIObjectVersioner{},
		aapiType:          reflect.TypeOf(aapi.RemoteClusterConfiguration{}),
		aapiListType:      reflect.TypeOf(aapi.RemoteClusterConfigurationList{}),
		libCalicoType:     reflect.TypeOf(libcalicoapi.RemoteClusterConfiguration{}),
		libCalicoListType: reflect.TypeOf(libcalicoapi.RemoteClusterConfigurationList{}),
		isNamespaced:      false,
		create:            createFn,
		update:            updateFn,
		get:               getFn,
		delete:            deleteFn,
		list:              listFn,
		watch:             watchFn,
		resourceName:      "RemoteClusterConfiguration",
		converter:         RemoteClusterConfigurationConverter{},
		licenseCache:      opts.LicenseCache,
		hasRestrictions:   hasRestrictionsFn,
	}, Codec: opts.RESTOptions.StorageConfig.Codec}
	return dryRunnableStorage, func() {}
}

type RemoteClusterConfigurationConverter struct {
}

func (gc RemoteClusterConfigurationConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiConfig := aapiObj.(*aapi.RemoteClusterConfiguration)
	lcgConfig := &libcalicoapi.RemoteClusterConfiguration{}
	lcgConfig.TypeMeta = aapiConfig.TypeMeta
	lcgConfig.ObjectMeta = aapiConfig.ObjectMeta
	lcgConfig.Kind = libcalicoapi.KindRemoteClusterConfiguration
	lcgConfig.APIVersion = libcalicoapi.GroupVersionCurrent
	lcgConfig.Spec = aapiConfig.Spec
	return lcgConfig
}

func (gc RemoteClusterConfigurationConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgConfig := libcalicoObject.(*libcalicoapi.RemoteClusterConfiguration)
	aapiConfig := aapiObj.(*aapi.RemoteClusterConfiguration)
	aapiConfig.Spec = lcgConfig.Spec
	aapiConfig.TypeMeta = lcgConfig.TypeMeta
	aapiConfig.ObjectMeta = lcgConfig.ObjectMeta
}

func (gc RemoteClusterConfigurationConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	lcgConfigList := libcalicoListObject.(*libcalicoapi.RemoteClusterConfigurationList)
	aapiConfigList := aapiListObj.(*aapi.RemoteClusterConfigurationList)
	if libcalicoListObject == nil {
		aapiConfigList.Items = []aapi.RemoteClusterConfiguration{}
		return
	}
	aapiConfigList.TypeMeta = lcgConfigList.TypeMeta
	aapiConfigList.ListMeta = lcgConfigList.ListMeta
	for _, item := range lcgConfigList.Items {
		aapiConfig := aapi.RemoteClusterConfiguration{}
		gc.convertToAAPI(&item, &aapiConfig)
		if matched, err := pred.Matches(&aapiConfig); err == nil && matched {
			aapiConfigList.Items = append(aapiConfigList.Items, aapiConfig)
		}
	}
}
