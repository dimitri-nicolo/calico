// Copyright (c) 2021 Tigera, Inc. All rights reserved.

//go:build tesla
// +build tesla

package elasticsearchconfiguration

import (
	"context"
	"fmt"
	"os"

	"github.com/projectcalico/calico/kube-controllers/pkg/resource"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var tenantID = os.Getenv("ELASTIC_INDEX_TENANT_ID")

// enableElasticsearchWatch disables watching the Elasticsearch CR in the Cloud/Tesla variant since
// the Elasticsearch is external.
var enableElasticsearchWatch = false

// reconcileConfigMap copies the tigera-secure-elasticsearch ConfigMap in the management cluster to the managed cluster,
// changing the clusterName data value to include the Tenant ID (to support multi-tenancy) and the cluster name this ConfigMap is being copied to
func (c *reconciler) reconcileConfigMap() error {
	if tenantID != "" {
		configMap, err := c.managementK8sCLI.CoreV1().ConfigMaps(c.managementOperatorNamespace).Get(context.Background(), resource.ElasticsearchConfigMapName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		configMap.ObjectMeta.Namespace = c.managedOperatorNamespace
		cp := resource.CopyConfigMap(configMap)
		cp.Data["clusterName"] = fmt.Sprintf("%s.%s", tenantID, c.clusterName)
		if err := resource.WriteConfigMapToK8s(c.managedK8sCLI, cp); err != nil {
			return err
		}
		return nil
	}

	return c.eeReconcileConfigMap()
}
