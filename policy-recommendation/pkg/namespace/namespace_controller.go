// Copyright (c) 2022 Tigera Inc. All rights reserved.

package namespace

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	k8scache "k8s.io/client-go/tools/cache"

	"github.com/projectcalico/calico/policy-recommendation/pkg/cache"
	"github.com/projectcalico/calico/policy-recommendation/pkg/constants"
	"github.com/projectcalico/calico/policy-recommendation/pkg/controller"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/client"

	log "github.com/sirupsen/logrus"
)

const KindNamespace = "namespaces"

// NamespaceController watches for namespace changes and syncs all corresponding resource caches.
type namespaceController struct {
	watcher controller.Watcher
	cancel  context.CancelFunc
}

func NewNamespaceController(kubernetes kubernetes.Interface,
	resourceCache cache.ObjectCache[*v1.Namespace],
	synchronizer client.QueryInterface,
) controller.Controller {
	namespaceReconciler := NewNamespaceReconciler(kubernetes, resourceCache, synchronizer)

	watcher := controller.NewWatcher(
		namespaceReconciler,
		k8scache.NewListWatchFromClient(
			kubernetes.CoreV1().RESTClient(),
			KindNamespace,
			constants.AllNamespaceKey,
			fields.Everything(),
		),
		&v1.Namespace{},
	)

	return &namespaceController{
		watcher: watcher,
	}
}

func (n *namespaceController) Run(parentCtx context.Context) {
	log.Info("Starting Namespace Controller")

	ctx, cancel := context.WithCancel(parentCtx)
	n.cancel = cancel

	go n.watcher.Run(ctx.Done())
}

func (n *namespaceController) Close() {
	if n.cancel != nil {
		n.cancel()
	}
}
