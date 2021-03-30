// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package managedcluster

import (
	"context"
	"time"

	"github.com/projectcalico/kube-controllers/pkg/elasticsearch/users"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	tigeraapi "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/projectcalico/kube-controllers/pkg/config"
	"github.com/projectcalico/kube-controllers/pkg/elasticsearch"

	"github.com/projectcalico/kube-controllers/pkg/controllers/controller"
	"github.com/projectcalico/kube-controllers/pkg/controllers/worker"
	relasticsearch "github.com/projectcalico/kube-controllers/pkg/resource/elasticsearch"
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
	createManagedk8sCLI func(string) (kubernetes.Interface, *tigeraapi.Clientset, error)
	calicoCLI           *tigeraapi.Clientset
	cfg                 config.ManagedClusterControllerConfig
	managementK8sCLI    *kubernetes.Clientset
	esk8sCLI            relasticsearch.RESTClient
	esClientBuilder     elasticsearch.ClientBuilder
}

func New(
	createManagedk8sCLI func(string) (kubernetes.Interface, *tigeraapi.Clientset, error),
	managementK8sCLI *kubernetes.Clientset,
	calicok8sCLI *tigeraapi.Clientset,
	esk8sCLI relasticsearch.RESTClient,
	esClientBuilder elasticsearch.ClientBuilder,
	cfg config.ManagedClusterControllerConfig) controller.Controller {

	return &managedClusterController{
		createManagedk8sCLI: createManagedk8sCLI,
		calicoCLI:           calicok8sCLI,
		cfg:                 cfg,
		managementK8sCLI:    managementK8sCLI,
		esClientBuilder:     esClientBuilder,
		esk8sCLI:            esk8sCLI,
	}
}

// fetchRegisteredManagedClustersNames returns the name for the managed cluster as set or an error
// if the requests to k8s API failed
func (c *managedClusterController) fetchRegisteredManagedClustersNames() (map[string]bool, error) {
	managedClusters, err := c.calicoCLI.ProjectcalicoV3().ManagedClusters().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	registeredClusters := make(map[string]bool)
	for _, managedCluster := range managedClusters.Items {
		registeredClusters[managedCluster.Name] = true
	}
	return registeredClusters, nil
}

// init will make sure that all the components are connected and functional before starting the workers
// to run reconciliation. It will try to establish connection to ElasticSearch and will not move forward
// until the connection is validated. It will create the workers (for managed and management clusters)
// and add watches to the primary resources.
func (c *managedClusterController) init(stop chan struct{}) (elasticsearch.Client, []worker.Worker) {

	// We first try to connect to ES; The workers will not start until this step is completed
	connectedToEs := false
	waitTime := 5 * time.Second
	var client elasticsearch.Client
	var err error

	for !connectedToEs {
		select {
		case <-stop:
			return nil, nil
		default:
			if client, err = c.esClientBuilder.Build(); err != nil {
				log.WithError(err).Error("Failed to connect to Elasticsearch")
				time.Sleep(waitTime)
				continue
			}
			connectedToEs = true
		}
	}

	// create the workers
	mcReconciler := &managedClusterESControllerReconciler{
		createManagedK8sCLI:      c.createManagedk8sCLI,
		managementK8sCLI:         c.managementK8sCLI,
		calicoCLI:                c.calicoCLI,
		esK8sCLI:                 c.esk8sCLI,
		managedClustersStopChans: make(map[string]chan struct{}),
		cfgEs:                    c.cfg.ElasticConfig,
		cfgLic:                   c.cfg.LicenseConfig,
		esClientBuilder:          c.esClientBuilder,
		esClient:                 client,
	}

	// Watch the ManagedCluster resources for changes
	managedClusterWorker := worker.New(mcReconciler)
	managedClusterWorker.AddWatch(
		cache.NewListWatchFromClient(c.calicoCLI.ProjectcalicoV3().RESTClient(), "managedclusters", "", fields.Everything()),
		&v3.ManagedCluster{},
	)

	return client, []worker.Worker{managedClusterWorker}
}

func (c *managedClusterController) Run(stop chan struct{}) {

	// Establish connection to Es and create workers
	esClient, workers := c.init(stop)

	if esClient == nil || workers == nil {
		return
	}

	// Delete users and roles for deleted managed clusters. This check is required to make sure the clean up is
	// performed when kube-controllers are not running at the same time as deletion occurs
	go func() {
		success := false
		waitTime := 5 * time.Second

		for !success {
			select {
			case <-stop:
				return
			default:
				if err := deleteUsersAtStarUp(c, esClient); err != nil {
					log.WithError(err).Error("Failed to clean up Elasticsearch users")
					time.Sleep(waitTime)
					continue
				}

				success = true
			}
		}

		log.Info("Successful ran Elasticsearch user clean up")
	}()

	for _, worker := range workers {
		go worker.Run(c.cfg.NumberOfWorkers, stop)
	}

	<-stop
}

func deleteUsersAtStarUp(c *managedClusterController, esClient elasticsearch.Client) error {
	// Fetch registered managed clusters
	registeredManagedClusters, err := c.fetchRegisteredManagedClustersNames()
	if err != nil {
		return err
	}

	cleaner := users.NewEsCleaner(esClient)
	return cleaner.DeleteAllResidueUsers(registeredManagedClusters)
}
