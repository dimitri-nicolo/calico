// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package elasticsearchconfiguration

import (
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/calico/kube-controllers/pkg/controllers/worker"
	"github.com/projectcalico/calico/kube-controllers/pkg/elasticsearch"
	"github.com/projectcalico/calico/kube-controllers/pkg/resource"
	relasticsearch "github.com/projectcalico/calico/kube-controllers/pkg/resource/elasticsearch"
)

func NewReconciler(
	esClientBuilder elasticsearch.ClientBuilder,
	managementK8sCLI kubernetes.Interface,
	managedK8sCLI kubernetes.Interface,
	esK8sCLI relasticsearch.RESTClient,
	restartChan chan string,
	setOpts func(*reconciler),
) worker.Reconciler {
	r := &reconciler{
		clusterName:                 "cluster",
		ownerReference:              "",
		management:                  true,
		managementK8sCLI:            managementK8sCLI,
		managementOperatorNamespace: resource.OperatorNamespace,
		managedK8sCLI:               managedK8sCLI,
		managedOperatorNamespace:    resource.OperatorNamespace,
		esK8sCLI:                    esK8sCLI,
		esClientBuilder:             esClientBuilder,
		restartChan:                 restartChan,
	}
	if setOpts != nil {
		setOpts(r)
	}
	return r
}
