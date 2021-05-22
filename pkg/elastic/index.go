// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package elastic

import (
	"fmt"
)

// Request properties to indicate the cluster used for proxying and RBAC.
const (
	DefaultClusterName  = "cluster"
	esAlertsIndexPrefix = "tigera_secure_ee_events"
	esDnsIndexPrefix    = "tigera_secure_ee_dns"
	esflowIndexPrefix   = "tigera_secure_ee_flows"
	esL7IndexPrefix     = "tigera_secure_ee_l7"
)

func GetDnsIndex(cluster string) string {
	return fmt.Sprintf("%s.%s.*", esDnsIndexPrefix, cluster)
}

func GetEventsIndex(cluster string) string {
	return fmt.Sprintf("%s.%s", esAlertsIndexPrefix, cluster)
}

func GetFlowsIndex(cluster string) string {
	return fmt.Sprintf("%s.%s.*", esflowIndexPrefix, cluster)
}

func GetL7FlowsIndex(cluster string) string {
	return fmt.Sprintf("%s.%s.*", esL7IndexPrefix, cluster)
}
