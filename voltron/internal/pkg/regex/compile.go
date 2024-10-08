// Copyright (c) 2020 Tigera, Inc. All rights reserved.

// Package regex has a set of regex related functions used across components
package regex

import (
	"fmt"
	"regexp"
)

// CompileRegexStrings takes the given slice of strings and attempts to compile
// each one into a Regexp. Returns a slice containing each of the compiled
// Regexp objects.
func CompileRegexStrings(patterns []string) ([]regexp.Regexp, error) {
	regexList := []regexp.Regexp{}
	for _, pattern := range patterns {
		result, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("PathRegexp failed: %s", err)
		}
		regexList = append(regexList, *result)
	}
	return regexList, nil
}
