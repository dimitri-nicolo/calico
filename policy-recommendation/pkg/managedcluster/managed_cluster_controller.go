// Copyright (c) 2022-2023 Tigera Inc. All rights reserved.

package managedcluster

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	log "github.com/sirupsen/logrus"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	linseed "github.com/projectcalico/calico/linseed/pkg/client"
	lmak8s "github.com/projectcalico/calico/lma/pkg/k8s"
	"github.com/projectcalico/calico/policy-recommendation/pkg/controller"
)

// ManagedClusterController watches for ManagedCluster and sets up the Controllers for each ManagedCluster
// attached
type managedClusterController struct {
	watcher controller.Watcher
	cancel  context.CancelFunc
}

func NewManagedClusterController(
	ctx context.Context,
	client ctrlclient.WithWatch,
	clientFactory lmak8s.ClientSetFactory,
	linseedClient linseed.Client,
	tenantNamespace string,
) controller.Controller {
	managedClusterReconciler := &managedClusterReconciler{
		client:           client,
		clientSetFactory: clientFactory,
		linseedClient:    linseedClient,
		cache:            make(map[string]*managedClusterState),
		TenantNamespace:  tenantNamespace,
	}

	listWatcher := newManagedClusterListWatcher(ctx, client, tenantNamespace)
	watcher := controller.NewWatcher(
		managedClusterReconciler,
		listWatcher,
		&v3.ManagedCluster{},
	)

	return &managedClusterController{
		watcher: watcher,
	}
}

func (m *managedClusterController) Run(parentCtx context.Context) {
	log.Info("Starting Managed Clusters Controller")

	ctx, cancel := context.WithCancel(parentCtx)
	m.cancel = cancel

	go m.watcher.Run(ctx.Done())
}

func (m *managedClusterController) Close() {
	m.cancel()
}

// newManagedClusterListWatcher returns an implementation of the ListWatch interface capable of being used to
// build an informer based on a controller-runtime client. Using the controller-runtime client allows us to build
// an Informer that works for both namespaced and cluster-scoped ManagedCluster resources regardless of whether
// it is a multi-tenant cluster or not.
func newManagedClusterListWatcher(ctx context.Context, c ctrlclient.WithWatch, namespace string) *cache.ListWatch {
	return &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			list := &v3.ManagedClusterList{}
			err := c.List(ctx, list, &ctrlclient.ListOptions{Raw: &options, Namespace: namespace})
			return list, err
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			list := &v3.ManagedClusterList{}
			return c.Watch(ctx, list, &ctrlclient.ListOptions{Raw: &options, Namespace: namespace})
		},
	}
}
