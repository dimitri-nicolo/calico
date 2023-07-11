// Copyright 2021 Tigera Inc. All rights reserved.

package managedcluster

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
)

// managedClusterController is responsible for watching ManagedCluster resource.
type managedClusterController struct {
	lsClient               client.Client
	tenantID               string
	calicoCLI              calicoclient.Interface
	createManagedCalicoCLI func(string) (calicoclient.Interface, error)
	cancel                 context.CancelFunc
	worker                 worker.Worker
}

// NewManagedClusterController returns a managedClusterController and returns health.Pinger for resources it watches and also
// returns another health.Pinger that monitors health of GlobalAlertController in each of the managed cluster.
func NewManagedClusterController(calicoCLI calicoclient.Interface, lsClient client.Client, k8sClient kubernetes.Interface, enableAnomalyDetection bool, anomalyTrainingController controller.AnomalyDetectionController, anomalyDetectionController controller.AnomalyDetectionController, namespace string, createManagedCalicoCLI func(string) (calicoclient.Interface, error), fipsModeEnabled bool, tenantID string) controller.Controller {
	m := &managedClusterController{
		lsClient:               lsClient,
		calicoCLI:              calicoCLI,
		createManagedCalicoCLI: createManagedCalicoCLI,
		tenantID:               tenantID,
	}

	// Create worker to watch ManagedCluster resource
	m.worker = worker.New(&managedClusterReconciler{
		createManagedCalicoCLI:          m.createManagedCalicoCLI,
		namespace:                       namespace,
		lsClient:                        lsClient,
		managementCalicoCLI:             m.calicoCLI,
		k8sClient:                       k8sClient,
		adTrainingController:            anomalyTrainingController,
		adDetectionController:           anomalyDetectionController,
		alertNameToAlertControllerState: map[string]alertControllerState{},
		enableAnomalyDetection:          enableAnomalyDetection,
		fipsModeEnabled:                 fipsModeEnabled,
		tenantID:                        tenantID,
	})

	m.worker.AddWatch(
		cache.NewListWatchFromClient(m.calicoCLI.ProjectcalicoV3().RESTClient(), "managedclusters", "", fields.Everything()),
		&v3.ManagedCluster{})

	log.Info("creating a new managed cluster controller")

	return m
}

// Run starts a ManagedCluster monitoring routine.
func (m *managedClusterController) Run(parentCtx context.Context) {
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(parentCtx)
	log.Info("Starting managed cluster controllers")
	go m.worker.Run(ctx.Done())
}

// Close cancels the ManagedCluster worker context and removes health check for all the objects that worker watches.
func (m *managedClusterController) Close() {
	log.Infof("closing a managed cluster controller %+v", m)
	m.worker.Close()
	m.cancel()
}
