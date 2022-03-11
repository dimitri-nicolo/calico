// Copyright 2021 Tigera Inc. All rights reserved.

package managedcluster

import (
	"context"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	es "github.com/tigera/intrusion-detection/controller/pkg/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/controllers/controller"
	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/worker"
	"github.com/tigera/intrusion-detection/controller/pkg/health"
	lma "github.com/tigera/lma/pkg/elastic"
)

// managedClusterController is responsible for watching ManagedCluster resource.
type managedClusterController struct {
	lmaESClient                        lma.Client
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
func NewManagedClusterController(calicoCLI calicoclient.Interface, lmaESClient lma.Client, k8sClient kubernetes.Interface,
	anomalyTrainingController controller.ADJobController, anomalyDetectionController controller.ADJobController,
	indexSettings es.IndexSettings, namespace string, createManagedCalicoCLI func(string) (calicoclient.Interface, error)) (controller.Controller, []health.Pinger) {
	m := &managedClusterController{
		lmaESClient:                        lmaESClient,
		indexSettings:                      indexSettings,
		calicoCLI:                          calicoCLI,
		createManagedCalicoCLI:             createManagedCalicoCLI,
		managedAlertControllerHealthPinger: health.NewPingPonger(),
		managedAlertControllerCh:           make(chan []health.Pinger, 100),
	}

	// Create worker to watch ManagedCluster resource
	m.worker = worker.New(&managedClusterReconciler{
		createManagedCalicoCLI:          m.createManagedCalicoCLI,
		namespace:                       namespace,
		indexSettings:                   m.indexSettings,
		lmaESClient:                     m.lmaESClient,
		managementCalicoCLI:             m.calicoCLI,
		k8sClient:                       k8sClient,
		anomalyTrainingController:       anomalyDetectionController,
		anomalyDetectionController:      anomalyDetectionController,
		alertNameToAlertControllerState: map[string]alertControllerState{},
		managedClusterAlertControllerCh: m.managedAlertControllerCh,
	})

	pinger := m.worker.AddWatch(
		cache.NewListWatchFromClient(m.calicoCLI.ProjectcalicoV3().RESTClient(), "managedclusters", "", fields.Everything()),
		&v3.ManagedCluster{})

	log.Info("creating a new managed cluster controller")

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
	log.Infof("closing a managed cluster controller %+v", m)
	m.worker.Close()
	m.cancel()
}

// pingAllManagedAlertController keeps track of health.Pinger of GlobalAlertController in each managed cluster,
// and calls Ping on individual health.Pinger, if all Ping succeeds it sends a pong back, else it continues without sending pong.
// If a new GlobalAlertController is created its health.Pinger is added to the list health.Pinger to Ping.
// If Ping receives error related to ping channel closed, corresponding health.Pinger is removed from the list health.Pinger to Ping.
func (m *managedClusterController) pingAllManagedAlertController(ctx context.Context) {
	pingers := []health.Pinger{}
	pingerMap := make(map[health.Pinger]struct{})
	for {
		select {
		case pong := <-m.managedAlertControllerHealthPinger.ListenForPings():
			pingSuccess := true
			// If pinger is removed from slice within a loop, the loop doesn't know about the underlying changes to slice.
			// Loop through all the pingers in reverse order, so if a pinger is removed all subsequent pingers
			// that are shifted are part of already processed slice.
			for i := len(pingers) - 1; i >= 0; i-- {
				hp := pingers[i]
				// IDS uses default timeoutSeconds(1s) for livenessProbe. Use the same timeout on individual Pings.
				chCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
				defer cancel()
				if err := hp.Ping(chCtx); err != nil {
					log.Warnf("error in pinging %+v", err)
					statusError, ok := err.(*errors.StatusError)
					if ok && statusError.Status().Message == health.PingChannelClosed {
						pingers = append(pingers[:i], pingers[i+1:]...)
						delete(pingerMap, hp)
					} else if ok && strings.Contains(statusError.Status().Message, health.PingChannelBusy) {
						if err := retryPingOnBusy(ctx, hp); err != nil {
							pingSuccess = false
							log.WithError(err).Error("received error on health ping")
						}
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
			for _, pinger := range newPingers {
				if _, ok := pingerMap[pinger]; !ok {
					log.Infof("adding new pinger for managed alert controller %+v", pinger)
					pingerMap[pinger] = struct{}{}
					pingers = append(pingers, pinger)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func retryPingOnBusy(ctx context.Context, hp health.Pinger) error {
	var err error
	for maxRetries := 5; maxRetries > 0; maxRetries-- {
		chCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()
		if err = hp.Ping(chCtx); err == nil {
			break
		}
		if statusError, ok := err.(*errors.StatusError); !ok || !strings.Contains(statusError.Status().Message, health.PingChannelBusy) {
			break
		}
		time.Sleep(5 * time.Second)
	}
	return err
}
