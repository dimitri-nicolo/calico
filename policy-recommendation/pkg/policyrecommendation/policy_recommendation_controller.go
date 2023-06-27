// Copyright (c) 2022-2023 Tigera Inc. All rights reserved.

package policyrecommendation

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"

	linseed "github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/policy-recommendation/pkg/constants"
	"github.com/projectcalico/calico/policy-recommendation/pkg/controller"
	"github.com/projectcalico/calico/policy-recommendation/pkg/syncer"
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
	watcher controller.Watcher
	calico  calicoclient.ProjectcalicoV3Interface
	cancel  context.CancelFunc
	clog    log.Entry
}

func NewPolicyRecommendationController(
	calico calicoclient.ProjectcalicoV3Interface,
	linseedClient linseed.Client,
	synchronizer client.QueryInterface,
	caches *syncer.CacheSet,
	cluster string,
	serviceNameSuffix string,
) controller.Controller {
	prReconciler := &policyRecommendationReconciler{
		calico:            calico,
		linseedClient:     linseedClient,
		synchronizer:      synchronizer,
		caches:            caches,
		cluster:           cluster,
		serviceNameSuffix: serviceNameSuffix,
		tickDuration:      make(chan time.Duration),
		clock:             &realClock{},
	}

	watcher := controller.NewWatcher(
		prReconciler,
		cache.NewListWatchFromClient(calico.RESTClient(), KindPolicyRecommendationScopes,
			constants.AllNamespaceKey, fields.Everything()),
		&v3.PolicyRecommendationScope{},
	)

	return &policyRecommendationController{
		watcher: watcher,
		calico:  calico,
		clog:    *log.WithField("cluster", cluster),
	}
}

func (pr *policyRecommendationController) Run(parentCtx context.Context) {
	pr.clog.Info("Starting Policy Recommendation Controller")

	ctx, cancel := context.WithCancel(parentCtx)
	pr.cancel = cancel

	go pr.watcher.Run(ctx.Done())
}

func (pr *policyRecommendationController) Close() {
	if pr.cancel != nil {
		pr.cancel()
	}
}
