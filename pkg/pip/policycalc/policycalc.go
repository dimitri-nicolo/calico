package policycalc

import log "github.com/sirupsen/logrus"

// PolicyCalculator implements the policy impact calculator.
type PolicyCalculator struct {
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
) *PolicyCalculator {
	// Create the selector handler. This is shared by both the before and after matcher factories - this is fine because
	// the labels on the endpoints are not being adjusted, and so a selector will return the same value in the before
	// and after configurations.
	selectors := NewEndpointSelectorHandler()

	// Create the PolicyCalculator.
	return &PolicyCalculator{
		Config:    cfg,
		Selectors: selectors,
		Before:    compile(cfg, resourceDataBefore, modified, selectors),
		After:     compile(cfg, resourceDataAfter, modified, selectors),
	}
}

// Action calculates the action before and after the configuration change for a specific flow.
// This method may be called simultaneously from multiple go routines if required.
func (fp *PolicyCalculator) Action(flow *Flow) (processed bool, before, after Action) {
	// Initialize selector caches
	flow.Source.cachedSelectorResults = fp.CreateSelectorCache()
	flow.Destination.cachedSelectorResults = fp.CreateSelectorCache()

	// Check if this flow is impacted.
	if !fp.Before.FlowImpacted(flow) && !fp.After.FlowImpacted(flow) {
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

	return true, before, after
}

// CreateSelectorCache creates the match type slice used to cache selector calculations for a particular flow
// endpoint.
func (fp *PolicyCalculator) CreateSelectorCache() []MatchType {
	return fp.Selectors.CreateSelectorCache()
}
