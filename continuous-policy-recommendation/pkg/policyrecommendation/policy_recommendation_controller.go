// Copyright (c) 2022 Tigera Inc. All rights reserved.

package policyrecommendation

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"

	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/constants"
	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/controller"
	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/syncer"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/client"
)

const (
	KindPolicyRecommendationScopes = "policyrecommendationscopes"
)

type realClock struct{}

// NowRFC3339 returns time Now() as a string, in RFC3339 format.
func (realClock) NowRFC3339() string {
	// TODO(dimitrin): Replace the function for getting the time Now(), to account for server side
	// timezone differences. If we use this a s base line to retieve logs from ES, we may not get
	// expected results

	// The rule is new and should be timestamped.
	return time.Now().UTC().Format(time.RFC3339)
}

type policyRecommendationController struct {
	watcher     controller.Watcher
	calico      calicoclient.ProjectcalicoV3Interface
	lmaESClient *lmaelastic.Client
	cancel      context.CancelFunc
}

func NewPolicyRecommendationController(
	calico calicoclient.ProjectcalicoV3Interface,
	lmaESClient *lmaelastic.Client, synchronizer client.QueryInterface,
	caches *syncer.CacheSet,
	cluster string,
) controller.Controller {
	prReconciler := &policyRecommendationReconciler{
		calico:       calico,
		lmaESClient:  lmaESClient,
		synchronizer: synchronizer,
		caches:       caches,
		cluster:      cluster,
		tickDuration: make(chan time.Duration),
		clock:        &realClock{},
	}

	watcher := controller.NewWatcher(
		prReconciler,
		cache.NewListWatchFromClient(calico.RESTClient(), KindPolicyRecommendationScopes,
			constants.AllNamespaceKey, fields.Everything()),
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
