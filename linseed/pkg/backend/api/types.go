// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package api

import (
	"fmt"
	"strings"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

// ClusterInfo defines the user who made the request.
type ClusterInfo struct {
	Cluster string
	Tenant  string
}

func (c *ClusterInfo) Valid() error {
	// Prevent multi-index queries.
	// https://www.elastic.co/guide/en/elasticsearch/reference/current/api-conventions.html#api-multi-index
	if strings.Contains(c.Tenant, "*") ||
		strings.HasPrefix(c.Tenant, "-") ||
		strings.Contains(c.Tenant, ",") {
		return fmt.Errorf("tenantID on request contains an unsupported symbol")
	}
	if strings.Contains(c.Cluster, "*") ||
		strings.HasPrefix(c.Cluster, "-") ||
		strings.Contains(c.Cluster, ",") {
		return fmt.Errorf("clusterID on request contains an unsupported symbol")
	}

	// A cluster ID is always required.
	if c.Cluster == "" {
		return fmt.Errorf("no cluster ID provided on request")
	}
	return nil
}

func (c *ClusterInfo) IsQueryMultipleClusters() bool {
	return v1.IsQueryMultipleClusters(c.Cluster)
}
