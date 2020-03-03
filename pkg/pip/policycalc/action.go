package policycalc

import (
	"github.com/tigera/lma/pkg/api"
)

// For pip we extend the action flags so that we can track multiple actions at once.

const (
	ActionFlagsAllPolicyActions = api.ActionFlagMax - 1
)

// Augmented set of action flags used by the policy calculation engine.
const (
	// This action flag is used in the policy calculation to specify that none of the rules matched the flow.
	ActionFlagNoMatch = api.ActionFlagMax << iota

	// These action flags provide additional information about how the action was determined, and how trustworthy the
	// result.
	// -  Whether the policy action was confirmed by the flow log policy hit
	// -  Whether the policy action was an exact match with the policy hit..
	// -  Whether the calculated action or actions did not agree with the policy hit in the flow log.
	ActionFlagFlowLogMatchesCalculated
	ActionFlagFlowLogRemovedUncertainty
	ActionFlagFlowLogConflictsWithCalculated
)

const (
	ActionFlagsAllowAndDeny               = api.ActionFlagAllow | api.ActionFlagDeny
	ActionFlagsAllCalculatedPolicyActions = api.ActionFlagAllow | api.ActionFlagDeny | api.ActionFlagNextTier | ActionFlagNoMatch
)

// Indeterminate returns true if the compiled rule actions indicate the action is indeterminate. This is true when
// a rule match is uncertain such that both Allow and Deny actions are possible with the limited available information
// in the flow data.
func Indeterminate(a api.ActionFlag) bool {
	// If uncertainty in the calculated action results in both Allow and Deny scenarios then the impact is
	// indeterminate.
	return a&ActionFlagsAllowAndDeny == ActionFlagsAllowAndDeny
}

// ActualFlowAction returns the real flow action flags from the full set of flags. The calculated
// set contains various other flags that augment that actual action info.
func ActualFlowAction(in api.ActionFlag) api.ActionFlag {
	return in & ActionFlagsAllowAndDeny
}

// ActualPolicyHitAction returns the real action flags from the full set of flags in a policy hit. The calculated
// set contains various other flags that augment that actual action info.
func ActualPolicyHitAction(in api.ActionFlag) api.ActionFlag {
	return in & (ActionFlagsAllowAndDeny | api.ActionFlagNextTier)
}
