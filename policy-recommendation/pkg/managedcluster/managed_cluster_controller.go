// Copyright (c) 2022-2023 Tigera Inc. All rights reserved.

package managedcluster

import (
	"context"

	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"

	linseed "github.com/projectcalico/calico/linseed/pkg/client"
	lmak8s "github.com/projectcalico/calico/lma/pkg/k8s"
	"github.com/projectcalico/calico/policy-recommendation/pkg/constants"
	"github.com/projectcalico/calico/policy-recommendation/pkg/controller"
)

const (
	KindManagedClusters = "managedclusters"
)

// ManagedClusterController watches for ManagedCluster and sets up the Controllers for each ManagedCluster
// attached
type managedClusterController struct {
	watcher                    controller.Watcher
	managementStandaloneCalico calicoclient.ProjectcalicoV3Interface
	cancel                     context.CancelFunc
}

func NewManagedClusterController(
	managementStandaloneCalico calicoclient.ProjectcalicoV3Interface,
	clientFactory lmak8s.ClientSetFactory,
	linceed linseed.Client,
) controller.Controller {
	managedClusterReconciler := &managedClusterReconciler{
		managementStandaloneCalico: managementStandaloneCalico,
		clientFactory:              clientFactory,
		linseed:                    linceed,
		cache:                      make(map[string]*managedClusterState),
	}

	watcher := controller.NewWatcher(
		managedClusterReconciler,
		cache.NewListWatchFromClient(managementStandaloneCalico.RESTClient(), KindManagedClusters, constants.AllNamespaceKey, fields.Everything()),
		&v3.ManagedCluster{},
	)

	return &managedClusterController{
		watcher:                    watcher,
		managementStandaloneCalico: managementStandaloneCalico,
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
