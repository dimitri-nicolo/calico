// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package xrefcache

import (
	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/k8s/conversion"
	"github.com/projectcalico/libcalico-go/lib/set"

	"github.com/tigera/compliance/pkg/ips"
	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
)

var (
	KindsServiceEndpoints = []metav1.TypeMeta{
		resources.TypeK8sEndpoints,
	}
	KindsServices = []metav1.TypeMeta{
		resources.TypeK8sServices,
	}
)

// VersionedServiceEndpointsResource is an extension of the VersionedResource interface, specific to handling service
// endpoints.
type VersionedServiceEndpointsResource interface {
	VersionedResource
	getIPAndEndpointIDs() (set.Set, error)
}

type CacheEntryK8sServiceEndpoints struct {
	// The versioned policy resource.
	VersionedServiceEndpointsResource

	// The corresponding Service for this Endpoints.
	Service apiv3.ResourceID

	// --- Internal data ---
	cacheEntryCommon

	//TODO(rlb): Might as well include the clog in the cacheEntryCommon thing.
	clog *log.Entry
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

func (v *versionedK8sServiceEndpoints) getIPAndEndpointIDs() (set.Set, error) {
	var lastErr error
	s := set.New()
	for ssIdx := range v.Endpoints.Subsets {
		for addrIdx := range v.Endpoints.Subsets[ssIdx].Addresses {
			if target := v.Endpoints.Subsets[ssIdx].Addresses[addrIdx].TargetRef; target != nil && target.Kind == "Pod" {
				pod := apiv3.ResourceID{
					TypeMeta:  resources.TypeK8sPods,
					Name:      target.Name,
					Namespace: target.Namespace,
				}.String()

				log.Debugf("Including %s in service endpoints: %s", pod, resources.GetResourceID(v.Endpoints))
				s.Add(pod)
			}

			ip, err := ips.NormalizeIP(v.Endpoints.Subsets[ssIdx].Addresses[addrIdx].IP)
			if err != nil {
				lastErr = err
				continue
			}

			log.Debugf("Including %s in service endpoints: %s", ip, resources.GetResourceID(v.Endpoints))
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

func (c *K8sServiceEndpointsEngine) kinds() []metav1.TypeMeta {
	return KindsServiceEndpoints
}

func (c *K8sServiceEndpointsEngine) newCacheEntry() CacheEntry {
	return &CacheEntryK8sServiceEndpoints{}
}

func (c *K8sServiceEndpointsEngine) convertToVersioned(res resources.Resource) (VersionedResource, error) {
	in := res.(*corev1.Endpoints)
	return &versionedK8sServiceEndpoints{Endpoints: in}, nil
}

func (c *K8sServiceEndpointsEngine) resourceAdded(id apiv3.ResourceID, entry CacheEntry) {
	x := entry.(*CacheEntryK8sServiceEndpoints)
	x.clog = log.WithField("id", id)

	// Set the Service ID since this is basically the same as the Endpoints with a different kind.
	x.Service = apiv3.ResourceID{
		TypeMeta:  resources.TypeK8sServices,
		Name:      id.Name,
		Namespace: id.Namespace,
	}

	c.resourceUpdated(id, entry, nil)
}

func (c *K8sServiceEndpointsEngine) resourceUpdated(id apiv3.ResourceID, entry CacheEntry, prev VersionedResource) {
	x := entry.(*CacheEntryK8sServiceEndpoints)
	i, err := x.getIPAndEndpointIDs()
	if err != nil {
		x.clog.Info("Unable to determine IP addresses or Pod IDs")
	}
	c.IPOrEndpointManager().SetClientKeys(x.Service, i)
}

func (c *K8sServiceEndpointsEngine) resourceDeleted(id apiv3.ResourceID, entry CacheEntry) {
	x := entry.(*CacheEntryK8sServiceEndpoints)
	c.IPOrEndpointManager().DeleteClient(x.Service)
}

// recalculate implements the resourceCacheEngine interface.
func (c *K8sServiceEndpointsEngine) recalculate(podId apiv3.ResourceID, podEntry CacheEntry) syncer.UpdateType {
	// We calculate all state in the resourceUpdated/resourceAdded callbacks.
	return 0
}
