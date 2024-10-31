// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package recommendation_scope_controller

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	rcache "github.com/projectcalico/calico/kube-controllers/pkg/cache"
	lsclient "github.com/projectcalico/calico/linseed/pkg/client"
	lmak8s "github.com/projectcalico/calico/lma/pkg/k8s"
	controller "github.com/projectcalico/calico/policy-recommendation/pkg/controllers/controller"
	reccontroller "github.com/projectcalico/calico/policy-recommendation/pkg/controllers/recommendation"
	recengine "github.com/projectcalico/calico/policy-recommendation/pkg/engine"
	"github.com/projectcalico/calico/policy-recommendation/pkg/flows"
	rectypes "github.com/projectcalico/calico/policy-recommendation/pkg/types"
)

type realClock struct{}

func (c realClock) NowRFC3339() string {
	return time.Now().Format(time.RFC3339)
}

const (
	// kindRecommendations is the kind of the recommendations resource.
	kindRecommendations = "recommendations"
)

type recommendationScopeReconciler struct {
	// ctx is the context.
	ctx context.Context

	// clusterID is the cluster ID.
	clusterID string

	// clientSet is the client-set that is used to interact with the Calico or Kubernetes API.
	clientSet lmak8s.ClientSet

	// linseed is the linseed client.
	linseed lsclient.Client

	// ctrl is the recommendation controller. Added to facilitate testing.
	ctrl controller.Controller

	// enabled is the current state of the recommendation engine.
	enabled v3.PolicyRecommendationNamespaceStatus

	// engine is the recommendation engine.
	engine *recengine.RecommendationEngine

	// stopChan is used to stop the controller.
	stopChan chan struct{}

	// clog is the logger for the controller.
	clog *log.Entry

	// mutex is used to synchronize access to the cache.
	mutex sync.Mutex
}

func newRecommendationScopeReconciler(
	ctx context.Context, clusterID string, clientSet lmak8s.ClientSet, linseed lsclient.Client, clog *log.Entry,
) *recommendationScopeReconciler {

	return &recommendationScopeReconciler{
		clog:      clog,
		ctx:       ctx,
		clusterID: clusterID,
		clientSet: clientSet,
		linseed:   linseed,
		enabled:   v3.PolicyRecommendationScopeDisabled,
	}
}

// Reconcile will be triggered by any changes performed on the PolicyRecommendation resource.
func (r *recommendationScopeReconciler) Reconcile(key types.NamespacedName) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// TODO(dimitrin): Remove this check once https://tigera.atlassian.net/browse/EV-4647 has been
	// merged in recommendation_scope_controller.go.
	if key.Name != rectypes.PolicyRecommendationScopeName {
		r.clog.Infof("Ignoring PolicyRecommendationScope %s", key.Name)
		return nil
	}

	if scope, err := r.clientSet.ProjectcalicoV3().PolicyRecommendationScopes().Get(r.ctx, key.Name, metav1.GetOptions{}); err == nil && scope != nil {
		status := scope.Spec.NamespaceSpec.RecStatus
		if r.enabled != status {
			if status == v3.PolicyRecommendationScopeEnabled {
				if r.ctrl == nil {
					// Create the cache to store recommendations in.
					cache := r.newRecommendationResourceCache()
					// Create the recommendation engine.
					r.engine = recengine.NewRecommendationEngine(
						r.ctx,
						r.clusterID,
						r.clientSet.ProjectcalicoV3(),
						r.linseed,
						flows.NewRecommendationFlowLogQuery(r.ctx, r.linseed, r.clusterID),
						cache,
						scope,
						realClock{},
					)
					// Create the recommendation controller.
					r.ctrl, err = reccontroller.NewRecommendationController(r.ctx, r.clusterID, r.clientSet, r.engine, cache)
					if err != nil {
						r.clog.WithError(err).Error("failed to create recommendation controller")
						return nil
					}
				}

				// Start the Recommendation controller, which will start the engine.
				r.stopChan = make(chan struct{})
				go r.ctrl.Run(r.stopChan)

				r.enabled = v3.PolicyRecommendationScopeEnabled
				r.clog.Info("Recommendation engine enabled")
			} else {
				// We expect to create a new one next time we enable the engine.
				r.ctrl = nil
				// Stop the Recommendation controller, which will stop the engine.
				close(r.stopChan)

				r.enabled = v3.PolicyRecommendationScopeDisabled
				r.clog.Info("Recommendation engine disabled")
			}
		}
		if r.enabled == v3.PolicyRecommendationScopeEnabled {
			r.clog.Info("Updating PolicyRecommendation settings")
			// Update the PolicyRecommendationScope context.
			if r.engine != nil {
				r.engine.UpdateChannel <- *scope
			}
		}
	}

	return nil
}

// newRecommendationResourceCache creates a new cache to store recommendations in.
func (r *recommendationScopeReconciler) newRecommendationResourceCache() rcache.ResourceCache {
	// Define the list of items handled by the policy recommendation cache.
	listFunc := func() (map[string]interface{}, error) {
		r.clog.Debug("Listing recommendations")

		snps, err := r.clientSet.ProjectcalicoV3().StagedNetworkPolicies(v1.NamespaceAll).List(r.ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", v3.LabelTier, rectypes.PolicyRecommendationTierName),
		})
		if err != nil {
			r.clog.WithError(err).Error("unexpected error querying staged network policies")
			return nil, err
		}

		snpMap := make(map[string]interface{})
		for _, snp := range snps.Items {
			r.clog.WithField("name", snp.Name).Debug("Cache recommendation")
			snpMap[snp.Namespace] = snp
		}

		return snpMap, nil
	}

	// Create a cache to store recommendations in.
	cacheArgs := rcache.ResourceCacheArgs{
		ListFunc:    listFunc,
		ObjectType:  reflect.TypeOf(v3.StagedNetworkPolicy{}),
		LogTypeDesc: kindRecommendations,
		ReconcilerConfig: rcache.ReconcilerConfig{
			DisableUpdateOnChange: true,
			DisableMissingInCache: true,
		},
	}

	return rcache.NewResourceCache(cacheArgs)
}

// stop stops the recommendation scope reconciler.
func (r *recommendationScopeReconciler) stop() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.ctrl != nil {
		close(r.stopChan)
	}
}
