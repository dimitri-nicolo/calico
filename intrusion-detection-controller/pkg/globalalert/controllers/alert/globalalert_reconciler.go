// Copyright 2021 Tigera Inc. All rights reserved.

package alert

import (
	"context"

	log "github.com/sirupsen/logrus"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/globalalert/alert"
	"github.com/projectcalico/calico/linseed/pkg/client"
)

// globalAlertReconciler creates a routine for each new GlobalAlert resource that queries Linseed on interval,
// processes, transforms the result and creates an event via Linseed and updates GlobalAlert status.
// If GlobalAlert resource is deleted or updated, cancel the current goroutine, and create a new one if resource is updated.
type globalAlertReconciler struct {
	linseedClient         client.Client
	kubeClientSet         kubernetes.Interface
	calicoClientSet       calicoclient.Interface
	alertNameToAlertState map[string]alertState
	clusterName           string
	tenantID              string
	namespace             string
}

// alertState has the alert and cancel function to stop the alert routine.
type alertState struct {
	alert  *alert.Alert
	cancel context.CancelFunc
}

// Reconcile gets the given GlobalAlert, if it is a new GlobalAlert resource creates a goroutine that periodically
// check Linseed data for alert condition.
// For GlobalAlert with an existing goroutine if spec is same, do nothing, else cancel the existing goroutine and
// recreate it with new specs from alert.
func (r *globalAlertReconciler) Reconcile(namespacedName types.NamespacedName) error {
	obj, err := r.calicoClientSet.ProjectcalicoV3().GlobalAlerts().Get(context.Background(),
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

	alert, err := alert.NewAlert(obj, r.calicoClientSet, r.linseedClient, r.kubeClientSet, r.clusterName, r.tenantID, r.namespace)
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
	log.WithFields(log.Fields{"tenant": r.tenantID, "cluster": r.clusterName, "alert": name}).Debug("Cancelling routine for alert")
	a := r.alertNameToAlertState[name]
	a.cancel()
	delete(r.alertNameToAlertState, name)
}
