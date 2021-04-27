// Copyright 2021 Tigera Inc. All rights reserved.

package managedcluster

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	es "github.com/tigera/intrusion-detection/controller/pkg/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/controllers/alert"
	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/controllers/controller"
	"github.com/tigera/intrusion-detection/controller/pkg/health"
	"github.com/tigera/intrusion-detection/controller/pkg/util"

	"github.com/olivere/elastic/v7"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v3 "github.com/projectcalico/apiserver/pkg/apis/projectcalico/v3"
	libv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	calicoclient "github.com/projectcalico/apiserver/pkg/client/clientset_generated/clientset"
)

// managedClusterReconciler is responsible for starting and managing GlobalAlertController for every managed cluster.
// For each managed cluster it creates a worker controllers that watches for GlobalAlert resource in that managed cluster,
// and adds the new controller to health.Pingers.
// If managed cluster is updated or deleted close the corresponding GlobalAlertController this in turn cancels all the goroutines.
type managedClusterReconciler struct {
	esCLI                                     *elastic.Client
	indexSettings                             es.IndexSettings
	managementCalicoCLI                       calicoclient.Interface
	createManagedCalicoCLI                    func(string) (calicoclient.Interface, error)
	alertNameToAlertControllerState           map[string]alertControllerState
	managedClusterAlertControllerHealthPinger health.Pinger
	managedClusterAlertControllerCh           chan []health.Pinger
}

// alertControllerState has the Controller and cancel function to stop the Controller.
type alertControllerState struct {
	alertController controller.Controller
	cancel          context.CancelFunc
}

// Reconcile gets the given ManagedCluster, if it is a new ManagedCluster resource it creates a GlobalAlertController for that
// managed cluster if the cluster is connected and adds it to the health.PingPongers to handle health checks,
// else it cancels the existing GlobalAlertController for that ManagedCluster.
func (r *managedClusterReconciler) Reconcile(namespacedName types.NamespacedName) error {
	mc, err := r.managementCalicoCLI.ProjectcalicoV3().ManagedClusters().Get(context.Background(), namespacedName.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if _, ok := r.alertNameToAlertControllerState[namespacedName.Name]; ok {
		r.cancelAlertController(namespacedName.Name)
	}

	if errors.IsNotFound(err) {
		// we are done closing the goroutine, noting more to do for deleted managed cluster
		return nil
	}

	if clusterConnected(mc) {
		if err := r.startManagedClusterAlertController(mc.Name); err != nil {
			return err
		}
	} else {
		log.Infof("Managed cluster %s is not connected", namespacedName.Name)
	}

	return nil
}

// startManagedClusterAlertController creates a client for the managed cluster, starts a new GlobalAlertController
// and sends the health.Pinger of this new GlobalAlertController managedClusterController to run liveness probe.
func (r *managedClusterReconciler) startManagedClusterAlertController(name string) error {
	ctx, cancel := context.WithCancel(context.Background())
	managedCLI, err := r.createManagedCalicoCLI(name)
	if err != nil {
		log.WithError(err).Debug("Error creating client for managed cluster.")
		return err
	}

	ch := make(chan struct{})
	if err := es.CreateOrUpdateIndex(ctx, r.esCLI, r.indexSettings, fmt.Sprintf(es.EventIndexPattern, name), es.EventMapping, ch); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"index": fmt.Sprintf(es.EventIndexPattern, name),
		}).Error("Could not create index")
	}
	if err := util.WaitForChannel(ctx, ch, es.CreateIndexWaitTimeout); err != nil {
		return err
	}

	alertController, alertHealthPingers := alert.NewGlobalAlertController(managedCLI, r.esCLI, name)
	select {
	case r.managedClusterAlertControllerCh <- alertHealthPingers:
	default:
		return fmt.Errorf("failed to add health Pinger for GlobalAlertController in cluster %s", name)
	}

	r.alertNameToAlertControllerState[name] = alertControllerState{
		alertController: alertController,
		cancel:          cancel,
	}

	go alertController.Run(ctx)
	return nil
}

// Close cancel all the internal goroutines.
func (r *managedClusterReconciler) Close() {
	for name, _ := range r.alertNameToAlertControllerState {
		r.cancelAlertController(name)
	}
}

// cancelAlertController cancels the context of GlobalAlertController and removes it from the map
func (r *managedClusterReconciler) cancelAlertController(name string) {
	log.Debugf("Cancelling controller for cluster %s", name)
	a := r.alertNameToAlertControllerState[name]
	a.alertController.Close()
	a.cancel()
	delete(r.alertNameToAlertControllerState, name)
}

func clusterConnected(managedCluster *v3.ManagedCluster) bool {
	for _, condition := range managedCluster.Status.Conditions {
		if condition.Type == libv3.ManagedClusterStatusTypeConnected && condition.Status == libv3.ManagedClusterStatusValueTrue {
			return true
		}
	}
	return false
}
