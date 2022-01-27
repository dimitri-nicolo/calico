package util

import (
	"fmt"
	"strings"

	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/selection"
)

const (
	policyDelim     = "."
	uiSettingsDelim = "."
	defaultTier     = "default"
)

func setDefaultTierSelector(options *metainternalversion.ListOptions) {
	defaultTierSelector := fields.SelectorFromSet(map[string]string{"spec.tier": defaultTier})
	if options.FieldSelector == nil {
		options.FieldSelector = defaultTierSelector
	} else {
		options.FieldSelector = fields.AndSelectors(options.FieldSelector, defaultTierSelector)
	}
}

func GetTierNameFromSelector(options *metainternalversion.ListOptions) (string, error) {
	if options.FieldSelector != nil {
		requirements := options.FieldSelector.Requirements()
		for _, requirement := range requirements {
			if requirement.Field == "spec.tier" {
				if requirement.Operator == selection.Equals ||
					requirement.Operator == selection.DoubleEquals {
					return requirement.Value, nil
				}
				return "", fmt.Errorf("Non equal selector operator not supported for field spec.tier")
			}
		}
	}

	if options.LabelSelector != nil {
		requirements, _ := options.LabelSelector.Requirements()
		for _, requirement := range requirements {
			if requirement.Key() == "projectcalico.org/tier" {
				if len(requirement.Values()) > 1 {
					return "", fmt.Errorf("multi-valued selector not supported")
				}
				tierName, ok := requirement.Values().PopAny()
				if ok && (requirement.Operator() == selection.Equals ||
					requirement.Operator() == selection.DoubleEquals) {
					return tierName, nil
				}
				return "", fmt.Errorf("Non equal selector operator not supported for label projectcalico.org/tier")
			}
		}
	}

	// Reaching here implies tier is 'default' and hasn't been explicitly set as part of the selectors.
	setDefaultTierSelector(options)
	return defaultTier, nil
}

// GetTierFromPolicyName extracts the Tier name from the policy name.
func GetTierFromPolicyName(policyName string) (string, string) {
	policySlice := strings.Split(policyName, policyDelim)
	if len(policySlice) < 2 {
		return "default", policySlice[0]
	}
	return policySlice[0], policySlice[1]
}

func GetUISettingsGroupNameFromSelector(options *metainternalversion.ListOptions) (string, error) {
	if options.FieldSelector != nil {
		requirements := options.FieldSelector.Requirements()
		for _, requirement := range requirements {
			if requirement.Field == "spec.group" {
				if requirement.Operator == selection.Equals ||
					requirement.Operator == selection.DoubleEquals {
					return requirement.Value, nil
				}
				return "", fmt.Errorf("Non equal selector operator not supported for field spec.group")
			}
		}
	}

	if options.LabelSelector != nil {
		requirements, _ := options.LabelSelector.Requirements()
		for _, requirement := range requirements {
			if requirement.Key() == "projectcalico.org/uisettingsgroup" {
				if len(requirement.Values()) > 1 {
					return "", fmt.Errorf("multi-valued selector not supported")
				}
				groupName, ok := requirement.Values().PopAny()
				if ok && (requirement.Operator() == selection.Equals ||
					requirement.Operator() == selection.DoubleEquals) {
					return groupName, nil
				}
				return "", fmt.Errorf("Non equal selector operator not supported for label projectcalico.org/uisettingsgroup")
			}
		}
	}

	// No group selector. Return "*" as the name - this will be used to check authorization. If authorized to allow
	// a settings called "*" it is assumed it is authorized to access all settings. Strictly, a RBAC role could be
	// set up matching on a name of "*" but I think this is a live with.
	return "*", fmt.Errorf("Require UISettingsGroup to be specified")
}

// GetUISettingsGroupFromUISettingsName extracts the UISettingsGroup name from the UISettings name.
func GetUISettingsGroupFromUISettingsName(uiSettingsName string) (string, error) {
	parts := strings.Split(uiSettingsName, uiSettingsDelim)
	if len(parts) < 2 {
		return "", fmt.Errorf("UISettings name format is incorrect - should be prefixed by the UISettingsGroup")
	}
	return parts[0], nil
}
