// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package xrefcache

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/k8s/conversion"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/backend/syncersv1/updateprocessors"

	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
)

var (
	KindsServiceAccount = []schema.GroupVersionKind{
		resources.ResourceTypeServiceAccounts,
	}
)

// VersionedServiceAccountResource is an extension of the VersionedResource interface, specific to handling ServiceAccounts.
type VersionedServiceAccountResource interface {
	VersionedResource
	getV1Profile() *model.Profile
	getV3Profile() *apiv3.Profile
}

// CacheEntryK8sServiceAccount implements the CacheEntry interface, and is what we stored in the ServiceAccounts cache.
type CacheEntryK8sServiceAccount struct {
	// The versioned policy resource.
	VersionedServiceAccountResource

	// --- Internal data ---
	cacheEntryCommon
}

// getVersionedResource implements the CacheEntry interface.
func (c *CacheEntryK8sServiceAccount) getVersionedResource() VersionedResource {
	return c.VersionedServiceAccountResource
}

// setVersionedResource implements the CacheEntry interface.
func (c *CacheEntryK8sServiceAccount) setVersionedResource(r VersionedResource) {
	c.VersionedServiceAccountResource = r.(VersionedServiceAccountResource)
}

// versionedK8sServiceAccount implements the VersionedServiceAccountResource interface.
type versionedK8sServiceAccount struct {
	*corev1.ServiceAccount
	v3 *apiv3.Profile
	v1 *model.Profile
}

// getV3 implements the VersionedServiceAccountResource interface.
func (v *versionedK8sServiceAccount) getV3() resources.Resource {
	return v.v3
}

// getV1 implements the VersionedServiceAccountResource interface.
func (v *versionedK8sServiceAccount) getV1() interface{} {
	return v.v1
}

// getV1Profile implements the VersionedServiceAccountResource interface.
func (v *versionedK8sServiceAccount) getV1Profile() *model.Profile {
	return v.v1
}

// getV3Profile implements the VersionedServiceAccountResource interface.
func (v *versionedK8sServiceAccount) getV3Profile() *apiv3.Profile {
	return v.v3
}

// newK8sServiceAccountsEngine creates a resourceCacheEngine used to handle the ServiceAccounts cache.
func newK8sServiceAccountsEngine() resourceCacheEngine {
	return &k8sServiceAccountEngine{}
}

// k8sServiceAccountEngine implements the resourceCacheEngine.
type k8sServiceAccountEngine struct {
	engineCache
	converter conversion.Converter
}

// register implements the resourceCacheEngine.
func (c *k8sServiceAccountEngine) register(cache engineCache) {
	c.engineCache = cache
}

// kinds implements the resourceCacheEngine.
func (c *k8sServiceAccountEngine) kinds() []schema.GroupVersionKind {
	return KindsServiceAccount
}

// newCacheEntry implements the resourceCacheEngine.
func (c *k8sServiceAccountEngine) newCacheEntry() CacheEntry {
	return &CacheEntryK8sServiceAccount{}
}

// convertToVersioned implements the resourceCacheEngine.
func (c *k8sServiceAccountEngine) convertToVersioned(res resources.Resource) (VersionedResource, error) {
	in := res.(*corev1.ServiceAccount)

	kvp, err := c.converter.ServiceAccountToProfile(in)
	if err != nil {
		return nil, err
	}

	v3 := kvp.Value.(*apiv3.Profile)
	v1, err := updateprocessors.ConvertProfileV3ToV1Value(v3)
	if err != nil {
		return nil, err
	}

	return &versionedK8sServiceAccount{
		ServiceAccount: in,
		v3:             v3,
		v1:             v1,
	}, nil
}

// resourceAdded implements the resourceCacheEngine.
func (c *k8sServiceAccountEngine) resourceAdded(id resources.ResourceID, entry CacheEntry) {
	_ = c.resourceUpdated(id, entry, nil)
}

// resourceUpdated implements the resourceCacheEngine.
func (c *k8sServiceAccountEngine) resourceUpdated(id resources.ResourceID, entry CacheEntry, prev VersionedResource) syncer.UpdateType {
	// Kubernetes namespaces are configured as Calico profiles. Use the V3 version of the name and the V1 version of the
	// labels since they will have been modified to match the selector modifications in the pod.
	x := entry.(*CacheEntryK8sServiceAccount)
	c.EndpointLabelSelector().UpdateParentLabels(x.getV3Profile().Name, x.getV1Profile().Labels)
	return 0
}

// resourceDeleted implements the resourceCacheEngine.
func (c *k8sServiceAccountEngine) resourceDeleted(id resources.ResourceID, entry CacheEntry) {
	// Kubernetes namespaces are configured as Calico profiles. Use the V3 version of the name since it will have been
	// modified to match the selector modifications in the pod.
	x := entry.(*CacheEntryK8sServiceAccount)
	c.EndpointLabelSelector().DeleteParentLabels(x.getV3Profile().Name)
}

// recalculate implements the resourceCacheEngine interface.
func (c *k8sServiceAccountEngine) recalculate(id resources.ResourceID, entry CacheEntry) syncer.UpdateType {
	// We don't store any additional ServiceAccount state at the moment.
	return 0
}
