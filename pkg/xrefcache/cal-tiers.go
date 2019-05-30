// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package xrefcache

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/backend/syncersv1/updateprocessors"

	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
)

var (
	KindsTier = []metav1.TypeMeta{
		resources.TypeCalicoTiers,
	}
)

type VersionedTierResource interface {
	VersionedResource
	getV1Tier() *model.Tier
}

type CacheEntryTier struct {
	// The versioned network set resource.
	VersionedTierResource

	// Augmented policy data.
	AugmentedTierData

	// --- Internal data ---
	cacheEntryCommon
}

func (c *CacheEntryTier) getVersionedResource() VersionedResource {
	return c.VersionedTierResource
}

func (c *CacheEntryTier) setVersionedResource(r VersionedResource) {
	c.VersionedTierResource = r.(VersionedTierResource)
}

type AugmentedTierData struct {
}

type versionedCalicoTier struct {
	*apiv3.Tier
	v1 *model.Tier
}

func (v *versionedCalicoTier) getV3() resources.Resource {
	return v.Tier
}

func (v *versionedCalicoTier) getV1() interface{} {
	return v.v1
}

func (v *versionedCalicoTier) getV1Tier() *model.Tier {
	return v.v1
}

func newTierHandler() resourceHandler {
	return &tierHandler{}
}

type tierHandler struct {
	CacheAccessor
}

func (c *tierHandler) register(cache CacheAccessor) {
	c.CacheAccessor = cache
}

func (c *tierHandler) kinds() []metav1.TypeMeta {
	return KindsTier
}

func (c *tierHandler) newCacheEntry() CacheEntry {
	return &CacheEntryTier{}
}

func (c *tierHandler) resourceAdded(id apiv3.ResourceID, entry CacheEntry) {
	c.resourceUpdated(id, entry, nil)
}

func (c *tierHandler) resourceUpdated(id apiv3.ResourceID, entry CacheEntry, prev VersionedResource) {
}

func (c *tierHandler) resourceDeleted(id apiv3.ResourceID, _ CacheEntry) {
}

// recalculate implements the resourceHandler interface.
func (c *tierHandler) recalculate(podId apiv3.ResourceID, podEntry CacheEntry) syncer.UpdateType {
	// We calculate all state in the resourceUpdated/resourceAdded callbacks.
	return 0
}

func (c *tierHandler) convertToVersioned(res resources.Resource) (VersionedResource, error) {
	in := res.(*apiv3.Tier)

	v1, err := updateprocessors.ConvertTierV3ToV1Value(in)
	if err != nil {
		return nil, err
	}

	return &versionedCalicoTier{
		Tier: in,
		v1:   v1.(*model.Tier),
	}, nil
}
