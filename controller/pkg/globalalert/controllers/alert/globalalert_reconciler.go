// Copyright 2021 Tigera Inc. All rights reserved.

package alert

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/olivere/elastic/v7"

	calicoclient "github.com/projectcalico/apiserver/pkg/client/clientset_generated/clientset"

	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/alert"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// globalAlertReconciler creates a routine for each new GlobalAlert resource that queries Elasticsearch on interval,
// processes, transforms the Elasticsearch result and updates the Elasticsearch events index and GlobalAlert status.
// If GlobalAlert resource is deleted or updated, cancel the current goroutine, and create a new one if resource is updated.
type globalAlertReconciler struct {
	esCLI                 *elastic.Client
	calicoCLI             calicoclient.Interface
	alertNameToAlertState map[string]alertState
	clusterName           string
}

// alertState has the alert and cancel function to stop the alert routine.
type alertState struct {
	alert  *alert.Alert
	cancel context.CancelFunc
}

// Reconcile gets the given GlobalAlert, if it is a new GlobalAlert resource creates a goroutine that periodically
// check Elasticsearch index data for alert condition.
// For GlobalAlert with an existing goroutine if spec is same, do nothing, else cancel the existing goroutine.
func (r *globalAlertReconciler) Reconcile(namespacedName types.NamespacedName) error {
	obj, err := r.calicoCLI.ProjectcalicoV3().GlobalAlerts().Get(context.Background(), namespacedName.Name, metav1.GetOptions{})
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

	alert, err := alert.NewAlert(obj, r.calicoCLI, r.esCLI, r.clusterName)
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
	for name, _ := range r.alertNameToAlertState {
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
