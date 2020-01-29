// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package api

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
