// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package elasticsearchconfiguration

import (
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/kube-controllers/pkg/controllers/worker"
	"github.com/projectcalico/kube-controllers/pkg/elasticsearch"
	relasticsearch "github.com/projectcalico/kube-controllers/pkg/resource/elasticsearch"
)

func NewReconciler(
	clusterName string,
	ownerReference string,
	esClientBuilder elasticsearch.ClientBuilder,
	management bool,
	managementK8sCLI kubernetes.Interface,
	managedK8sCLI kubernetes.Interface,
	esK8sCLI relasticsearch.RESTClient) worker.Reconciler {
	return &reconciler{
		clusterName:      clusterName,
		ownerReference:   ownerReference,
		management:       management,
		managementK8sCLI: managementK8sCLI,
		managedK8sCLI:    managedK8sCLI,
		esK8sCLI:         esK8sCLI,
		esClientBuilder:  esClientBuilder,
	}
}
