// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package flows

import (
	"fmt"
	"net/http"
)

// Request properties to indicate the cluster used for proxying and RBAC.
const (
	clusterIdHeader    = "x-cluster-id"
	defaultClusterName = "cluster"
	esflowIndexPrefix  = "tigera_secure_ee_flows"
)

func GetFlowsIndex(req *http.Request) string {
	cluster := req.Header.Get(clusterIdHeader)
	if cluster == "" {
		cluster = defaultClusterName
	}
	return fmt.Sprintf("%s.%s.*", esflowIndexPrefix, cluster)
}
