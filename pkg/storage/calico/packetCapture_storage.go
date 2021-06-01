// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package calico

import (
	"reflect"

	"golang.org/x/net/context"

	licClient "github.com/tigera/licensing/client"
	features "github.com/tigera/licensing/client/features"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/storage"
	etcd "k8s.io/apiserver/pkg/storage/etcd3"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"

	aapi "github.com/projectcalico/apiserver/pkg/apis/projectcalico"

	libcalicoapi "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/watch"
)

// NewPacketCaptureStorage creates a new libcalico-based storage.Interface implementation for PacketCaptures
func NewPacketCaptureStorage(opts Options) (registry.DryRunnableStorage, factory.DestroyFunc) {
	c := CreateClientFromConfig()
	createFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.PacketCapture)
		return c.PacketCaptures().Create(ctx, res, oso)
	}
	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.PacketCapture)
		return c.PacketCaptures().Update(ctx, res, oso)
	}
	getFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.PacketCaptures().Get(ctx, ns, name, ogo)
	}
	deleteFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		odo := opts.(options.DeleteOptions)
		return c.PacketCaptures().Delete(ctx, ns, name, odo)
	}
	listFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (resourceListObject, error) {
		olo := opts.(options.ListOptions)
		return c.PacketCaptures().List(ctx, olo)
	}
	watchFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (watch.Interface, error) {
		olo := opts.(options.ListOptions)
		return c.PacketCaptures().Watch(ctx, olo)
	}
	hasRestrictionsFn := func(obj resourceObject, claims *licClient.LicenseClaims) bool {
		return !claims.ValidateFeature(features.PacketCapture)
	}
	// TODO(doublek): Inject codec, client for nicer testing.
	dryRunnableStorage := registry.DryRunnableStorage{Storage: &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         etcd.APIObjectVersioner{},
		aapiType:          reflect.TypeOf(aapi.PacketCapture{}),
		aapiListType:      reflect.TypeOf(aapi.PacketCaptureList{}),
		libCalicoType:     reflect.TypeOf(libcalicoapi.PacketCapture{}),
		libCalicoListType: reflect.TypeOf(libcalicoapi.PacketCaptureList{}),
		isNamespaced:      true,
		create:            createFn,
		update:            updateFn,
		get:               getFn,
		delete:            deleteFn,
		list:              listFn,
		watch:             watchFn,
		resourceName:      "PacketCapture",
		converter:         PacketCaptureConverter{},
		licenseCache:      opts.LicenseCache,
		hasRestrictions:   hasRestrictionsFn,
	}, Codec: opts.RESTOptions.StorageConfig.Codec}
	return dryRunnableStorage, func() {}
}

type PacketCaptureConverter struct {
}

func (gc PacketCaptureConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiPacketCapture := aapiObj.(*aapi.PacketCapture)
	lcgPacketCapture := &libcalicoapi.PacketCapture{}
	lcgPacketCapture.TypeMeta = aapiPacketCapture.TypeMeta
	lcgPacketCapture.ObjectMeta = aapiPacketCapture.ObjectMeta
	lcgPacketCapture.Kind = libcalicoapi.KindPacketCapture
	lcgPacketCapture.APIVersion = libcalicoapi.GroupVersionCurrent
	lcgPacketCapture.Spec = aapiPacketCapture.Spec
	lcgPacketCapture.Status = aapiPacketCapture.Status
	return lcgPacketCapture
}

func (gc PacketCaptureConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgPacketCapture := libcalicoObject.(*libcalicoapi.PacketCapture)
	aapiPacketCapture := aapiObj.(*aapi.PacketCapture)
	aapiPacketCapture.Spec = lcgPacketCapture.Spec
	aapiPacketCapture.Status = lcgPacketCapture.Status
	aapiPacketCapture.TypeMeta = lcgPacketCapture.TypeMeta
	aapiPacketCapture.ObjectMeta = lcgPacketCapture.ObjectMeta
}

func (gc PacketCaptureConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	lcgPacketCaptureList := libcalicoListObject.(*libcalicoapi.PacketCaptureList)
	aapiPacketCaptureList := aapiListObj.(*aapi.PacketCaptureList)
	if libcalicoListObject == nil {
		aapiPacketCaptureList.Items = []aapi.PacketCapture{}
		return
	}
	aapiPacketCaptureList.TypeMeta = lcgPacketCaptureList.TypeMeta
	aapiPacketCaptureList.ListMeta = lcgPacketCaptureList.ListMeta
	for _, item := range lcgPacketCaptureList.Items {
		aapiPacketCapture := aapi.PacketCapture{}
		gc.convertToAAPI(&item, &aapiPacketCapture)
		if matched, err := pred.Matches(&aapiPacketCapture); err == nil && matched {
			aapiPacketCaptureList.Items = append(aapiPacketCaptureList.Items, aapiPacketCapture)
		}
	}
}
