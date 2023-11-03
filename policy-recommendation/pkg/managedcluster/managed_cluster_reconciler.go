// Copyright (c) 2022-2023 Tigera Inc. All rights reserved.

package managedcluster

import (
	"context"

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	log "github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	linseed "github.com/projectcalico/calico/linseed/pkg/client"
	lmak8s "github.com/projectcalico/calico/lma/pkg/k8s"
	"github.com/projectcalico/calico/policy-recommendation/pkg/cache"
	"github.com/projectcalico/calico/policy-recommendation/pkg/controller"
	"github.com/projectcalico/calico/policy-recommendation/pkg/namespace"
	"github.com/projectcalico/calico/policy-recommendation/pkg/policyrecommendation"
	"github.com/projectcalico/calico/policy-recommendation/pkg/stagednetworkpolicies"
	"github.com/projectcalico/calico/policy-recommendation/pkg/syncer"
	"github.com/projectcalico/calico/policy-recommendation/utils"
)

type managedClusterReconciler struct {
	client           ctrlclient.WithWatch
	clientSetFactory lmak8s.ClientSetFactory
	linseedClient    linseed.Client
	cache            map[string]*managedClusterState
	TenantNamespace  string
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

	mc := &v3.ManagedCluster{}
	err := r.client.Get(context.Background(), types.NamespacedName{Name: namespacedName.Name, Namespace: r.TenantNamespace}, mc)
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
	clog := log.WithField("cluster", mc.Name)

	ctx, cancel := context.WithCancel(context.Background())
	clog.Info("Starting policy recommendation")
	clientSet, err := r.clientSetFactory.NewClientSetForApplication(mc.Name)
	if err != nil {
		clog.WithError(err).Errorf("failed to create Calico client for managed cluster %s", mc.Name)
		cancel()
		return err
	}

	// TODO(dimitrin): Get the managed cluster clusterDomain from each managed cluster's
	// /etc/resolv.conf contents.
	// We set the cluster domain for each managed cluster to a const value, until an approach has
	// been implemented to address accessing the files contents for each managed cluster.
	serviceNameSuffix := utils.GetServiceNameSuffix(utils.DefaultClusterDomain)

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
	cacheSynchronizer := syncer.NewCacheSynchronizer(clientSet, *caches, utils.SuffixGenerator)

	suffixGenerator := utils.SuffixGenerator
	policyRecController := policyrecommendation.NewPolicyRecommendationController(
		clientSet.ProjectcalicoV3(),
		clientSet,
		r.linseedClient,
		cacheSynchronizer,
		caches,
		mc.Name,
		serviceNameSuffix,
		&suffixGenerator,
	)
	stagednetworkpoliciesController := stagednetworkpolicies.NewStagedNetworkPolicyController(
		clientSet.ProjectcalicoV3(),
		snpResourceCache,
	)
	namespaceController := namespace.NewNamespaceController(
		clientSet,
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
