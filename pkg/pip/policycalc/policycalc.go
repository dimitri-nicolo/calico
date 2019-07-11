package policycalc

import log "github.com/sirupsen/logrus"

type PolicyCalculator interface {
	Action(flow *Flow) (processed bool, before, after Action)
}

// policyCalculator implements the policy impact calculator.
type policyCalculator struct {
	Config    *Config
	Before    *CompiledTiersAndPolicies
	After     *CompiledTiersAndPolicies
	Selectors *EndpointSelectorHandler
}

// NewPolicyCalculator returns a new policyCalculator.
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
		Before:    compile(cfg, resourceDataBefore, modified, selectors),
		After:     compile(cfg, resourceDataAfter, modified, selectors),
	}
}

// Action calculates the action before and after the configuration change for a specific flow.
// This method may be called simultaneously from multiple go routines f or different flowsif required.
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
