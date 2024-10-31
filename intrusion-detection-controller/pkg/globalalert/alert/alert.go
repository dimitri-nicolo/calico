// Copyright 2021 Tigera Inc. All rights reserved.

package alert

import (
	"context"
	"errors"
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/globalalert/query"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/globalalert/reporting"
	"github.com/projectcalico/calico/linseed/pkg/client"
)

const (
	DefaultPeriod      = 5 * time.Minute
	MinimumAlertPeriod = 5 * time.Second

	GlobalAlertSpecTypeFieldName = "Type"
)

type Alert struct {
	alert       *v3.GlobalAlert
	calicoCLI   calicoclient.Interface
	service     query.Service
	clusterName string
	tenantID    string
}

// NewAlert sets and returns an Alert, builds Linseed query that will be used periodically to query Elasticsearch data.
func NewAlert(globalAlert *v3.GlobalAlert, calicoCLI calicoclient.Interface, linseedClient client.Client, k8sClient kubernetes.Interface, clusterName string, tenantID string, namespace string) (*Alert, error) {
	globalAlert.Status.Active = true
	globalAlert.Status.LastUpdate = &metav1.Time{Time: time.Now()}

	// extract by Reflect to handle GlobalAlert on managed clusters with CE version before Type field exists
	globalAlertSpec := reflect.ValueOf(&globalAlert.Spec).Elem()

	globalAlertType, ok := globalAlertSpec.FieldByName(GlobalAlertSpecTypeFieldName).Interface().(v3.GlobalAlertType)

	alert := &Alert{
		alert:       globalAlert,
		calicoCLI:   calicoCLI,
		clusterName: clusterName,
		tenantID:    tenantID,
	}

	if !ok || globalAlertType != v3.GlobalAlertTypeAnomalyDetection {
		service, err := query.NewService(linseedClient, clusterName, globalAlert)
		if err != nil {
			return nil, err
		}

		alert.service = service

	} else {
		return nil, errors.New("GlobalAlert for Anomaly Detection is no longer supported")
	}

	return alert, nil
}

func (a *Alert) Execute(ctx context.Context) {
	log.Debugf("Handling of type: %s.", a.alert.Spec.Type)

	// extract by Reflect to handle GlobalAlert on managed clusters with CE version before Type field exists
	globalAlertSpec := reflect.ValueOf(&a.alert.Spec).Elem()

	globalAlertType, ok := globalAlertSpec.FieldByName(GlobalAlertSpecTypeFieldName).Interface().(v3.GlobalAlertType)

	if !ok || globalAlertType != v3.GlobalAlertTypeAnomalyDetection {
		a.ExecuteQuery(ctx)
	}
}

// ExecuteQuery periodically queries Linseed, updates GlobalAlert status
// and adds alerts to events index if alert conditions are met.
// If parent context is cancelled, updates the GlobalAlert status and returns.
func (a *Alert) ExecuteQuery(ctx context.Context) {
	if err := reporting.UpdateGlobalAlertStatusWithRetryOnConflict(a.alert, a.clusterName, a.calicoCLI, ctx); err != nil {
		log.WithError(err).Warnf(`failed to update globalalert "%s" status when executing linseed query`, a.alert.Name)
	}

	for {
		timer := time.NewTimer(a.getDurationUntilNextAlert())
		select {
		case <-timer.C:
			a.alert.Status = a.service.ExecuteAlert(ctx, a.alert)
			if err := reporting.UpdateGlobalAlertStatusWithRetryOnConflict(a.alert, a.clusterName, a.calicoCLI, ctx); err != nil {
				log.WithError(err).Warnf(`failed to update globalalert "%s" status when executing linseed query`, a.alert.Name)
			}
			timer.Stop()
		case <-ctx.Done():
			a.alert.Status.Active = false
			if err := reporting.UpdateGlobalAlertStatusWithRetryOnConflict(a.alert, a.clusterName, a.calicoCLI, ctx); err != nil {
				log.WithError(err).Warnf(`failed to update globalalert "%s" status when executing linseed query`, a.alert.Name)
			}
			timer.Stop()
			return
		}
	}
}

// getDurationUntilNextAlert returns the duration after which next alert should run.
// If Status.LastExecuted is set on alert use that to calculate time for next alert execution,
// else uses Spec.Period value.
func (a *Alert) getDurationUntilNextAlert() time.Duration {
	alertPeriod := DefaultPeriod
	if a.alert.Spec.Period != nil {
		alertPeriod = a.alert.Spec.Period.Duration
	}

	if a.alert.Status.LastExecuted != nil {
		now := time.Now()
		durationSinceLastExecution := now.Sub(a.alert.Status.LastExecuted.Local())
		if durationSinceLastExecution < 0 {
			log.Errorf("last executed alert is in the future")
			return MinimumAlertPeriod
		}
		timeUntilNextRun := alertPeriod - durationSinceLastExecution
		if timeUntilNextRun <= 0 {
			// return MinimumAlertPeriod instead of 0s to guarantee that we would never have a tight loop
			// that burns through our pod resources and spams Linseed.
			return MinimumAlertPeriod
		}
		return timeUntilNextRun
	}
	return alertPeriod
}

// EqualAlertSpec does reflect.DeepEqual on give spec and cached alert spec.
func (a *Alert) EqualAlertSpec(spec v3.GlobalAlertSpec) bool {
	return reflect.DeepEqual(a.alert.Spec, spec)
}
