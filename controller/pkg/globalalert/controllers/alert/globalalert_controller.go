// Copyright 2021 Tigera Inc. All rights reserved.

package alert

import (
	"context"

	"github.com/tigera/intrusion-detection/controller/pkg/health"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/controllers/controller"
	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/worker"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"

	v3 "github.com/projectcalico/apiserver/pkg/apis/projectcalico/v3"
	calicoclient "github.com/projectcalico/apiserver/pkg/client/clientset_generated/clientset"
)

// globalAlertController is responsible for watching GlobalAlert resource in a cluster.
type globalAlertController struct {
	esCLI       *elastic.Client
	calicoCLI   calicoclient.Interface
	clusterName string
	cancel      context.CancelFunc
	worker      worker.Worker
}

// NewGlobalAlertController returns a globalAlertController and for each object it watches,
// a health.Pinger object is created returned for health check.
func NewGlobalAlertController(calicoCLI calicoclient.Interface, esCLI *elastic.Client, clusterName string) (controller.Controller, []health.Pinger) {
	c := &globalAlertController{
		esCLI:       esCLI,
		calicoCLI:   calicoCLI,
		clusterName: clusterName,
	}

	// Create worker to watch GlobalAlert resource in the cluster
	c.worker = worker.New(
		&globalAlertReconciler{
			esCLI:                 c.esCLI,
			calicoCLI:             c.calicoCLI,
			alertNameToAlertState: map[string]alertState{},
			clusterName:           c.clusterName,
		})

	pinger := c.worker.AddWatch(
		cache.NewListWatchFromClient(c.calicoCLI.ProjectcalicoV3().RESTClient(), "globalalerts", "", fields.Everything()),
		&v3.GlobalAlert{})

	return c, []health.Pinger{pinger}
}

// Run starts the GlobalAlert monitoring routine.
func (c *globalAlertController) Run(parentCtx context.Context) {
	var ctx context.Context
	ctx, c.cancel = context.WithCancel(parentCtx)
	log.Infof("Starting alert controller for cluster %s", c.clusterName)
	go c.worker.Run(ctx.Done())
}

// Close cancels the GlobalAlert worker context and removes health check for all the objects that worker watches.
func (c *globalAlertController) Close() {
	c.worker.Close()
	c.cancel()
}
