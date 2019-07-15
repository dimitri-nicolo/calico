package pip

import (
	"context"

	"github.com/tigera/es-proxy/pkg/pip/policycalc"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/compliance/pkg/config"
	"github.com/tigera/compliance/pkg/list"
	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
	"github.com/tigera/compliance/pkg/xrefcache"
)

var (
	// These are the resource types that we need to query from the k8s API to populate our internal cache.
	requiredTypes = []v1.TypeMeta{
		resources.TypeCalicoTiers,
		resources.TypeCalicoNetworkPolicies,
		resources.TypeCalicoGlobalNetworkPolicies,
		resources.TypeCalicoGlobalNetworkSets,
		resources.TypeK8sNamespaces,
		resources.TypeK8sServiceAccounts,
	}
)

// New returns a new PIP instance.
func New(cfg *policycalc.Config, listSrc list.Source) PIP {
	p := &pip{
		listSrc: listSrc,
		cfg:     cfg,
	}
	return p
}

// pip implements the PIP interface.
type pip struct {
	listSrc list.Source
	cfg     *policycalc.Config
}

// GetPolicyCalculator loads the initial configuration and updated configuration and returns a primed policyCalculator
// used for checking flow impact.
func (s *pip) GetPolicyCalculator(ctx context.Context, r []ResourceChange) (policycalc.PolicyCalculator, error) {
	// Create a new x-ref cache. Use a blank compliance config for the config settings since the XrefCache currently
	// requires it but doesn't use any fields except the istio config (which we're not concerned with in the pip use
	// case).
	//
	// We just use the xref cache to determine the ordered set of tiers and policies before and after the updates. Set
	// in-sync immediately since we aren't interested in callbacks.
	xc := xrefcache.NewXrefCache(&config.Config{}, func() {})
	xc.OnStatusUpdate(syncer.NewStatusUpdateComplete())

	// Load the initial set of policy.
	if err := s.loadInitialPolicy(xc); err != nil {
		return nil, err
	}

	// Extract the current set of config from the xrefcache.
	resourceDataBefore := s.resourceDataFromXrefCache(xc)

	// Apply the preview changes to the xref cache. This also constructs the set of modified resources for use by the
	// policy calculator.
	modified, err := s.applyPolicyChanges(xc, r)
	if err != nil {
		return nil, err
	}

	// Extract the updated set of config from the xrefcache.
	resourceDataAfter := s.resourceDataFromXrefCache(xc)

	// Create the policy calculator.
	calc := policycalc.NewPolicyCalculator(s.cfg, resourceDataBefore, resourceDataAfter, modified)

	return calc, nil
}

// loadInitialPolicy will load the initial set of policy and related resources from the k8s API into the xrefcache.
func (s *pip) loadInitialPolicy(xc xrefcache.XrefCache) error {
	for _, t := range requiredTypes {
		l, err := s.listSrc.RetrieveList(t)
		if err != nil {
			return err
		}

		err = meta.EachListItem(l.ResourceList, func(obj runtime.Object) error {
			res := obj.(resources.Resource)
			xc.OnUpdates([]syncer.Update{{
				Type:       syncer.UpdateTypeSet,
				Resource:   res,
				ResourceID: resources.GetResourceID(res),
			}})
			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
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

// Create the policy configuration from the data stored in the xrefcache.
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
