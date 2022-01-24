// Copyright (c) 2018-2022 Tigera, Inc. All rights reserved.

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

package waf

import (
	"regexp"
	"strings"
)

const ParserDelim = " "
const ParserEmpty = ""
const ParserEscape = "\""
const ParserPrefix = "ModSecurity: "
const ParserMatchAll = -1
const NumElements = 2

const ParserFile = "file"
const ParserLine = "line"
const ParserId = "id"
const ParserMsg = "msg"
const ParserData = "data"
const ParserSeverity = "severity"
const ParserVersion = "ver"
const ParserHostname = "hostname"
const ParserUri = "uri"
const ParserUniqueId = "unique_id"

func ParseLog(payload string) map[string]string {

	dictionary := make(map[string]string)
	regex := regexp.MustCompile(`\[([^\[\]]*)]`)

	submatchall := regex.FindAllString(payload, ParserMatchAll)
	for _, element := range submatchall {
		element = strings.Trim(element, "[")
		element = strings.Trim(element, "]")

		// Record only entries in payload that conform to key / value.
		splitN := strings.SplitAfterN(element, ParserDelim, NumElements)
		if len(splitN) != NumElements {
			continue
		}

		key := strings.Trim(splitN[0], ParserDelim)
		value := strings.Replace(splitN[1], ParserEscape, ParserEmpty, ParserMatchAll)

		dictionary[key] = value
	}

	// Default MSG with ModSecurity preliminary text if payload MSG is empty.
	if len(dictionary[ParserMsg]) == 0 {
		index := strings.Index(payload, "[")
		msg := strings.TrimSpace(payload[:index])
		msg = strings.Replace(msg, ParserPrefix, ParserEmpty, ParserMatchAll)
		dictionary[ParserMsg] = msg
	}

	return dictionary
}
