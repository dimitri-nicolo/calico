// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package calico

import (
	"reflect"

	"golang.org/x/net/context"

	licClient "github.com/tigera/licensing/client"
	features "github.com/tigera/licensing/client/features"
	"k8s.io/apiserver/pkg/registry/generic/registry"
	etcd "k8s.io/apiserver/pkg/storage/etcd3"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"

	aapi "github.com/projectcalico/apiserver/pkg/apis/projectcalico"

	libcalicoapi "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
)

// NewDeepPacketInspectionStatusStorage creates a new libcalico-based storage.Interface implementation for DeepPacketInspections
func NewDeepPacketInspectionStatusStorage(opts Options) (registry.DryRunnableStorage, factory.DestroyFunc) {
	c := CreateClientFromConfig()
	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.DeepPacketInspection)
		return c.DeepPacketInspections().UpdateStatus(ctx, res, oso)
	}
	getFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.DeepPacketInspections().Get(ctx, ns, name, ogo)
	}
	hasRestrictionsFn := func(obj resourceObject, claims *licClient.LicenseClaims) bool {
		return !claims.ValidateFeature(features.ThreatDefense)
	}
	// TODO(doublek): Inject codec, client for nicer testing.
	dryRunnableStorage := registry.DryRunnableStorage{Storage: &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         etcd.APIObjectVersioner{},
		aapiType:          reflect.TypeOf(aapi.DeepPacketInspection{}),
		aapiListType:      reflect.TypeOf(aapi.DeepPacketInspectionList{}),
		libCalicoType:     reflect.TypeOf(libcalicoapi.DeepPacketInspection{}),
		libCalicoListType: reflect.TypeOf(libcalicoapi.DeepPacketInspectionList{}),
		isNamespaced:      true,
		update:            updateFn,
		get:               getFn,
		resourceName:      "DeepPacketInspectionStatus",
		converter:         DeepPacketInspectionConverter{},
		licenseCache:      opts.LicenseCache,
		hasRestrictions:   hasRestrictionsFn,
	}, Codec: opts.RESTOptions.StorageConfig.Codec}
	return dryRunnableStorage, func() {}
}
