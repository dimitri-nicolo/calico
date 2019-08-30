package policycalc

// ActionFlag is used to calculate effective actions while traversing the compiled tiers.
type ActionFlag byte

const (
	// The compiled rule actions. This is a set of bitwise flags so that multiple actions can be built up when a rule
	// match is uncertain.
	ActionFlagAllow ActionFlag = 1 << iota
	ActionFlagDeny
	ActionFlagNextTier
	ActionFlagDidNotMatchTier
)

const (
	ActionFlagsAllowAndDeny     = ActionFlagAllow | ActionFlagDeny
	ActionFlagsAllPolicyActions = ActionFlagAllow | ActionFlagDeny | ActionFlagNextTier
)

// Indeterminate returns true if the compiled rule actions indicate the action is indeterminate. This is true when
// a rule match is uncertain such that both Allow and Deny actions are possible with the limited available information
// in the flow data.
func (a ActionFlag) Indeterminate() bool {
	// If uncertainty in the calculated action results in both Allow and Deny scenarios then the impact is
	// indeterminate.
	return a&ActionFlagsAllowAndDeny == ActionFlagsAllowAndDeny
}

// ToAction() converts the final calculated action flags to the equivalent Action.
func (a ActionFlag) ToAction() Action {
	switch a & ActionFlagsAllPolicyActions {
	case ActionFlagAllow:
		return ActionAllow
	case ActionFlagDeny:
		return ActionDeny
	case ActionFlagNextTier:
		return ActionNextTier
	}
	if a.Indeterminate() {
		return ActionUnknown
	}
	return ActionInvalid
}

func ActionFlagFromAction(a Action) ActionFlag {
	switch a {
	case ActionDeny:
		return ActionFlagDeny
	case ActionNextTier:
		return ActionFlagNextTier
	case ActionAllow:
		return ActionFlagAllow
	}
	return 0
}
