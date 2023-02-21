// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package api

// Action represents an action that a policy hit may take on.
type Action string

// ActionFlag is used to calculate effective actions while traversing the compiled tiers.
type ActionFlag byte

const (
	// The compiled rule action names.
	ActionUnknown       Action = "unknown"
	ActionAllow         Action = "allow"
	ActionDeny          Action = "deny"
	ActionEndOfTierDeny Action = "eot-deny"
	ActionNextTier      Action = "pass"
	// TODO figure out if log should be here, it doesn't seem like log actions appear in the flow logs.
	// Used to flag an action as invalid when calling ActionFromString
	ActionInvalid Action = ""
)

const (
	// The compiled rule actions. This is a set of bitwise flags so that multiple actions can be built up when a rule
	// match is uncertain.
	ActionFlagAllow ActionFlag = 1 << iota
	ActionFlagDeny
	ActionFlagEndOfTierDeny
	ActionFlagNextTier
	ActionFlagMax
)

// ToFlag is a convenience function to convert the Action to an ActionFlag. It basically just wraps ActionFlagFromString.
func (a Action) ToFlag() ActionFlag {
	return ActionFlagFromString(string(a))
}

// AllActions returns the list of actions a policy hit action can currently take have.
func AllActions() []Action {
	return []Action{ActionUnknown, ActionAllow, ActionDeny, ActionEndOfTierDeny, ActionNextTier}
}

// ActionFromString verifies that the given string is a valid action (an action in the list returned by AllActions()) and
// returns the Action representation of the string.
func ActionFromString(actionStr string) Action {
	for _, action := range AllActions() {
		if actionStr == string(action) {
			return action
		}
	}

	return ActionInvalid
}

// ToFlowActionString() converts the action flags to the equivalent flow action string. This is used for the final
// flow action verdict and will therefore returns one of allow, deny or unknown (in the event both allow and deny are
// set).
func (a ActionFlag) ToFlowActionString() string {
	allow := a&ActionFlagAllow != 0
	deny := a&(ActionFlagDeny|ActionFlagEndOfTierDeny) != 0

	if allow && deny {
		return string(ActionUnknown)
	} else if allow {
		return string(ActionAllow)
	} else if deny {
		return string(ActionDeny)
	}
	return ""
}

// ToActionStrings() converts the action flags to the full set of action strings.
// This will be consist of allow, deny, eot-deny or pass.
func (a ActionFlag) ToActionStrings() []string {
	var actions []string
	if a&ActionFlagAllow != 0 {
		actions = append(actions, string(ActionAllow))
	}
	if a&ActionFlagDeny != 0 {
		actions = append(actions, string(ActionDeny))
	}
	if a&ActionFlagNextTier != 0 {
		actions = append(actions, string(ActionNextTier))
	}
	if a&ActionFlagEndOfTierDeny != 0 {
		actions = append(actions, string(ActionEndOfTierDeny))
	}
	return actions
}

// ActionFlagFromString converts the action string found in either the flow action or the policy hit string into the
// equivalent ActionFlag.
func ActionFlagFromString(a string) ActionFlag {
	switch Action(a) {
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
