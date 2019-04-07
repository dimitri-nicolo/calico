// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package xrefcache

import (
	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/projectcalico/libcalico-go/lib/backend/k8s/conversion"
	"github.com/projectcalico/libcalico-go/lib/set"

	"github.com/tigera/compliance/pkg/ips"
	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
)

var (
	KindsServiceEndpoints = []schema.GroupVersionKind{
		resources.ResourceTypeEndpoints,
	}
)

// VersionedServiceEndpointsResource is an extension of the VersionedResource interface, specific to handling service
// endpoints.
type VersionedServiceEndpointsResource interface {
	VersionedResource
	getIPs() (set.Set, error)
}

type CacheEntryK8sServiceEndpoints struct {
	// The versioned policy resource.
	VersionedServiceEndpointsResource

	// --- Internal data ---
	cacheEntryCommon

	//TODO(rlb): Might as well include the clog in the cacheEntryCommon thing.
	clog *log.Entry
}

type AugmentedK8sServiceEndpointsData struct {
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

func (c *CacheEntryK8sServiceEndpoints) getVersionedResource() VersionedResource {
	return c.VersionedServiceEndpointsResource
}

func (c *CacheEntryK8sServiceEndpoints) setVersionedResource(r VersionedResource) {
	c.VersionedServiceEndpointsResource = r.(VersionedServiceEndpointsResource)
}

type versionedK8sServiceEndpoints struct {
	*corev1.Endpoints
}

func (v *versionedK8sServiceEndpoints) getV3() resources.Resource {
	return nil
}

func (v *versionedK8sServiceEndpoints) getV1() interface{} {
	return nil
}

func (v *versionedK8sServiceEndpoints) getIPs() (set.Set, error) {
	var lastErr error
	s := set.New()
	for ssIdx := range v.Endpoints.Subsets {
		for addrIdx := range v.Endpoints.Subsets[ssIdx].Addresses {
			ip, err := ips.NormalizeIP(v.Endpoints.Subsets[ssIdx].Addresses[addrIdx].IP)
			if err != nil {
				lastErr = err
				continue
			}
			s.Add(ip)
		}
	}
	return s, lastErr
}

func newK8sServiceEndpointsEngine() resourceCacheEngine {
	return &K8sServiceEndpointsEngine{}
}

type K8sServiceEndpointsEngine struct {
	engineCache
	converter conversion.Converter
}

func (c *K8sServiceEndpointsEngine) register(cache engineCache) {
	c.engineCache = cache
}

func (c *K8sServiceEndpointsEngine) kinds() []schema.GroupVersionKind {
	return KindsServiceEndpoints
}

func (c *K8sServiceEndpointsEngine) newCacheEntry() CacheEntry {
	return &CacheEntryK8sServiceEndpoints{}
}

func (c *K8sServiceEndpointsEngine) convertToVersioned(res resources.Resource) (VersionedResource, error) {
	in := res.(*corev1.Endpoints)
	return &versionedK8sServiceEndpoints{Endpoints: in}, nil
}

func (c *K8sServiceEndpointsEngine) resourceAdded(id resources.ResourceID, entry CacheEntry) {
	x := entry.(*CacheEntryK8sServiceEndpoints)
	x.clog = log.WithField("id", id)
	c.resourceUpdated(id, entry, nil)
}

func (c *K8sServiceEndpointsEngine) resourceUpdated(id resources.ResourceID, entry CacheEntry, prev VersionedResource) {
	x := entry.(*CacheEntryK8sServiceEndpoints)
	i, err := x.getIPs()
	if err != nil {
		x.clog.Info("Unable to determine IP addresses")
	}
	c.IPManager().SetClientKeys(id, i)
}

func (c *K8sServiceEndpointsEngine) resourceDeleted(id resources.ResourceID, entry CacheEntry) {
	c.IPManager().DeleteClient(id)
}

// recalculate implements the resourceCacheEngine interface.
func (c *K8sServiceEndpointsEngine) recalculate(podId resources.ResourceID, podEntry CacheEntry) syncer.UpdateType {
	// We calculate all state in the resourceUpdated/resourceAdded callbacks.
	return 0
}
