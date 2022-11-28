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

	policyRecController := policyrecommendation.NewPolicyRecommendationController(clientSetForCluster.ProjectcalicoV3(), mcLMAElasticClient)
	stagednetworkpoliciesController := stagednetworkpolicies.NewStagedNetworkPolicyController(clientSetForCluster.ProjectcalicoV3(), snpResourceCache)
	namespaceController := namespace.NewNamespaceController(clientSetForCluster, namespaceCache)

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

	if err := lmaESClient.CreateEventsIndex(ctx); err != nil {
		log.WithError(err).Errorf("failed to create events index for managed cluster %s", clusterName)
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
