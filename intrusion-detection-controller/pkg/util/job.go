// Copyright 2022 Tigera Inc. All rights reserved.
package util

import (
	"fmt"
	"regexp"
	"strings"

	k8svalidation "k8s.io/apimachinery/pkg/util/validation"
)

const (
	// Hash constants.
	hashShortenedPrefix                 = "-"
	numHashChars                        = 5
	lenOfMaxValidInitialTrainingJobName = k8svalidation.DNS1123LabelMaxLength - (len(hashShortenedPrefix) + numHashChars)
)

// GetValidInitialTrainingJobName returns a job name that is RFC 1123 compliant. It assumes that a
// hash value will be concatenated as a suffix at a later stage with length equal to
// len(len(hashShortenedPrefix) + numHashChars).
//
// The resulting name must:
// - start with an alphabetic character
// - end with an alphanumeric character
// - contain at most 57 characters
// - contain only lowercase alphanumeric characters or '-' or '.'
func GetValidInitialTrainingJobName(clusterName, detectorName, suffix string) string {
	name := fmt.Sprintf("%s-%s-%s", clusterName, detectorName, suffix)

	// If the job name is in a valid RFC1123 label format, return.
	if len(name) <= lenOfMaxValidInitialTrainingJobName && isValidJobName(name) {
		return name
	}

	// The name is not in RFC1123 label format, at least one conversion has to occur to make it valid.
	// An empty string will contain a wildcard 'z' character.

	validClusterName := convertToValidJobName(clusterName)
	validDetectorName := convertToValidJobName(detectorName)

	rfcName := fmt.Sprintf("%s-%s-initial-training", validClusterName, validDetectorName)

	// If the length of the name exceeds the length of lenOfMaxValidInitialTrainingJobName,
	// then cut the length of the rfcName, so that its length with the hash is less than the length of
	// DNS1123LabelMaxLength.
	if len(rfcName) > lenOfMaxValidInitialTrainingJobName {
		if rfcName[lenOfMaxValidInitialTrainingJobName-1] == '-' {
			// If the last character of the substring of rfcName is '-', remove it.
			rfcName = rfcName[:lenOfMaxValidInitialTrainingJobName-1]
		} else {
			rfcName = rfcName[:lenOfMaxValidInitialTrainingJobName]
		}
	}

	return rfcName
}

// convertToValidJobName converts all characters to lower case and removes all invalid Job name
// characters. A wild card, 'z', is used in case all characters of the name are invalid and are
// removed.
func convertToValidJobName(name string) string {
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

// isValidJobName return true if the name is a valid job name with respect to the characters, and
// does not validate the length.
// The name must:
// - start with an alphabetic character
// - end with an alphanumeric character
// - contain only lowercase alphanumeric characters or '-' or '.'
// - does not contain ".-" or "-." substrings.
func isValidJobName(name string) bool {
	nameRFC1123PolicyLabelFmt := "[a-z0-9]([-a-z0-9]*[a-z0-9])?"
	nameRFC1123PolicySubdomainFmt := nameRFC1123PolicyLabelFmt + "(" + nameRFC1123PolicyLabelFmt + ")*"
	// Names must follow a simple subdomain DNS1123 format.
	isValidLabelNameFmt := regexp.MustCompile("^" + nameRFC1123PolicySubdomainFmt + "$").MatchString

	return isValidLabelNameFmt(name)
}
