// Copyright (c) 2022 Tigera Inc. All rights reserved.

package managedcluster

import (
	"context"

	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	lmak8s "github.com/projectcalico/calico/lma/pkg/k8s"

	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/cache"
	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/controller"
	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/namespace"
	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/policyrecommendation"
	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/stagednetworkpolicies"
	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/syncer"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"

	log "github.com/sirupsen/logrus"
)

type managedClusterReconciler struct {
	managementStandaloneCalico calicoclient.ProjectcalicoV3Interface
	clientFactory              lmak8s.ClientSetFactory
	elasticClientFactory       lmaelastic.ClusterContextClientFactory
	cache                      map[string]*managedClusterState
}

type managedClusterState struct {
	clusterName string
	controllers []controller.Controller
	cancel      context.CancelFunc
}

// Reconcile listens for ManagedCluster Resource and creates the caches and Controllers for the attached ManagedCluster
// based on the ClientSet created for the ManagedCluster.  Controllers created for the ManagedCluster will watch Kubernetes
// resources only on their assigned ManagedClusters.  All connections opened by the Controllers for the ManagedCluster
// will go through the Voltron - Guardian tunnel.
func (r *managedClusterReconciler) Reconcile(namespacedName types.NamespacedName) error {
	mc, err := r.managementStandaloneCalico.ManagedClusters().Get(context.Background(), namespacedName.Name, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	if k8serrors.IsNotFound(err) || !r.isManagedClusterConnected(*mc) {
		// we are done closing the goroutine, nothing more to do for deleted managed cluster
		clusterState, ok := r.cache[namespacedName.Name]
		if ok {
			r.cancelPolicyRecControllerForManagedCluster(namespacedName.Name, clusterState)
		}
		return nil
	}

	return r.startRecommendationPolicyControllerForManagedCluster(*mc)
}

func (r *managedClusterReconciler) isManagedClusterConnected(mc v3.ManagedCluster) bool {
	for _, condition := range mc.Status.Conditions {
		if condition.Type == v3.ManagedClusterStatusTypeConnected && condition.Status == v3.ManagedClusterStatusValueTrue {
			return true
		}
	}
	return false
}

func (r *managedClusterReconciler) startRecommendationPolicyControllerForManagedCluster(mc v3.ManagedCluster) error {
	ctx, cancel := context.WithCancel(context.Background())

	clientSetForCluster, err := r.clientFactory.NewClientSetForApplication(mc.Name)
	if err != nil {
		log.WithError(err).Errorf("failed to create Calico client for managed cluster %s", mc.Name)
		cancel()
		return err
	}

	mcLMAElasticClient, err := r.createLMAElasticClientForManagedCluster(ctx, mc.Name)
	if err != nil {
		log.WithError(err).Errorf("failed to create Elastic client for managed cluster %s", mc.Name)
		cancel()
		return err
	}

	// SNP cache
	snpResourceCache := cache.NewSynchronizedObjectCache[*v3.StagedNetworkPolicy]()

	// Namespace cache
	namespaceCache := cache.NewSynchronizedObjectCache[*v1.Namespace]()

	// NetworkSets Cache
	networkSetCache := cache.NewSynchronizedObjectCache[*v3.NetworkSet]()

	caches := &syncer.CacheSet{
		Namespaces:            namespaceCache,
		NetworkSets:           networkSetCache,
		StagedNetworkPolicies: snpResourceCache,
	}

	// Setup Synchronizer
	cacheSynchronizer := syncer.NewCacheSynchronizer(clientSetForCluster, *caches)

	policyRecController := policyrecommendation.NewPolicyRecommendationController(
		clientSetForCluster.ProjectcalicoV3(),
		&mcLMAElasticClient,
		cacheSynchronizer,
		caches,
		mc.Name,
	)
	stagednetworkpoliciesController := stagednetworkpolicies.NewStagedNetworkPolicyController(
		clientSetForCluster.ProjectcalicoV3(),
		snpResourceCache,
	)
	namespaceController := namespace.NewNamespaceController(
		clientSetForCluster,
		namespaceCache,
		cacheSynchronizer,
	)

	controllers := []controller.Controller{policyRecController, stagednetworkpoliciesController, namespaceController}

	go func() {
		for _, controller := range controllers {
			controller.Run(ctx)
		}
	}()

	r.cache[mc.Name] = &managedClusterState{
		clusterName: mc.Name,
		controllers: controllers,
		cancel:      cancel,
	}

	return nil
}

func (r *managedClusterReconciler) createLMAElasticClientForManagedCluster(ctx context.Context, clusterName string) (lmaelastic.Client, error) {
	envCfg := lmaelastic.MustLoadConfig()
	envCfg.ElasticIndexSuffix = clusterName
	lmaESClient, err := r.elasticClientFactory.ClientForCluster(clusterName)
	if err != nil {
		return nil, err
	}

	return lmaESClient, nil
}

func (r *managedClusterReconciler) Close() {
	for key, state := range r.cache {
		r.cancelPolicyRecControllerForManagedCluster(key, state)
	}
}

func (r *managedClusterReconciler) cancelPolicyRecControllerForManagedCluster(key string, state *managedClusterState) {
	for _, controller := range state.controllers {
		controller.Close()
	}

	delete(r.cache, key)
	state.cancel()
}
