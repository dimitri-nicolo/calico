// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

package elasticsearchconfiguration

import (
	"fmt"

	relasticsearch "github.com/projectcalico/kube-controllers/pkg/resource/elasticsearch"

	"k8s.io/apimachinery/pkg/fields"

	"github.com/projectcalico/kube-controllers/pkg/controllers/worker"

	"github.com/projectcalico/kube-controllers/pkg/controllers/controller"
	"github.com/projectcalico/kube-controllers/pkg/resource"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	esalpha1 "github.com/elastic/cloud-on-k8s/pkg/apis/elasticsearch/v1alpha1"
)

const (
	UserChangeHashLabel        = "tigera-change-hash"
	ElasticsearchUserNameLabel = "tigera-elasticsearch-user"
)

// esConfigController is responsible managing the elasticsearch configuration for a particular cluster (management, standalone,
// managed). In this controller, we have the notion of a managed cluster and a management cluster. The management cluster
// can be treated like a managed cluster by using the kube config for the management cluster in the  managedK8sCli. In this
// case the "management" flag should be set, as the elasticsearch configuration that needs to be created / copied differs
// between a management (same as standalone) and a managed cluster. Depending on if the management flag is set, this controller
// does the following:
//
// If the management flag is false:
// - Creates the elasticsearch users and roles in elasticsearch for the components in the the managed cluster and stores
//   them in secrets in the managed cluster. There are certain components that only run in the management cluster, like
//   the Manager and the ComplianceServer, and those users and roles will not be created
// - Copies over the Secret in the management cluster that contains the elasticsearch tls certificate
// - Copies the ConfigMap that has other elasticsearch related configuration that a managed cluster needs
//
// If the management flag is true:
// - Creates the elasticsearch users and roles in elasticsearch for the components in the the management cluster and stores
//   them in secrets in the management cluster. There are certain components that only run in the management cluster, like
//   the Manager and the ComplianceServer, and those users and roles will be created
//
// A note on when the Reconcile function is run:
// Regardless of whether the management flag is true or false, we add watches using the managedK8sCli to watch the component
// user secrets created, the Elasticsearch tls secret, and the Elasticsearch config map. If the the management flag is set
// to true, we also add a watch for changes in Elasticsearch. If the management flag is set to false, it is assumed that
// this controller is being used to reconcile a managed cluster, and in this case the ManagedCluster controller watches
// elasticsearch in the management cluster and restarts the ElasticsearchConfiguration controllers for the managed clusters
// if there is a significant change in elasticsearch.
//
// When this controller starts it runs the Reconcile function of the reconciler in this package, which creates / updates
// everything that the components in the managed cluster need to access / update elasticsearch. Watches are added to the
// k8s components created in the managed cluster, and if any of them change then the reconcilers Reconcile function is run
// and the changed components are likely updated.
//
// Note that this controller does not react to changes in the management cluster (unless, of course, the managedK8sCLI points
// to the management cluster). If something changes in the management cluster, this controller should just be recreated
// and re run.
type esConfigController struct {
	clusterName string
	r           *reconciler
	worker      worker.Worker
}

func New(
	clusterName string,
	esServiceURL string,
	managedK8sCLI kubernetes.Interface,
	managementK8sCLI kubernetes.Interface,
	esK8sCLI relasticsearch.RESTClient,
	management bool) controller.Controller {
	r := &reconciler{
		clusterName:      clusterName,
		esServiceURL:     esServiceURL,
		managementK8sCLI: managementK8sCLI,
		managedK8sCLI:    managedK8sCLI,
		esK8sCLI:         esK8sCLI,
		management:       management,
	}
	w := worker.New(r)

	w.AddWatch(
		cache.NewFilteredListWatchFromClient(managedK8sCLI.CoreV1().RESTClient(), "secrets", resource.OperatorNamespace, func(options *metav1.ListOptions) {
			options.LabelSelector = ElasticsearchUserNameLabel
		}),
		&corev1.Secret{},
		worker.ResourceWatchUpdate, worker.ResourceWatchDelete,
	)

	notifications := []worker.ResourceWatch{worker.ResourceWatchUpdate, worker.ResourceWatchDelete}
	// if this is for a managed cluster this controller adds the public cert secret and the config map so we don't need
	// to be notified when it's added
	if management {
		notifications = append(notifications, worker.ResourceWatchAdd)
	}

	w.AddWatch(
		cache.NewListWatchFromClient(managedK8sCLI.CoreV1().RESTClient(), "secrets", resource.OperatorNamespace,
			fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", resource.ElasticsearchCertSecret))),
		&corev1.Secret{},
		notifications...,
	)

	w.AddWatch(
		cache.NewListWatchFromClient(managedK8sCLI.CoreV1().RESTClient(), "configmaps", resource.OperatorNamespace,
			fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", resource.ElasticsearchConfigMapName))),
		&corev1.ConfigMap{},
		notifications...,
	)

	if management {
		// if this is a management cluster then we need to watch elasticsearch for changes. The manage cluster controller
		// does that in the case the this isn't for a management cluster
		w.AddWatch(
			cache.NewListWatchFromClient(esK8sCLI, "elasticsearches", resource.TigeraElasticsearchNamespace,
				fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", resource.DefaultTSEEInstanceName))),
			&esalpha1.Elasticsearch{},
		)

		w.AddWatch(
			cache.NewListWatchFromClient(managementK8sCLI.CoreV1().RESTClient(), "secrets", resource.TigeraElasticsearchNamespace,
				fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", resource.ElasticsearchUserSecret))),
			&corev1.Secret{},
		)
	}

	return &esConfigController{
		clusterName: clusterName,
		r:           r,
		worker:      w,
	}
}

func (c *esConfigController) Run(threadiness int, reconcilerPeriod string, stop chan struct{}) {
	logger := log.WithField("cluster", c.clusterName)
	logger.Info("Starting Elasticsearch configuration controller")
	// kick off the reconciler so we know all the es credentials are created
	if err := c.r.Reconcile(types.NamespacedName{}); err != nil {
		logger.WithError(err).Error("failed initial reconcile")
	}

	go c.worker.Run(threadiness, stop)

	<-stop
}
