// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package managedcluster

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	tigeraapi "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/projectcalico/calico/kube-controllers/pkg/config"
	"github.com/projectcalico/calico/kube-controllers/pkg/controllers/controller"
	"github.com/projectcalico/calico/kube-controllers/pkg/controllers/worker"
)

type Controller interface {
	New(clusterName, ownerReference string, managedK8sCLI,
		managementK8sCLI kubernetes.Interface,
		managedCalicoCLI, managementCalicoCLI tigeraapi.Interface,
		management bool, restartChan chan<- string) controller.Controller
	HandleManagedClusterRemoved(clusterName string)
	Initialize(stop chan struct{}, clusters ...string)
}

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
	createManagedK8sCLI func(string) (kubernetes.Interface, *tigeraapi.Clientset, error)
	calicoCLI           *tigeraapi.Clientset
	cfg                 config.ManagedClusterControllerConfig
	managementK8sCLI    *kubernetes.Clientset
	restartChan         chan<- string
	controllers         []Controller
}

func New(
	createManagedK8sCLI func(string) (kubernetes.Interface, *tigeraapi.Clientset, error),
	managementK8sCLI *kubernetes.Clientset,
	calicok8sCLI *tigeraapi.Clientset,
	cfg config.ManagedClusterControllerConfig,
	restartChan chan<- string,
	controllers []Controller,
) controller.Controller {

	return &managedClusterController{
		createManagedK8sCLI: createManagedK8sCLI,
		calicoCLI:           calicok8sCLI,
		cfg:                 cfg,
		managementK8sCLI:    managementK8sCLI,
		restartChan:         restartChan,
		controllers:         controllers,
	}
}

// fetchRegisteredManagedClustersNames returns the name for the managed cluster as set or an error
// if the requests to k8s API failed
func (c *managedClusterController) fetchRegisteredManagedClustersNames(stop chan struct{}) []string {
	success := false
	waitTime := 5 * time.Second

	var err error
	var managedClusters *v3.ManagedClusterList
	for !success {
		select {
		case <-stop:
			return nil
		default:
			if managedClusters, err = c.calicoCLI.ProjectcalicoV3().ManagedClusters().List(context.Background(), metav1.ListOptions{}); err != nil {
				log.WithError(err).Error("Failed to clean up Elasticsearch users")
				time.Sleep(waitTime)
				continue
			}

			success = true
		}
	}

	var registeredClusters []string
	for _, managedCluster := range managedClusters.Items {
		registeredClusters = append(registeredClusters, managedCluster.Name)
	}

	return registeredClusters
}

func (c *managedClusterController) Run(stop chan struct{}) {
	clusterNames := c.fetchRegisteredManagedClustersNames(stop)
	for _, controller := range c.controllers {
		controller.Initialize(stop, clusterNames...)
	}

	mcReconciler := &reconciler{
		createManagedK8sCLI:      c.createManagedK8sCLI,
		managementK8sCLI:         c.managementK8sCLI,
		managedClustersStopChans: make(map[string]chan struct{}),
		restartChan:              c.restartChan,
		calicoCLI:                c.calicoCLI,
	}

	// Watch the ManagedCluster resources for changes
	managedClusterWorker := worker.New(mcReconciler)
	managedClusterWorker.AddWatch(
		cache.NewListWatchFromClient(c.calicoCLI.ProjectcalicoV3().RESTClient(), "managedclusters", "", fields.Everything()),
		&v3.ManagedCluster{},
	)

	go managedClusterWorker.Run(c.cfg.NumberOfWorkers, stop)

	<-stop
}
