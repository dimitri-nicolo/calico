// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

package managedcluster

import (
	"context"
	"sync"

	"github.com/projectcalico/kube-controllers/pkg/resource"

	log "github.com/sirupsen/logrus"
	tigeraapi "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	relasticsearch "github.com/projectcalico/kube-controllers/pkg/resource/elasticsearch"
)

// managementClusterChangeReconciler watches for changes in the management cluster that should trigger a recreation of
// all the elasticsearch configuration controllers for the ManagedClusters
type managementClusterChangeReconciler struct {
	sync.Mutex
	managementK8sCLI kubernetes.Interface
	calicoCLI        tigeraapi.Interface
	esK8sCLI         relasticsearch.RESTClient
	changeNotify     chan bool
	changeHash       string
}

func newManagementClusterChangeReconciler(
	managementk8sCLI kubernetes.Interface,
	calicok8sCLI tigeraapi.Interface,
	esk8sCLI relasticsearch.RESTClient,
	changeNotify chan bool,
) *managementClusterChangeReconciler {
	r := &managementClusterChangeReconciler{
		managementK8sCLI: managementk8sCLI,
		calicoCLI:        calicok8sCLI,
		esK8sCLI:         esk8sCLI,
		changeNotify:     changeNotify,
	}

	return r
}

func (c *managementClusterChangeReconciler) Reconcile(name types.NamespacedName) error {
	reqLogger := log.WithField("request", name)
	reqLogger.Info("Reconcile elasticsearch")

	// This allows us to detect if Elasticsearch, the Elasticsearch cert secret, or the Elasticsearch config map have changed
	changeHash, err := c.calculateChangeHash(reqLogger)
	if err != nil {
		return err
	}

	currentChangeHash := c.getChangeHash()
	if currentChangeHash == "" {
		c.setChangeHash(changeHash)
	} else if currentChangeHash != changeHash {
		reqLogger.Info("Elasticsearch configuration change detected")
		select {
		case c.changeNotify <- true:
		default:
			// If we're blocked from writing to the listener that means there's already an event queued and the managed
			// cluster watches will be regenerated
		}

		c.setChangeHash(changeHash)
	}

	return nil
}

func (c *managementClusterChangeReconciler) calculateChangeHash(reqLogger *log.Entry) (string, error) {
	esHash, err := c.esK8sCLI.CalculateTigeraElasticsearchHash()
	if err != nil {
		return "", err
	}

	esConfigMap, err := c.managementK8sCLI.CoreV1().ConfigMaps(resource.OperatorNamespace).Get(context.Background(), resource.ElasticsearchConfigMapName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	certSecret, err := c.managementK8sCLI.CoreV1().Secrets(resource.OperatorNamespace).Get(context.Background(), resource.ElasticsearchCertSecret, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	// This allows us to detect if Elasticsearch, the Elasticsearch cert secret, or the Elasticsearch config map have changed
	return resource.CreateHashFromObject([]interface{}{esHash, esConfigMap.Data, certSecret.Data})
}

func (c *managementClusterChangeReconciler) setChangeHash(hash string) {
	c.Lock()
	defer c.Unlock()

	c.changeHash = hash
}

func (c *managementClusterChangeReconciler) getChangeHash() string {
	c.Lock()
	defer c.Unlock()

	return c.changeHash
}
