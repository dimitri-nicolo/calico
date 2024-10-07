package policycalc

import (
	"github.com/projectcalico/calico/lma/pkg/api"
)

// For pip we extend the action flags so that we can track multiple actions at once.

const (
	ActionFlagsAllPolicyActions = api.ActionFlagMax - 1
)

// Augmented set of action flags used by the policy calculation engine.
const (
	// This action flag is used in the policy calculation to specify that none of the rules matched the flow. This may
	// be used in conjunction with an allow or deny action, in which case this is an "uncertain" no match.
	ActionFlagNoMatch = api.ActionFlagMax << iota

	// These action flags provide additional information about how the action was determined and, therefore, how
	// trustworthy the calculated result is
	// -  Whether the calculated policy hit is exact (i.e. there is no uncertainty in the calculation) *and* exactly
	//    matches the policy hit in the flow log.
	// -  Whether calculated policy hit is uncertain (due to insufficient data), but the flow log policy hit matches one
	//    of the possible calculated hits and can therefore be used to select the same calculated value.
	// -  Whether a calculated action did not agree with the policy hit in the flow log. For example, the policy
	//    calculation may determine an allow hit on policy X in tier A, but the flow log data actually has a deny hit or
	//    does not match that policy at all.
	//
	// Note that is a policy is adjusted as part of the preview action then flow log data that directly or indirectly
	// involves that policy cannot be used to augment the calculation.  For example:
	// -  Policy A in tier X is adjusted. Flow log shows policy B in tier X is "hit".  Policy B is ordered after A.
	//    We can infer that policy A is a no-match in the original flow.
	//    -  If calculation for policy A is uncertain then we can use the indirectly determined "no-match" iff the
	//       policy was not modified.
	// This applies to staged policies as well - these can provide some clarity for no-matches.
	ActionFlagFlowLogMatchesCalculated
	ActionFlagFlowLogRemovedUncertainty
	ActionFlagFlowLogConflictsWithCalculated
)

const (
	ActionFlagsAllowAndDeny               = api.ActionFlagAllow | api.ActionFlagDeny
	ActionFlagsAllCalculatedPolicyActions = api.ActionFlagAllow | api.ActionFlagDeny | api.ActionFlagNextTier | ActionFlagNoMatch
	ActionFlagsVerified                   = ActionFlagFlowLogMatchesCalculated | ActionFlagFlowLogRemovedUncertainty
	ActionFlagsMeasured                   = ActionFlagsVerified | ActionFlagFlowLogConflictsWithCalculated
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
