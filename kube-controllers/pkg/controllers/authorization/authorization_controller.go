// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package authorization

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/projectcalico/calico/kube-controllers/pkg/config"
	"github.com/projectcalico/calico/kube-controllers/pkg/controllers/controller"
	"github.com/projectcalico/calico/kube-controllers/pkg/controllers/worker"
	"github.com/projectcalico/calico/kube-controllers/pkg/elasticsearch"
	"github.com/projectcalico/calico/kube-controllers/pkg/elasticsearch/users"
	"github.com/projectcalico/calico/kube-controllers/pkg/resource"
)

// authorizationController synchronizes kubernetes RBAC authorization with Elasticsearch authorization. At a high level,
// this controller gathers information about all the ClusterRoles and ClusterRoleBindings in the kubernetes cluster, and
// creates the appropriate role mappings in Elasticsearch so that an authenticated Elasticsearch user will have the permissions
// designated to them through kubernetes RBAC.
type authorizationController struct {
	k8sCLI                 kubernetes.Interface
	esClientBuilder        elasticsearch.ClientBuilder
	numWorkers             int
	resyncPeriod           time.Duration
	usernamePrefix         string
	groupPrefix            string
	enableESOIDCWorkaround bool
}

func New(k8sCLI kubernetes.Interface, esClientBuilder elasticsearch.ClientBuilder, config *config.AuthorizationControllerCfg) controller.Controller {
	// TODO remove this default when this is properly hooked up to using KubeControllerConfiguration
	resyncPeriod := config.ReconcilerPeriod
	if config.ReconcilerPeriod == 0 {
		resyncPeriod = 5 * time.Minute
	}

	return &authorizationController{
		k8sCLI:                 k8sCLI,
		esClientBuilder:        esClientBuilder,
		numWorkers:             config.NumberOfWorkers,
		resyncPeriod:           resyncPeriod,
		usernamePrefix:         config.OIDCAuthUsernamePrefix,
		groupPrefix:            config.OIDCAuthGroupPrefix,
		enableESOIDCWorkaround: config.EnableElasticsearchOIDCWorkaround,
	}
}

func (c *authorizationController) Run(stop chan struct{}) {
	log.Info("Starting authorization controller.")

	resourceUpdatesChan := make(chan resourceUpdate, 100)
	defer close(resourceUpdatesChan)

	var esCLI elasticsearch.Client
	var err error

	// Don't proceed until the an Elasticsearch client can be created. An Elasticsearch client will likely not be created
	// in the event Elasticsearch is not yet running
	log.Debug("Initializing Elasticsearch client.")
	stopped := retry(stop, 5*time.Second, "Waiting for Elasticsearch to initialize", func() error {
		esCLI, err = c.esClientBuilder.Build()
		return err
	})
	if stopped {
		return
	}

	stopped = retry(stop, 5*time.Second, "Failed to create the authorizations roles", func() error {
		roles := users.GetAuthorizationRoles(rbacv1.ResourceAll)
		roles = append(roles, users.GetGlobalAuthorizationRoles()...)
		return esCLI.CreateRoles(roles...)
	})
	if stopped {
		return
	}

	clusterRoleBindingWorker := worker.New(&clusterRoleBindingReconciler{
		k8sCLI:          c.k8sCLI,
		resourceUpdates: resourceUpdatesChan,
	})
	clusterRoleBindingWorker.AddWatch(
		cache.NewListWatchFromClient(c.k8sCLI.RbacV1().RESTClient(), "clusterrolebindings", "", fields.Everything()),
		&rbacv1.ClusterRoleBinding{},
	)

	clusterRoleWorker := worker.New(&clusterRoleReconciler{
		k8sCLI:          c.k8sCLI,
		resourceUpdates: resourceUpdatesChan,
	})
	clusterRoleWorker.AddWatch(
		cache.NewListWatchFromClient(c.k8sCLI.RbacV1().RESTClient(), "clusterroles", "", fields.Everything()),
		&rbacv1.ClusterRole{},
	)

	log.Debug("Starting ClusterRole and ClusterRoleBinding workers.")
	// Start the workers before initialising the cache and removing the stale values so we don't lose any deletion updates.
	go clusterRoleWorker.Run(c.numWorkers, stop)
	go clusterRoleBindingWorker.Run(c.numWorkers, stop)

	if c.enableESOIDCWorkaround {

		configMapWorker := worker.New(&configMapReconciler{
			k8sCLI:          c.k8sCLI,
			resourceUpdates: resourceUpdatesChan,
		})
		configMapWorker.AddWatch(
			cache.NewListWatchFromClient(c.k8sCLI.CoreV1().RESTClient(), "configmaps", resource.TigeraElasticsearchNamespace,
				fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", resource.OIDCUsersConfigMapName))),
			&corev1.ConfigMap{},
		)

		secretWorker := worker.New(&secretReconciler{
			k8sCLI:          c.k8sCLI,
			resourceUpdates: resourceUpdatesChan,
		})

		secretWorker.AddWatch(
			cache.NewListWatchFromClient(c.k8sCLI.CoreV1().RESTClient(), "secrets", resource.TigeraElasticsearchNamespace,
				fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", resource.OIDCUsersEsSecreteName))),
			&corev1.Secret{},
		)

		log.Debug("Starting ConfigMap & Secret workers.")
		go configMapWorker.Run(c.numWorkers, stop)
		go secretWorker.Run(c.numWorkers, stop)
	}

	updateHandler := c.initializeK8sUpdateHandler(stop, esCLI, resourceUpdatesChan)
	go updateHandler.listenAndSynchronize()

	ticker := time.NewTicker(c.resyncPeriod)
	defer ticker.Stop()

	done := false
	for !done {
		select {
		case <-ticker.C:
			log.Info("Resyncing")
			updateHandler.stop()
			updateHandler = c.initializeK8sUpdateHandler(stop, esCLI, resourceUpdatesChan)

			go updateHandler.listenAndSynchronize()

			log.Info("Finished Resyncing")
		case <-stop:
			done = true
		}
	}

	<-stop
}

// initializeK8sUpdateHandler initializes a k8sUpdateHandler and resyncs the backend with cache.
func (c *authorizationController) initializeK8sUpdateHandler(stop chan struct{}, esCLI elasticsearch.Client, resourceUpdatesChan chan resourceUpdate) *k8sUpdateHandler {
	var synchronizer k8sRBACSynchronizer

	if c.enableESOIDCWorkaround {
		synchronizer = newNativeUserSynchronizer(stop, esCLI, c.k8sCLI)
	} else {
		synchronizer = newRoleMappingSynchronizer(stop, esCLI, c.k8sCLI, c.usernamePrefix, c.groupPrefix)
	}

	if synchronizer == nil {
		return nil
	}

	updateHandler := &k8sUpdateHandler{
		stopChan:        make(chan chan struct{}),
		resourceUpdates: resourceUpdatesChan,
		synchronizer:    synchronizer,
	}

	// Clean up any items in backend that don't have an associated cached item and also adds missing item to backend if associated cached item is available.
	stopped := retry(stop, 5*time.Second, "failed to remove stale Elasticsearch mappings", updateHandler.synchronizer.resync)
	if stopped {
		return nil
	}

	return updateHandler
}

// retry continuously runs the given function f until it succeeds, i.e. doesn't return an error. If f eventually succeeds,
// false is returned. If a signal is sent down the stop channel the function returns immediately with a value of true. The
// return value signals whether this function stopped trying to run f because a stop signal was sent, or because f succeeded.
func retry(stop chan struct{}, waitTime time.Duration, message string, f func() error) bool {
	done := false
	for !done {
		select {
		case <-stop:
			return true
		default:
			if err := f(); err != nil {
				log.WithError(err).Error(message)
				time.Sleep(waitTime)
				continue
			}

			done = true
		}
	}

	return false
}
