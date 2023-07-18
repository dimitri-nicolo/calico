// Copyright 2022-2023 Tigera Inc. All rights reserved.
package util

import (
	"regexp"
	"strings"
)

const (
	MaxJobNameLen = 52
	NumHashChars  = 5
)

func MakeADJobName(jobTypePrefix, tenantID, clusterName, alertOrDetectorName string) string {
	// Form a name incorporating all the pieces.
	fullNameWithoutHash := strings.ToLower(jobTypePrefix + "-" + Unify(tenantID, clusterName))
	if alertOrDetectorName != "" {
		fullNameWithoutHash = fullNameWithoutHash + "-" + alertOrDetectorName
	}

	// Truncate if necessary to ensure that the eventual job name will be less than
	// MaxJobNameLen chars.
	maxTruncatedNameLen := MaxJobNameLen - (1 + NumHashChars)
	truncatedName := fullNameWithoutHash
	if len(truncatedName) > maxTruncatedNameLen {
		truncatedName = fullNameWithoutHash[:maxTruncatedNameLen]
	}

	// Ensure it's RFC1123 compliant.
	truncatedName = ConvertToValidName(truncatedName)

	// Ensure it's unique by adding a hash of the full name.
	jobName := truncatedName + "-" + ComputeSha256HashWithLimit(fullNameWithoutHash, NumHashChars)

	return jobName
}

// ConvertToValidName converts all characters to lower case and removes all invalid Job name
// characters. A wild card, 'z', is used in case all characters of the name are invalid and are
// removed.
func ConvertToValidName(name string) string {
	rfcWildcard := "z"

	// Convert all uppercase to lower case.
	rfcName := strings.ToLower(name)

	// Remove all characters that are not alphanumeric or '-' or '.'.
	regexInvalidChars := regexp.MustCompile(`[^a-z0-9\\-\\.]+`)
	rfcName = regexInvalidChars.ReplaceAllString(rfcName, "-")
	// Collapse all consecutive strings of '.' with a single '.'.
	regexPrefixSuffix := regexp.MustCompile("[.]*[.]")
	rfcName = regexPrefixSuffix.ReplaceAllString(rfcName, ".")
	// Collapse all consecutive strings of '-' with a single '-'.
	regexPrefixSuffix = regexp.MustCompile("[-]*[-]")
	rfcName = regexPrefixSuffix.ReplaceAllString(rfcName, "-")
	// Remove all '-' and '.' from the prefix and suffix of the name.
	regexPrefixSuffix = regexp.MustCompile("^[.-]*|[.-]*$")
	rfcName = regexPrefixSuffix.ReplaceAllString(rfcName, "")

	// If all characters have been removed, replace the empty string with a 'z'.
	if len(rfcName) == 0 {
		rfcName = rfcWildcard
	}

	return rfcName
}
