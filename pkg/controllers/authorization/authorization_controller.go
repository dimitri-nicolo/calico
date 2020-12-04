// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

package authorization

import (
	"context"
	"time"

	"github.com/projectcalico/kube-controllers/pkg/config"

	"github.com/projectcalico/kube-controllers/pkg/elasticsearch/users"

	"github.com/projectcalico/kube-controllers/pkg/controllers/controller"
	"github.com/projectcalico/kube-controllers/pkg/controllers/worker"
	"github.com/projectcalico/kube-controllers/pkg/elasticsearch"
	"github.com/projectcalico/kube-controllers/pkg/rbaccache"
	"github.com/projectcalico/kube-controllers/pkg/resource"
	relasticsearch "github.com/projectcalico/kube-controllers/pkg/resource/elasticsearch"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	log "github.com/sirupsen/logrus"
)

// authorizationController synchronizes kubernetes RBAC authorization with Elasticsearch authorization. At a high level,
// this controller gathers information about all the ClusterRoles and ClusterRoleBindings in the kubernetes cluster, and
// creates the appropriate role mappings in Elasticsearch so that an authenticated Elasticsearch user will have the permissions
// designated to them through kubernetes RBAC.
type authorizationController struct {
	k8sCLI         kubernetes.Interface
	esServiceURL   string
	numWorkers     int
	resyncPeriod   time.Duration
	usernamePrefix string
	groupPrefix    string
}

func New(k8sCLI kubernetes.Interface, esServiceURL string, config *config.AuthorizationControllerCfg) controller.Controller {
	// TODO remove this default when this is properly hooked up to using KubeControllerConfiguration
	resyncPeriod := config.ReconcilerPeriod
	if config.ReconcilerPeriod == 0 {
		resyncPeriod = 5 * time.Minute
	}

	return &authorizationController{
		k8sCLI:         k8sCLI,
		esServiceURL:   esServiceURL,
		numWorkers:     config.NumberOfWorkers,
		resyncPeriod:   resyncPeriod,
		usernamePrefix: config.OIDCAuthUsernamePrefix,
		groupPrefix:    config.OIDCAuthGroupPrefix,
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
		esCLI, err = getESClient(c.k8sCLI)
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
	// Start the role workers before initialising the cache and removing the stale mappings so we don't lose any deletion
	// updates.
	go clusterRoleWorker.Run(c.numWorkers, stop)
	go clusterRoleBindingWorker.Run(c.numWorkers, stop)

	log.Debug("Synchronizing K8s RBAC with Elasticsearch role mappings")

	synchronizer := c.initializeRoleMappingSynchronizer(stop, esCLI, resourceUpdatesChan)
	go synchronizer.synchronizeRoleMappings()

	ticker := time.NewTicker(c.resyncPeriod)
	defer ticker.Stop()

	done := false
	for !done {
		select {
		case <-ticker.C:
			log.Info("Resyncing role mappings")
			synchronizer.stop()
			synchronizer = c.initializeRoleMappingSynchronizer(stop, esCLI, resourceUpdatesChan)

			go synchronizer.synchronizeRoleMappings()

			log.Info("Finished Resyncing role mappings")
		case <-stop:
			done = true
		}
	}

	<-stop
}

func (c *authorizationController) initializeRoleMappingSynchronizer(stop chan struct{}, esCLI elasticsearch.Client, resourceUpdatesChan chan resourceUpdate) *esRoleMappingSynchronizer {
	var clusterRoleCache rbaccache.ClusterRoleCache
	var err error

	log.Debug("Initializing ClusterRole cache.")
	// Initialize the cache so removeStaleMappings can calculate what Elasticsearch role mappings should be removed.
	stopped := retry(stop, 5*time.Second, "failed to initialize cache", func() error {
		clusterRoleCache, err = initializeRolesCache(c.k8sCLI)
		return err
	})
	if stopped {
		return nil
	}

	synchronizer := &esRoleMappingSynchronizer{
		stopChan:        make(chan chan struct{}),
		roleCache:       clusterRoleCache,
		esCLI:           esCLI,
		resourceUpdates: resourceUpdatesChan,
		usernamePrefix:  c.usernamePrefix,
		groupPrefix:     c.groupPrefix,
	}

	// Clean up any Elasticsearch mappings that don't have an appropriate ClusterRole with ClusterRoleBinding
	stopped = retry(stop, 5*time.Second, "failed to remove stale Elasticsearch mappings", synchronizer.removeStaleMappings)
	if stopped {
		return nil
	}

	// Clean up any Elasticsearch mappings that don't have an appropriate ClusterRole with ClusterRoleBinding
	stopped = retry(stop, 5*time.Second, "failed to sync Elasticsearch mappings", synchronizer.syncBoundClusterRoles)
	if stopped {
		return nil
	}

	return synchronizer
}

// initializeRolesCache creates and fills the rbaccache.ClusterRoleCache with the available ClusterRoles and ClusterRoleBindings.
func initializeRolesCache(k8sCLI kubernetes.Interface) (rbaccache.ClusterRoleCache, error) {
	ctx := context.Background()

	clusterRolesCache := rbaccache.NewClusterRoleCache([]string{rbacv1.UserKind, rbacv1.GroupKind}, []string{"lma.tigera.io"})

	clusterRoleBindings, err := k8sCLI.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	clusterRoles, err := k8sCLI.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, binding := range clusterRoleBindings.Items {
		clusterRolesCache.AddClusterRoleBinding(&binding)
	}

	for _, role := range clusterRoles.Items {
		clusterRolesCache.AddClusterRole(&role)
	}

	return clusterRolesCache, nil
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

func getESClient(k8sCLI kubernetes.Interface) (elasticsearch.Client, error) {
	user, password, roots, err := relasticsearch.ClientCredentialsFromK8sCLI(k8sCLI)
	if err != nil {
		return nil, err
	}

	esCli, err := elasticsearch.NewClient(resource.ElasticsearchServiceURL, user, password, roots)
	if err != nil {
		return nil, err
	}

	return esCli, nil
}
