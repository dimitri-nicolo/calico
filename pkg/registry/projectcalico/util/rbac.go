package util

import (
	"fmt"
	"strings"

	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/selection"
)

const (
	policyDelim = "."
	defaultTier = "default"
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

// Check the user is allowed to "get" the tier.
// This is required to be allowed to perform actions on policies.
func GetTierPolicy(policyName string) (string, string) {
	policySlice := strings.Split(policyName, policyDelim)
	if len(policySlice) < 2 {
		return "default", policySlice[0]
	}
	return policySlice[0], policySlice[1]
}
