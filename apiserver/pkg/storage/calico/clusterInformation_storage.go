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

// NewClusterInformationStorage creates a new libcalico-based storage.Interface implementation for ClusterInformation
func NewClusterInformationStorage(opts Options) (registry.DryRunnableStorage, factory.DestroyFunc) {
	c := CreateClientFromConfig()
	getFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.ClusterInformation().Get(ctx, name, ogo)
	}
	listFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (resourceListObject, error) {
		olo := opts.(options.ListOptions)
		return c.ClusterInformation().List(ctx, olo)
	}
	watchFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (watch.Interface, error) {
		olo := opts.(options.ListOptions)
		return c.ClusterInformation().Watch(ctx, olo)
	}
	hasRestrictionsFn := func(obj resourceObject) bool {
		return false
	}

	dryRunnableStorage := registry.DryRunnableStorage{Storage: &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         APIObjectVersioner{},
		aapiType:          reflect.TypeOf(v3.ClusterInformation{}),
		aapiListType:      reflect.TypeOf(v3.ClusterInformationList{}),
		libCalicoType:     reflect.TypeOf(v3.ClusterInformation{}),
		libCalicoListType: reflect.TypeOf(v3.ClusterInformationList{}),
		isNamespaced:      false,
		get:               getFn,
		list:              listFn,
		watch:             watchFn,
		resourceName:      "ClusterInformation",
		converter:         ClusterInformationConverter{},
		hasRestrictions:   hasRestrictionsFn,
	}, Codec: opts.RESTOptions.StorageConfig.Codec}
	return dryRunnableStorage, func() {}
}

type ClusterInformationConverter struct {
}

func (gc ClusterInformationConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiClusterInformation := aapiObj.(*v3.ClusterInformation)
	lcgClusterInformation := &v3.ClusterInformation{}
	lcgClusterInformation.TypeMeta = aapiClusterInformation.TypeMeta
	lcgClusterInformation.ObjectMeta = aapiClusterInformation.ObjectMeta
	lcgClusterInformation.Kind = v3.KindClusterInformation
	lcgClusterInformation.APIVersion = v3.GroupVersionCurrent
	lcgClusterInformation.Spec = aapiClusterInformation.Spec
	return lcgClusterInformation
}

func (gc ClusterInformationConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgClusterInformation := libcalicoObject.(*v3.ClusterInformation)
	aapiClusterInformation := aapiObj.(*v3.ClusterInformation)
	aapiClusterInformation.Spec = lcgClusterInformation.Spec
	aapiClusterInformation.TypeMeta = lcgClusterInformation.TypeMeta
	aapiClusterInformation.ObjectMeta = lcgClusterInformation.ObjectMeta
}

func (gc ClusterInformationConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	lcgClusterInformationList := libcalicoListObject.(*v3.ClusterInformationList)
	aapiClusterInformationList := aapiListObj.(*v3.ClusterInformationList)
	if libcalicoListObject == nil {
		aapiClusterInformationList.Items = []v3.ClusterInformation{}
		return
	}
	aapiClusterInformationList.TypeMeta = lcgClusterInformationList.TypeMeta
	aapiClusterInformationList.ListMeta = lcgClusterInformationList.ListMeta
	for _, item := range lcgClusterInformationList.Items {
		aapiClusterInformation := v3.ClusterInformation{}
		gc.convertToAAPI(&item, &aapiClusterInformation)
		if matched, err := pred.Matches(&aapiClusterInformation); err == nil && matched {
			aapiClusterInformationList.Items = append(aapiClusterInformationList.Items, aapiClusterInformation)
		}
	}
}
