package cache

import (
	log "github.com/sirupsen/logrus"

	"strings"

	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/libcalico-go/lib/apis/v3"
	bapi "github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/set"
	"github.com/tigera/calicoq/web/pkg/querycache/api"
	"github.com/tigera/calicoq/web/pkg/querycache/dispatcherv1v3"
	"github.com/tigera/calicoq/web/pkg/querycache/labelhandler"
)

type PolicyCache interface {
	TotalPolicies() api.PolicyCounts
	OnUpdate(update dispatcherv1v3.Update)
	GetPolicy(model.Key) api.Policy
	GetTier(model.Key) api.Tier
	GetOrderedPolicies(set.Set) []api.Tier
	PolicyEndpointMatch(matchType labelhandler.MatchType, selector labelhandler.SelectorID, endpoint model.Key)
}

func NewPolicyCache() PolicyCache {
	return &policyCache{
		globalNetworkPolicies: make(map[model.Key]*policyData, 0),
		networkPolicies:       make(map[model.Key]*policyData, 0),
		tiers:                 make(map[string]*tierData, 0),
		policySorter:          calc.NewPolicySorter(),
	}
}

type policyCache struct {
	globalNetworkPolicies map[model.Key]*policyData
	networkPolicies       map[model.Key]*policyData
	tiers                 map[string]*tierData
	policySorter          *calc.PolicySorter
	orderedTiers          []api.Tier
}

func (c *policyCache) OnUpdate(update dispatcherv1v3.Update) {
	uv1 := update.UpdateV1
	uv3 := update.UpdateV3

	// Manage our internal tier and policy cache first.
	switch v1k := uv1.Key.(type) {
	case model.TierKey:
		name := v1k.Name
		switch uv3.UpdateType {
		case bapi.UpdateTypeKVNew:
			c.tiers[name] = &tierData{
				name:     name,
				resource: uv3.Value.(api.Resource),
			}
		case bapi.UpdateTypeKVUpdated:
			c.tiers[name].resource = uv3.Value.(api.Resource)
		case bapi.UpdateTypeKVDeleted:
			delete(c.tiers, name)
		}
	case model.PolicyKey:
		m := c.getMap(uv3.Key)
		if m == nil {
			return
		}
		switch uv3.UpdateType {
		case bapi.UpdateTypeKVNew:
			pv1 := uv1.Value.(*model.Policy)
			pd := &policyData{
				resource: uv3.Value.(api.Resource),
			}
			pd.ruleEndpoints.Ingress = make([]api.RuleDirection, len(pv1.InboundRules))
			pd.ruleEndpoints.Egress = make([]api.RuleDirection, len(pv1.OutboundRules))
			m[uv3.Key] = pd
		case bapi.UpdateTypeKVUpdated:
			pv1 := uv1.Value.(*model.Policy)
			existing := m[uv3.Key]
			existing.resource = uv3.Value.(api.Resource)
			// Extend or shrink our rules slices if necessary.
			deltaIngress := len(pv1.InboundRules) - len(existing.ruleEndpoints.Ingress)
			deltaEgress := len(pv1.OutboundRules) - len(existing.ruleEndpoints.Egress)
			if deltaIngress > 0 {
				existing.ruleEndpoints.Ingress = append(
					existing.ruleEndpoints.Ingress,
					make([]api.RuleDirection, deltaIngress)...,
				)
			} else if deltaEgress < 0 {
				existing.ruleEndpoints.Ingress = existing.ruleEndpoints.Ingress[:len(pv1.InboundRules)]
			}
			if deltaEgress > 0 {
				existing.ruleEndpoints.Egress = append(
					existing.ruleEndpoints.Egress,
					make([]api.RuleDirection, deltaEgress)...,
				)
			} else if deltaEgress < 0 {
				existing.ruleEndpoints.Egress = existing.ruleEndpoints.Egress[:len(pv1.OutboundRules)]
			}
		case bapi.UpdateTypeKVDeleted:
			delete(m, uv3.Key)
		}
	}

	// Update the policy sorter, invalidating our ordered tiers if the policy order needs
	// recalculating.
	if c.policySorter.OnUpdate(*uv1) {
		c.orderedTiers = nil
	}
}

func (c *policyCache) PolicyEndpointMatch(matchType labelhandler.MatchType, selector labelhandler.SelectorID, epKey model.Key) {
	erk := epKey.(model.ResourceKey)
	epd := c.getPolicy(selector.Policy())
	var epc *api.EndpointCounts
	if selector.IsRule() {
		var recs []api.RuleDirection
		var reec *api.RuleEntity
		if selector.IsIngress() {
			recs = epd.ruleEndpoints.Ingress
		} else {
			recs = epd.ruleEndpoints.Egress
		}
		if selector.Index() > len(recs) {
			// If the rules length has decreased we will get deleted updates for rules
			// that we no longer have cached.
			return
		}
		rec := &recs[selector.Index()]
		if selector.IsSource() {
			reec = &rec.Source
		} else {
			reec = &rec.Destination
		}
		if selector.IsNegated() {
			epc = &reec.NotSelector
		} else {
			epc = &reec.Selector
		}
	} else {
		epc = &epd.endpoints
	}

	switch erk.Kind {
	case v3.KindHostEndpoint:
		epc.NumHostEndpoints += matchTypeToDelta[matchType]
	case v3.KindWorkloadEndpoint:
		epc.NumWorkloadEndpoints += matchTypeToDelta[matchType]
	default:
		log.WithField("key", selector.Policy()).Error("Unexpected resource in event type, expecting a v3 policy type")
	}
}

func (c *policyCache) TotalPolicies() api.PolicyCounts {
	return api.PolicyCounts{
		NumGlobalNetworkPolicies: len(c.globalNetworkPolicies),
		NumNetworkPolicies:       len(c.networkPolicies),
	}
}

func (c *policyCache) GetPolicy(key model.Key) api.Policy {
	if policy := c.getPolicy(key); policy != nil {
		return policy
	}
	return nil
}

func (c *policyCache) GetTier(key model.Key) api.Tier {
	c.orderPolicies()
	return c.tiers[key.(model.ResourceKey).Name]
}

func (c *policyCache) GetOrderedPolicies(keys set.Set) []api.Tier {
	c.orderPolicies()
	if keys == nil {
		return c.orderedTiers
	}

	tiers := make([]api.Tier, 0)
	for _, t := range c.orderedTiers {
		td := &tierData{
			resource: t.(*tierData).resource,
			name:     t.(*tierData).name,
		}
		for _, p := range t.GetOrderedPolicies() {
			if keys.Contains(p.(*policyData).getKey()) {
				td.orderedPolicies = append(td.orderedPolicies, p)
			}
		}
		if len(td.orderedPolicies) > 0 {
			tiers = append(tiers, td)
		}
	}

	return tiers
}

func (c *policyCache) orderPolicies() {
	if c.orderedTiers != nil {
		return
	}
	tiers := c.policySorter.Sorted()
	c.orderedTiers = make([]api.Tier, 0, len(tiers))
	for _, tier := range tiers {
		td := c.tiers[tier.Name]
		if td == nil {
			td = &tierData{name: tier.Name}
		}
		c.orderedTiers = append(c.orderedTiers, td)

		for _, policy := range tier.OrderedPolicies {
			policyData := c.getPolicyFromV1Key(policy.Key)
			td.orderedPolicies = append(td.orderedPolicies, policyData)
		}
	}
}

func (c *policyCache) getPolicyFromV1Key(key model.PolicyKey) *policyData {
	parts := strings.Split(key.Name, "/")
	if len(parts) == 1 {
		return c.globalNetworkPolicies[model.ResourceKey{
			Kind: v3.KindGlobalNetworkPolicy,
			Name: parts[0],
		}]
	}
	return c.networkPolicies[model.ResourceKey{
		Kind:      v3.KindNetworkPolicy,
		Namespace: parts[0],
		Name:      parts[1],
	}]
}

func (c *policyCache) getPolicy(key model.Key) *policyData {
	m := c.getMap(key)
	if m == nil {
		return nil
	}
	return m[key]
}

func (c *policyCache) getMap(polKey model.Key) map[model.Key]*policyData {
	if rKey, ok := polKey.(model.ResourceKey); ok {
		switch rKey.Kind {
		case v3.KindGlobalNetworkPolicy:
			return c.globalNetworkPolicies
		case v3.KindNetworkPolicy:
			return c.networkPolicies
		}
	}

	log.WithField("key", polKey).Error("Unexpected resource in event type, expecting a v3 policy type")
	return nil
}

// policyData is used to hold policy data in the cache, and also implements the Policy interface
// for returning on queries.
type policyData struct {
	resource      api.Resource
	endpoints     api.EndpointCounts
	ruleEndpoints api.Rule
}

func (d *policyData) GetEndpointCounts() api.EndpointCounts {
	return d.endpoints
}

func (d *policyData) GetRuleEndpointCounts() api.Rule {
	return d.ruleEndpoints
}

func (d *policyData) GetResource() api.Resource {
	return d.resource
}

func (d *policyData) GetTier() string {
	switch r := d.resource.(type) {
	case *v3.NetworkPolicy:
		return r.Spec.Tier
	case *v3.GlobalNetworkPolicy:
		return r.Spec.Tier
	}
	return ""
}

func (d *policyData) getKey() model.Key {
	return model.ResourceKey{
		Kind:      d.resource.GetObjectKind().GroupVersionKind().Kind,
		Name:      d.resource.GetObjectMeta().GetName(),
		Namespace: d.resource.GetObjectMeta().GetNamespace(),
	}
}

// tierData is used to hold policy data in the cache, and also implements the Policy interface
// for returning on queries.
type tierData struct {
	name            string
	resource        api.Resource
	orderedPolicies []api.Policy
}

func (d *tierData) GetOrderedPolicies() []api.Policy {
	return d.orderedPolicies
}

func (d *tierData) GetName() string {
	return d.name
}

func (d *tierData) GetResource() api.Resource {
	return d.resource
}
