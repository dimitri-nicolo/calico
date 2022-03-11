package managedcluster

import (
	"time"

	tigeraapi "github.com/tigera/api/pkg/client/clientset_generated/clientset"

	"github.com/projectcalico/calico/kube-controllers/pkg/config"
	"github.com/projectcalico/calico/kube-controllers/pkg/controllers/controller"
	"github.com/projectcalico/calico/kube-controllers/pkg/controllers/elasticsearchconfiguration"
	"github.com/projectcalico/calico/kube-controllers/pkg/elasticsearch"
	"github.com/projectcalico/calico/kube-controllers/pkg/elasticsearch/users"
	relasticsearch "github.com/projectcalico/calico/kube-controllers/pkg/resource/elasticsearch"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

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
) Controller {
	return &Elasticsearch{
		esK8sCLI:        esK8sCLI,
		esClientBuilder: esClientBuilder,
		cfg:             cfg,
	}
}

func (e *Elasticsearch) New(
	clusterName, ownerReference string,
	managedK8sCLI, managementK8sCLI kubernetes.Interface,
	managedCalicoCLI, managementCalicoCLI tigeraapi.Interface,
	management bool, restartChan chan<- string) controller.Controller {

	return elasticsearchconfiguration.New(
		clusterName, ownerReference, managedK8sCLI, managementK8sCLI, e.esK8sCLI,
		e.esClientBuilder, management, e.cfg, restartChan)
}

func (e *Elasticsearch) HandleManagedClusterRemoved(clusterName string) {
	cleaner := users.NewEsCleaner(e.esClient)
	cleaner.DeleteResidueUsers(clusterName)
}

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

func (e *Elasticsearch) getClient(stop chan struct{}) elasticsearch.Client {
	var client elasticsearch.Client
	var err error

	connectedToEs := false
	waitTime := 5 * time.Second
	for !connectedToEs {
		select {
		case <-stop:
			return nil
		default:
			if client, err = e.esClientBuilder.Build(); err != nil {
				log.WithError(err).Error("Failed to connect to Elasticsearch")
				time.Sleep(waitTime)
				continue
			}
			connectedToEs = true
		}
	}

	return client
}
