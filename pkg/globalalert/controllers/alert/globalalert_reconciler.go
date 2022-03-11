// Copyright 2021 Tigera Inc. All rights reserved.

package alert

import (
	"context"

	log "github.com/sirupsen/logrus"

	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"

	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/alert"
	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/controllers/controller"
	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/podtemplate"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	lma "github.com/tigera/lma/pkg/elastic"
)

// globalAlertReconciler creates a routine for each new GlobalAlert resource that queries Elasticsearch on interval,
// processes, transforms the Elasticsearch result and updates the Elasticsearch events index and GlobalAlert status.
// If GlobalAlert resource is deleted or updated, cancel the current goroutine, and create a new one if resource is updated.
type globalAlertReconciler struct {
	lmaESClient                lma.Client
	k8sClient                  kubernetes.Interface
	calicoCLI                  calicoclient.Interface
	podTemplateQuery           podtemplate.ADPodTemplateQuery
	anomalyDetectionController controller.ADJobController
	alertNameToAlertState      map[string]alertState
	clusterName                string
	namespace                  string
}

// alertState has the alert and cancel function to stop the alert routine.
type alertState struct {
	alert  *alert.Alert
	cancel context.CancelFunc
}

// Reconcile gets the given GlobalAlert, if it is a new GlobalAlert resource creates a goroutine that periodically
// check Elasticsearch index data for alert condition.
// For GlobalAlert with an existing goroutine if spec is same, do nothing, else cancel the existing goroutine and
// recreate it with new specs from alert.
func (r *globalAlertReconciler) Reconcile(namespacedName types.NamespacedName) error {
	obj, err := r.calicoCLI.ProjectcalicoV3().GlobalAlerts().Get(context.Background(),
		namespacedName.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	if a, ok := r.alertNameToAlertState[namespacedName.Name]; ok {
		if a.alert.EqualAlertSpec(obj.Spec) {
			log.Debug("Spec unchanged.")
			return nil
		}
		r.cancelAlertRoutine(namespacedName.Name)
	}

	if errors.IsNotFound(err) {
		// GlobalAlert doesn't exist, we are done closing the goroutine, return.
		return nil
	}

	alert, err := alert.NewAlert(obj, r.calicoCLI, r.lmaESClient, r.k8sClient, r.podTemplateQuery, r.anomalyDetectionController,
		r.clusterName, r.namespace)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.alertNameToAlertState[namespacedName.Name] = alertState{
		alert:  alert,
		cancel: cancel,
	}

	go alert.Execute(ctx)
	return nil
}

// Close cancel all the internal goroutines.
func (r *globalAlertReconciler) Close() {
	for name := range r.alertNameToAlertState {
		r.cancelAlertRoutine(name)
	}
}

// cancelAlertRoutine cancels the context of alert goroutine and removes it from the map
func (r *globalAlertReconciler) cancelAlertRoutine(name string) {
	log.Debugf("Cancelling routine for alert %s in cluster %s", name, r.clusterName)
	a := r.alertNameToAlertState[name]
	a.cancel()
	delete(r.alertNameToAlertState, name)
}
