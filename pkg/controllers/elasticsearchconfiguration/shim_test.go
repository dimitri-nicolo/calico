// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

package elasticsearchconfiguration

import (
	"github.com/projectcalico/kube-controllers/pkg/controllers/worker"
	relasticsearch "github.com/projectcalico/kube-controllers/pkg/resource/elasticsearch"
	"k8s.io/client-go/kubernetes"
)

func NewReconciler(
	clusterName string,
	esServiceURL string,
	management bool,
	managementK8sCLI kubernetes.Interface,
	managedK8sCLI kubernetes.Interface,
	esK8sCLI relasticsearch.RESTClient) worker.Reconciler {
	return &reconciler{
		clusterName:      clusterName,
		management:       management,
		managementK8sCLI: managementK8sCLI,
		managedK8sCLI:    managedK8sCLI,
		esK8sCLI:         esK8sCLI,
		esServiceURL:     esServiceURL,
	}
}
