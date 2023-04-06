// Copyright (c) 2022 Tigera Inc. All rights reserved.
package networksets

import (
	"context"

	k8scache "k8s.io/client-go/tools/cache"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
	"k8s.io/apimachinery/pkg/fields"

	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/cache"
	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/constants"
	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/controller"

	log "github.com/sirupsen/logrus"
)

// NetworkSetController restores the NetworkSets created by the Policy Recommendation Engine
type NetworkSetController struct {
	watcher controller.Watcher
	calico  calicoclient.ProjectcalicoV3Interface
	cancel  context.CancelFunc
}

func NewNetworkSetController(calico calicoclient.ProjectcalicoV3Interface, resourceCache cache.ObjectCache[*v3.NetworkSet]) controller.Controller {
	networksetReconciler := &networksetReconciler{
		resourceCache: resourceCache,
		calico:        calico,
	}

	watcher := controller.NewWatcher(
		networksetReconciler,
		k8scache.NewListWatchFromClient(
			calico.RESTClient(),
			v3.KindNetworkSet,
			constants.AllNamespaceKey,
			fields.Everything(),
		),
		&v3.NetworkSet{},
	)

	return &NetworkSetController{
		watcher: watcher,
		calico:  calico,
	}
}

func (sc *NetworkSetController) Run(parentCtx context.Context) {
	log.Info("Starting NetworkSet Controller")

	ctx, cancel := context.WithCancel(parentCtx)
	sc.cancel = cancel

	go sc.watcher.Run(ctx.Done())
}

func (sc *NetworkSetController) Close() {
	if sc.cancel != nil {
		sc.cancel()
	}
}
