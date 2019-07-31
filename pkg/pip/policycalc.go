package pip

import (
	"context"
	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"sync"

	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	compcfg "github.com/tigera/compliance/pkg/config"
	"github.com/tigera/compliance/pkg/replay"
	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
	"github.com/tigera/compliance/pkg/xrefcache"

	"github.com/tigera/es-proxy/pkg/pip/policycalc"
)

var (
	// These are the resource types that we need to query from the k8s API to populate our internal cache.
	requiredPolicyTypes = []metav1.TypeMeta{
		resources.TypeCalicoTiers,
		resources.TypeCalicoNetworkPolicies,
		resources.TypeCalicoGlobalNetworkPolicies,
		resources.TypeCalicoGlobalNetworkSets,
		resources.TypeK8sNamespaces,
		resources.TypeK8sServiceAccounts,
	}
	// These are the resource types that we need to query from the k8s API to populate our internal cache.
	requiredEndpointTypes = []metav1.TypeMeta{
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
	xc := xrefcache.NewXrefCache(&compcfg.Config{}, func() {})
	xc.OnStatusUpdate(syncer.NewStatusUpdateComplete())

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
	resourceDataBefore := s.resourceDataFromXrefCache(xc)

	// Apply the preview changes to the xref cache. This also constructs the set of modified resources for use by the
	// policy calculator.
	modified, err := s.applyPolicyChanges(xc, params.ResourceActions)
	if err != nil {
		return nil, err
	}

	// Extract the updated set of config from the xrefcache.
	resourceDataAfter := s.resourceDataFromXrefCache(xc)

	// Wait for the archived endpoint query to complete. We don't track if the endpoint cache population errors since
	// we can still do a PIP query without it, however, chance of indeterminate calculations will be higher.
	wgEps.Wait()

	// Create the policy calculator.
	calc := policycalc.NewPolicyCalculator(s.cfg, ec, resourceDataBefore, resourceDataAfter, modified)

	return calc, nil
}

// applyPolicyChanges applies the supplied resource updates on top of the loaded configuration in the xrefcache.
func (s *pip) applyPolicyChanges(xc xrefcache.XrefCache, rs []ResourceChange) (policycalc.ModifiedResources, error) {
	modified := make(policycalc.ModifiedResources)

	for _, r := range rs {
		id := resources.GetResourceID(r.Resource)
		modified.Add(r.Resource)

		switch r.Action {
		case "update":
			xc.OnUpdates([]syncer.Update{{
				Type:       syncer.UpdateTypeSet,
				Resource:   r.Resource,
				ResourceID: id,
			}})

		case "delete":
			xc.OnUpdates([]syncer.Update{{
				Type:       syncer.UpdateTypeDeleted,
				Resource:   r.Resource,
				ResourceID: id,
			}})
		case "create":
			xc.OnUpdates([]syncer.Update{{
				Type:       syncer.UpdateTypeSet,
				Resource:   r.Resource,
				ResourceID: id,
			}})
		}
	}
	return modified, nil
}

// resourceDataFromXrefCache creates the policy configuration from the data stored in the xrefcache.
func (s *pip) resourceDataFromXrefCache(xc xrefcache.XrefCache) *policycalc.ResourceData {
	// Create an empty config.
	rd := &policycalc.ResourceData{}

	// Grab the ordered tiers and policies from the xrefcache and convert to the required type.
	xrefTiers := xc.GetOrderedTiersAndPolicies()
	rd.Tiers = make(policycalc.Tiers, len(xrefTiers))
	for i := range xrefTiers {
		for _, t := range xrefTiers[i].OrderedPolicies {
			rd.Tiers[i] = append(rd.Tiers[i], t.GetCalicoV3())
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
	errs := make(chan error, len(requiredPolicyTypes))
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
