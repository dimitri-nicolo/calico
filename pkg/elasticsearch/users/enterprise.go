// Copyright (c) 2021 Tigera, Inc. All rights reserved.

// +build !tesla

package users

import "fmt"

func indexPattern(prefix, cluster, suffix string) string {
	return fmt.Sprintf("%s.%s%s", prefix, cluster, suffix)
}
