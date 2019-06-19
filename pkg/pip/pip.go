package pip

import (
	"context"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	networkingv1 "k8s.io/api/networking/v1"

	"github.com/tigera/compliance/pkg/config"
	"github.com/tigera/compliance/pkg/list"
	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
	"github.com/tigera/compliance/pkg/xrefcache"
	"github.com/tigera/es-proxy/pkg/pip/flow"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type pip struct {
	xc      xrefcache.XrefCache
	listSrc list.Source

	inScope map[v3.ResourceID]*xrefcache.CacheEntryEndpoint
}

func New(listSrc list.Source) PIP {
	return &pip{
		listSrc: listSrc,
	}
}

// CalculateFlowImpact is the meat of the PIP API. It loads all network policy data from k8s, runs the passed-in flows
// through the policy chains, and then determines the Action of the flow. It then does the same operation a second time,
// after making the passed-in NetworkPolicyChanges to the policy chain. The end-result is a set of flows with a new
// attribute on each: "preview_action".
func (s *pip) CalculateFlowImpact(ctx context.Context, npcs []NetworkPolicyChange, flows []flow.Flow) ([]flow.Flow, error) {
	// Create a new x-ref cache. Use a blank compliance config for the config settings since the XrefCache currently
	// requires it but doesn't use any fields except the istio config (which we're not concerned with in the pip use case).
	s.xc = xrefcache.NewXrefCache(&config.Config{}, func() {})

	// Set the in-scope endpoints by combining (OR-ing) the selectors of all of the modified policies.
	s.xc.RegisterInScopeEndpoints(&v3.EndpointsSelection{
		Selector: buildSelector(npcs),
	})

	// Register for notification of in-scope endpoints.
	for _, k := range xrefcache.KindsEndpoint {
		s.xc.RegisterOnUpdateHandler(k, xrefcache.EventInScope, s.onUpdate)
	}

	// Load the initial set of policy.
	if err := s.loadInitialPolicy(); err != nil {
		return flows, err
	}

	// Set in-sync, so we get updates as we feed in the endpoints.
	s.xc.OnStatusUpdate(syncer.NewStatusUpdateComplete())

	// TODO: modify flows using policy before returning them
	return flows, nil
}

func (s *pip) onUpdate(update syncer.Update) {
	s.inScope[update.ResourceID] = update.Resource.(*xrefcache.CacheEntryEndpoint)
}

func (s *pip) loadInitialPolicy() error {
	list, err := s.listSrc.RetrieveList(resources.TypeCalicoTiers)
	if err != nil {
		return err
	}
	for i := range list.ResourceList.(*v3.TierList).Items {
		res := &list.ResourceList.(*v3.TierList).Items[i]
		s.xc.OnUpdates([]syncer.Update{{
			Type:       syncer.UpdateTypeSet,
			Resource:   res,
			ResourceID: resources.GetResourceID(res),
		}})
	}

	list, err = s.listSrc.RetrieveList(resources.TypeK8sNetworkPolicies)
	if err != nil {
		return err
	}
	for i := range list.ResourceList.(*networkingv1.NetworkPolicyList).Items {
		res := &list.ResourceList.(*networkingv1.NetworkPolicyList).Items[i]
		s.xc.OnUpdates([]syncer.Update{{
			Type:       syncer.UpdateTypeSet,
			Resource:   res,
			ResourceID: resources.GetResourceID(res),
		}})
	}

	list, err = s.listSrc.RetrieveList(resources.TypeCalicoNetworkPolicies)
	if err != nil {
		return err
	}
	for i := range list.ResourceList.(*v3.NetworkPolicyList).Items {
		res := &list.ResourceList.(*v3.NetworkPolicyList).Items[i]
		s.xc.OnUpdates([]syncer.Update{{
			Type:       syncer.UpdateTypeSet,
			Resource:   res,
			ResourceID: resources.GetResourceID(res),
		}})
	}

	list, err = s.listSrc.RetrieveList(resources.TypeCalicoGlobalNetworkPolicies)
	if err != nil {
		return err
	}
	for i := range list.ResourceList.(*v3.GlobalNetworkPolicyList).Items {
		res := &list.ResourceList.(*v3.GlobalNetworkPolicyList).Items[i]
		s.xc.OnUpdates([]syncer.Update{{
			Type:       syncer.UpdateTypeSet,
			Resource:   res,
			ResourceID: resources.GetResourceID(res),
		}})
	}

	list, err = s.listSrc.RetrieveList(resources.TypeCalicoGlobalNetworkSets)
	if err != nil {
		return err
	}
	for i := range list.ResourceList.(*v3.GlobalNetworkSetList).Items {
		res := &list.ResourceList.(*v3.GlobalNetworkSetList).Items[i]
		s.xc.OnUpdates([]syncer.Update{{
			Type:       syncer.UpdateTypeSet,
			Resource:   res,
			ResourceID: resources.GetResourceID(res),
		}})
	}

	return nil
}

func buildSelector(npcs []NetworkPolicyChange) string {
	// TODO: loop through policy change, create a set, and union the results in this format: "() | () | ()"
	return "all()"
}

//Because... where does this come from ?
type DummySource struct {
}

func (d *DummySource) RetrieveList(kind metav1.TypeMeta) (*list.TimestampedResourceList, error) {
	return &list.TimestampedResourceList{}, nil
}
