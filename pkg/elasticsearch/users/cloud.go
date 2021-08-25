// Copyright (c) 2021 Tigera, Inc. All rights reserved.

// +build tesla

package users

import (
	"fmt"
	"os"
)

var tenantID = os.Getenv("ELASTIC_INDEX_TENANT_ID")

func indexPattern(prefix, cluster, suffix string) string {
	return fmt.Sprintf("%s.%s.%s%s", prefix, tenantID, cluster, suffix)
}
