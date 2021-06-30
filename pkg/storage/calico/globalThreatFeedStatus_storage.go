// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package calico

import (
	"reflect"

	"golang.org/x/net/context"
	"k8s.io/apiserver/pkg/registry/generic/registry"
	etcd "k8s.io/apiserver/pkg/storage/etcd3"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"

	licClient "github.com/tigera/licensing/client"
	"github.com/tigera/licensing/client/features"

	libcalicoapi "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"

	aapi "github.com/projectcalico/apiserver/pkg/apis/projectcalico"
)

// NewGlobalThreatFeedStatusStorage creates a new libcalico-based storage.Interface implementation for GlobalThreatFeedsStatus
func NewGlobalThreatFeedStatusStorage(opts Options) (registry.DryRunnableStorage, factory.DestroyFunc) {
	c := CreateClientFromConfig()
	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.GlobalThreatFeed)
		return c.GlobalThreatFeeds().UpdateStatus(ctx, res, oso)
	}
	getFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.GlobalThreatFeeds().Get(ctx, name, ogo)
	}
	hasRestrictionsFn := func(obj resourceObject, claims *licClient.LicenseClaims) bool {
		return !claims.ValidateFeature(features.ThreatDefense)
	}
	// TODO(doublek): Inject codec, client for nicer testing.
	dryRunnableStorage := registry.DryRunnableStorage{Storage: &resourceStore{
		client:          c,
		codec:           opts.RESTOptions.StorageConfig.Codec,
		versioner:       etcd.APIObjectVersioner{},
		aapiType:        reflect.TypeOf(aapi.GlobalThreatFeed{}),
		isNamespaced:    false,
		update:          updateFn,
		get:             getFn,
		resourceName:    "GlobalThreatFeedStatus",
		converter:       GlobalThreatFeedConverter{},
		licenseCache:    opts.LicenseCache,
		hasRestrictions: hasRestrictionsFn,
	}, Codec: opts.RESTOptions.StorageConfig.Codec}
	return dryRunnableStorage, func() {}
}
