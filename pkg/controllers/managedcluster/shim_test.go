// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

package managedcluster

import (
	"github.com/projectcalico/kube-controllers/pkg/controllers/worker"
	relasticsearch "github.com/projectcalico/kube-controllers/pkg/resource/elasticsearch"
	tigeraapi "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	"k8s.io/client-go/kubernetes"
)

func NewManagementClusterChangeReconciler(
	managementk8sCLI kubernetes.Interface,
	calicok8sCLI tigeraapi.Interface,
	esk8sCLI relasticsearch.RESTClient,
	changeNotify chan bool,
) worker.Reconciler {
	return newManagementClusterChangeReconciler(managementk8sCLI, calicok8sCLI, esk8sCLI, changeNotify)
}
