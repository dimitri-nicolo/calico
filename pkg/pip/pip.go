package pip

import (
	"context"
	"math/rand"
	"strings"

	log "github.com/sirupsen/logrus"

	libv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
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
	s.xc.RegisterInScopeEndpoints(&libv3.EndpointsSelection{
		Selector: "all()",
	})

	// Load the initial set of policy.
	if err := s.loadInitialPolicy(); err != nil {
		return flows, err
	}

	// apply the preview changes
	if err := s.applyPolicyChanges(npcs); err != nil {
		return flows, err
	}

	// Set in-sync, so we get updates as we feed in the endpoints.
	s.xc.OnStatusUpdate(syncer.NewStatusUpdateComplete())

	// create a selector set which is used later to determine if each flow is impacted by this policy change.
	ss := NewSelectorSet(npcs)

	var retFlows = []flow.Flow{}
	for _, f := range flows {
		clog := log.WithFields(log.Fields{
			"flowSrc":  f.Src_name,
			"flowDest": f.Dest_name,
		})

		// before we process this endpoint, check if it's even selected by any of our input policies.
		if !ss.anySelectorSelects(f.Src_labels) && !!ss.anySelectorSelects(f.Dest_labels) {
			clog.Debug("skipping flow because no policy applies to it")
			continue
		}

		// each flow falls under one of three categories:
		// 1. the source is an endpoint
		// 2. the dest is a endpoint
		// 3. both the src & dest are endpoints
		// For the third case, the policymap of the request leaving the source is different
		// than the policymap of the request arriving at the dest. as such, we'll check
		// the flow against the source first to see if it would be allowed. if it is, we
		// still have to check the dest next.
		var predictedAction string
		orderedTiersAndPolicies := s.xc.GetOrderedTiersAndPolicies()

		// if the flow came from the cluster, see if it would have left the source.
		if f.Src_type != flow.EndpointTypeNet {
			if srcEp := getSrcResource(f); srcEp != nil {
				predictedAction = computeAction(f, orderedTiersAndPolicies)
			} else {
				clog.WithField("srcType", f.Src_type).Warn("skipping flow with unexpected source type")
				continue
			}
		}

		// a blank predictedAction
		if predictedAction == "" ||
			predictedAction == PreviewActionAllow ||
			predictedAction == PreviewActionPass {
			if destEp := getDstResource(f); destEp != nil {
				predictedAction = computeAction(f, orderedTiersAndPolicies)
			} else {
				clog.WithField("destType", f.Dest_type).Warn("skipping flow with unexpected dest type")
				continue
			}
		}

		clog.WithField("predictedAction", predictedAction).Debug("computed flow action")
		f.Action = predictedAction
		retFlows = append(retFlows, f)
	}

	return retFlows, nil
}

func (s *pip) applyPolicyChanges(npcs []NetworkPolicyChange) error {
	for _, npc := range npcs {
		switch npc.ChangeAction {
		case "update":
			s.xc.OnUpdates([]syncer.Update{{
				Type:       syncer.UpdateTypeSet,
				Resource:   &npc.NetworkPolicy,
				ResourceID: resources.GetResourceID(&npc.NetworkPolicy),
			}})

		case "delete":
			s.xc.OnUpdates([]syncer.Update{{
				Type:       syncer.UpdateTypeDeleted,
				Resource:   &npc.NetworkPolicy,
				ResourceID: resources.GetResourceID(&npc.NetworkPolicy),
			}})
		case "create":
			s.xc.OnUpdates([]syncer.Update{{
				Type:       syncer.UpdateTypeSet,
				Resource:   &npc.NetworkPolicy,
				ResourceID: resources.GetResourceID(&npc.NetworkPolicy),
			}})
		}
	}
	return nil
}

func (s *pip) loadInitialPolicy() error {
	list, err := s.listSrc.RetrieveList(resources.TypeCalicoTiers)
	if err != nil {
		return err
	}
	for i := range list.ResourceList.(*v3.TierList).Items {
		clientRes := &list.ResourceList.(*v3.TierList).Items[i]
		res := &libv3.Tier{
			TypeMeta:   resources.TypeCalicoTiers,
			ObjectMeta: clientRes.ObjectMeta,
			Spec:       clientRes.Spec,
		}
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
		res.TypeMeta = resources.TypeK8sNetworkPolicies
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
		clientRes := &list.ResourceList.(*v3.NetworkPolicyList).Items[i]
		res := &libv3.NetworkPolicy{
			TypeMeta:   resources.TypeCalicoNetworkPolicies,
			ObjectMeta: clientRes.ObjectMeta,
			Spec:       clientRes.Spec,
		}
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
		clientRes := &list.ResourceList.(*v3.GlobalNetworkPolicyList).Items[i]
		res := &libv3.GlobalNetworkPolicy{
			TypeMeta:   resources.TypeCalicoGlobalNetworkPolicies,
			ObjectMeta: clientRes.ObjectMeta,
			Spec:       clientRes.Spec,
		}
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
		clientRes := &list.ResourceList.(*v3.GlobalNetworkSetList).Items[i]
		res := &libv3.GlobalNetworkSet{
			TypeMeta:   resources.TypeCalicoGlobalNetworkSets,
			ObjectMeta: clientRes.ObjectMeta,
			Spec:       clientRes.Spec,
		}
		s.xc.OnUpdates([]syncer.Update{{
			Type:       syncer.UpdateTypeSet,
			Resource:   res,
			ResourceID: resources.GetResourceID(res),
		}})
	}

	return nil
}

// getSrcResource generates a 'pretend' resource from the source data of a flow.
// It returns nil if the source is not a known resource type or if the traffic originated
// from outside the cluster.
func getSrcResource(f flow.Flow) resources.Resource {
	switch f.Src_type {
	case flow.EndpointTypeWep:
		return &corev1.Pod{
			ObjectMeta: v1.ObjectMeta{
				Name:      f.Src_name,
				Namespace: f.Src_NS,
				Labels:    f.Src_labels,
			},
			Status: corev1.PodStatus{
				PodIP: f.Src_IP,
			},
		}
	case flow.EndpointTypeHep:
		return &libv3.HostEndpoint{
			ObjectMeta: v1.ObjectMeta{
				Name:      f.Src_name,
				Namespace: f.Src_NS,
				Labels:    f.Src_labels,
			},
			Spec: libv3.HostEndpointSpec{
				Node: strings.TrimSuffix(f.Src_name, "-*"),
			},
		}
	}
	return nil
}

// getDstResource generates a 'pretend' resource from the destination data of a flow.
// It returns nil if the destination is not a known resource type or if the traffic was sent
// outside the cluster.
func getDstResource(f flow.Flow) resources.Resource {
	switch f.Dest_type {
	case flow.EndpointTypeWep:
		return &corev1.Pod{
			ObjectMeta: v1.ObjectMeta{
				Name:      f.Dest_name,
				Namespace: f.Dest_NS,
				Labels:    f.Dest_labels,
			},
			Status: corev1.PodStatus{
				PodIP: f.Dest_IP,
			},
		}
	case flow.EndpointTypeHep:
		return &libv3.HostEndpoint{
			ObjectMeta: v1.ObjectMeta{
				Name:      f.Dest_name,
				Namespace: f.Dest_NS,
				Labels:    f.Dest_labels,
			},
			Spec: libv3.HostEndpointSpec{
				Node: strings.TrimSuffix(f.Src_name, "-*"),
			},
		}
	}
	return nil
}

func buildSelector(npcs []NetworkPolicyChange) string {
	// TODO: loop through policy change, create a set, and union the results in this format: "() | () | ()"
	return "all()"
}

// TODO: compute action instead of returning a random action
func computeAction(f flow.Flow, tops []*xrefcache.TierWithOrderedPolicies) string {
	return []string{PreviewActionAllow, PreviewActionDeny, PreviewActionPass, PreviewActionUnknown}[rand.Int()%4]
}
