// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package xrefcache

import (
	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/k8s/conversion"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/backend/syncersv1/updateprocessors"

	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
)

const (
	// The set of pod flags that are updated directly from the network policy flags associated with the pod.
	CacheEntryEndpointAndNetworkPolicy = CacheEntryFlagsEndpoint & CacheEntryFlagsNetworkPolicy
)

var (
	KindsEndpoint = []schema.GroupVersionKind{
		resources.ResourceTypeHostEndpoints,
		resources.ResourceTypePods,
	}
)

// VersionedEndpointResource is an extension of the VersionedResource interface, specific to handling Pods.
type VersionedEndpointResource interface {
	VersionedResource
	getV1Labels() map[string]string
	getV1Profiles() []string
}

// CacheEntryEndpoint implements the CacheEntry interface, and is what we stored in the Pods cache.
type CacheEntryEndpoint struct {
	// The versioned policy resource.
	VersionedEndpointResource

	// Boolean values associated with this pod. Valid flags defined by CacheEntryFlagsEndpoint.
	Flags CacheEntryFlags

	// Policies applied to this pod.
	AppliedPolicies resources.Set

	// --- Internal data ---
	cacheEntryCommon
	clog *log.Entry
}

// getVersionedResource implements the CacheEntry interface.
func (c *CacheEntryEndpoint) getVersionedResource() VersionedResource {
	return c.VersionedEndpointResource
}

// setVersionedResource implements the CacheEntry interface.
func (c *CacheEntryEndpoint) setVersionedResource(r VersionedResource) {
	c.VersionedEndpointResource = r.(VersionedEndpointResource)
}

// versionedK8sNamespace implements the VersionedEndpointResource interface.
type versionedK8sPod struct {
	*corev1.Pod
	v3 *apiv3.WorkloadEndpoint
	v1 *model.WorkloadEndpoint
}

// getV3 implements the VersionedEndpointResource interface.
func (v *versionedK8sPod) getV3() resources.Resource {
	return v.v3
}

// getV1 implements the VersionedEndpointResource interface.
func (v *versionedK8sPod) getV1() interface{} {
	return v.v1
}

// getLabels implements the VersionedEndpointResource interface.
func (v *versionedK8sPod) getV1Labels() map[string]string {
	return v.v1.Labels
}

// getLabels implements the VersionedEndpointResource interface.
func (v *versionedK8sPod) getV1Profiles() []string {
	return v.v1.ProfileIDs
}

// versionedCalicoHostEndpoint implements the VersionedEndpointResource interface.
type versionedCalicoHostEndpoint struct {
	*apiv3.HostEndpoint
	v1 *model.HostEndpoint
}

// getV3 implements the VersionedEndpointResource interface.
func (v *versionedCalicoHostEndpoint) getV3() resources.Resource {
	return v.HostEndpoint
}

// getV1 implements the VersionedEndpointResource interface.
func (v *versionedCalicoHostEndpoint) getV1() interface{} {
	return v.v1
}

// getLabels implements the VersionedEndpointResource interface.
func (v *versionedCalicoHostEndpoint) getV1Labels() map[string]string {
	return v.v1.Labels
}

// getLabels implements the VersionedEndpointResource interface.
func (v *versionedCalicoHostEndpoint) getV1Profiles() []string {
	return v.v1.ProfileIDs
}

// newEndpointsEngine creates a resourceCacheEngine used to handle the Pods cache.
func newEndpointsEngine() resourceCacheEngine {
	return &endpointEngine{}
}

// endpointEngine implements the resourceCacheEngine.
type endpointEngine struct {
	engineCache
	converter conversion.Converter
}

// kinds implements the resourceCacheEngine interface.
func (c *endpointEngine) kinds() []schema.GroupVersionKind {
	return KindsEndpoint
}

// register implements the resourceCacheEngine interface.
func (c *endpointEngine) register(cache engineCache) {
	c.engineCache = cache
	c.EndpointLabelSelector().RegisterCallbacks(c.kinds(), c.policyMatchStarted, c.policyMatchStopped)

	// Register for updates for all NetworkPolicy events. We don't care about Added/Deleted/Updated events as any
	// changes to the cross-referencing will result in a notification here where we will requeue any changed endpoints.
	for _, kind := range KindsNetworkPolicy {
		c.RegisterOnUpdateHandler(
			kind,
			syncer.UpdateType(CacheEntryFlagsNetworkPolicy),
			c.queueEndpointsForRecalculation,
		)
	}
}

// newCacheEntry implements the resourceCacheEngine interface.
func (c *endpointEngine) newCacheEntry() CacheEntry {
	return &CacheEntryEndpoint{
		AppliedPolicies: resources.NewSet(),
	}
}

// convertToVersioned implements the resourceCacheEngine interface.
func (c *endpointEngine) convertToVersioned(res resources.Resource) (VersionedResource, error) {
	switch in := res.(type) {
	case *apiv3.HostEndpoint:
		v1, err := updateprocessors.ConvertHostEndpointV3ToV1(&model.KVPair{
			Key: model.ResourceKey{
				Kind: apiv3.KindHostEndpoint,
				Name: in.Name,
			},
			Value: in,
		})
		if err != nil {
			return nil, err
		}

		return &versionedCalicoHostEndpoint{
			HostEndpoint: in,
			v1:           v1.Value.(*model.HostEndpoint),
		}, nil
	case *corev1.Pod:
		kvp, err := c.converter.PodToWorkloadEndpoint(in)
		if err != nil {
			return nil, err
		}

		v3 := kvp.Value.(*apiv3.WorkloadEndpoint)
		v1, err := updateprocessors.ConvertWorkloadEndpointV3ToV1Value(v3)
		if err != nil {
			return nil, err
		}

		return &versionedK8sPod{
			Pod: in,
			v3:  v3,
			v1:  v1.(*model.WorkloadEndpoint),
		}, nil
	}

	return nil, nil
}

// resourceAdded implements the resourceCacheEngine interface.
func (c *endpointEngine) resourceAdded(id resources.ResourceID, entry CacheEntry) {
	x := entry.(*CacheEntryEndpoint)
	x.clog = log.WithField("id", id)
	c.resourceUpdated(id, entry, nil)
}

// resourceUpdated implements the resourceCacheEngine interface.
func (c *endpointEngine) resourceUpdated(id resources.ResourceID, entry CacheEntry, prev VersionedResource) {
	x := entry.(*CacheEntryEndpoint)

	// Update the labels associated with this pod. Use the labels and profile from the v1 model since these are
	// modified to include namespace and service account details.
	c.EndpointLabelSelector().UpdateLabels(id, x.getV1Labels(), x.getV1Profiles())
}

// resourceDeleted implements the resourceCacheEngine interface.
func (c *endpointEngine) resourceDeleted(id resources.ResourceID, _ CacheEntry) {
	// Delete the labels associated with this pod. Default cache processing will remove this cache entry.
	c.EndpointLabelSelector().DeleteLabels(id)
}

// recalculate implements the resourceCacheEngine interface.
func (c *endpointEngine) recalculate(podId resources.ResourceID, epEntry CacheEntry) syncer.UpdateType {
	x := epEntry.(*CacheEntryEndpoint)

	// ------
	// See note in flags.go for details of the bitwise operations for boolean values and their associated update type.
	// ------

	// Store the current set of flags.
	oldFlags := x.Flags

	// Clear the set of flags that will be reset from the applied network Policies.
	x.Flags &^= CacheEntryEndpointAndNetworkPolicy

	// Iterate through the applied network Policies and recalculate the flags that the network policy applies to the
	// x.
	x.AppliedPolicies.Iter(func(polId resources.ResourceID) error {
		policy, ok := c.GetFromXrefCache(polId).(*CacheEntryNetworkPolicy)

		if !ok {
			// The applied Policies should always be in the cache since deletion of the underlying policy should remove
			// the reference from the set.
			log.Errorf("%s applied policy is missing from cache: %s", podId, polId)
			return nil
		}

		// The x flags are the combined set of flags from the applied Policies filtered by the allowed set of
		// flags for a Pod.
		x.Flags |= policy.Flags & CacheEntryEndpointAndNetworkPolicy

		// If all flags that the policy can set in the x are now set then exit without checking the other Policies.
		if x.Flags&CacheEntryEndpointAndNetworkPolicy == CacheEntryEndpointAndNetworkPolicy {
			return resources.StopIteration
		}

		return nil
	})

	// Return the delta between the old and new flags as a set up UpdateType flags.
	changed := syncer.UpdateType(oldFlags ^ x.Flags)
	x.clog.Debugf("Recalculated, returning update: %d", changed)

	return changed
}

func (c *endpointEngine) queueEndpointsForRecalculation(update syncer.Update) {
	x := update.Resource.(*CacheEntryNetworkPolicy)
	x.SelectedPods.Iter(func(podId resources.ResourceID) error {
		c.QueueRecalculation(podId, nil, update.Type)
		return nil
	})
	x.SelectedHostEndpoints.Iter(func(hepId resources.ResourceID) error {
		c.QueueRecalculation(hepId, nil, update.Type)
		return nil
	})
}

// policyMatchStarted is called synchronously from the policy or pod resource update methods when a policy<->pod match
// has started. We update  our set of applied Policies and then queue for asynchronous recalculation - this ensures we
// wait until all related changes to have occurred further up the casading chain of events before we recalculate.
func (c *endpointEngine) policyMatchStarted(policyId, podId resources.ResourceID) {
	p, ok := c.GetFromOurCache(podId).(*CacheEntryEndpoint)
	if !ok {
		// This is called synchronously from the resource update methods, so we don't expect the entries to have been
		// removed from the cache at this point.
		log.Errorf("Match started on pod, but pod is not in cache: %s matches %s", policyId, podId)
		return
	}
	// Update the policy list in our pod data and queue a recalculation.
	p.AppliedPolicies.Add(policyId)
	c.QueueRecalculation(podId, p, EventPolicyMatchStarted)
}

// policyMatchStopped is called synchronously from the policy or pod resource update methods when a policy<->pod match
// has stopped. We update  our set of applied Policies and then queue for asynchronous recalculation - this ensures we
// wait until all related changes to have occurred further up the chain of events before we recalculate.
func (c *endpointEngine) policyMatchStopped(policyId, podId resources.ResourceID) {
	p, ok := c.GetFromOurCache(podId).(*CacheEntryEndpoint)
	if !ok {
		// This is called synchronously from the resource update methods, so we don't expect the entries to have been
		// removed from the cache at this point.
		log.Errorf("Match stopped on pod, but pod is not in cache: %s no longer matches %s", policyId, podId)
		return
	}
	// Update the policy list in our pod data and queue a recalculation.
	p.AppliedPolicies.Discard(policyId)
	c.QueueRecalculation(podId, p, EventPolicyMatchStopped)
}
