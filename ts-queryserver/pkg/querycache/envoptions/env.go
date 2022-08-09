// Copyright (c) 2018-2019 Tigera, Inc. All rights reserved.
package envoptions

import (
	"os"
	"strings"
)

func OutputPerfStats() bool {
	return isTrue(os.Getenv("OUTPUT_PERF_STATS"))
}

func isTrue(v string) bool {
	lv := strings.ToLower(v)
	return lv == "true" || lv == "yes" || lv == "t" || lv == "y" || lv == "1"
}
