// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

package hashutils

import (
	"crypto/sha256"
	"encoding/base32"
	"strings"
)

const (
	shortenedPrefix = "-"
	numHashChars    = 8
)

// GetLengthLimitedName returns a valid k8s name that consists of the name, or, if that would exceed the length limit,
// a string with the name prefix and a cryptographic hash of the name as a suffix, truncated to the required length.
func GetLengthLimitedName(value string, maxLength int) string {
	if len(value) > maxLength {
		// Value is too long, shorten it.
		hasher := sha256.New()
		_, _ = hasher.Write([]byte(value))
		enc := base32.HexEncoding.WithPadding('Z')
		hash := strings.ToLower(enc.EncodeToString(hasher.Sum(nil)))
		value = value[:maxLength-numHashChars] + shortenedPrefix + hash[0:numHashChars-len(shortenedPrefix)]
	}
	return value
}
