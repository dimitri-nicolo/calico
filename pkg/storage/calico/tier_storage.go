// Copyright (c) 2017-2019 Tigera, Inc. All rights reserved.

package calico

import (
	"reflect"

	"golang.org/x/net/context"

	licClient "github.com/tigera/licensing/client"
	"github.com/tigera/licensing/client/features"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/storage"
	etcd "k8s.io/apiserver/pkg/storage/etcd3"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"

	aapi "github.com/projectcalico/apiserver/pkg/apis/projectcalico"

	libcalicoapi "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/watch"
)

// NewTierStorage creates a new libcalico-based storage.Interface implementation for Tiers
func NewTierStorage(opts Options) (registry.DryRunnableStorage, factory.DestroyFunc) {
	c := CreateClientFromConfig()
	createFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.Tier)
		return c.Tiers().Create(ctx, res, oso)
	}
	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.Tier)
		return c.Tiers().Update(ctx, res, oso)
	}
	getFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.Tiers().Get(ctx, name, ogo)
	}
	deleteFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		odo := opts.(options.DeleteOptions)
		return c.Tiers().Delete(ctx, name, odo)
	}
	listFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (resourceListObject, error) {
		olo := opts.(options.ListOptions)
		return c.Tiers().List(ctx, olo)
	}
	watchFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (watch.Interface, error) {
		olo := opts.(options.ListOptions)
		return c.Tiers().Watch(ctx, olo)
	}
	hasRestrictionsFn := func(obj resourceObject, claims *licClient.LicenseClaims) bool {
		return !claims.ValidateFeature(features.Tiers)
	}

	// TODO(doublek): Inject codec, client for nicer testing.
	dryRunnableStorage := registry.DryRunnableStorage{Storage: &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         etcd.APIObjectVersioner{},
		aapiType:          reflect.TypeOf(aapi.Tier{}),
		aapiListType:      reflect.TypeOf(aapi.TierList{}),
		libCalicoType:     reflect.TypeOf(libcalicoapi.Tier{}),
		libCalicoListType: reflect.TypeOf(libcalicoapi.TierList{}),
		isNamespaced:      false,
		create:            createFn,
		update:            updateFn,
		get:               getFn,
		delete:            deleteFn,
		list:              listFn,
		watch:             watchFn,
		resourceName:      "Tier",
		converter:         TierConverter{},
		licenseCache:      opts.LicenseCache,
		hasRestrictions:   hasRestrictionsFn,
	}, Codec: opts.RESTOptions.StorageConfig.Codec}
	return dryRunnableStorage, func() {}
}

type TierConverter struct {
}

func (tc TierConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiTier := aapiObj.(*aapi.Tier)
	lcgTier := &libcalicoapi.Tier{}
	lcgTier.TypeMeta = aapiTier.TypeMeta
	lcgTier.ObjectMeta = aapiTier.ObjectMeta
	lcgTier.Kind = libcalicoapi.KindTier
	lcgTier.APIVersion = libcalicoapi.GroupVersionCurrent
	lcgTier.Spec = aapiTier.Spec
	return lcgTier
}

func (tc TierConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgTier := libcalicoObject.(*libcalicoapi.Tier)
	aapiTier := aapiObj.(*aapi.Tier)
	aapiTier.Spec = lcgTier.Spec
	aapiTier.TypeMeta = lcgTier.TypeMeta
	aapiTier.ObjectMeta = lcgTier.ObjectMeta
}

func (tc TierConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	lcgTierList := libcalicoListObject.(*libcalicoapi.TierList)
	aapiTierList := aapiListObj.(*aapi.TierList)
	if libcalicoListObject == nil {
		aapiTierList.Items = []aapi.Tier{}
		return
	}
	aapiTierList.TypeMeta = lcgTierList.TypeMeta
	aapiTierList.ListMeta = lcgTierList.ListMeta
	for _, item := range lcgTierList.Items {
		aapiTier := aapi.Tier{}
		tc.convertToAAPI(&item, &aapiTier)
		if matched, err := pred.Matches(&aapiTier); err == nil && matched {
			aapiTierList.Items = append(aapiTierList.Items, aapiTier)
		}
	}
}
