// Copyright (c) 2019 Tigera Inc. All rights reserved.

package util

import (
	"regexp"
	"strings"
)

// PainlessFmt formats Painless code so that it looks readable when encoded as JSON
func PainlessFmt(s string) string {
	var res []string
	lines := strings.Split(s, "\n")
	for idx, line := range lines {
		line = strings.TrimSpace(line)
		if idx > 0 && regexp.MustCompile(`^[{}]`).MatchString(line) {
			line = " " + line
		}
		if idx < len(lines)-1 && regexp.MustCompile(`[,{};]$`).MatchString(line) {
			line = line + " "
		}
		if line != "" {
			res = append(res, line)
		}
	}
	return strings.Join(res, "")
}
