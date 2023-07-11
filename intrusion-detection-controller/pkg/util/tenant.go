// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package util

import "fmt"

func Unify(tenantID string, clusterName string) string {
	// We need to take into account Calico Cloud setup that functions in a multi-tenant flavour
	// In order to keep backwards compatibility, anomaly detection name will have key <tenant_id.cluster_name>
	// in multi-tenant setup and <cluster_name> for Enterprise

	if tenantID == "" {
		return clusterName
	}

	return fmt.Sprintf("%s.%s", tenantID, clusterName)
}
