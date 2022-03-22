// Copyright (c) 2021 Tigera, Inc. All rights reserved.

//go:build tesla
// +build tesla

package elastic

import (
	"fmt"
	"os"
)

var tenantID = os.Getenv("ELASTIC_INDEX_TENANT_ID")

// AddIndexInfix is a hook to add any extra substring to the index pattern. For Cloud/Tesla,
// we currently add an extra Tenant ID prefix.
func AddIndexInfix(index string) string {
	if tenantID != "" {
		return fmt.Sprintf("%s.%s", tenantID, index)
	}

	return index
}
