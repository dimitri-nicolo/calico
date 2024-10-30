// Copyright 2021-2022 Tigera Inc. All rights reserved.

package managedcluster

import (
	"context"
	"fmt"

	lsclient "github.com/projectcalico/calico/linseed/pkg/client"

	log "github.com/sirupsen/logrus"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/controller"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/globalalert/controllers/alert"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/globalalert/controllers/waf"
	lmak8s "github.com/projectcalico/calico/lma/pkg/k8s"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"
)

// managedClusterReconciler is responsible for starting and managing GlobalAlertController for every managed cluster.
// For each managed cluster it creates a worker controllers that watches for GlobalAlert resource in that managed cluster,
// and adds the new controller to health.Pingers.
// If managed cluster is updated or deleted close the corresponding GlobalAlertController this in turn cancels all the goroutines.
type managedClusterReconciler struct {
	namespace                       string
	tenantID                        string
	lsClient                        lsclient.Client
	k8sClient                       kubernetes.Interface
	managementCalicoCLI             calicoclient.Interface
	client                          ctrlclient.WithWatch
	clientSetFactory                lmak8s.ClientSetFactory
	alertNameToAlertControllerState map[string]alertControllerState
	tenantNamespace                 string
}

// alertControllerState has the Controller and cancel function to stop the Controller.
type alertControllerState struct {
	alertController controller.Controller
	clusterName     string
	tenantID        string
	cancel          context.CancelFunc
}

// Reconcile gets the given ManagedCluster, if it is a new ManagedCluster resource it creates a GlobalAlertController for that
// managed cluster if the cluster is connected and adds it to the health.PingPongers to handle health checks,
// else it cancels the existing GlobalAlertController for that ManagedCluster.
func (r *managedClusterReconciler) Reconcile(namespacedName types.NamespacedName) error {
	mc := &v3.ManagedCluster{}
	err := r.client.Get(context.Background(), types.NamespacedName{Name: namespacedName.Name, Namespace: r.tenantNamespace}, mc)
	if err != nil {
		return err
	}

	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	if _, ok := r.alertNameToAlertControllerState[namespacedName.Name]; ok {
		r.cancelAlertController(namespacedName.Name)
	}

	wafClustername := fmt.Sprintf("waf-%v", namespacedName.Name)
	if _, ok := r.alertNameToAlertControllerState[wafClustername]; ok {
		r.cancelAlertController(wafClustername)
	}

	if k8serrors.IsNotFound(err) {
		// we are done closing the goroutine
		return nil
	}

	if clusterConnected(mc) {
		if err := r.startManagedClusterAlertController(mc.Name); err != nil {
			return err
		}
		if err := r.startManagedClusterWafController(mc.Name); err != nil {
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
	// Create clientSet for application for the managed cluster indexed by the clusterID.
	clientSet, err := r.clientSetFactory.NewClientSetForApplication(name)
	if err != nil {
		log.WithError(err).Errorf("failed to create application client for cluster: %s", name)
		cancel()
		return err
	}

	clusterName := name

	// create the GlobalAlertController for the managed cluster - this controller will monitor all GlobalAlert operations
	// of the assigned managedcluster
	// This will create global alerts per managed cluster
	alertController, _ := alert.NewGlobalAlertController(clientSet, r.lsClient, r.k8sClient, clusterName, r.tenantID, r.namespace, r.tenantNamespace)

	r.alertNameToAlertControllerState[clusterName] = alertControllerState{
		alertController: alertController,
		clusterName:     clusterName,
		tenantID:        r.tenantID,
		cancel:          cancel,
	}

	go alertController.Run(ctx)
	return nil
}

// startManagedClusterWafController creates a client for the managed cluster, starts a new WafAlertController.
func (r *managedClusterReconciler) startManagedClusterWafController(name string) error {
	ctx, cancel := context.WithCancel(context.Background())

	clusterName := name

	// create the WafAlertController for the managed cluster - this controller will monitor all waf logs
	// of the assigned managedcluster
	wafAlertController := waf.NewWafAlertController(r.lsClient, clusterName, r.tenantID, r.namespace)

	wafAlertControllerName := fmt.Sprintf("waf-%s", clusterName)
	r.alertNameToAlertControllerState[wafAlertControllerName] = alertControllerState{
		alertController: wafAlertController,
		clusterName:     clusterName,
		tenantID:        r.tenantID,
		cancel:          cancel,
	}

	go wafAlertController.Run(ctx)
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
