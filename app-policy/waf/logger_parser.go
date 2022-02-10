// Copyright (c) 2018-2022 Tigera, Inc. All rights reserved.

package waf

import (
	"fmt"
	"regexp"
	"strings"
)

// Logger Parser is used to parse the payload sent by the ModSecurity callback.
// The payload is a stream of text as a flattened dictionary of key / value pairs
// for Warning information generated by violations in Core Rule Sets files
// e.g. which OWASP file generated the warning, which line, the severity etc.

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

// ParseLog() function is the code that actually parses the stream of text and creates
// an on-the-fly dictionary that is returned to WAF code which can be logged to Kibana.
func ParseLog(payload string) map[string]string {

	dictionary := make(map[string]string)
	regex := regexp.MustCompile(`\[([^\[\]]*)]`)

	for _, groups := range regex.FindAllStringSubmatch(payload, ParserMatchAll) {
		kv := groups[1]

		// Record only entries in payload that conform to key / value.
		splitN := strings.SplitN(kv, ParserDelim, NumElements)
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

// FormatMap() function flattens the dictionary key values into single string of text.
func FormatMap(dictionary map[string]string) string {

	owaspHost := dictionary[ParserHostname]
	owaspFile := dictionary[ParserFile]
	owaspLine := dictionary[ParserLine]
	owaspId := dictionary[ParserId]
	owaspData := dictionary[ParserData]
	owaspSeverity := dictionary[ParserSeverity]
	owaspVersion := dictionary[ParserVersion]

	return fmt.Sprintf("Host:'%s' File:'%s' Line:'%s' ID:'%s' Data:'%s' Severity:'%s' Version:'%s'", owaspHost, owaspFile, owaspLine, owaspId, owaspData, owaspSeverity, owaspVersion)
}
