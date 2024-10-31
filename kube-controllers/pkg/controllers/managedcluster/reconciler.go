// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package managedcluster

import (
	"context"
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	tigeraapi "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type reconciler struct {
	sync.Mutex
	createManagedK8sCLI func(string) (kubernetes.Interface, *tigeraapi.Clientset, error)
	kubeClientSet       kubernetes.Interface
	clientSetFactory    tigeraapi.Interface

	client ctrlclient.WithWatch

	// The only information we need for a ManagedCluster is the channel to stop it. The exists of this channel can tell
	// us if we have a controller for a ManagedCluster and the only action we would want to take on one is to stop it.
	managedClustersStopChans map[string]chan struct{}
	restartChan              chan<- string

	controllers     []ControllerManager
	TenantNamespace string
}

// Reconcile finds the ManagedCluster resource specified by the name and either passes the information to the underlying
// ControllerManagers. If the ManagedCluster doesn't exist or is no longer connected then the ControllerManagers are
// notified by calling HandleManagedClusterRemoved. If this a newly added ManagedCluster then the CreateController is
// called on every controller manager and the return controller is run for the new managed cluster.
func (c *reconciler) Reconcile(name types.NamespacedName) error {
	reqLogger := log.WithField("request", name)
	reqLogger.Info("Reconciling ManagedClusters")

	mc := &v3.ManagedCluster{}
	err := c.client.Get(context.Background(), types.NamespacedName{Name: name.Name, Namespace: c.TenantNamespace}, mc)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("ManagedCluster not found")
			for _, controller := range c.controllers {
				controller.HandleManagedClusterRemoved(name.Name)
			}
			return nil
		}
		return err
	}
	log.WithField("mc", mc).Info("cluster")
	if !clusterConnected(mc) {
		reqLogger.Info("Attempting to stop watch on disconnected cluster")
		c.removeManagedClusterWatch(mc)
		return nil
	}

	reqLogger.Info("Attempting to start watch on connected cluster")
	if err := c.startManagedClusterWatch(mc); err != nil {
		return err
	}

	return nil
}

func clusterConnected(managedCluster *v3.ManagedCluster) bool {
	for _, condition := range managedCluster.Status.Conditions {
		if condition.Type == v3.ManagedClusterStatusTypeConnected && condition.Status == v3.ManagedClusterStatusValueTrue {
			return true
		}
	}
	return false
}

func (c *reconciler) startManagedClusterWatch(mc *v3.ManagedCluster) error {
	managedK8sCLI, managedCalicoCLI, err := c.createManagedK8sCLI(mc.Name)
	if err != nil {
		return err
	}

	c.removeManagedClusterWatch(mc)
	c.addManagedClusterWatch(mc, managedK8sCLI, managedCalicoCLI)

	return nil
}

func (c *reconciler) removeManagedClusterWatch(mc *v3.ManagedCluster) {
	c.Lock()
	defer c.Unlock()

	log.Infof("Removing cluster watch for %s", mc.Name)
	if st, exists := c.managedClustersStopChans[mc.Name]; exists {
		close(st)
		delete(c.managedClustersStopChans, mc.Name)
	}
}

func (c *reconciler) addManagedClusterWatch(mc *v3.ManagedCluster, managedK8sCLI kubernetes.Interface, managedCalicoCLI *tigeraapi.Clientset) {
	c.Lock()
	defer c.Unlock()

	log.Infof("Adding cluster watch for %s", mc.Name)
	// If this happens it's a programming error, setManagerClusterWatch should never be called if the managed cluster
	// already has an entry
	if _, exists := c.managedClustersStopChans[mc.Name]; exists {
		panic(fmt.Sprintf("a watch for managed cluster %s already exists", mc.Name))
	}

	stop := make(chan struct{})
	for _, controller := range c.controllers {
		controller := controller.CreateController(mc.Name, string(mc.UID), managedK8sCLI, c.kubeClientSet, managedCalicoCLI, c.clientSetFactory, c.restartChan)
		go controller.Run(stop)
	}

	c.managedClustersStopChans[mc.Name] = stop
}
