// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package elastic

import (
	"fmt"
)

// Request properties to indicate the cluster used for proxying and RBAC.
const (
	DefaultClusterName    = "cluster"
	esEventsIndexPrefix   = "tigera_secure_ee_events"
	esDNSLogsIndexPrefix  = "tigera_secure_ee_dns"
	esFlowLogsIndexPrefix = "tigera_secure_ee_flows"
	esL7LogsIndexPrefix   = "tigera_secure_ee_l7"
)

func GetDNSLogsIndex(cluster string) string {
	return fmt.Sprintf("%s.%s.*", esDNSLogsIndexPrefix, cluster)
}

func GetEventsIndex(cluster string) string {
	return fmt.Sprintf("%s.%s", esEventsIndexPrefix, cluster)
}

func GetFlowLogsIndex(cluster string) string {
	return fmt.Sprintf("%s.%s.*", esFlowLogsIndexPrefix, cluster)
}

func GetL7LogsIndex(cluster string) string {
	return fmt.Sprintf("%s.%s.*", esL7LogsIndexPrefix, cluster)
}
