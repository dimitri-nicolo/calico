// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package xrefcache

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/projectcalico/libcalico-go/lib/backend/k8s/conversion"

	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
)

var (
	KindsEndpoints = []schema.GroupVersionKind{
		resources.ResourceTypeEndpoints,
	}
)

type CacheEntryK8sEndpoints struct {
	// The versioned policy resource.
	VersionedResource

	// --- Internal data ---
	cacheEntryCommon
}

type AugmentedK8sEndpointsData struct {
	// Whether this endpoints exposes ingress or egress to another endpoints. This is calculated directly from the rule
	// configuration and therefore can be calculated outside any cross-reference processing.
	IngressFromOtherEndpoints bool
	EgressToOtherEndpoints    bool

	// Whether this endpoints has ingress or egress protection.
	IngressProtected bool
	EgressProtected  bool

	// Whether this endpoints exposes ingress or egress to the internet.
	IngressFromInternet bool
	EgressToInternet    bool
}

func (c *CacheEntryK8sEndpoints) getVersionedResource() VersionedResource {
	return c.VersionedResource
}

func (c *CacheEntryK8sEndpoints) setVersionedResource(r VersionedResource) {
	c.VersionedResource = r
}

type versionedK8sEndpoints struct {
	*corev1.Endpoints
}

func (v *versionedK8sEndpoints) getV3() resources.Resource {
	return nil
}

func (v *versionedK8sEndpoints) getV1() interface{} {
	return nil
}

func newK8sEndpointsEngine() resourceCacheEngine {
	return &k8sEndpointsEngine{}
}

type k8sEndpointsEngine struct {
	engineCache
	converter conversion.Converter
}

func (c *k8sEndpointsEngine) register(cache engineCache) {
	c.engineCache = cache
}

func (c *k8sEndpointsEngine) kinds() []schema.GroupVersionKind {
	return KindsEndpoints
}

func (c *k8sEndpointsEngine) newCacheEntry() CacheEntry {
	return &CacheEntryK8sEndpoints{}
}

func (c *k8sEndpointsEngine) convertToVersioned(res resources.Resource) (VersionedResource, error) {
	in := res.(*corev1.Endpoints)
	return &versionedK8sEndpoints{Endpoints: in}, nil
}

func (c *k8sEndpointsEngine) resourceAdded(id resources.ResourceID, entry CacheEntry) {
	c.resourceUpdated(id, entry, nil)
}

func (c *k8sEndpointsEngine) resourceUpdated(id resources.ResourceID, entry CacheEntry, prev VersionedResource) syncer.UpdateType {
	return 0
}

func (c *k8sEndpointsEngine) resourceDeleted(id resources.ResourceID, entry CacheEntry) {
}

// recalculate implements the resourceCacheEngine interface.
func (c *k8sEndpointsEngine) recalculate(podId resources.ResourceID, podEntry CacheEntry) syncer.UpdateType {
	// We calculate all state in the resourceUpdated/resourceAdded callbacks.
	return 0
}
