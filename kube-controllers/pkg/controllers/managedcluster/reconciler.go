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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

type reconciler struct {
	sync.Mutex
	createManagedK8sCLI func(string) (kubernetes.Interface, *tigeraapi.Clientset, error)
	managementK8sCLI    kubernetes.Interface
	calicoCLI           tigeraapi.Interface
	// the only information we need about an elasticsearch cluster controller for a ManagedCluster is the channel to stop
	// it. The exists of this channel can tell us if we have a controller for a ManagedCluster and the only action we would
	// want to take on one is to stop it
	managedClustersStopChans map[string]chan struct{}
	restartChan              chan<- string

	controllers []Controller
}

// Reconcile finds the ManagedCluster resource specified by the name and either adds, removes, or recreates the elasticsearch
// configuration controller for that managed cluster. If the ManagedCluster that's being reconciled exists is connected
// then the elasticsearch configuration controller for that managed cluster is added or recreated. If the ManagedCluster
// doesn't exist or is no longer connected then the Elasticsearch configuration controller is stopped for that ManagedCluster,
// if there is one running. In addition to reconciling Elasticsearch configuration changes, the controller will also reconcile
// license changes in the managed cluster
func (c *reconciler) Reconcile(name types.NamespacedName) error {
	reqLogger := log.WithField("request", name)
	reqLogger.Info("Reconciling ManagedClusters")

	mc, err := c.calicoCLI.ProjectcalicoV3().ManagedClusters().Get(context.Background(), name.Name, metav1.GetOptions{})
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
		controller := controller.New(mc.Name, string(mc.UID), managedK8sCLI, c.managementK8sCLI, managedCalicoCLI, c.calicoCLI, false, c.restartChan)
		go controller.Run(stop)
	}

	c.managedClustersStopChans[mc.Name] = stop
}
