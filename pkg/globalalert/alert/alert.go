// Copyright 2021 Tigera Inc. All rights reserved.

package alert

import (
	"context"
	"reflect"
	"time"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"

	ad "github.com/tigera/intrusion-detection/controller/pkg/globalalert/anomalydetection"
	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/controllers/controller"
	es "github.com/tigera/intrusion-detection/controller/pkg/globalalert/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/podtemplate"
	lma "github.com/tigera/lma/pkg/elastic"
)

const (
	DefaultPeriod      = 5 * time.Minute
	MinimumAlertPeriod = 5 * time.Second

	GlobalAlertSpecTypeFieldName = "Type"
)

type Alert struct {
	alert       *v3.GlobalAlert
	calicoCLI   calicoclient.Interface
	es          es.Service
	adj         ad.ADService
	clusterName string
}

// NewAlert sets and returns an Alert, builds Elasticsearch query that will be used periodically to query Elasticsearch data.
func NewAlert(globalAlert *v3.GlobalAlert, calicoCLI calicoclient.Interface, lmaESClient lma.Client, k8sClient kubernetes.Interface,
	podTemplateQuery podtemplate.ADPodTemplateQuery, adDetectionController controller.AnomalyDetectionController,
	adTrainingController controller.AnomalyDetectionController, clusterName string, namespace string) (*Alert, error) {
	globalAlert.Status.Active = true
	globalAlert.Status.LastUpdate = &metav1.Time{Time: time.Now()}

	// extract by Reflect to handle GlobalAlert on managed clusters with CE version before Type field exists
	globalAlertSpec := reflect.ValueOf(&globalAlert.Spec).Elem()

	globalAlertType, ok := globalAlertSpec.FieldByName(GlobalAlertSpecTypeFieldName).Interface().(v3.GlobalAlertType)

	alert := &Alert{
		alert:       globalAlert,
		calicoCLI:   calicoCLI,
		clusterName: clusterName,
	}

	if !ok || globalAlertType != v3.GlobalAlertTypeAnomalyDetection {
		elastic, err := es.NewService(lmaESClient, clusterName, globalAlert)
		if err != nil {
			return nil, err
		}

		alert.es = elastic

	} else {
		adj, err := ad.NewService(calicoCLI, k8sClient, podTemplateQuery, adDetectionController, adTrainingController, clusterName, namespace, globalAlert)
		if err != nil {
			return nil, err
		}

		alert.adj = adj
	}

	return alert, nil
}

func (a *Alert) Execute(ctx context.Context) {
	log.Debugf("Handling of type: %s.", a.alert.Spec.Type)

	// extract by Reflect to handle GlobalAlert on managed clusters with CE version before Type field exists
	globalAlertSpec := reflect.ValueOf(&a.alert.Spec).Elem()

	globalAlertType, ok := globalAlertSpec.FieldByName(GlobalAlertSpecTypeFieldName).Interface().(v3.GlobalAlertType)

	if !ok || globalAlertType != v3.GlobalAlertTypeAnomalyDetection {
		a.ExecuteElasticQuery(ctx)
	} else {
		a.ExecuteAnomalyDetection(ctx)
	}
}

// ExecuteAnomalyDetection starts the service for GlobalAlerts
// specified for anomaly detection.  The scheduling of training
// and anomaly detection of the jobs are done in the service itself.
func (a *Alert) ExecuteAnomalyDetection(ctx context.Context) {
	a.alert.Status = a.adj.Start(ctx)
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error { return a.updateStatus(ctx) }); err != nil {
		log.WithError(err).Errorf("failed to update status GlobalAlert %s in cluster %s, maximum retries reached", a.alert.Name, a.clusterName)
	}

	for {
		<-ctx.Done()
		a.stopAnomalyDetectionService(ctx)
		return
	}
}

func (a *Alert) stopAnomalyDetectionService(ctx context.Context) {
	a.alert.Status.Active = false
	a.alert.Status = a.adj.Stop()

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error { return a.updateStatus(ctx) }); err != nil {
		log.WithError(err).Errorf("failed to update status GlobalAlert %s in cluster %s, maximum retries reached", a.alert.Name, a.clusterName)
	}
}

// ExecuteElasticQuery periodically queries the Elasticsearch, updates GlobalAlert status
// and adds alerts to events index if alert conditions are met.
// If parent context is cancelled, updates the GlobalAlert status and returns.
// It also deletes any existing elastic watchers for the cluster.
func (a *Alert) ExecuteElasticQuery(ctx context.Context) {
	a.es.DeleteElasticWatchers(ctx)
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error { return a.updateStatus(ctx) }); err != nil {
		log.WithError(err).Errorf("failed to update status GlobalAlert %s in cluster %s, maximum retries reached", a.alert.Name, a.clusterName)
	}

	for {
		timer := time.NewTimer(a.getDurationUntilNextAlert())
		select {
		case <-timer.C:

			a.alert.Status = a.es.ExecuteAlert(a.alert)
			if err := retry.RetryOnConflict(retry.DefaultRetry, func() error { return a.updateStatus(ctx) }); err != nil {
				log.WithError(err).Errorf("failed to update status GlobalAlert %s in cluster %s, maximum retries reached", a.alert.Name, a.clusterName)
			}

			timer.Stop()
		case <-ctx.Done():
			a.alert.Status.Active = false

			if err := retry.RetryOnConflict(retry.DefaultRetry, func() error { return a.updateStatus(ctx) }); err != nil {
				log.WithError(err).Errorf("failed to update status GlobalAlert %s in cluster %s, maximum retries reached", a.alert.Name, a.clusterName)
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
			// that burns through our pod resources and spams Elasticsearch.
			return MinimumAlertPeriod
		}
		return timeUntilNextRun
	}
	return alertPeriod
}

// updateStatus gets the latest GlobalAlert and updates its status.
func (a *Alert) updateStatus(ctx context.Context) error {
	log.Debugf("Updating status of GlobalAlert %s in cluster %s", a.alert.Name, a.clusterName)
	alert, err := a.calicoCLI.ProjectcalicoV3().GlobalAlerts().Get(ctx, a.alert.Name, metav1.GetOptions{})
	if err != nil {
		log.WithError(err).Errorf("could not get GlobalAlert %s in cluster %s", a.alert.Name, a.clusterName)
		return err
	}
	alert.Status = a.alert.Status
	_, err = a.calicoCLI.ProjectcalicoV3().GlobalAlerts().UpdateStatus(ctx, alert, metav1.UpdateOptions{})
	if err != nil {
		log.WithError(err).Errorf("could not update status of GlobalAlert %s in cluster %s", a.alert.Name, a.clusterName)
		return err
	}
	return nil
}

// EqualAlertSpec does reflect.DeepEqual on give spec and cached alert spec.
func (a *Alert) EqualAlertSpec(spec v3.GlobalAlertSpec) bool {
	return reflect.DeepEqual(a.alert.Spec, spec)
}
