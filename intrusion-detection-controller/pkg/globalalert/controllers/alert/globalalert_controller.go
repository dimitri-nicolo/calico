// Copyright 2021 Tigera Inc. All rights reserved.

package alert

import (
	"context"

	"github.com/projectcalico/calico/linseed/pkg/client"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/globalalert/controllers/controller"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/globalalert/worker"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/health"
)

const (
	GlobalAlertResourceName = "globalalerts"
)

// globalAlertController is responsible for watching GlobalAlert resource in a cluster.
type globalAlertController struct {
	linseedClient   client.Client
	calicoClientSet calicoclient.Interface
	kubeClientSet   kubernetes.Interface
	clusterName     string
	tenantID        string
	namespace       string
	cancel          context.CancelFunc
	worker          worker.Worker
	tenantNamespace string
}

// NewGlobalAlertController returns a globalAlertController and for each object it watches,
// a health.Pinger object is created returned for health check.
func NewGlobalAlertController(calicoClientSet calicoclient.Interface, linseedClient client.Client, kubeClientSet kubernetes.Interface, clusterName string, tenantID string, namespace string, tenantNamespace string) (controller.Controller, []health.Pinger) {
	c := &globalAlertController{
		linseedClient:   linseedClient,
		calicoClientSet: calicoClientSet,
		kubeClientSet:   kubeClientSet,
		clusterName:     clusterName,
		tenantID:        tenantID,
		namespace:       namespace,
		tenantNamespace: tenantNamespace,
	}

	// Create worker to watch GlobalAlert resource in the cluster
	c.worker = worker.New(
		&globalAlertReconciler{
			linseedClient:         c.linseedClient,
			calicoClientSet:       c.calicoClientSet,
			kubeClientSet:         kubeClientSet,
			alertNameToAlertState: map[string]alertState{},
			clusterName:           c.clusterName,
			tenantID:              c.tenantID,
			namespace:             namespace,
		})

	pinger := c.worker.AddWatch(
		cache.NewListWatchFromClient(c.calicoClientSet.ProjectcalicoV3().RESTClient(), GlobalAlertResourceName, "", fields.Everything()),
		&v3.GlobalAlert{})

	return c, []health.Pinger{pinger}
}

// Run starts the GlobalAlert monitoring routine.
func (c *globalAlertController) Run(parentCtx context.Context) {
	var ctx context.Context
	ctx, c.cancel = context.WithCancel(parentCtx)
	log.Infof("[Global Alert] Starting alert controller for cluster %s", c.clusterName)
	go c.worker.Run(ctx.Done())
}

// Close cancels the GlobalAlert worker context and removes health check for all the objects that worker watches.
func (c *globalAlertController) Close() {
	c.worker.Close()
	// check if the cancel function has been called by another goroutine
	if c.cancel != nil {
		c.cancel()
	}
}
