// Copyright (c) 2024-2025 Tigera, Inc. All rights reserved.
package recommendation_scope_controller

import (
	"context"
	"errors"

	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	uruntime "k8s.io/apimachinery/pkg/util/runtime"
	k8swatch "k8s.io/apimachinery/pkg/watch"
	k8scache "k8s.io/client-go/tools/cache"

	lsclient "github.com/projectcalico/calico/linseed/pkg/client"
	lmak8s "github.com/projectcalico/calico/lma/pkg/k8s"
	"github.com/projectcalico/calico/policy-recommendation/pkg/controllers/controller"
	"github.com/projectcalico/calico/policy-recommendation/pkg/controllers/watcher"
	"github.com/projectcalico/calico/policy-recommendation/pkg/types"
	"github.com/projectcalico/calico/policy-recommendation/utils"
)

const KindPolicyRecommendationScopes = "policyrecommendationscopes"

type recommendationScopeController struct {
	// The context for the controller.
	ctx context.Context

	// The cluster ID.
	clusterID string

	// The clientSet used to access the Calico or Kubernetes API.
	clientSet lmak8s.ClientSet

	// The linseed client.
	linseed lsclient.Client

	// The enabled flag is used keep track of the engine status.
	enabled v3.PolicyRecommendationNamespaceStatus

	// The reconciler is used to reconcile the recommendation scope resource.
	reconciler *recommendationScopeReconciler

	// The watcher is used to watch for updates to the PolicyRecommendationScope resource.
	watcher watcher.Watcher

	// clog is the logger for the controller. This has been added for convenience of testing.
	clog *log.Entry
}

// NewRecommendationScopeController returns a controller which manages updates for the
// PolicyRecommendationScope resource. The resource is responsible for enabling/disabling the
// recommendation engine, and for defining the scope of the engine.
func NewRecommendationScopeController(
	ctx context.Context,
	clusterID string,
	clientSet lmak8s.ClientSet,
	linseed lsclient.Client,
	minPollInterval metav1.Duration,
) (controller.Controller, error) {
	logEntry := log.WithField("clusterID", utils.GetLogClusterID(clusterID))

	reconciler := newRecommendationScopeReconciler(ctx, clusterID, clientSet, linseed, minPollInterval, logEntry)
	if reconciler == nil {
		return nil, errors.New("failed to create recommendation scope reconciler")
	}

	return &recommendationScopeController{
		clog:       logEntry,
		ctx:        ctx,
		clientSet:  clientSet,
		clusterID:  clusterID,
		linseed:    linseed,
		enabled:    v3.PolicyRecommendationScopeDisabled,
		reconciler: reconciler,
		watcher: watcher.NewWatcher(
			reconciler,
			// The FieldSelector does not work, reported as a known issue (https://tigera.atlassian.net/browse/EV-4647).
			// The reconciler ignores all recommendation scope resources except the default one.
			&k8scache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					options.FieldSelector = fields.OneTermEqualSelector("metadata.name", types.PolicyRecommendationScopeName).String() // defaultNameFieldLabel
					return clientSet.ProjectcalicoV3().PolicyRecommendationScopes().List(context.Background(), options)
				},
				WatchFunc: func(options metav1.ListOptions) (k8swatch.Interface, error) {
					options.FieldSelector = fields.OneTermEqualSelector("metadata.name", types.PolicyRecommendationScopeName).String() // defaultNameFieldLabel
					return clientSet.ProjectcalicoV3().PolicyRecommendationScopes().Watch(context.Background(), options)
				},
			},
			&v3.PolicyRecommendationScope{},
		),
	}, nil
}

// Run starts the PolicyRecommendationScope controller. This blocks until we've been asked to stop.
func (c *recommendationScopeController) Run(stopChan chan struct{}) {
	defer uruntime.HandleCrash()

	// Start the PolicyRecommendationScope watcher
	go c.watcher.Run(stopChan)

	c.clog.Info("Started RecommendationScope controller")

	// Listen for the stop signal. Blocks until we receive a stop signal.
	<-stopChan

	c.reconciler.stop()

	c.clog.Info("Stopped RecommendationScope controller")
}
