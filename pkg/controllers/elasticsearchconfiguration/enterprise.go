// Copyright (c) 2021 Tigera, Inc. All rights reserved.

// +build !tesla

package elasticsearchconfiguration

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/kube-controllers/pkg/resource"
)

// enableElasticsearchWatch enables watching the Elasticsearch CR in the Enterprise variant.
var enableElasticsearchWatch = true

// reconcileConfigMap copies the tigera-secure-elasticsearch ConfigMap in the management cluster to the managed cluster,
// changing the clusterName data value to the cluster name this ConfigMap is being copied to
func (c *reconciler) reconcileConfigMap() error {
	configMap, err := c.managementK8sCLI.CoreV1().ConfigMaps(c.managementOperatorNamespace).Get(context.Background(), resource.ElasticsearchConfigMapName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	configMap.ObjectMeta.Namespace = c.managedOperatorNamespace
	cp := resource.CopyConfigMap(configMap)
	cp.Data["clusterName"] = c.clusterName
	if err := resource.WriteConfigMapToK8s(c.managedK8sCLI, cp); err != nil {
		return err
	}
	return nil
}
