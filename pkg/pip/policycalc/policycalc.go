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

// PolicyCalculator is used to determine the calculated behavior from a configuration change for a given flow.
type PolicyCalculator interface {
	Calculate(flow *Flow) (before, after *Response)
}

type Response struct {
	// The calculated response for the source endpoint.
	Source EndpointResponse

	// The calculated response for the destination endpoint.
	Destination EndpointResponse
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

	// The calculated action at the endpoint for the supplied flow.
	Action Action

	// The set of policies applied to this flow. The format of each entry is as follows.
	// For policy matches:
	// -  <tierIdx>|<tierName>|<namespaceName>/<policyName>|<action>
	// -  <tierIdx>|<tierName>|<policyName>|<action>
	//
	// For end of tier implicit drop (where policy is the last matching policy that did not match the rule):
	// -  <tierIdx>|<tierName>|<namespaceName>/<policyName>|deny
	// -  <tierIdx>|<tierName>|<policyName>|deny
	//
	// End of tiers allow for Pods (in Kubernetes):
	// -  <tierIdx>|__PROFILE__|__PROFILE__.kns.<namespaceName>|allow
	//
	// End of tiers drop for HostEndpoints:
	// -  <tierIdx>|__PROFILE__|__PROFILE__.__NO_MACH__|allow
	Policies []string
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

// Calculate calculates the action before and after the configuration change for a specific flow.
// This method may be called simultaneously from multiple go routines for different flows if required.
func (fp *policyCalculator) Calculate(flow *Flow) (before, after *Response) {
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
		pcr := flow.getUnchangedResponse()
		return pcr, pcr
	}

	// If we want to calculate the original action from the initial config rather than using the value in the flow log
	// then do that first.
	if fp.Config.CalculateOriginalAction {
		log.Debug("Calculate original action")
		before = fp.Before.Calculate(flow)
	} else {
		log.Debug("Keep original action from flow log")
		before = flow.getUnchangedResponse()
	}

	// Calculate the action with the updated config.
	log.Debug("Calculate new action")
	after = fp.After.Calculate(flow)

	if log.IsLevelEnabled(log.DebugLevel) {
		if before.Source.Include {
			clog.WithFields(log.Fields{
				"calculatedBeforeSourceAction":   before.Source.Action,
				"calculatedAfterSourceAction":    after.Source.Action,
				"calculatedBeforeSourcePolicies": before.Source.Policies,
				"calculatedAfterSourcePolicies":  after.Source.Policies,
			}).Debug("Including source flow")
		}
		if before.Destination.Include {
			clog.WithFields(log.Fields{
				"calculatedBeforeDestAction":   before.Destination.Action,
				"calculatedAfterDestAction":    after.Destination.Action,
				"calculatedBeforeDestPolicies": before.Destination.Policies,
				"calculatedAfterDestPolicies":  after.Destination.Policies,
			}).Debug("Including destination flow")
		}
	}

	return before, after
}

// CreateSelectorCache creates the match type slice used to cache selector calculations for a particular flow
// endpoint.
func (fp *policyCalculator) CreateSelectorCache() []MatchType {
	return fp.Selectors.CreateSelectorCache()
}
