// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package client

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/apis/v3"
	bapi "github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/backend/syncersv1/updateprocessors"
	"github.com/projectcalico/libcalico-go/lib/backend/watchersyncer"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/errors"
	"github.com/projectcalico/libcalico-go/lib/set"
	"github.com/tigera/compliance/pkg/querycache/api"
	"github.com/tigera/compliance/pkg/querycache/cache"
	"github.com/tigera/compliance/pkg/querycache/dispatcherv1v3"
	"github.com/tigera/compliance/pkg/querycache/labelhandler"
)

// NewQueryInterface returns a queryable resource cache.
func NewQueryInterface(ci clientv3.Interface) QueryInterface {
	cq := &cachedQuery{
		policies:          cache.NewPoliciesCache(),
		endpoints:         cache.NewEndpointsCache(),
		nodes:             cache.NewNodeCache(),
		networksets:       cache.NewNetworkSetCache(),
		polEplabelHandler: labelhandler.NewLabelHandler(),
		wepConverter: dispatcherv1v3.NewConverterFromSyncerUpdateProcessor(
			updateprocessors.NewWorkloadEndpointUpdateProcessor(),
		),
		gnpConverter: dispatcherv1v3.NewConverterFromSyncerUpdateProcessor(
			updateprocessors.NewGlobalNetworkPolicyUpdateProcessor(),
		),
		npConverter: dispatcherv1v3.NewConverterFromSyncerUpdateProcessor(
			updateprocessors.NewNetworkPolicyUpdateProcessor(),
		),
	}

	// We want to watch the v3 resource types (so that we can cache the actual configured
	// data), but we need the v1 version of several of the resources to feed into the various
	// Felix helper helper modules. The dispatcherv1v3 converts the updates from v3 to v1 (using
	// the watchersyncer update processor functionality used by Felix), and fans out an update
	// containing both the v1 and v3 data to any handlers registered for notifications.
	dispatcherTypes := []dispatcherv1v3.Resource{
		{
			// We need to convert the GNP for use with the policy sorter, and to get the
			// correct selectors for the labelhandler.
			Kind:      v3.KindGlobalNetworkPolicy,
			Converter: cq.gnpConverter,
		},
		{
			// We need to convert the NP for use with the policy sorter, and to get the
			// correct selectors for the labelhandler.
			Kind:      v3.KindNetworkPolicy,
			Converter: cq.npConverter,
		},
		{
			// We need to convert the Tier for use with the policy sorter.
			Kind: v3.KindTier,
			Converter: dispatcherv1v3.NewConverterFromSyncerUpdateProcessor(
				updateprocessors.NewTierUpdateProcessor(),
			),
		},
		{
			// We need to convert the WEP to get the corrected labels for the labelhandler.
			Kind:      v3.KindWorkloadEndpoint,
			Converter: cq.wepConverter,
		},
		{
			Kind: v3.KindHostEndpoint,
		},
		{
			Kind: v3.KindProfile,
		},
		{
			// We don't need these to be converted.
			Kind: v3.KindNode,
		},
		{
			Kind: v3.KindGlobalNetworkSet,
		},
	}
	dispatcher := dispatcherv1v3.New(dispatcherTypes)

	// Register the caches for updates from the dispatcher.
	cq.endpoints.RegisterWithDispatcher(dispatcher)
	cq.policies.RegisterWithDispatcher(dispatcher)
	cq.nodes.RegisterWithDispatcher(dispatcher)
	cq.networksets.RegisterWithDispatcher(dispatcher)

	// Register the label handlers *after* the actual resource caches (since the
	// resource caches register for updates from the label handler)
	cq.polEplabelHandler.RegisterWithDispatcher(dispatcher)

	// Register the policy and endpoint caches for updates from the label handler.
	cq.endpoints.RegisterWithLabelHandler(cq.polEplabelHandler)
	cq.policies.RegisterWithLabelHandler(cq.polEplabelHandler)

	// Create a SyncerQueryHandler which ensures syncer updates and query requests are
	// serialized. This handler will pass syncer updates to the dispatcher (see below),
	// and pass query update into this cahcedQuery instance.
	scb, qi := NewSerializedSyncerQuery(dispatcher, cq)

	// Create a watchersyncer for the same resource types that the dispatcher handles.
	// The syncer will call into the SyncerQuerySerializer.
	wsResourceTypes := make([]watchersyncer.ResourceType, 0, len(dispatcherTypes))
	for _, r := range dispatcherTypes {
		wsResourceTypes = append(wsResourceTypes, watchersyncer.ResourceType{
			ListInterface: model.ResourceListOptions{Kind: r.Kind},
		})
	}
	syncer := watchersyncer.New(
		ci.(backend).Backend(),
		wsResourceTypes,
		scb,
	)

	// Start the syncer and return the synchronized query interface.
	syncer.Start()
	return qi
}

// We know the calico clients implement the Backend() method, so define an interface
// to allow us to access that method.
type backend interface {
	Backend() bapi.Client
}

// cachedQuery implements the QueryInterface.
type cachedQuery struct {
	// A cache of all loaded policy (keyed off name) and endpoint resources (keyed off key).
	// The cache includes Tiers, GNPs and NPs.
	policies cache.PoliciesCache

	// A cache of all loaded endpoints. The cache includes both HEPs and WEPs.
	endpoints cache.EndpointsCache

	// A cache of all loaded nodes. The cache includes directly configured node resources, as
	// well as those configured indirectly via WEPs and HEPs.
	nodes cache.NodeCache

	// A cache of all loaded networksets.
	networksets cache.NetworkSetCache

	// polEplabelHandler handles the relationship between policy and rule selectors
	// and endpoint and networkset labels.
	polEplabelHandler labelhandler.Interface

	// Converters for some of the resources.
	wepConverter dispatcherv1v3.Converter
	gnpConverter dispatcherv1v3.Converter
	npConverter  dispatcherv1v3.Converter
}

// RunQuery is a callback from the SyncerQuerySerializer to run a query.  It is guaranteed
// not to be called at the same time as OnUpdates and OnStatusUpdated.
func (c *cachedQuery) RunQuery(cxt context.Context, req interface{}) (interface{}, error) {
	switch qreq := req.(type) {
	case QueryClusterReq:
		return c.runQuerySummary(cxt, qreq)
	case QueryEndpointsReq:
		return c.runQueryEndpoints(cxt, qreq)
	case QueryPoliciesReq:
		return c.runQueryPolicies(cxt, qreq)
	case QueryNodesReq:
		return c.runQueryNodes(cxt, qreq)
	default:
		return nil, fmt.Errorf("unhandled query type: %#v", req)
	}
}

func (c *cachedQuery) runQuerySummary(cxt context.Context, req QueryClusterReq) (*QueryClusterResp, error) {
	// Get the summary counts for the endpoints, summing up the per namespace counts.
	hepSummary := c.endpoints.TotalHostEndpoints()
	totWEP := 0
	numUnlabelledWEP := 0
	numUnprotectedWEP := 0
	namespaceSummary := make(map[string]QueryClusterNamespaceCounts)
	for ns, weps := range c.endpoints.TotalWorkloadEndpointsByNamespace() {
		totWEP += weps.Total
		numUnlabelledWEP += weps.NumWithNoLabels
		numUnprotectedWEP += weps.NumWithNoPolicies
		namespaceSummary[ns] = QueryClusterNamespaceCounts{
			NumWorkloadEndpoints:            weps.Total,
			NumUnlabelledWorkloadEndpoints:  weps.NumWithNoLabels,
			NumUnprotectedWorkloadEndpoints: weps.NumWithNoPolicies,
		}
	}

	// Get the summary counts for policies, summing up the per namespace counts.
	gnpSummary := c.policies.TotalGlobalNetworkPolicies()
	totNP := 0
	numUnmatchedNP := 0
	for ns, nps := range c.policies.TotalNetworkPoliciesByNamespace() {
		totNP += nps.Total
		numUnmatchedNP += nps.NumUnmatched

		// Update the existing entry with the NP counts, or create a new one if it doesn't exist.
		counts := namespaceSummary[ns]
		counts.NumNetworkPolicies = nps.Total
		counts.NumUnmatchedNetworkPolicies = nps.NumUnmatched
		namespaceSummary[ns] = counts
	}

	resp := &QueryClusterResp{
		NumGlobalNetworkPolicies:          gnpSummary.Total,
		NumNetworkPolicies:                totNP,
		NumHostEndpoints:                  hepSummary.Total,
		NumWorkloadEndpoints:              totWEP,
		NumUnmatchedGlobalNetworkPolicies: gnpSummary.NumUnmatched,
		NumUnmatchedNetworkPolicies:       numUnmatchedNP,
		NumUnlabelledHostEndpoints:        hepSummary.NumWithNoLabels,
		NumUnlabelledWorkloadEndpoints:    numUnlabelledWEP,
		NumUnprotectedHostEndpoints:       hepSummary.NumWithNoPolicies,
		NumUnprotectedWorkloadEndpoints:   numUnprotectedWEP,
		NumNodes:                          c.nodes.TotalNodes(),
		NumNodesWithNoEndpoints:           c.nodes.TotalNodesWithNoEndpoints(),
		NumNodesWithNoWorkloadEndpoints:   c.nodes.TotalNodesWithNoWorkloadEndpoints(),
		NumNodesWithNoHostEndpoints:       c.nodes.TotalNodesWithNoHostEndpoints(),
		NamespaceCounts:                   namespaceSummary,
	}
	return resp, nil
}

func (c *cachedQuery) runQueryEndpoints(cxt context.Context, req QueryEndpointsReq) (*QueryEndpointsResp, error) {
	// If an endpoint was specified, just return that (if it exists).
	if req.Endpoint != nil {
		ep := c.endpoints.GetEndpoint(req.Endpoint)
		if ep == nil {
			// Endpoint does not exist, return no results.
			return nil, errors.ErrorResourceDoesNotExist{
				Identifier: req.Endpoint,
			}
		}
		return &QueryEndpointsResp{
			Count: 1,
			Items: []Endpoint{
				*c.apiEndpointToQueryEndpoint(ep),
			},
		}, nil
	}

	var err error
	selector := req.Selector
	if req.Policy != nil {
		selector, err = c.getPolicySelector(
			req.Policy, req.RuleDirection, req.RuleIndex, req.RuleEntity, req.RuleNegatedSelector,
		)
		if err != nil {
			return nil, err
		}
	}

	epkeys, err := c.polEplabelHandler.QueryEndpoints(selector)
	if err != nil {
		return nil, err
	}

	items := make([]Endpoint, 0, len(epkeys))
	for _, result := range epkeys {
		ep := c.endpoints.GetEndpoint(result)
		if req.Node != "" && ep.GetNode() != req.Node {
			continue
		}
		if req.Unprotected && ep.IsProtected() {
			continue
		}
		if req.Unlabelled && ep.IsLabelled() {
			continue
		}
		items = append(items, *c.apiEndpointToQueryEndpoint(ep))
	}
	sortEndpoints(items, req.Sort)

	count := len(items)
	if req.Page != nil {
		fromIdx, toIdx, err := getPageFromToIdx(req.Page, count)
		if err != nil {
			return nil, err
		}
		items = items[fromIdx:toIdx]
	}

	return &QueryEndpointsResp{
		Count: count,
		Items: items,
	}, nil
}

func (c *cachedQuery) apiEndpointToQueryEndpoint(ep api.Endpoint) *Endpoint {
	pc := ep.GetPolicyCounts()
	res := ep.GetResource()
	e := &Endpoint{
		Kind:      res.GetObjectKind().GroupVersionKind().Kind,
		Name:      res.GetObjectMeta().GetName(),
		Namespace: res.GetObjectMeta().GetNamespace(),
		Node:      ep.GetNode(),
		NumGlobalNetworkPolicies: pc.NumGlobalNetworkPolicies,
		NumNetworkPolicies:       pc.NumNetworkPolicies,
		Labels:                   res.GetObjectMeta().GetLabels(),
	}

	switch rt := res.(type) {
	case *v3.WorkloadEndpoint:
		e.Workload = rt.Spec.Workload
		e.Orchestrator = rt.Spec.Orchestrator
		e.Pod = rt.Spec.Pod
		e.InterfaceName = rt.Spec.InterfaceName
		e.IPNetworks = rt.Spec.IPNetworks
	case *v3.HostEndpoint:
		e.InterfaceName = rt.Spec.InterfaceName
		e.IPNetworks = rt.Spec.ExpectedIPs
	}

	return e
}

func (c *cachedQuery) runQueryPolicies(cxt context.Context, req QueryPoliciesReq) (*QueryPoliciesResp, error) {
	// If a policy was specified, just return that (if it exists).
	if req.Policy != nil {
		p := c.policies.GetPolicy(req.Policy)
		if p == nil {
			return nil, errors.ErrorResourceDoesNotExist{
				Identifier: req.Policy,
			}
		}
		return &QueryPoliciesResp{
			Count: 1,
			Items: []Policy{
				*c.apiPolicyToQueryPolicy(p, 0),
			},
		}, nil
	}

	var policySet set.Set

	// If an endpoint has been specified, determine the labels on the endpoint.
	if req.Endpoint != nil {
		// Endpoint is requested, get the labels and profiles and calculate the matching
		// policies.
		labels, profiles, err := c.getEndpointLabelsAndProfiles(req.Endpoint)
		if err != nil {
			return nil, err
		}
		policySet = c.queryPoliciesByLabel(labels, profiles, nil)
		log.WithField("policySet", policySet).Debug("Endpoint query")
	}

	// If a networkset has been specified, determine the labels on the networkset.
	if req.NetworkSet != nil {
		// NetworkSet is requested, get the labels and calculate the matching
		// policies.
		labels, err := c.getNetworkSetLabels(req.NetworkSet)
		if err != nil {
			return nil, err
		}
		// Query policies for the rule selectors matching the networkset labels.
		policySet = c.queryPoliciesByLabelMatchingRule(labels, nil, policySet)
		log.WithField("policySet", policySet).Debug("NetworkSet query")
	}

	if len(req.Labels) > 0 {
		// Labels have been specified, calculate the matching policies. If we matched on endpoint
		// then only filter in the common elements.
		policySet = c.queryPoliciesByLabel(req.Labels, nil, policySet)
		log.WithField("policySet", policySet).Debug("Labels query")
	}

	var ordered []api.Tier
	if policySet == nil && req.Tier != "" {
		// If a tier has been specified, but no other query parameters then we can request just
		// the policies associated with a Tier as a minor finesse.
		tier := c.policies.GetTier(model.ResourceKey{
			Kind: v3.KindTier,
			Name: req.Tier,
		})
		if tier != nil {
			ordered = append(ordered, tier)
		}
	} else {
		// Get the required policies ordered by tier and policy Order parameter. If the policy set is
		// empty this will return all policies across all tiers.
		ordered = c.policies.GetOrderedPolicies(policySet)
	}
	log.WithField("ordered", ordered).Info("Pre filter list")

	// Compile a flat list of policies from the ordered set, filtering out based on the remaining
	// request parameters.
	items := make([]Policy, 0)
	for _, t := range ordered {
		op := t.GetOrderedPolicies()
		// If a tier is specified, filter out policies that are not in the requested tier.
		if req.Tier != "" && t.GetName() != req.Tier {
			log.Info("Filter out wrong tier")
			continue
		}

		for _, p := range op {
			if req.Unmatched && !p.IsUnmatched() {
				log.Info("Filter out matched policy")
				continue
			}
			items = append(items, *c.apiPolicyToQueryPolicy(p, len(items)))
		}
	}

	if req.Sort != nil {
		// User has specified a different sort order, so re-order the policies according to the sort fields.
		sortPolicies(items, req.Sort)
	}

	// If we are paging results then return the required page-worths of results.
	count := len(items)
	if req.Page != nil {
		fromIdx, toIdx, err := getPageFromToIdx(req.Page, count)
		if err != nil {
			return nil, err
		}
		items = items[fromIdx:toIdx]
	}

	return &QueryPoliciesResp{
		Count: count,
		Items: items,
	}, nil
}

func (c *cachedQuery) apiPolicyToQueryPolicy(p api.Policy, idx int) *Policy {
	ep := p.GetEndpointCounts()
	res := p.GetResource()
	return &Policy{
		Index:                idx,
		Name:                 res.GetObjectMeta().GetName(),
		Namespace:            res.GetObjectMeta().GetNamespace(),
		Kind:                 res.GetObjectKind().GroupVersionKind().Kind,
		Tier:                 p.GetTier(),
		NumHostEndpoints:     ep.NumHostEndpoints,
		NumWorkloadEndpoints: ep.NumWorkloadEndpoints,
		Ingress:              c.convertRules(p.GetRuleEndpointCounts().Ingress),
		Egress:               c.convertRules(p.GetRuleEndpointCounts().Egress),
	}
}

func (c *cachedQuery) convertRules(apiRules []api.RuleDirection) []RuleDirection {
	r := make([]RuleDirection, len(apiRules))
	for i, ar := range apiRules {
		r[i] = RuleDirection{
			Source: RuleEntity{
				NumWorkloadEndpoints: ar.Source.NumWorkloadEndpoints,
				NumHostEndpoints:     ar.Source.NumHostEndpoints,
			},
			Destination: RuleEntity{
				NumWorkloadEndpoints: ar.Destination.NumWorkloadEndpoints,
				NumHostEndpoints:     ar.Destination.NumHostEndpoints,
			},
		}
	}
	return r
}

func (c *cachedQuery) runQueryNodes(cxt context.Context, req QueryNodesReq) (*QueryNodesResp, error) {
	// If a policy was specified, just return that (if it exists).
	if req.Node != nil {
		n := c.nodes.GetNode(req.Node.(model.ResourceKey).Name)
		if n == nil {
			// Node does not exist.
			return nil, errors.ErrorResourceDoesNotExist{
				Identifier: req.Node,
			}
		}
		return &QueryNodesResp{
			Count: 1,
			Items: []Node{
				*c.apiNodeToQueryNode(n),
			},
		}, nil
	}

	// Sort the nodes by name (which is the only current option).
	nodes := c.nodes.GetNodes()

	items := make([]Node, 0, len(nodes))
	for _, n := range nodes {
		items = append(items, *c.apiNodeToQueryNode(n))
	}
	sortNodes(items, req.Sort)

	// If we are paging the results then only keep the required page worth of results.
	count := len(nodes)
	if req.Page != nil {
		fromIdx, toIdx, err := getPageFromToIdx(req.Page, count)
		if err != nil {
			return nil, err
		}
		items = items[fromIdx:toIdx]
	}

	return &QueryNodesResp{
		Count: count,
		Items: items,
	}, nil
}

func (c *cachedQuery) apiNodeToQueryNode(n api.Node) *Node {
	ep := n.GetEndpointCounts()
	node := &Node{
		Name:                 n.GetName(),
		NumHostEndpoints:     ep.NumHostEndpoints,
		NumWorkloadEndpoints: ep.NumWorkloadEndpoints,
	}
	r := n.GetResource()
	if r != nil {
		nr := r.(*v3.Node)
		if nr.Spec.BGP != nil {
			if len(nr.Spec.BGP.IPv4Address) > 0 {
				node.BGPIPAddresses = append(node.BGPIPAddresses, nr.Spec.BGP.IPv4Address)
			}
			if len(nr.Spec.BGP.IPv6Address) > 0 {
				node.BGPIPAddresses = append(node.BGPIPAddresses, nr.Spec.BGP.IPv6Address)
			}
		}
	}
	return node
}

func (c *cachedQuery) getEndpointLabelsAndProfiles(key model.Key) (map[string]string, []string, error) {
	ep := c.endpoints.GetEndpoint(key)
	if ep == nil {
		return nil, nil, errors.ErrorResourceDoesNotExist{
			Identifier: key,
		}
	}

	// For host endpoints, return the labels unchanged.
	var labels map[string]string
	var profiles []string
	if hep, ok := ep.GetResource().(*v3.HostEndpoint); ok {
		labels = hep.Labels
		profiles = hep.Spec.Profiles
	} else {
		// For workload endpoints we need to convert the resource to ensure our labels are
		// cleaned of any potentially conflicting overridden values.
		epv1 := c.wepConverter.ConvertV3ToV1(&bapi.Update{
			UpdateType: bapi.UpdateTypeKVNew,
			KVPair: model.KVPair{
				Key:   key,
				Value: ep.GetResource(),
			},
		})
		// If the WEP has been filtered out, then the value may be nil.
		if epv1 == nil {
			return nil, nil, fmt.Errorf("endpoint %s is not valid: no policy is enforced on this endpoint", key)
		}
		wep := epv1.Value.(*model.WorkloadEndpoint)
		labels = wep.Labels
		profiles = wep.ProfileIDs
	}

	// If labels are nil, convert to an empty map.
	if labels == nil {
		labels = make(map[string]string)
	}

	return labels, profiles, nil
}

func (c *cachedQuery) getNetworkSetLabels(key model.Key) (map[string]string, error) {
	netset := c.networksets.GetNetworkSet(key)
	if netset == nil {
		return nil, errors.ErrorResourceDoesNotExist{
			Identifier: key,
		}
	}

	return netset.GetObjectMeta().GetLabels(), nil
}

func (c *cachedQuery) queryPoliciesByLabel(labels map[string]string, profiles []string, filterIn set.Set) set.Set {
	policies := c.polEplabelHandler.QueryPolicies(labels, profiles)

	// Filter out the rule matches, and only filter in those in the supplied set (if supplied).
	results := set.New()
	for _, p := range policies {
		if filterIn != nil && !filterIn.Contains(p) {
			continue
		}
		results.Add(p)
	}
	log.WithField("NumResults", results.Len()).Info("Returning policies from label query")
	return results
}

func (c *cachedQuery) queryPoliciesByLabelMatchingRule(labels map[string]string, profiles []string, filterIn set.Set) set.Set {
	selectors := c.polEplabelHandler.QueryRuleSelectors(labels, profiles)

	// Convert the selectors to a set of the policy matches.
	results := set.New()
	// Iterate over all the selectors and join the sets
	for _, selector := range selectors {
		matching := c.policies.GetPolicyKeySetByRuleSelector(selector)
		matching.Iter(func(item interface{}) error {
			converted, ok := item.(model.Key)
			if !ok {
				return fmt.Errorf("object: %v stored against selector: %s is not a policy key", item, selector)
			}

			// Only filter policies in if they are in the supplied set (if supplied).
			if filterIn != nil && !filterIn.Contains(converted) {
				return nil
			}

			results.Add(converted)
			return nil
		})
	}
	log.WithField("NumResults", results.Len()).Info("Returning policies from label query against rule selectors")
	return results
}

func (c *cachedQuery) getPolicySelector(key model.Key, direction string, index int, entity string, negatedSelector bool) (string, error) {
	p := c.policies.GetPolicy(key)
	if p == nil {
		return "", errors.ErrorResourceDoesNotExist{
			Identifier: key,
		}
	}
	pr := p.GetResource()

	// We need to convert the policy to the v1 equivalent so that we get the correct converted
	// selector.
	var converted *bapi.Update
	switch pr.GetObjectKind().GroupVersionKind().Kind {
	case v3.KindNetworkPolicy:
		converted = c.npConverter.ConvertV3ToV1(&bapi.Update{
			UpdateType: bapi.UpdateTypeKVNew,
			KVPair: model.KVPair{
				Key:   key,
				Value: pr,
			},
		})
	case v3.KindGlobalNetworkPolicy:
		converted = c.gnpConverter.ConvertV3ToV1(&bapi.Update{
			UpdateType: bapi.UpdateTypeKVNew,
			KVPair: model.KVPair{
				Key:   key,
				Value: pr,
			},
		})
	}

	if converted == nil {
		return "", fmt.Errorf("unable to process resource: %s", key.String())
	}

	// Extract selector from the indexed rule. This safely handles bad input parameters since they
	// are provided over the API.
	pv1 := converted.Value.(*model.Policy)
	var rd []model.Rule
	switch direction {
	case "":
		return pv1.Selector, nil
	case RuleDirectionIngress:
		rd = pv1.InboundRules
	case RuleDirectionEgress:
		rd = pv1.OutboundRules
	default:
		return "", fmt.Errorf("rule direction not valid: %s", direction)
	}

	if len(rd) == 0 {
		return "", fmt.Errorf("there are no %s rules configured", direction)
	}

	if index < 0 || index >= len(rd) {
		return "", fmt.Errorf("rule index out of range, expected: 0-%d; requested index: %d", len(rd)-1, index)
	}

	r := rd[index]
	switch entity {
	case RuleEntitySource:
		switch negatedSelector {
		case false:
			return r.SrcSelector, nil
		case true:
			return r.NotSrcSelector, nil
		}
	case RuleEntityDestination:
		switch negatedSelector {
		case false:
			return r.DstSelector, nil
		case true:
			return r.NotDstSelector, nil
		}
	}
	return "", fmt.Errorf("rule entity not valid: %s", entity)
}

func getPageFromToIdx(p *Page, numItems int) (int, int, error) {
	// Perform simple policing of the page number and num per page. This should already be policed
	// by the HTTP server, but we'll police here to be safe.
	perPage := p.NumPerPage
	pageNum := p.PageNum
	if perPage <= 0 {
		return 0, 0, fmt.Errorf("number of results must be >0, requested number: %d", perPage)
	}
	if pageNum < 0 {
		return 0, 0, fmt.Errorf("page number should be an integer >=0, requested number: %d", pageNum)
	}

	// Check if the requested page number is out of range, we can only do this once we collate our results.
	// We don't treat this as an error since it could be a timing window where the number of results has
	// changed. Also, by returning a valid response the consumer is able to find out how any results there
	// are.
	maxPageNum := (numItems - 1) / perPage
	if pageNum > maxPageNum {
		return 0, 0, nil
	}

	// Calculate the from and to indexes from our page number and per page.
	fromIdx := p.PageNum * perPage
	toIdx := fromIdx + perPage

	// Ensure the toIdx does not exceed the length of the slice, capping at numItems if it does.
	if toIdx > numItems {
		toIdx = numItems
	}

	return fromIdx, toIdx, nil
}
