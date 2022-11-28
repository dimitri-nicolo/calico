// Copyright (c) 2022 Tigera Inc. All rights reserved.

package policyrecommendation

import (
	"context"
	"sync"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"

	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/engine"
	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/resources"

	log "github.com/sirupsen/logrus"
)

const (
	defaultPolicyRecEngineInterval = 150 * time.Second
	minimumInterval                = 5 * time.Second

	policyRecommendationLastUpdatedKey = "policyrecommendation.tigera.io/lastUpdated"
	policyRecommendationTimeFormat     = time.RFC3339
)

type policyRecommendationReconciler struct {
	stateLock sync.Mutex
	state     *policyRecommendationScopeState
	calico    calicoclient.ProjectcalicoV3Interface
}

type policyRecommendationScopeState struct {
	object v3.PolicyRecommendationScope
	cancel context.CancelFunc
}

func (pr *policyRecommendationReconciler) Reconcile(namespacedName types.NamespacedName) error {
	prScope, err := pr.calico.PolicyRecommendationScopes().Get(
		context.Background(),
		namespacedName.Name,
		metav1.GetOptions{},
	)

	if err != nil {
		if k8serrors.IsNotFound(err) {
			pr.cancelRecommondationRoutine()
			return nil
		}
		return err
	}

	if pr.state == nil {
		ctx, cancel := context.WithCancel(context.Background())

		state := policyRecommendationScopeState{
			object: *prScope,
			cancel: cancel,
		}

		// create go routine for engine
		pr.stateLock.Lock()
		defer pr.stateLock.Unlock()

		pr.state = &state

		go pr.runRecommendationEngine(ctx)

		return nil
	}

	// updates state as to not reset the recommendation engine
	if !resources.DeepEqual(prScope.Spec, pr.state.object.Spec) {
		pr.stateLock.Lock()
		defer pr.stateLock.Unlock()

		// updates the state with the new
		pr.state.object = *prScope
	}

	return nil
}

func (pr *policyRecommendationReconciler) runRecommendationEngine(ctx context.Context) {
	for {
		pr.stateLock.Lock()
		if pr.state == nil {
			// break once policyrec has been disabled or is not tracked anymore
			return
		}
		timer := time.NewTimer(getDuratioUntilNextIteration(pr.state.object))
		pr.stateLock.Unlock()

		select {
		case <-timer.C:
			pr.stateLock.Lock()
			defer pr.stateLock.Unlock()

			// run engine and updated status
			status := engine.RunEngine(pr.state.object)

			pr.state.object.Status = status
			pr.state.object.Annotations[policyRecommendationLastUpdatedKey] = time.Now().Format(policyRecommendationTimeFormat)

			_, err := pr.calico.PolicyRecommendationScopes().UpdateStatus(
				context.Background(),
				&pr.state.object,
				metav1.UpdateOptions{},
			)

			log.WithError(err).Warnf("failed to udpate status of %s with %+v", pr.state.object.Name, status)
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

func getDuratioUntilNextIteration(policyRecResource v3.PolicyRecommendationScope) time.Duration {
	interval := defaultPolicyRecEngineInterval

	if policyRecResource.Spec.Interval != nil {
		interval = policyRecResource.Spec.Interval.Duration
	}

	lastUpdatedValue := policyRecResource.Annotations[policyRecommendationLastUpdatedKey]

	lastUpdated, err := time.Parse(
		time.RFC3339,
		lastUpdatedValue,
	)

	if err != nil {
		return interval
	}

	now := time.Now()
	durationSinceLastExecution := now.Sub(lastUpdated.Local())
	if durationSinceLastExecution < 0 {
		log.Errorf("last executed alert is in the future")
		return minimumInterval
	}

	timeUntilNextRun := interval - durationSinceLastExecution
	if timeUntilNextRun <= 0 {
		// return MinimumInterval instead of 0s to guarantee that we would never have a tight loop
		// that burns through our pod resources.
		return minimumInterval
	}
	return timeUntilNextRun

}

func (pr *policyRecommendationReconciler) Close() {
	// clsoes the engine
	pr.cancelRecommondationRoutine()
}

func (pr *policyRecommendationReconciler) cancelRecommondationRoutine() {
	pr.stateLock.Lock()
	defer pr.stateLock.Unlock()

	if pr.state == nil {
		return
	}

	pr.state.cancel()
	pr.state = nil
}
