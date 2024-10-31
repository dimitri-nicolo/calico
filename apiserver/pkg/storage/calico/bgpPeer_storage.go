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

// NewBGPPeerStorage creates a new libcalico-based storage.Interface implementation for BGPPeers
func NewBGPPeerStorage(opts Options) (registry.DryRunnableStorage, factory.DestroyFunc) {
	c := CreateClientFromConfig()
	createFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*v3.BGPPeer)
		return c.BGPPeers().Create(ctx, res, oso)
	}
	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*v3.BGPPeer)
		return c.BGPPeers().Update(ctx, res, oso)
	}
	getFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.BGPPeers().Get(ctx, name, ogo)
	}
	deleteFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		odo := opts.(options.DeleteOptions)
		return c.BGPPeers().Delete(ctx, name, odo)
	}
	listFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (resourceListObject, error) {
		olo := opts.(options.ListOptions)
		return c.BGPPeers().List(ctx, olo)
	}
	watchFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (watch.Interface, error) {
		olo := opts.(options.ListOptions)
		return c.BGPPeers().Watch(ctx, olo)
	}
	hasRestrictionsFn := func(obj resourceObject) bool {
		return false
	}

	dryRunnableStorage := registry.DryRunnableStorage{Storage: &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         APIObjectVersioner{},
		aapiType:          reflect.TypeOf(v3.BGPPeer{}),
		aapiListType:      reflect.TypeOf(v3.BGPPeerList{}),
		libCalicoType:     reflect.TypeOf(v3.BGPPeer{}),
		libCalicoListType: reflect.TypeOf(v3.BGPPeerList{}),
		isNamespaced:      false,
		create:            createFn,
		update:            updateFn,
		get:               getFn,
		delete:            deleteFn,
		list:              listFn,
		watch:             watchFn,
		resourceName:      "BGPPeer",
		converter:         BGPPeerConverter{},
		hasRestrictions:   hasRestrictionsFn,
	}, Codec: opts.RESTOptions.StorageConfig.Codec}
	return dryRunnableStorage, func() {}
}

type BGPPeerConverter struct {
}

func (gc BGPPeerConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiBGPPeer := aapiObj.(*v3.BGPPeer)
	lcgBGPPeer := &v3.BGPPeer{}
	lcgBGPPeer.TypeMeta = aapiBGPPeer.TypeMeta
	lcgBGPPeer.ObjectMeta = aapiBGPPeer.ObjectMeta
	lcgBGPPeer.Kind = v3.KindBGPPeer
	lcgBGPPeer.APIVersion = v3.GroupVersionCurrent
	lcgBGPPeer.Spec = aapiBGPPeer.Spec
	return lcgBGPPeer
}

func (gc BGPPeerConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgBGPPeer := libcalicoObject.(*v3.BGPPeer)
	aapiBGPPeer := aapiObj.(*v3.BGPPeer)
	aapiBGPPeer.Spec = lcgBGPPeer.Spec
	aapiBGPPeer.TypeMeta = lcgBGPPeer.TypeMeta
	aapiBGPPeer.ObjectMeta = lcgBGPPeer.ObjectMeta
}

func (gc BGPPeerConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	lcgBGPPeerList := libcalicoListObject.(*v3.BGPPeerList)
	aapiBGPPeerList := aapiListObj.(*v3.BGPPeerList)
	if libcalicoListObject == nil {
		aapiBGPPeerList.Items = []v3.BGPPeer{}
		return
	}
	aapiBGPPeerList.TypeMeta = lcgBGPPeerList.TypeMeta
	aapiBGPPeerList.ListMeta = lcgBGPPeerList.ListMeta
	for _, item := range lcgBGPPeerList.Items {
		aapiBGPPeer := v3.BGPPeer{}
		gc.convertToAAPI(&item, &aapiBGPPeer)
		if matched, err := pred.Matches(&aapiBGPPeer); err == nil && matched {
			aapiBGPPeerList.Items = append(aapiBGPPeerList.Items, aapiBGPPeer)
		}
	}
}
