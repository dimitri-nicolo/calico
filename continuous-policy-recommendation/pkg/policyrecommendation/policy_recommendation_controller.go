// Copyright (c) 2022 Tigera Inc. All rights reserved.

package policyrecommendation

import (
	"context"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"

	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"

	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/constants"
	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/controller"

	log "github.com/sirupsen/logrus"
)

const (
	KindPolicyRecommendationScopes = "policyrecommendationscopes"
)

type policyRecommendationController struct {
	watcher     controller.Watcher
	calico      calicoclient.ProjectcalicoV3Interface
	lmaESClient lmaelastic.Client
	cancel      context.CancelFunc
}

func NewPolicyRecommendationController(calico calicoclient.ProjectcalicoV3Interface, lmaESClient lmaelastic.Client) controller.Controller {
	prReconciler := &policyRecommendationReconciler{
		calico: calico,
	}

	watcher := controller.NewWatcher(
		prReconciler,
		cache.NewListWatchFromClient(calico.RESTClient(), KindPolicyRecommendationScopes, constants.AllNamespaceKey, fields.Everything()),
		&v3.PolicyRecommendationScope{},
	)

	return &policyRecommendationController{
		watcher:     watcher,
		calico:      calico,
		lmaESClient: lmaESClient,
	}
}

func (pr *policyRecommendationController) Run(parentCtx context.Context) {
	log.Info("Starting Policy Recommendation Controller")

	ctx, cancel := context.WithCancel(parentCtx)
	pr.cancel = cancel

	go pr.watcher.Run(ctx.Done())
}

func (pr *policyRecommendationController) Close() {
	if pr.cancel != nil {
		pr.cancel()
	}
}
