package policycalc

import (
	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/compliance/pkg/resources"
)

// ------
// This file contains all of the struct definitions that are used as input when instantiating a new policy calculator.
// ------

// The tiers containing the ordered set of Calico v3 resource types.
type Tier []resources.Resource
type Tiers []Tier

// The consistent set of configuration used for calculating policy impact.
type ResourceData struct {
	Tiers           Tiers
	Namespaces      []*corev1.Namespace
	ServiceAccounts []*corev1.ServiceAccount
}

// ModifiedResources is essentially a set of resource IDs used to track which resources were modified in the proposed
// update.
type ModifiedResources map[v3.ResourceID]bool

// Add adds a resource to the set of modified resources.
func (m ModifiedResources) Add(r resources.Resource) {
	m[resources.GetResourceID(r)] = true
}

// IsModified returns true if the specified resource is one of the resources that was modified in the proposed update.
func (m ModifiedResources) IsModified(r resources.Resource) bool {
	return m[resources.GetResourceID(r)]
}

// PolicyCalculator is used to determine the calculated action from a configuration change for a given flow.
type PolicyCalculator interface {
	Action(flow *Flow) (processed bool, before, after Action)
}

// policyCalculator implements the PolicyCalculator interface.
type policyCalculator struct {
	Config    *Config
	Before    *CompiledTiersAndPolicies
	After     *CompiledTiersAndPolicies
	Selectors *EndpointSelectorHandler
}

// NewPolicyCalculator returns a new PolicyCalculator.
func NewPolicyCalculator(
	cfg *Config,
	resourceDataBefore *ResourceData,
	resourceDataAfter *ResourceData,
	modified ModifiedResources,
) PolicyCalculator {
	// Create the selector handler. This is shared by both the before and after matcher factories - this is fine because
	// the labels on the endpoints are not being adjusted, and so a selector will return the same value in the before
	// and after configurations.
	selectors := NewEndpointSelectorHandler()

	// Create the policyCalculator.
	return &policyCalculator{
		Config:    cfg,
		Selectors: selectors,
		Before:    newCompiledTiersAndPolicies(cfg, resourceDataBefore, modified, selectors),
		After:     newCompiledTiersAndPolicies(cfg, resourceDataAfter, modified, selectors),
	}
}

// Action calculates the action before and after the configuration change for a specific flow.
// This method may be called simultaneously from multiple go routines for different flows if required.
func (fp *policyCalculator) Action(flow *Flow) (processed bool, before, after Action) {
	clog := log.WithFields(log.Fields{
		"flowSrc":        flow.Source.Name,
		"flowDest":       flow.Destination.Name,
		"originalAction": flow.Action,
	})

	// Initialize selector caches
	flow.Source.cachedSelectorResults = fp.CreateSelectorCache()
	flow.Destination.cachedSelectorResults = fp.CreateSelectorCache()

	// Check if this flow is impacted.
	if !fp.Before.FlowSelectedByModifiedPolicies(flow) && !fp.After.FlowSelectedByModifiedPolicies(flow) {
		log.Debug("Flow unaffected")
		return false, flow.Action, flow.Action
	}

	// If we want to calculate the original action from the initial config rather than using the value in the flow log
	// then do that first.
	if fp.Config.CalculateOriginalAction {
		log.Debug("Calculate original action")
		before = fp.Before.Action(flow)
	} else {
		log.Debug("Keep original action from flow log")
		before = flow.Action
	}

	// Calculate the action with the updated config.
	log.Debug("Calculate new action")
	after = fp.After.Action(flow)

	clog.WithFields(log.Fields{
		"calculatedBefore": before,
		"calculatedAfter":  after,
	}).Debug("Compute flow action")

	return true, before, after
}

// CreateSelectorCache creates the match type slice used to cache selector calculations for a particular flow
// endpoint.
func (fp *policyCalculator) CreateSelectorCache() []MatchType {
	return fp.Selectors.CreateSelectorCache()
}
