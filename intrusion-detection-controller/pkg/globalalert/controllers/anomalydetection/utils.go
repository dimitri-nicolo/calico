package anomalydetection

import v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

func clusterConnected(managedCluster *v3.ManagedCluster) bool {
	for _, condition := range managedCluster.Status.Conditions {
		if condition.Type == v3.ManagedClusterStatusTypeConnected && condition.Status == v3.ManagedClusterStatusValueTrue {
			return true
		}
	}
	return false
}
