package policycalc

// ActionFlag is used to calculate effective actions while traversing the compiled tiers.
type ActionFlag byte

const (
	// The compiled rule actions. This is a set of bitwise flags so that multiple actions can be built up when a rule
	// match is uncertain.
	ActionFlagAllow ActionFlag = 1 << iota
	ActionFlagDeny
	ActionFlagNextTier
)

const ActionFlagsAllowAndDeny = ActionFlagAllow | ActionFlagDeny

// Indeterminate returns true if the compile rule actions indicate the action is indeterminate. This is true when
// a rule match is uncertain such that both Allow and Deny actions are possible with the limited available information
// in the flow data.
func (a ActionFlag) Indeterminate() bool {
	// If uncertainty in the calculated action results in both Allow and Deny scenarios then the impact is
	// indeterminate.
	return a&ActionFlagsAllowAndDeny == ActionFlagsAllowAndDeny
}

// Deny returns true if the bitwise values indicate a deny action and not allow.
func (a ActionFlag) Deny() bool {
	return a&ActionFlagsAllowAndDeny == ActionFlagDeny
}

// Allow returns true if the bitwise values indicate an allow action and not deny.
func (a ActionFlag) Allow() bool {
	return a&ActionFlagsAllowAndDeny == ActionFlagAllow
}
