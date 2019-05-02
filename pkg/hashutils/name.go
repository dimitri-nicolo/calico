// Copyright (c) 2016-2017 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
		// Value is too long, shorten it
		hasher := sha256.New()
		hasher.Write([]byte(value))
		enc := base32.HexEncoding.WithPadding('Z')
		hash := strings.ToLower(enc.EncodeToString(hasher.Sum(nil)))
		value = value[:maxLength-numHashChars] + shortenedPrefix + hash[0:numHashChars-len(shortenedPrefix)]
	}
	// No need to shorten.
	return value
}
