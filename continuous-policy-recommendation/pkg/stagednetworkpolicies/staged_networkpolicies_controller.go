// Copyright (c) 2022 Tigera Inc. All rights reserved.

package stagednetworkpolicies

import (
	"context"

	"k8s.io/apimachinery/pkg/fields"
	k8scache "k8s.io/client-go/tools/cache"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"

	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/cache"
	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/constants"
	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/controller"

	log "github.com/sirupsen/logrus"
)

const KindStagedNetworkPolicies = "stagednetworkpolicies"

type StagedNetworkPolicyController struct {
	watcher controller.Watcher
	calico  calicoclient.ProjectcalicoV3Interface
	cancel  context.CancelFunc
}

func NewStagedNetworkPolicyController(calico calicoclient.ProjectcalicoV3Interface, resourceCache cache.ObjectCache[*v3.StagedNetworkPolicy]) controller.Controller {
	stagednetworkpoliciesReconciler := &stagednetworkpoliciesReconciler{
		resourceCache: resourceCache,
		calico:        calico,
	}

	watcher := controller.NewWatcher(
		stagednetworkpoliciesReconciler,
		k8scache.NewListWatchFromClient(calico.RESTClient(), KindStagedNetworkPolicies, constants.AllNamespaceKey, fields.Everything()),
		&v3.StagedNetworkPolicy{},
	)

	return &StagedNetworkPolicyController{
		watcher: watcher,
		calico:  calico,
	}
}

func (sc *StagedNetworkPolicyController) Run(parentCtx context.Context) {
	log.Info("Starting StagedNetworkPolicy Controller")

	ctx, cancel := context.WithCancel(parentCtx)
	sc.cancel = cancel

	go sc.watcher.Run(ctx.Done())
}

func (sc *StagedNetworkPolicyController) Close() {
	if sc.cancel != nil {
		sc.cancel()
	}
}
