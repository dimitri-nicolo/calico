// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package xrefcache

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/k8s/conversion"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/backend/syncersv1/updateprocessors"

	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
)

var (
	KindsNamespace = []metav1.TypeMeta{
		resources.TypeK8sNamespaces,
	}
)

// VersionedNamespaceResource is an extension of the VersionedResource interface, specific to handling Namespaces.
type VersionedNamespaceResource interface {
	VersionedResource
	getV1Profile() *model.Profile
	getV3Profile() *apiv3.Profile
}

// CacheEntryK8sNamespace implements the CacheEntry interface, and is what we stored in the Namespaces cache.
type CacheEntryK8sNamespace struct {
	// The versioned policy resource.
	VersionedNamespaceResource

	// --- Internal data ---
	cacheEntryCommon
}

// getVersionedResource implements the CacheEntry interface.
func (c *CacheEntryK8sNamespace) getVersionedResource() VersionedResource {
	return c.VersionedNamespaceResource
}

// setVersionedResource implements the CacheEntry interface.
func (c *CacheEntryK8sNamespace) setVersionedResource(r VersionedResource) {
	c.VersionedNamespaceResource = r.(VersionedNamespaceResource)
}

// versionedK8sNamespace implements the VersionedNamespaceResource interface.
type versionedK8sNamespace struct {
	*corev1.Namespace
	v3 *apiv3.Profile
	v1 *model.Profile
}

// getV3 implements the VersionedNamespaceResource interface.
func (v *versionedK8sNamespace) getV3() resources.Resource {
	return v.v3
}

// getV1 implements the VersionedNamespaceResource interface.
func (v *versionedK8sNamespace) getV1() interface{} {
	return v.v1
}

// getV1Profile implements the VersionedNamespaceResource interface.
func (v *versionedK8sNamespace) getV1Profile() *model.Profile {
	return v.v1
}

// getV3Profile implements the VersionedNamespaceResource interface.
func (v *versionedK8sNamespace) getV3Profile() *apiv3.Profile {
	return v.v3
}

// newK8sNamespacesEngine creates a resourceCacheEngine used to handle the Namespaces cache.
func newK8sNamespacesEngine() resourceCacheEngine {
	return &k8sNamespaceEngine{}
}

// k8sNamespaceEngine implements the resourceCacheEngine.
type k8sNamespaceEngine struct {
	engineCache
	converter conversion.Converter
}

// kinds implements the resourceCacheEngine interface.
func (c *k8sNamespaceEngine) kinds() []metav1.TypeMeta {
	return KindsNamespace
}

// register implements the resourceCacheEngine interface.
func (c *k8sNamespaceEngine) register(cache engineCache) {
	c.engineCache = cache
}

// newCacheEntry implements the resourceCacheEngine interface.
func (c *k8sNamespaceEngine) newCacheEntry() CacheEntry {
	return &CacheEntryK8sNamespace{}
}

// convertToVersioned implements the resourceCacheEngine interface.
func (c *k8sNamespaceEngine) convertToVersioned(res resources.Resource) (VersionedResource, error) {
	in := res.(*corev1.Namespace)

	kvp, err := c.converter.NamespaceToProfile(in)
	if err != nil {
		return nil, err
	}

	v3 := kvp.Value.(*apiv3.Profile)
	v1, err := updateprocessors.ConvertProfileV3ToV1Value(v3)
	if err != nil {
		return nil, err
	}

	return &versionedK8sNamespace{
		Namespace: in,
		v3:        v3,
		v1:        v1,
	}, nil
}

// resourceAdded implements the resourceCacheEngine interface.
func (c *k8sNamespaceEngine) resourceAdded(id apiv3.ResourceID, entry CacheEntry) {
	c.resourceUpdated(id, entry, nil)
}

// resourceUpdated implements the resourceCacheEngine interface.
func (c *k8sNamespaceEngine) resourceUpdated(id apiv3.ResourceID, entry CacheEntry, prev VersionedResource) {
	// Kubernetes namespaces are configured as Calico profiles. Use the V3 version of the name and the V1 version of the
	// labels since they will have been modified to match the selector modifications in the pod.
	x := entry.(*CacheEntryK8sNamespace)
	c.EndpointLabelSelector().UpdateParentLabels(x.getV3Profile().Name, x.getV1Profile().Labels)
}

// resourceDeleted implements the resourceCacheEngine interface.
func (c *k8sNamespaceEngine) resourceDeleted(id apiv3.ResourceID, entry CacheEntry) {
	// Kubernetes namespaces are configured as Calico profiles. Use the V3 version of the name since it will have been
	// modified to match the selector modifications in the pod.
	x := entry.(*CacheEntryK8sNamespace)
	c.EndpointLabelSelector().DeleteParentLabels(x.getV3Profile().Name)
}

// recalculate implements the resourceCacheEngine interface.
func (c *k8sNamespaceEngine) recalculate(id apiv3.ResourceID, res CacheEntry) syncer.UpdateType {
	// We don't store any additional Namespace state at the moment.
	return 0
}
