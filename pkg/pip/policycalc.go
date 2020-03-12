package pip

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/resources"
	validator "github.com/projectcalico/libcalico-go/lib/validator/v3"

	compcfg "github.com/tigera/compliance/pkg/config"
	"github.com/tigera/compliance/pkg/replay"
	"github.com/tigera/compliance/pkg/syncer"
	"github.com/tigera/compliance/pkg/xrefcache"

	"github.com/tigera/es-proxy/pkg/pip/policycalc"
)

var (
	// These are the resource types that we need to query from the k8s API to populate our internal cache.
	requiredPolicyTypes = []metav1.TypeMeta{
		resources.TypeCalicoStagedKubernetesNetworkPolicies,
		resources.TypeCalicoStagedGlobalNetworkPolicies,
		resources.TypeCalicoStagedNetworkPolicies,
		resources.TypeCalicoTiers,
		resources.TypeCalicoNetworkPolicies,
		resources.TypeCalicoGlobalNetworkPolicies,
		resources.TypeK8sNetworkPolicies,
		resources.TypeK8sNamespaces,
		resources.TypeK8sServiceAccounts,
	}
	// These are the resource types that we need to query from the k8s API to populate our internal cache.
	requiredEndpointTypes = []metav1.TypeMeta{
		resources.TypeCalicoNetworkSets,
		resources.TypeCalicoGlobalNetworkSets,
		resources.TypeCalicoHostEndpoints,
		resources.TypeK8sPods,
	}
)

// GetPolicyCalculator loads the initial configuration and updated configuration and returns a primed PolicyCalculator
// used for checking flow impact.
func (s *pip) GetPolicyCalculator(ctx context.Context, params *PolicyImpactParams) (policycalc.PolicyCalculator, error) {
	// Create a new x-ref cache. Use a blank compliance config for the config settings since the XrefCache currently
	// requires it but doesn't use any fields except the istio config (which we're not concerned with in the pip use
	// case).
	//
	// We just use the xref cache to determine the ordered set of tiers and policies before and after the updates. Set
	// in-sync immediately since we aren't interested in callbacks.
	xc := xrefcache.NewXrefCache(&compcfg.Config{IncludeStagedNetworkPolicies: true}, func() {})
	xc.OnStatusUpdate(syncer.NewStatusUpdateInSync())

	// Populate the endpoint cache. Run this on a go-routine so we can double up with the other queries.
	// Depending on configuration, the endpoint cache may be populated from historical data (snapshots and audit logs),
	// and/or from current endpoint configuration. The default is neither - we only use flow log data for our
	// calculations.
	ec := policycalc.NewEndpointCache()
	wgEps := sync.WaitGroup{}
	wgEps.Add(1)
	go func() {
		defer wgEps.Done()
		if s.cfg.AugmentFlowLogDataWithAuditLogData {
			log.Debug("Augmenting flow log data with audit log data")
			s.syncFromArchive(ctx, params, ec)
		}
		if s.cfg.AugmentFlowLogDataWithCurrentConfiguration {
			log.Debug("Augmenting flow log data with current datastore configuration")
			_ = s.syncFromDatastore(ctx, requiredEndpointTypes, ec)
		}
	}()

	// Load the initial set of policy. If this errors we cannot continue.
	if err := s.syncFromDatastore(ctx, requiredPolicyTypes, xc); err != nil {
		return nil, err
	}

	// Extract the current set of config from the xrefcache.
	resourceDataBefore := resourceDataFromXrefCache(xc)

	// Apply the preview changes to the xref cache. This also constructs the set of impacted resources for use by the
	// policy calculator.
	impacted, err := ApplyPIPPolicyChanges(xc, params.ResourceActions)
	if err != nil {
		return nil, err
	}

	// Extract the updated set of config from the xrefcache.
	resourceDataAfter := resourceDataFromXrefCache(xc)

	// Wait for the archived endpoint query to complete. We don't track if the endpoint cache population errors since
	// we can still do a PIP query without it, however, chance of indeterminate calculations will be higher.
	wgEps.Wait()

	// Create the policy calculator.
	calc := policycalc.NewPolicyCalculator(s.cfg, ec, resourceDataBefore, resourceDataAfter, impacted)

	return calc, nil
}

// syncFromArchive will load archived configuration and invoke the syncer callbacks.
func (s *pip) syncFromArchive(cxt context.Context, params *PolicyImpactParams, cb syncer.SyncerCallbacks) {
	// If we could not determine the time interval, then we can't populate the cache from archived data.
	if params.FromTime == nil || params.ToTime == nil {
		log.Debug("No From/To time available, so cannot load archived data")
		return
	}

	// Populate the cache from the replayer.
	r := replay.New(*params.FromTime, *params.ToTime, s.esClient, s.esClient, cb)
	r.Start(cxt)
	return
}

// syncFromDatastore will load the current set of configuration from the datastore and invoke the syncer callbacks.
// This is used to populate both the xrefcache and the EndpointCache which both implement the syncer callbacks
// interface.
func (s *pip) syncFromDatastore(ctx context.Context, types []metav1.TypeMeta, cb syncer.SyncerCallbacks) error {
	wg := sync.WaitGroup{}
	lock := sync.Mutex{}
	errs := make(chan error, len(types))
	defer close(errs)

	for _, t := range types {
		// If we are running in an FV framework then skip config load of Calico resources which require an AAPIS.
		if s.cfg.RunningFunctionalVerification && t.APIVersion == v3.GroupVersionCurrent {
			log.Warningf("Running functional verification - skipping config load from datastore for %s", t.Kind)
			return nil
		}

		wg.Add(1)
		go func(t metav1.TypeMeta) {
			defer wg.Done()

			// List current resource configuration for this type.
			l, err := s.listSrc.RetrieveList(t)
			if err != nil {
				errs <- err
				return
			}

			// Invoke the syncer callbacks for each item in the list. We need to lock around the callbacks because the
			// syncer interfaces are assumed not to be go-routine safe.
			lock.Lock()
			err = meta.EachListItem(l.ResourceList, func(obj runtime.Object) error {
				res := obj.(resources.Resource)
				cb.OnUpdates([]syncer.Update{{
					Type:       syncer.UpdateTypeSet,
					Resource:   res,
					ResourceID: resources.GetResourceID(res),
				}})
				return nil
			})
			lock.Unlock()

			if err != nil {
				errs <- err
				return
			}
		}(t)
	}
	wg.Wait()

	// Return the first error if there is one. Use non-blocking version of the channel operator.
	select {
	case err := <-errs:
		log.WithError(err).Warning("Hit error loading configuration from datastore")
		cb.OnStatusUpdate(syncer.StatusUpdate{
			Type:  syncer.StatusTypeFailed,
			Error: err,
		})
		return err
	default:
		log.Info("Loaded configuration from datastore")
		return nil
	}
}

// resourceDataFromXrefCache creates the policy configuration from the data stored in the xrefcache.
func resourceDataFromXrefCache(xc xrefcache.XrefCache) *policycalc.ResourceData {
	// Create an empty config.
	rd := &policycalc.ResourceData{}

	// Grab the ordered tiers and policies from the xrefcache and convert to the required type.
	xrefTiers := xc.GetOrderedTiersAndPolicies()
	rd.Tiers = make(policycalc.Tiers, len(xrefTiers))
	for i := range xrefTiers {
		for _, t := range xrefTiers[i].OrderedPolicies {
			rd.Tiers[i] = append(rd.Tiers[i], policycalc.Policy{
				Policy: t.GetCalicoV3(),
				Staged: t.IsStaged(),
			})
		}
	}

	// Grab the namespaces and the service accounts.
	_ = xc.EachCacheEntry(resources.TypeK8sNamespaces, func(ce xrefcache.CacheEntry) error {
		rd.Namespaces = append(rd.Namespaces, ce.GetPrimary().(*corev1.Namespace))
		return nil
	})
	_ = xc.EachCacheEntry(resources.TypeK8sServiceAccounts, func(ce xrefcache.CacheEntry) error {
		rd.ServiceAccounts = append(rd.ServiceAccounts, ce.GetPrimary().(*corev1.ServiceAccount))
		return nil
	})

	return rd
}

// ApplyPolicyChanges applies the supplied resource updates on top of the loaded configuration in the xrefcache.
func ApplyPIPPolicyChanges(xc xrefcache.XrefCache, rs []ResourceChange) (policycalc.ImpactedResources, error) {
	impacted := make(policycalc.ImpactedResources)
	var err error

	for _, r := range rs {
		// Get the resource ID.
		id := resources.GetResourceID(r.Resource)

		// Trace the input resource.
		log.Debugf("Applying resource update: %s", id)

		// Convert staged policies to non-staged/enforced policies. Note that we don't care about the ordering
		// here since we're currently only passing one resource at a time, and that we skip "delete" actions
		// for staged policies since we don't process staged policies in the "after" processing.
		var stagedAction v3.StagedAction

		// Unless determined otherwise, the resource is modified and enforced.
		staged := false
		modified := true

		// Extract the resource. If this is a staged resource, convert to the enforced equivalent.
		resource := r.Resource
		action := r.Action

		// Locate the resource in the xrefcache if it exists and work out if it has been modified. We do this for
		// staged and enforced policies.
		existing := xc.Get(id)
		if existing != nil && resource != nil {
			log.Debug("Check if resource is modified")
			modified, err = IsResourceModifiedForPIP(existing.GetPrimary(), resource)
			if err != nil {
				return nil, err
			}
		}

		switch np := resource.(type) {
		case *v3.StagedNetworkPolicy:
			log.Debug("Enforcing StagedNetworkPolicy")
			stagedAction, resource = v3.ConvertStagedPolicyToEnforced(np)
			staged = true
		case *v3.StagedGlobalNetworkPolicy:
			log.Debug("Enforcing StagedGlobalNetworkPolicy")
			stagedAction, resource = v3.ConvertStagedGlobalPolicyToEnforced(np)
			staged = true
		case *v3.StagedKubernetesNetworkPolicy:
			log.Debug("Enforcing StagedKubernetesNetworkPolicy")
			stagedAction, resource = v3.ConvertStagedKubernetesPolicyToK8SEnforced(np)
			staged = true
		}

		if staged {
			// Update and trace the resource ID.
			id = resources.GetResourceID(resource)
			log.Debugf("Converted staged resource update: %s", id)

			if action == "delete" {
				log.Debug("Staged policy deleted - no op")
				continue
			}

			switch stagedAction {
			case v3.StagedActionDelete:
				// If the staged action was delete then set the resource action to delete with the enforced resource.
				action = "delete"
			case v3.StagedActionSet, "":
				// If the staged action was set then set the resource action update with the enforced resource. Note
				// that update and create are handled identically.
				action = "update"
			default:
				log.Warningf("Invalid staged action: %s", stagedAction)
				return nil, fmt.Errorf("invalid staged action in preview request: %s", stagedAction)
			}
		}

		switch action {
		case "update", "create":
			log.Debugf("Update or create resource: id=%s; modified=%v; staged=%v; deleted=false", id, modified, staged)
			impacted.Add(id, policycalc.Impact{UseStaged: staged, Modified: modified, Deleted: false})
			xc.OnUpdates([]syncer.Update{{
				Type:       syncer.UpdateTypeSet,
				Resource:   resource,
				ResourceID: id,
			}})

			if xcres := xc.Get(id); xcres == nil {
				// The xrefcache will delete resources that could not be converted (which may be the case with incorrect
				// data). Check the resource is present, and if not, error.
				log.Infof("Invalid resource data: %v", resource)
				return nil, fmt.Errorf("invalid resource in preview request: %s", id.String())
			} else if v3res := xcres.GetCalicoV3(); v3res != nil {
				// Validate the calico representation of the resource.
				log.Debug("Validating Calico v3 resource")
				if err := validator.Validate(v3res); err != nil {
					log.WithError(err).Info("previous resource failed validation")
					return nil, err
				}
			}
		case "delete":
			log.Debugf("Delete resource: id=%s; modified=false; staged=%v; deleted=true", id, staged)
			impacted.Add(id, policycalc.Impact{UseStaged: staged, Modified: false, Deleted: true})
			xc.OnUpdates([]syncer.Update{{
				Type:       syncer.UpdateTypeDeleted,
				Resource:   resource,
				ResourceID: id,
			}})
		default:
			log.Warningf("Invalid preview action: %s", action)
			return nil, fmt.Errorf("invalid action in preview request: %s", action)
		}
	}

	// Remove the actual staged resources - these are useful for the "before" calculation when we cache data from the
	// flow logs, but we do not want them in the "after" calculation where we ignore staged policies except for those
	// we are explicitly performing policy impact on - and in that case we convert the staged policies to the enforced
	// equivalent.
	DeleteStagedResources(xc)

	return impacted, nil
}

// DeleteStagedResources removes all staged resources from the xref cache. We do this for the "after" processing
// because we don't need to process staged resources in that case.
func DeleteStagedResources(xc xrefcache.XrefCache) {
	log.WithField("xc", xc).Debug("Deleting staged resources from xrefcache")
	_ = xc.EachCacheEntry(resources.TypeCalicoStagedNetworkPolicies, func(ce xrefcache.CacheEntry) error {
		log.WithField("ce", ce).Debug("Sending delete update")
		xc.OnUpdates([]syncer.Update{
			{Type: syncer.UpdateTypeDeleted, ResourceID: resources.GetResourceID(ce)},
		})
		return nil
	})
	_ = xc.EachCacheEntry(resources.TypeCalicoStagedGlobalNetworkPolicies, func(ce xrefcache.CacheEntry) error {
		log.WithField("ce", ce).Debug("Sending delete update")
		xc.OnUpdates([]syncer.Update{
			{Type: syncer.UpdateTypeDeleted, ResourceID: resources.GetResourceID(ce)},
		})
		return nil
	})
	_ = xc.EachCacheEntry(resources.TypeCalicoStagedKubernetesNetworkPolicies, func(ce xrefcache.CacheEntry) error {
		log.WithField("ce", ce).Debug("Sending delete update")
		xc.OnUpdates([]syncer.Update{
			{Type: syncer.UpdateTypeDeleted, ResourceID: resources.GetResourceID(ce)},
		})
		return nil
	})
}

// IsResourceModifiedForPIP compares the before and after resource to determine if the settings have been modified in a
// way that will impact the policy calculation of this specific resource. If modified then we cannot use historical data
// in the flow log to augment the pip calculation.
//
// Note that for policies, we don't care about order changes because the order doesn't impact whether or not the policy
// itself will match a flow. This is a minor finesse for the situation where we decrease the order of a policy but don't
// change anything else - in this case we can still use the match data in the flow log for this policy (if we have any)
// to augment the calculation.
func IsResourceModifiedForPIP(r1, r2 resources.Resource) (bool, error) {
	if r1 == nil || r2 == nil {
		// A resource is not modified if either is nil.  Created or deleted, but not modified (in the sense that we
		// mean.
		log.Debug("At least one resource is nil - not modified")
		return false, nil
	}

	if reflect.TypeOf(r1) != reflect.TypeOf(r2) {
		// Resource types do not match.  This indicates a bug rather than an abuse of the API, but return an error
		// up the stack for debugging purposes.
		log.Errorf("Resource types do not match: %v != %v", reflect.TypeOf(r1), reflect.TypeOf(r2))
		return false, fmt.Errorf("Resource type before and after do not match: %v != %v", reflect.TypeOf(r1), reflect.TypeOf(r2))
	}

	// Copy the resources since we modify them to do the comparison.
	r1 = r1.DeepCopyObject().(resources.Resource)
	r2 = r2.DeepCopyObject().(resources.Resource)
	mod := true

	switch rc1 := r1.(type) {
	case *v3.NetworkPolicy:
		log.Debug("Compare v3.NetworkPolicy")
		rc2 := r2.(*v3.NetworkPolicy)

		// For the purposes of PIP we don't care if the order changed since that doesn't impact the policy rule matches,
		// so nil out the order before comparing.  We only need to compare the spec for policies.
		rc1.Spec.Order = nil
		rc2.Spec.Order = nil
		mod = !reflect.DeepEqual(rc1.Spec, rc2.Spec)
	case *v3.StagedNetworkPolicy:
		log.Debug("Compare v3.StagedNetworkPolicy")
		rc2 := r2.(*v3.StagedNetworkPolicy)

		// For the purposes of PIP we don't care if the order changed since that doesn't impact the policy rule matches,
		// so nil out the order.
		rc1.Spec.Order = nil
		rc2.Spec.Order = nil
		mod = !reflect.DeepEqual(rc1.Spec, rc2.Spec)
	case *v3.GlobalNetworkPolicy:
		log.Debug("Compare v3.GlobalNetworkPolicy")
		rc2 := r2.(*v3.GlobalNetworkPolicy)

		// For the purposes of PIP we don't care if the order changed since that doesn't impact the policy rule matches,
		// so nil out the order before comparing.  We only need to compare the spec for policies.
		rc1.Spec.Order = nil
		rc2.Spec.Order = nil
		mod = !reflect.DeepEqual(rc1.Spec, rc2.Spec)
	case *v3.StagedGlobalNetworkPolicy:
		log.Debug("Compare v3.StagedGlobalNetworkPolicy")
		rc2 := r2.(*v3.StagedGlobalNetworkPolicy)

		// For the purposes of PIP we don't care if the order changed since that doesn't impact the policy rule matches,
		// so nil out the order before comparing.  We only need to compare the spec for policies.
		rc1.Spec.Order = nil
		rc2.Spec.Order = nil
		mod = !reflect.DeepEqual(rc1.Spec, rc2.Spec)
	case *networkingv1.NetworkPolicy:
		log.Debug("Compare networkingv1.NetworkPolicy")
		rc2 := r2.(*networkingv1.NetworkPolicy)

		// We only need to compare the spec for policies. Kubernetes policies do not have an order.
		mod = !reflect.DeepEqual(rc1.Spec, rc2.Spec)
	case *v3.StagedKubernetesNetworkPolicy:
		log.Debug("Compare v3.StagedKubernetesNetworkPolicy")

		// We only need to compare the spec for policies. Kubernetes policies do not have an order.
		rc2 := r2.(*v3.StagedKubernetesNetworkPolicy)
		mod = !reflect.DeepEqual(rc1.Spec, rc2.Spec)
	default:
		log.Infof("Unhandled resource type for policy impact preview: %s", resources.GetResourceID(r1))
		return false, fmt.Errorf("resource type not valid for policy preview: %v", resources.GetResourceID(r1))
	}

	// Not a supported resource update type. Assume it changed.
	log.Debugf("Resource modified: %v", mod)
	return mod, nil
}
