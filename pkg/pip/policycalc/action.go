package policycalc

// ActionFlag is used to calculate effective actions while traversing the compiled tiers.
type ActionFlag byte

const (
	// The compiled rule actions. This is a set of bitwise flags so that multiple actions can be built up when a rule
	// match is uncertain.
	ActionFlagAllow ActionFlag = 1 << iota
	ActionFlagDeny
	ActionFlagEndOfTierDeny
	ActionFlagNextTier
	ActionFlagMax
)

const (
	ActionFlagsAllPolicyActions = ActionFlagMax - 1
)

// Augmented set of action flags used by the policy calculation engine.
const (
	// This action flag is used in the policy calculation to specify that none of the rules matched the flow.
	ActionFlagNoMatch = ActionFlagMax << iota

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
	ActionFlagsAllowAndDeny               = ActionFlagAllow | ActionFlagDeny
	ActionFlagsAllCalculatedPolicyActions = ActionFlagAllow | ActionFlagDeny | ActionFlagNextTier | ActionFlagNoMatch
)

// Indeterminate returns true if the compiled rule actions indicate the action is indeterminate. This is true when
// a rule match is uncertain such that both Allow and Deny actions are possible with the limited available information
// in the flow data.
func (a ActionFlag) Indeterminate() bool {
	// If uncertainty in the calculated action results in both Allow and Deny scenarios then the impact is
	// indeterminate.
	return a&ActionFlagsAllowAndDeny == ActionFlagsAllowAndDeny
}

// ToFlowActionString() converts the action flags to the equivalent flow action string. This is used for the final
// flow action verdict and will therefore returns one of allow, deny or unknown (in the event both allow and deny are
// set).
func (a ActionFlag) ToFlowActionString() string {
	allow := a&ActionFlagAllow != 0
	deny := a&(ActionFlagDeny|ActionFlagEndOfTierDeny) != 0

	if allow && deny {
		return ActionUnknown
	} else if allow {
		return ActionAllow
	} else if deny {
		return ActionDeny
	}
	return ""
}

// ToActionStrings() converts the action flags to the full set of action strings.
// This will be consist of allow, deny, eot-deny or pass.
func (a ActionFlag) ToActionStrings() []string {
	var actions []string
	if a&ActionFlagAllow != 0 {
		actions = append(actions, ActionAllow)
	}
	if a&ActionFlagDeny != 0 {
		actions = append(actions, ActionDeny)
	}
	if a&ActionFlagNextTier != 0 {
		actions = append(actions, ActionNextTier)
	}
	if a&ActionFlagEndOfTierDeny != 0 {
		actions = append(actions, ActionEndOfTierDeny)
	}
	return actions
}

// ActionFlagFromString converts the action string found in either the flow action or the policy hit string into the
// equivalent ActionFlag.
func ActionFlagFromString(a string) ActionFlag {
	switch a {
	case ActionAllow:
		return ActionFlagAllow
	case ActionDeny:
		return ActionFlagDeny
	case ActionEndOfTierDeny:
		return ActionFlagEndOfTierDeny
	case ActionNextTier:
		return ActionFlagNextTier
	}
	return 0
}
