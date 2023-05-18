// Copyright 2021 Tigera Inc. All rights reserved.

//go:build tesla
// +build tesla

package managedcluster

import (
	"fmt"
	"os"
)

var tenantID = os.Getenv("TENANT_ID")

// This is the Cloud/Tesla variant of this function. For multi-tenancy we need to scope all indices by tenantID.
func getVariantSpecificClusterName(name string) string {
	if tenantID != "" {
		return fmt.Sprintf("%s.%s", tenantID, name)
	}

	return name
}
