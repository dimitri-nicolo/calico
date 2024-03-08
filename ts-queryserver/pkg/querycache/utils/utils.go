package utils

import (
	"errors"
	"regexp"
	"strings"
)

// BuildSubstringRegexMatcher creates a regex from a list to help with faster substring searching.
//
// the list should contain at least one value. If the list is empty it fails to create regex pattern.
func BuildSubstringRegexMatcher(list []string) (*regexp.Regexp, error) {
	if len(list) > 0 {
		regexPattern := strings.Join(list, "|")
		epListRegex, err := regexp.Compile(regexPattern)
		if err != nil {
			return nil, err
		}

		return epListRegex, nil
	}
	return nil, errors.New("vague input: cannot create regex pattern from empty list")
}
