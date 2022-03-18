// Copyright 2021-2022 Tigera Inc. All rights reserved.

package managedcluster

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	es "github.com/tigera/intrusion-detection/controller/pkg/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/controllers/alert"

	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/controllers/controller"
	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/podtemplate"
	"github.com/tigera/intrusion-detection/controller/pkg/health"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"

	lma "github.com/tigera/lma/pkg/elastic"
)

// managedClusterReconciler is responsible for starting and managing GlobalAlertController for every managed cluster.
// For each managed cluster it creates a worker controllers that watches for GlobalAlert resource in that managed cluster,
// and adds the new controller to health.Pingers.
// If managed cluster is updated or deleted close the corresponding GlobalAlertController this in turn cancels all the goroutines.
type managedClusterReconciler struct {
	namespace                       string
	lmaESClient                     lma.Client
	k8sClient                       kubernetes.Interface
	indexSettings                   es.IndexSettings
	managementCalicoCLI             calicoclient.Interface
	podTemplateQuery                podtemplate.ADPodTemplateQuery
	createManagedCalicoCLI          func(string) (calicoclient.Interface, error)
	alertNameToAlertControllerState map[string]alertControllerState

	adDetectionController controller.AnomalyDetectionController
	adTrainingController  controller.AnomalyDetectionController

	managedClusterAlertControllerHealthPinger health.Pinger
	managedClusterAlertControllerCh           chan []health.Pinger
}

// alertControllerState has the Controller and cancel function to stop the Controller.
type alertControllerState struct {
	alertController controller.Controller
	clusterName     string
	cancel          context.CancelFunc
}

// Reconcile gets the given ManagedCluster, if it is a new ManagedCluster resource it creates a GlobalAlertController for that
// managed cluster if the cluster is connected and adds it to the health.PingPongers to handle health checks,
// else it cancels the existing GlobalAlertController for that ManagedCluster.
func (r *managedClusterReconciler) Reconcile(namespacedName types.NamespacedName) error {
	mc, err := r.managementCalicoCLI.ProjectcalicoV3().ManagedClusters().Get(context.Background(), namespacedName.Name, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	if _, ok := r.alertNameToAlertControllerState[namespacedName.Name]; ok {
		r.cancelAlertController(namespacedName.Name)
	}

	if k8serrors.IsNotFound(err) {
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
		cancel()
		return err
	}

	clusterName := getVariantSpecificClusterName(name)

	// Create a managedCluster specific lma client
	envCfg := lma.MustLoadConfig()
	envCfg.ElasticIndexSuffix = clusterName
	lmaESClient, err := lma.NewFromConfig(envCfg)
	if err != nil {
		log.WithError(err).Errorf("failed to create Elastic client for managed cluster %s", clusterName)
		cancel()
		return err
	}
	if err := lmaESClient.CreateEventsIndex(ctx); err != nil {
		log.WithError(err).Errorf("failed to create events index for managed cluster %s", clusterName)
		cancel()
		return err
	}

	// setup training AD cronjobs to run on the management cluster for the managed cluster
	// also adds the cronjob to the AD training controller that will reconcile / maange it
	err = r.adTrainingController.AddDetector(clusterName)
	if err != nil {
		log.WithError(err).Debug("Error creating training cronjob for managed cluster %s.", clusterName)
		return err
	}

	// create the GlobalAlertController for the managed cluster - this controller will monitor all GlobalAlert operations
	// of the assigned managedcluster
	alertController, alertHealthPingers := alert.NewGlobalAlertController(managedCLI, lmaESClient, r.k8sClient,
		r.podTemplateQuery, r.adDetectionController, r.adTrainingController, clusterName, r.namespace)

	successSendingPinger := false
	for maxRetries := 5; maxRetries > 0; maxRetries-- {
		select {
		case r.managedClusterAlertControllerCh <- alertHealthPingers:
			successSendingPinger = true
		default:
			log.Infof("Failed to add health Pinger for GlobalAlertController in cluster %s, retries left %d", clusterName, maxRetries)
			time.Sleep(5 * time.Second)
		}

		if successSendingPinger {
			break
		}
	}

	if !successSendingPinger {
		log.Errorf("failed to add health Pinger for GlobalAlertController in cluster %s, after retries", clusterName)
		cancel()
		return fmt.Errorf("failed to add health Pinger for GlobalAlertController in cluster %s, after retries", clusterName)
	}

	r.alertNameToAlertControllerState[clusterName] = alertControllerState{
		alertController: alertController,
		clusterName:     clusterName,
		cancel:          cancel,
	}

	go alertController.Run(ctx)
	return nil
}

// Close cancel all the internal goroutines.
func (r *managedClusterReconciler) Close() {
	for name := range r.alertNameToAlertControllerState {
		r.cancelAlertController(name)
	}
}

// cancelAlertController cancels the context of GlobalAlertController and removes it from the map
func (r *managedClusterReconciler) cancelAlertController(name string) {
	log.Debugf("Cancelling controller for cluster %s", name)
	a := r.alertNameToAlertControllerState[name]

	r.adTrainingController.RemoveDetector(a.clusterName)
	a.alertController.Close()
	a.cancel()
	delete(r.alertNameToAlertControllerState, name)
}

func clusterConnected(managedCluster *v3.ManagedCluster) bool {
	for _, condition := range managedCluster.Status.Conditions {
		if condition.Type == v3.ManagedClusterStatusTypeConnected && condition.Status == v3.ManagedClusterStatusValueTrue {
			return true
		}
	}
	return false
}
