package managedcluster

import (
	"time"

	log "github.com/sirupsen/logrus"
	tigeraapi "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/calico/kube-controllers/pkg/config"
	"github.com/projectcalico/calico/kube-controllers/pkg/controllers/controller"
	"github.com/projectcalico/calico/kube-controllers/pkg/controllers/elasticsearchconfiguration"
	"github.com/projectcalico/calico/kube-controllers/pkg/elasticsearch"
	"github.com/projectcalico/calico/kube-controllers/pkg/elasticsearch/users"
	relasticsearch "github.com/projectcalico/calico/kube-controllers/pkg/resource/elasticsearch"
)

// Elasticsearch is a ControllerManager implementation responsible for managing elasticsearch resources for a manage
// cluster.
type Elasticsearch struct {
	esK8sCLI        relasticsearch.RESTClient
	esClientBuilder elasticsearch.ClientBuilder
	cfg             config.ElasticsearchCfgControllerCfg
	esClient        elasticsearch.Client
}

func NewElasticsearchController(
	esK8sCLI relasticsearch.RESTClient,
	esClientBuilder elasticsearch.ClientBuilder,
	cfg config.ElasticsearchCfgControllerCfg,
) ControllerManager {
	return &Elasticsearch{
		esK8sCLI:        esK8sCLI,
		esClientBuilder: esClientBuilder,
		cfg:             cfg,
	}
}

func (e *Elasticsearch) CreateController(
	clusterName, ownerReference string,
	managedK8sCLI, managementK8sCLI kubernetes.Interface,
	managedCalicoCLI, managementCalicoCLI tigeraapi.Interface,
	restartChan chan<- string) controller.Controller {

	return elasticsearchconfiguration.New(
		clusterName, ownerReference, managedK8sCLI, managementK8sCLI, e.esK8sCLI,
		e.esClientBuilder, false, e.cfg, restartChan)
}

// HandleManagedClusterRemoved cleans up the Elasticsearch users and roles for the managed cluster that was removed.
func (e *Elasticsearch) HandleManagedClusterRemoved(clusterName string) {
	cleaner := users.NewEsCleaner(e.esClient)
	cleaner.DeleteResidueUsers(clusterName)
}

// Initialize cleans up any users and roles that exist in Elasticsearch for managed cluster that are not longer around.
func (e *Elasticsearch) Initialize(stop chan struct{}, clusters ...string) {
	connectedToEs := false
	waitTime := 5 * time.Second

	var err error

	for !connectedToEs {
		select {
		case <-stop:
			return
		default:
			if e.esClient, err = e.esClientBuilder.Build(); err != nil {
				log.WithError(err).Error("Failed to connect to Elasticsearch")
				time.Sleep(waitTime)
				continue
			}
			connectedToEs = true
		}
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
				if err := e.deleteUsersAtStarUp(clusters...); err != nil {
					log.WithError(err).Error("Failed to clean up Elasticsearch users")
					time.Sleep(waitTime)
					continue
				}

				success = true
			}
		}

		log.Info("Successful ran Elasticsearch user clean up")
	}()
}

func (e *Elasticsearch) deleteUsersAtStarUp(clusterNames ...string) error {
	clusterNameMap := map[string]bool{}
	for _, name := range clusterNames {
		clusterNameMap[name] = true
	}

	cleaner := users.NewEsCleaner(e.esClient)
	return cleaner.DeleteAllResidueUsers(clusterNameMap)
}
