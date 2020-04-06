// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

package managedcluster

import (
	"fmt"

	"github.com/projectcalico/kube-controllers/pkg/config"

	esv1 "github.com/elastic/cloud-on-k8s/pkg/apis/elasticsearch/v1"
	"github.com/projectcalico/kube-controllers/pkg/controllers/controller"
	"github.com/projectcalico/kube-controllers/pkg/controllers/worker"
	"github.com/projectcalico/kube-controllers/pkg/resource"
	relasticsearch "github.com/projectcalico/kube-controllers/pkg/resource/elasticsearch"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	tigeraapi "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// managedClusterController is responsible for controllers (from the elasticsearchconfiguration package) for every managed
// cluster it finds to managed the elasticsearch configuration for a cluster. This controller watches the ManagedCluster
// resources and it runs a controller for each connected ManagedCluster it finds.
//
// This controller watches various other components in the management cluster, like elasticsearch, and recreates the watches
// if those components have changed in a way that effects the Elasticsearch configuration for the managed clusters. For
// instance, if Elasticsearch is completely recreated, we need to regenerate the users / roles, so recreating the Elasticsearch
// configuration controllers for the managed clusters will kick off the Reconcile functions of those controllers which will
// compare the Elasticsearch hash in the user secrets in the cluster to the hash of the new Elasticsearch cluster and recreate
// the users and secrets if they differ (and they will if the Elasticsearch cluster has been recreated)
type managedClusterController struct {
	managedClusterWorker worker.Worker
	mgmtChangeWorker     worker.Worker
	calicoCLI            *tigeraapi.Clientset
	cfg                  config.ManagedClusterControllerConfig
}

func New(
	createManagedk8sCLI func(string) (kubernetes.Interface, error),
	esServiceURL string,
	managementK8sCLI *kubernetes.Clientset,
	calicok8sCLI *tigeraapi.Clientset,
	esk8sCLI relasticsearch.RESTClient,
	cfg config.ManagedClusterControllerConfig) controller.Controller {

	mcReconciler := &managedClusterESControllerReconciler{
		createManagedK8sCLI:      createManagedk8sCLI,
		esServiceURL:             esServiceURL,
		managementK8sCLI:         managementK8sCLI,
		calicoCLI:                calicok8sCLI,
		esK8sCLI:                 esk8sCLI,
		managedClustersStopChans: make(map[string]chan struct{}),
		cfg:                      cfg.ElasticConfig,
	}
	mgmtChangeReconciler := newManagementClusterChangeReconciler(managementK8sCLI, calicok8sCLI, esk8sCLI, mcReconciler.listenForRebootNotify())
	// Watch the ManagedCluster resources for changes
	managedClusterWorker := worker.New(mcReconciler)
	managedClusterWorker.AddWatch(
		cache.NewListWatchFromClient(calicok8sCLI.ProjectcalicoV3().RESTClient(), "managedclusters", "", fields.Everything()),
		&v3.ManagedCluster{},
	)

	mgmtChangeWorker := worker.New(mgmtChangeReconciler)
	mgmtChangeWorker.AddWatch(
		cache.NewListWatchFromClient(esk8sCLI, "elasticsearches", resource.TigeraElasticsearchNamespace,
			fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", resource.DefaultTSEEInstanceName))),
		&esv1.Elasticsearch{},
	)

	mgmtChangeWorker.AddWatch(
		cache.NewListWatchFromClient(managementK8sCLI.CoreV1().RESTClient(), "secrets", resource.OperatorNamespace,
			fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", resource.ElasticsearchCertSecret))),
		&corev1.Secret{},
	)

	mgmtChangeWorker.AddWatch(
		cache.NewListWatchFromClient(managementK8sCLI.CoreV1().RESTClient(), "secrets", resource.TigeraElasticsearchNamespace,
			fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", resource.ElasticsearchUserSecret))),
		&corev1.Secret{},
	)

	mgmtChangeWorker.AddWatch(
		cache.NewListWatchFromClient(managementK8sCLI.CoreV1().RESTClient(), "configmaps", resource.OperatorNamespace,
			fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", resource.ElasticsearchConfigMapName))),
		&corev1.ConfigMap{},
	)

	return &managedClusterController{
		managedClusterWorker: managedClusterWorker,
		mgmtChangeWorker:     mgmtChangeWorker,
		calicoCLI:            calicok8sCLI,
		cfg:                  cfg,
	}
}

func (c *managedClusterController) Run(stop chan struct{}) {
	go c.managedClusterWorker.Run(c.cfg.NumberOfWorkers, stop)
	go c.mgmtChangeWorker.Run(c.cfg.NumberOfWorkers, stop)

	<-stop
}
