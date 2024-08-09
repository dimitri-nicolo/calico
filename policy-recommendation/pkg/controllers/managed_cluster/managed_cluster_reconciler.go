// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package managed_cluster_controller

import (
	"context"
	"sync"

	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	lsclient "github.com/projectcalico/calico/linseed/pkg/client"
	lmak8s "github.com/projectcalico/calico/lma/pkg/k8s"
	"github.com/projectcalico/calico/policy-recommendation/pkg/controllers/controller"
)

type managedClusterReconciler struct {
	// ctx is the context.
	ctx context.Context

	// client is the client for the managed cluster resource.
	client ctrlclient.WithWatch

	// clientFactory is the client factory interface.
	clientFactory lmak8s.ClientSetFactory

	// linseed is the linseed client.
	linseed lsclient.Client

	// managedClusters is the map of clisterIDs to PolicyRecommendationScope controllers.
	managedClusters map[string]*managedClusterCtrlContext

	// newRecommendationScopeController is the function that creates a new PolicyRecommendationScope
	// controller. This has been added for convenience of testing.
	newRecommendationScopeController func(context.Context, string, lmak8s.ClientSet, lsclient.Client) (controller.Controller, error)

	// tenantNamespace is the namespace where the tenant resources are stored.
	tenantNamespace string

	// mutex is used to synchronize updates from managed clusters.
	mutex sync.Mutex
}

func newManagedClusterReconciler(
	ctx context.Context,
	client ctrlclient.WithWatch,
	clientFactory lmak8s.ClientSetFactory,
	linseed lsclient.Client,
	managedClusters map[string]*managedClusterCtrlContext,
	newScopeController func(context.Context, string, lmak8s.ClientSet, lsclient.Client) (controller.Controller, error),
	tenantNamespace string,
) controller.Reconciler {
	return &managedClusterReconciler{
		ctx:                              ctx,
		client:                           client,
		clientFactory:                    clientFactory,
		linseed:                          linseed,
		managedClusters:                  managedClusters,
		newRecommendationScopeController: newScopeController,
		tenantNamespace:                  tenantNamespace,
	}
}

// Reconcile will be triggered by any changes performed on the watched managed clusters.
func (r *managedClusterReconciler) Reconcile(key types.NamespacedName) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	mc := &v3.ManagedCluster{}
	err := r.client.Get(context.Background(), types.NamespacedName{Name: key.Name, Namespace: r.tenantNamespace}, mc)
	if err != nil {
		if kerrors.IsNotFound(err) {
			// The item has been deleted from the store, delete it from the cache of managed clusters.
			if mc, ok := r.managedClusters[key.Name]; ok {
				// Close the stop channel and delete the key
				close(mc.stopChan)
				delete(r.managedClusters, key.Name)
				log.WithField("clusterID", key.Name).Info("ManagedCluster has been deleted")
			}
			return nil
		} else {
			return err
		}
	}

	if _, ok := r.managedClusters[key.Name]; ok {
		// No need to handle updates to the ManagedCluster resources. The controller has been previously
		// created and is running.
		return nil
	}

	clusterID := mc.Name
	log.WithField("clusterID", clusterID).Info("Adding ManagedCluster")

	// Create clientSet for application for the managed cluster indexed by the clusterID.
	clientSet, err := r.clientFactory.NewClientSetForApplication(clusterID)
	if err != nil {
		log.WithError(err).Errorf("failed to create application client for cluster: %s", clusterID)
		return err
	}

	ctrl, err := r.newRecommendationScopeController(r.ctx, clusterID, clientSet, r.linseed)
	if err != nil {
		log.WithError(err).Error("failed to create PolicyRecommendationScope controller")
		return err
	}

	r.managedClusters[clusterID] = &managedClusterCtrlContext{
		ctrl:     ctrl,
		stopChan: make(chan struct{}),
	}
	// Run the policy recommendation scope controller for the new managed cluster.
	go r.managedClusters[clusterID].ctrl.Run(r.managedClusters[clusterID].stopChan)

	return nil
}
