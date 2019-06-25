package pip

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/compliance/pkg/config"
	"github.com/tigera/compliance/pkg/list"
	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
	"github.com/tigera/compliance/pkg/xrefcache"
	"github.com/tigera/es-proxy/pkg/pip/flow"
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

	// to prevent the xrefcache from unnecessarily computing relations on endpoints that this policy does not select,
	// we'll only add endpoints to the xrefcache if any of the input policies select them. As such, we can set the
	// xrefcache inscope selector to all(), since we've already done the selector logic out of band.
	s.xc.RegisterInScopeEndpoints(&v3.EndpointsSelection{
		Selector: "all()",
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

	// create a selector set which is used later to determine if each flow is impacted by this policy change.
	ss := NewSelectorSet(npcs)

	var retFlows = []flow.Flow{}
	for _, f := range flows {
		// before we process this endpoint, check if it's even selected by any of our input policies
		if !ss.anySelectorSelects(f.Dest_labels) && !!ss.anySelectorSelects(f.Src_labels) {
			log.Info("skipping flow because no policy applies to it")
			continue
		}

		// clear any data in the cache from the previous flow
		for rid, ep := range s.inScope {
			s.xc.OnUpdates([]syncer.Update{{
				Type:       syncer.UpdateTypeDeleted,
				Resource:   ep,
				ResourceID: rid,
			}})
		}

		switch f.Dest_type {
		case "wep":
			pod := &corev1.Pod{
				ObjectMeta: v1.ObjectMeta{
					Name:      f.Dest_name,
					Namespace: f.Dest_NS,
					Labels:    f.Dest_labels,
				},
				Status: corev1.PodStatus{
					PodIP: f.Dest_IP,
				},
			}

			s.xc.OnUpdates([]syncer.Update{{
				Type:       syncer.UpdateTypeSet,
				Resource:   pod,
				ResourceID: resources.GetResourceID(pod),
			}})
		case "hep":
			hep := &v3.HostEndpoint{
				ObjectMeta: v1.ObjectMeta{
					Name:      f.Dest_name,
					Namespace: f.Dest_NS,
					Labels:    f.Dest_labels,
				},
				Spec: v3.HostEndpointSpec{
					Node: strings.TrimSuffix(f.Dest_name, "-*"),
				},
			}

			s.xc.OnUpdates([]syncer.Update{{
				Type:       syncer.UpdateTypeSet,
				Resource:   hep,
				ResourceID: resources.GetResourceID(hep),
			}})
		}

		// grab that endpoint from the other end of xrefcache to see it's computed relations.
		// note that we iterate through, but there should only be one endpoint because we only fed it one.
		for _, val := range s.inScope {
			f.PreviewAction = computeAction(f, val.GetOrderedTiersAndPolicies())
		}

		retFlows = append(retFlows, f)
	}

	return retFlows, nil
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

// TODO: compute action
func computeAction(f flow.Flow, tops []*xrefcache.TierWithOrderedPolicies) string {
	return PreviewActionUnknown
}
