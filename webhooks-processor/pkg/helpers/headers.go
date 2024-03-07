// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package helpers

import (
	"strings"
)

func ProcessHeaders(rawHeaders string) map[string]string {
	headers := make(map[string]string)
	for _, line := range strings.Split(rawHeaders, "\n") {
		if keyValue := strings.SplitN(line, ":", 2); len(keyValue) == 2 {
			headers[keyValue[0]] = keyValue[1]
		}
	}
	return headers
}
