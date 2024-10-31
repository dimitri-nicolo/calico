// Copyright 2021 Tigera Inc. All rights reserved.
package util

import (
	"regexp"

	log "github.com/sirupsen/logrus"
	k8svalidation "k8s.io/apimachinery/pkg/util/validation"
)

const (
	// Non RFC1123 compliant characters
	nameRFC1123LabelFmt                  = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"
	nameRFC1123SubdomainFmtExcludePeriod = nameRFC1123LabelFmt + "(" + nameRFC1123LabelFmt + ")*"

	// Hash constants.
	hashShortenedPrefix        = "-"
	numHashChars               = 5
	lenOfMaxRfc1123WithoutHash = k8svalidation.DNS1123LabelMaxLength - (len(hashShortenedPrefix) + numHashChars)
)

// IsValidTierName return true if the name is a valid RFC1123 name and of length less than half the .
// The name must:
//   - start with an alphabetic character
//   - end with an alphanumeric character
//   - contain only lowercase alphanumeric characters or '-'
//   - does not contain ".-" or "-." substrings.
//   - have a character length that is a maximum 28 characters long, i.e. half the
//     lenOfMaxRfc1123WithoutHash=57.
func IsValidTierName(name string) bool {
	// Validate the length of the tier name. The tier name is used as a prefix to a policy's name and
	// thus the combined size must not exceed lenOfMaxRfc1123WithoutHash, where we define the max size
	// to be half that.
	if len(name) >= lenOfMaxRfc1123WithoutHash/2 {
		log.Debugf("tier name, with length %d, exceeds allowed max length", len(name))
		return false
	}
	// Names must follow a simple subdomain DNS1123 format.
	isValidLabelNameFmt := regexp.MustCompile("^" + nameRFC1123SubdomainFmtExcludePeriod + "$").MatchString

	return isValidLabelNameFmt(name)
}
