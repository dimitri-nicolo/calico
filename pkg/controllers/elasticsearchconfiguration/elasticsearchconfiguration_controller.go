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
)

const (
	UserChangeHashLabel        = "tigera-change-hash"
	ElasticsearchUserNameLabel = "tigera-elasticsearch-user"
)

// esConfigController is responsible managing the elasticsearch configuration for a particular cluster. In this controller,
// we have the notion of a managed cluster and a management cluster. Elasticsearch runs in the management cluster, and this
// controller does the follow:
// - Creates the elasticsearch users and roles in elasticsearch for the components in the the managed cluster and stores
//   them in secrets in the managed cluster
// - Copies over the secret in the management cluster that contains the elasticsearch tls certificate
// - Copies the ConfigMap that has other elasticsearch related configuration that a managed cluster needs
//
// When this controller starts it runs the Reconcile function of the reconciler in this package, which creates / updates
// everything that the components in the managed cluster need to access / update elasticsearch. Watches are added to the
// k8s components created in the managed cluster, and if any of them change then the reconcilers Reconcile function is run
// and the changed components are likely updated.
//
// Something to note is that this controller can run on a standalone / management cluster, by setting the managedK8sCLI
// equal to the managementK8sCLI and setting the management flag to true.
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

	if !management {
		w.AddWatch(
			cache.NewListWatchFromClient(managedK8sCLI.CoreV1().RESTClient(), "secrets", resource.OperatorNamespace,
				fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", resource.ElasticsearchCertSecret))),
			&corev1.Secret{},
			worker.ResourceWatchUpdate, worker.ResourceWatchDelete,
		)

		w.AddWatch(
			cache.NewListWatchFromClient(managedK8sCLI.CoreV1().RESTClient(), "configmaps", resource.OperatorNamespace,
				fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", resource.ElasticsearchConfigMapName))),
			&corev1.ConfigMap{},
			worker.ResourceWatchUpdate, worker.ResourceWatchDelete,
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
