// Copyright 2021 Tigera Inc. All rights reserved.

package alert

import (
	"context"
	"reflect"
	"time"

	libcalicov3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	v3 "github.com/projectcalico/apiserver/pkg/apis/projectcalico/v3"
	calicoclient "github.com/projectcalico/apiserver/pkg/client/clientset_generated/clientset"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	log "github.com/sirupsen/logrus"

	"github.com/olivere/elastic/v7"
	es "github.com/tigera/intrusion-detection/controller/pkg/globalalert/elastic"
)

type Alert struct {
	alert       *v3.GlobalAlert
	calicoCLI   calicoclient.Interface
	es          *es.Service
	clusterName string
}

const (
	DefaultPeriod = 5 * time.Minute
)

// NewAlert sets and returns an Alert, builds Elasticsearch query that will be used periodically to query Elasticsearch data.
func NewAlert(alert *v3.GlobalAlert, calicoCLI calicoclient.Interface, esCli *elastic.Client, clusterName string) (*Alert, error) {
	alert.Status.Active = true
	alert.Status.LastUpdate = &metav1.Time{Time: time.Now()}

	es, err := es.NewService(esCli, clusterName, alert)
	if err != nil {
		return nil, err
	}

	return &Alert{
		alert:       alert,
		calicoCLI:   calicoCLI,
		es:          es,
		clusterName: clusterName,
	}, nil
}

// Execute periodically queries the Elasticsearch, updates GlobalAlert status
// and adds alerts to events index if alert conditions are met.
// If parent context is cancelled, updates the GlobalAlert status and returns.
// It also deletes any existing elastic watchers for the cluster.
func (a *Alert) Execute(ctx context.Context) {
	ticker := time.NewTicker(a.getAlertDuration())
	a.es.DeleteElasticWatchers(ctx)
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error { return a.updateStatus(ctx) }); err != nil {
		log.WithError(err).Errorf("failed to update status GlobalAlert %s in cluster %s, maximum retries reached", a.alert.Name, a.clusterName)
	}
	for {
		select {
		case <-ticker.C:
			a.alert.Status = a.es.ExecuteAlert(a.alert)
			if err := retry.RetryOnConflict(retry.DefaultRetry, func() error { return a.updateStatus(ctx) }); err != nil {
				log.WithError(err).Errorf("failed to update status GlobalAlert %s in cluster %s, maximum retries reached", a.alert.Name, a.clusterName)
			}
		case <-ctx.Done():
			a.alert.Status.Active = false
			if err := retry.RetryOnConflict(retry.DefaultRetry, func() error { return a.updateStatus(ctx) }); err != nil {
				log.WithError(err).Errorf("failed to update status GlobalAlert %s in cluster %s, maximum retries reached", a.alert.Name, a.clusterName)
			}
			ticker.Stop()
			return
		}
	}
}

// getAlertDuration returns the duration interval at which Elasticsearch should be queried.
func (a *Alert) getAlertDuration() time.Duration {
	if a.alert.Spec.Period == nil {
		return DefaultPeriod
	}
	return a.alert.Spec.Period.Duration
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
func (a *Alert) EqualAlertSpec(spec libcalicov3.GlobalAlertSpec) bool {
	return reflect.DeepEqual(a.alert.Spec, spec)
}
