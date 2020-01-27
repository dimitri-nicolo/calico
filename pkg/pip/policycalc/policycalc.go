package policycalc

import (
	"fmt"
	"sort"

	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"

	"github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/resources"

	"github.com/tigera/lma/pkg/api"

	pipcfg "github.com/tigera/es-proxy/pkg/pip/config"
)

// ------
// This file contains all of the struct definitions that are used as input when instantiating a new policy calculator.
// ------

// The tiers containing the ordered set of Calico v3 policy resource types.
type Policy struct {
	Policy resources.Resource
	Staged bool
}

func (p Policy) String() string {
	return fmt.Sprintf("%s; Staged=%v", resources.GetResourceID(p.Policy), p.Staged)
}

type Tier []Policy
type Tiers []Tier

// The consistent set of configuration used for calculating policy impact.
type ResourceData struct {
	Tiers           Tiers
	Namespaces      []*corev1.Namespace
	ServiceAccounts []*corev1.ServiceAccount
}

type Impact struct {
	// The resource was deleted.
	Deleted bool

	// Whether or not to use the staged policy data from the flow log to augment the calculation. This will be true for
	// actual staged policies, or for staged policies that have been converted to enforced policies.
	UseStaged bool

	// The resource was modified. This is used to determine whether data from the flow logs can be used to augment
	// calculated data. If the resource was modified then we cannot use flow log data.
	// Note that for an enforced-staged resource, modified pertains to the staged resource type.
	Modified bool
}

// ImpactedResources is a set of impacts for updated resources.
type ImpactedResources map[v3.ResourceID]Impact

// Add adds a resource to the set of modified resources.
func (m ImpactedResources) Add(rid v3.ResourceID, impact Impact) {
	m[rid] = impact

	// For K8s NP, also add the equivalent Calico NP resource ID.
	//TODO(rlb): This is hacky. Need to rethink how we handle converted resources and the modified resources map.
	if rid.TypeMeta == resources.TypeK8sNetworkPolicies {
		rid = v3.ResourceID{
			TypeMeta:  resources.TypeCalicoNetworkPolicies,
			Namespace: rid.Namespace,
			Name:      "knp.default." + rid.Name,
		}
		m[rid] = impact
	}
}

// Impact returns impact for a particular resource.
func (m ImpactedResources) Impact(r resources.Resource) (Impact, bool) {
	impact, isImpacted := m[resources.GetResourceID(r)]
	return impact, isImpacted
}

// IsModified returns if the resource is modified.
func (m ImpactedResources) IsModified(r resources.Resource) bool {
	return m[resources.GetResourceID(r)].Modified
}

// IsDeleted returns if the resource is deleted.
func (m ImpactedResources) IsDeleted(r resources.Resource) bool {
	return m[resources.GetResourceID(r)].Deleted
}

// UseStaged returns if the flow log data from the staged policy should be used to augment the calculation.
func (m ImpactedResources) UseStaged(r resources.Resource) bool {
	return m[resources.GetResourceID(r)].UseStaged
}

// PolicyCalculator is used to determine the calculated behavior from a configuration change for a given flow.
type PolicyCalculator interface {
	CalculateSource(source *api.Flow) (processed bool, before, after EndpointResponse)
	CalculateDest(dest *api.Flow, srcActionBefore, srcActionAfter api.ActionFlag) (processed bool, before, after EndpointResponse)
}

type EndpointResponse struct {
	// Whether to include the result in the final aggregated data set. For Calico->Calico endpoint flows we may need to
	// massage the data a little:
	// - For source-reported flows whose action changes from denied to allowed or unknown, we explicitly add the
	//   equivalent data at the destination, since the associated flow data should be missing from the original set.
	// - For destination-reported flows whose source action changes from allowed->denied, we remove the flow completely
	//   as it should not get reported.
	// This means the calculation response can have 0, 1 or 2 results to include in the aggregated data.
	Include bool

	// The calculated action flags at the endpoint for the supplied flow.  This can be a combination of ActionFlagAllow
	// and/or ActionFlagDeny with any of ActionFlagFlowLogMatchesCalculated, ActionFlagFlowLogRemovedUncertainty and
	// ActionFlagFlowLogConflictsWithCalculated.
	Action api.ActionFlag

	// The set of policies applied to this flow.
	Policies OrderedPolicyHits
}

type OrderedPolicyHits []api.PolicyHit

func (o OrderedPolicyHits) FlowLogPolicyStrings() []string {
	s := make([]string, 0, len(o))
	for i := range o {
		s = append(s, o[i].ToFlowLogPolicyStrings()...)
	}
	return s
}

// policyCalculator implements the PolicyCalculator interface.
type policyCalculator struct {
	Config    *pipcfg.Config
	Selectors *EndpointSelectorHandler
	Endpoints *EndpointCache
	Ingress   CompiledTierAndPolicyChangeSet
	Egress    CompiledTierAndPolicyChangeSet
}

// NewPolicyCalculator returns a new PolicyCalculator.
func NewPolicyCalculator(
	cfg *pipcfg.Config,
	endpoints *EndpointCache,
	resourceDataBefore *ResourceData,
	resourceDataAfter *ResourceData,
	impacted ImpactedResources,
) PolicyCalculator {
	// Create the selector handler. This is shared by both the before and after matcher factories - this is fine because
	// the labels on the endpoints are not being adjusted, and so a selector will return the same value in the before
	// and after configurations.
	selectors := NewEndpointSelectorHandler()

	// Calculate the before/after ingress/egress compiled tier and policy data.
	ingressBefore, egressBefore := calculateCompiledTiersAndImpactedPolicies(cfg, resourceDataBefore, impacted, selectors, false)
	ingressAfter, egressAfter := calculateCompiledTiersAndImpactedPolicies(cfg, resourceDataAfter, impacted, selectors, true)

	// Create the policyCalculator.
	return &policyCalculator{
		Config:    cfg,
		Selectors: selectors,
		Endpoints: endpoints,
		Ingress: CompiledTierAndPolicyChangeSet{
			Before: ingressBefore,
			After:  ingressAfter,
		},
		Egress: CompiledTierAndPolicyChangeSet{
			Before: egressBefore,
			After:  egressAfter,
		},
	}
}

// CalculateSource calculates the action before and after the configuration change for a specific source reported flow.
// This method may be called simultaneously from multiple go routines for different flows if required. If the source
// action changes from deny to allow then we also have to calculate the destination action since we will not have flow
// data to work from.
func (fp *policyCalculator) CalculateSource(flow *api.Flow) (modified bool, before, after EndpointResponse) {
	return fp.calculateBeforeAfterResponse(flow, &fp.Egress, 0, 0)
}

// Calculate calculates the action before and after the configuration change for a specific destination reported flow.
// This method may be called simultaneously from multiple go routines for different flows if required.
func (fp *policyCalculator) CalculateDest(flow *api.Flow, sourceActionBefore, sourceActionAfter api.ActionFlag) (modified bool, before, after EndpointResponse) {
	return fp.calculateBeforeAfterResponse(flow, &fp.Ingress, sourceActionBefore, sourceActionAfter)
}

// calculateBeforeAfterResponse calculates the action before and after the configuration change for a specific reported
// flow.
func (fp *policyCalculator) calculateBeforeAfterResponse(
	flow *api.Flow, changeset *CompiledTierAndPolicyChangeSet, beforeSrcAction, afterSrcAction api.ActionFlag,
) (modified bool, before, after EndpointResponse) {
	// Initialize logger for this flow, and initialize selector caches.
	clog := log.WithFields(log.Fields{
		"reporter":        flow.Reporter,
		"sourceName":      flow.Source.Name,
		"sourceNamespace": flow.Source.Namespace,
		"destName":        flow.Destination.Name,
		"destNamespace":   flow.Destination.Namespace,
		"beforeSrcAction": beforeSrcAction,
		"afterSrcAction":  afterSrcAction,
	})

	// Initialize flow for the calculation.
	fp.initializeFlowForCalculations(flow)

	// Initialize the per-flow cache.
	cache := fp.newFlowCache(flow)

	// If the flow is not impacted return the unmodified response.
	if !changeset.FlowSelectedByImpactedPolicies(flow, cache) {
		clog.Debug("Flow unaffected")
		if beforeSrcAction != api.ActionFlagDeny {
			before = getUnchangedResponse(flow)
		}
		if afterSrcAction != api.ActionFlagDeny {
			after = getUnchangedResponse(flow)
		}
		return beforeSrcAction != afterSrcAction, before, after
	}

	// Calculate the before impact. We don't necessarily use the calculated value, but it pre-populates the cache for
	// the after response.
	if beforeSrcAction != api.ActionFlagDeny {
		// We only bother calculating the before flow for the destination if the before source action is not deny.
		clog.Debug("Calculate before impact")
		before = changeset.Before.Calculate(flow, cache, true)
	}

	// If we are not requested to calculate the original action then replace with the unchanged source response.
	//
	// If we are requested to calculate the original action it's possible the verdict has changed from deny to
	// allow or unknown. In that case if the destination is a calico-managed endpoint we'll need to add a fake
	// destination and calculate the impact of that - this handles the fact that we won't have remote data to use.
	if !fp.Config.CalculateOriginalAction && beforeSrcAction != api.ActionFlagDeny {
		clog.Debug("Keep original action from flow log")
		before = getUnchangedResponse(flow)

		// Sort the original set of flows so that we can compare to the "after" set to see if anything has actually
		// changed. We don't need to sort the calculated policies since these will already be sorted.
		sort.Sort(api.SortablePolicyHits(before.Policies))
	}

	if afterSrcAction != api.ActionFlagDeny {
		// Calculate the after impact.
		clog.Debug("Calculate after impact")
		after = changeset.After.Calculate(flow, cache, false)
	}

	if log.IsLevelEnabled(log.DebugLevel) {
		if before.Include {
			clog.WithFields(log.Fields{
				"calculatedBeforeAction":   before.Action,
				"calculatedBeforePolicies": before.Policies,
			}).Debug("Including flow before")
		} else {
			clog.Debug("Not including flow before")
		}
		if after.Include {
			clog.WithFields(log.Fields{
				"calculatedAfterAction":   after.Action,
				"calculatedAfterPolicies": after.Policies,
			}).Debug("Including flow after")
		} else {
			clog.Debug("Not including flow after")
		}
	}

	modified = before.Include != after.Include ||
		before.Action != after.Action ||
		(len(before.Policies) != 0 && api.PolicyHitsEqual(before.Policies, after.Policies))

	return modified, before, after
}

func (fp *policyCalculator) initializeFlowForCalculations(flow *api.Flow) {
	// If either source or destination are calico endpoints initialize the selector cache and use the datastore cache to
	// augment the flow data (if some data is missing).
	if flow.Source.IsCalicoManagedEndpoint() {
		if ed := fp.Endpoints.Get(flow.Source.Namespace, flow.Source.Name); ed != nil {
			log.Debug("Found source endpoint in cache")

			if flow.Source.ServiceAccount == nil {
				log.Debugf("Augmenting source endpoint flow data with cached service account: %v", ed.ServiceAccount)
				flow.Source.ServiceAccount = ed.ServiceAccount
			}

			if flow.Source.NamedPorts == nil {
				log.Debugf("Augmenting source endpoint flow data with cached named ports: %v", ed.NamedPorts)
				flow.Source.NamedPorts = ed.NamedPorts
			}

			if len(flow.Source.Labels) == 0 {
				log.Debugf("Augmenting source endpoint flow data with cached labels: %v", ed.Labels)
				flow.Source.Labels = ed.Labels
			}
		}
	}
	if flow.Destination.IsCalicoManagedEndpoint() {
		if ed := fp.Endpoints.Get(flow.Destination.Namespace, flow.Destination.Name); ed != nil {
			log.Debug("Found destination endpoint in cache")

			if flow.Destination.ServiceAccount == nil {
				log.Debugf("Augmenting destination endpoint flow data with cached service account: %v", ed.ServiceAccount)
				flow.Destination.ServiceAccount = ed.ServiceAccount
			}

			if flow.Destination.NamedPorts == nil {
				log.Debugf("Augmenting destination endpoint flow data with cached named ports: %v", ed.NamedPorts)
				flow.Destination.NamedPorts = ed.NamedPorts
			}

			if len(flow.Destination.Labels) == 0 {
				log.Debugf("Augmenting destination endpoint flow data with cached labels: %v", ed.Labels)
				flow.Destination.Labels = ed.Labels
			}
		}
	}
}

func (fp *policyCalculator) newFlowCache(flow *api.Flow) *flowCache {
	flowCache := &flowCache{}

	// Initialize the caches if required.
	if flow.Source.Labels != nil {
		flowCache.source.selectors = fp.CreateSelectorCache()
	}
	if flow.Destination.Labels != nil {
		flowCache.destination.selectors = fp.CreateSelectorCache()
	}
	flowCache.policies = make(map[string]api.ActionFlag)
	return flowCache
}

// CreateSelectorCache creates the match type slice used to cache selector calculations for a particular flow
// endpoint.
func (fp *policyCalculator) CreateSelectorCache() []MatchType {
	return fp.Selectors.CreateSelectorCache()
}

// getUnchangedSourceResponse returns a policy calculation Response based on the original source flow data.
func getUnchangedResponse(f *api.Flow) EndpointResponse {
	// Filter out staged policies from the original data.
	var filtered []api.PolicyHit
	for _, p := range f.Policies {
		if !p.Staged {
			filtered = append(filtered, p)
		}
	}

	return EndpointResponse{
		Include:  true,
		Action:   f.ActionFlag,
		Policies: filtered,
	}
}

// CompiledTierAndPolicyChangeSet contains the before/after tier and policy data for a given flow direction (i.e.
// ingress or egress).
type CompiledTierAndPolicyChangeSet struct {
	// The compiled set of tiers and policies before the change.
	Before CompiledTiersAndImpactedPolicies

	// The compiled set of tiers and policies after the change.
	After CompiledTiersAndImpactedPolicies
}

// FlowSelectedByImpactedPolicies returns whether the flow is selected by any of the impacted policies before or after
// the change is applied.
func (c CompiledTierAndPolicyChangeSet) FlowSelectedByImpactedPolicies(flow *api.Flow, cache *flowCache) bool {
	return c.Before.FlowSelectedByImpactedPolicies(flow, cache) || c.After.FlowSelectedByImpactedPolicies(flow, cache)
}
