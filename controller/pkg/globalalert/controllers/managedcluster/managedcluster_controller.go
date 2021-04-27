// Copyright 2021 Tigera Inc. All rights reserved.

package managedcluster

import (
	"context"
	"net/http"

	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"

	es "github.com/tigera/intrusion-detection/controller/pkg/elastic"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	v3 "github.com/projectcalico/apiserver/pkg/apis/projectcalico/v3"
	calicoclient "github.com/projectcalico/apiserver/pkg/client/clientset_generated/clientset"

	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/controllers/controller"
	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/worker"
	"github.com/tigera/intrusion-detection/controller/pkg/health"
)

// managedClusterController is responsible for watching ManagedCluster resource.
type managedClusterController struct {
	esCLI                              *elastic.Client
	indexSettings                      es.IndexSettings
	calicoCLI                          calicoclient.Interface
	createManagedCalicoCLI             func(string) (calicoclient.Interface, error)
	cancel                             context.CancelFunc
	worker                             worker.Worker
	managedAlertControllerHealthPinger health.PingPonger
	managedAlertControllerCh           chan []health.Pinger
}

// NewManagedClusterController returns a managedClusterController and returns health.Pinger for resources it watches and also
// returns another health.Pinger that monitors health of GlobalAlertController in each of the managed cluster.
func NewManagedClusterController(calicoCLI calicoclient.Interface, esCLI *elastic.Client, indexSettings es.IndexSettings, createManagedCalicoCLI func(string) (calicoclient.Interface, error)) (controller.Controller, []health.Pinger) {
	m := &managedClusterController{
		esCLI:                              esCLI,
		indexSettings:                      indexSettings,
		calicoCLI:                          calicoCLI,
		createManagedCalicoCLI:             createManagedCalicoCLI,
		managedAlertControllerHealthPinger: health.NewPingPonger(),
		managedAlertControllerCh:           make(chan []health.Pinger),
	}

	// Create worker to watch ManagedCluster resource
	m.worker = worker.New(&managedClusterReconciler{
		createManagedCalicoCLI:          m.createManagedCalicoCLI,
		indexSettings:                   m.indexSettings,
		esCLI:                           m.esCLI,
		managementCalicoCLI:             m.calicoCLI,
		alertNameToAlertControllerState: map[string]alertControllerState{},
		managedClusterAlertControllerCh: m.managedAlertControllerCh,
	})

	pinger := m.worker.AddWatch(
		cache.NewListWatchFromClient(m.calicoCLI.ProjectcalicoV3().RESTClient(), "managedclusters", "", fields.Everything()),
		&v3.ManagedCluster{})

	return m, []health.Pinger{pinger, m.managedAlertControllerHealthPinger}
}

// Run starts a ManagedCluster monitoring routine.
func (m *managedClusterController) Run(parentCtx context.Context) {
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(parentCtx)
	log.Info("Starting managed cluster controllers")
	go m.pingAllManagedAlertController(ctx)
	go m.worker.Run(ctx.Done())
}

// Close cancels the ManagedCluster worker context and removes health check for all the objects that worker watches.
func (m *managedClusterController) Close() {
	m.worker.Close()
	m.cancel()
}

// pingAllManagedAlertController keeps track of health.Pinger of GlobalAlertController in each managed cluster,
// and calls Ping on individual health.Pinger, if all Ping succeeds it sends a pong back, else it continues without sending pong.
// If a new GlobalAlertController is created its health.Pinger is added to the list health.Pinger to Ping.
// If Ping receives error related to ping channel closed, corresponding health.Pinger is removed from the list health.Pinger to Ping.
func (m *managedClusterController) pingAllManagedAlertController(ctx context.Context) {
	pingers := []health.Pinger{}
	for {
		select {
		case pong := <-m.managedAlertControllerHealthPinger.ListenForPings():
			pingSuccess := true
			for i, hp := range pingers {
				if err := hp.Ping(ctx); err != nil {
					if statusError, isStatus := err.(*errors.StatusError); isStatus && statusError.Status().Code == http.StatusGone &&
						statusError.Status().Message == health.PingChannelClosed {
						pingers = append((pingers)[:i], (pingers)[i+1:]...)
					} else {
						pingSuccess = false
						log.WithError(err).Error("received error on health ping")
					}
				}
			}
			if pingSuccess {
				pong.Pong()
			}
		case newPingers := <-m.managedAlertControllerCh:
			pingers = append(pingers, newPingers...)
		case <-ctx.Done():
			return
		}
	}
}
