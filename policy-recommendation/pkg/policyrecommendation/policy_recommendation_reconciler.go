// Copyright (c) 2022-2023 Tigera Inc. All rights reserved.

package policyrecommendation

import (
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"

	"github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/calico/libcalico-go/lib/set"
	linseed "github.com/projectcalico/calico/linseed/pkg/client"
	calicores "github.com/projectcalico/calico/policy-recommendation/pkg/calico-resources"
	"github.com/projectcalico/calico/policy-recommendation/pkg/engine"
	"github.com/projectcalico/calico/policy-recommendation/pkg/resources"
	"github.com/projectcalico/calico/policy-recommendation/pkg/syncer"
	prtypes "github.com/projectcalico/calico/policy-recommendation/pkg/types"
	"github.com/projectcalico/calico/policy-recommendation/utils"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/client"
)

const (
	defaultPolicyRecEngineInterval               = 150 * time.Second
	defaultPolicyRecEngineInitialLookback        = 24 * time.Hour
	defaultPolicyRecEngineStabilizationPeriod    = 10 * time.Minute
	defaultPolicyRecEngineMaxRules               = 20
	defaultPolicyRecEnginePoliciesLearningCutOff = 20

	minimumInterval = 30 * time.Second
	tierOrder       = math.MaxFloat64
)

type policyRecommendationReconciler struct {
	stateLock         sync.Mutex
	state             *policyRecommendationScopeState
	calico            calicoclient.ProjectcalicoV3Interface
	linseedClient     linseed.Client
	synchronizer      client.QueryInterface
	caches            *syncer.CacheSet
	cluster           string
	serviceNameSuffix string
	tickDuration      chan time.Duration
	clock             engine.Clock
	ticker            *time.Ticker
	suffixGenerator   *func() string
}

type policyRecommendationScopeState struct {
	object v3.PolicyRecommendationScope
	cancel context.CancelFunc
}

func NewPolicyRecommendationReconciler(
	calico calicoclient.ProjectcalicoV3Interface,
	linseedClient linseed.Client,
	synchronizer client.QueryInterface,
	caches *syncer.CacheSet,
	clock engine.Clock,
	serviceSuffixName string,
	suffixGenerator *func() string,
) *policyRecommendationReconciler {
	td := new(chan time.Duration)

	return &policyRecommendationReconciler{
		state:             nil,
		calico:            calico,
		linseedClient:     linseedClient,
		synchronizer:      synchronizer,
		caches:            caches,
		tickDuration:      *td,
		clock:             clock,
		serviceNameSuffix: serviceSuffixName,
		suffixGenerator:   suffixGenerator,
	}
}

func (pr *policyRecommendationReconciler) Reconcile(name types.NamespacedName) error {
	if pr.calico == nil {
		err := errors.New("calico client is nil, unable to access datastore")
		log.WithError(err)

		return err
	}

	scope, err := pr.calico.PolicyRecommendationScopes().Get(
		context.Background(), name.Name, metav1.GetOptions{})
	if err != nil {
		log.Error(err)
		if k8serrors.IsNotFound(err) {
			// PolicyRecommendationScope resource not found
			query := getPolicyRecommendationScopeQuery(name.Name, nil, api.UpdateTypeKVDeleted)
			if _, err = pr.synchronizer.RunQuery(context.Background(), query); err != nil {
				return err
			}

			pr.cancelRecommendationRoutine()
			return nil
		}

		return err
	}

	// Set the scope's parameters to default values, if not already defined
	setPolicyRecommendationScopeDefaults(scope)

	// The state has not previously been set, this will be a new recommendation
	if pr.state == nil {
		ctx, cancel := context.WithCancel(context.Background())

		state := policyRecommendationScopeState{
			object: *scope,
			cancel: cancel,
		}

		pr.stateLock.Lock()
		defer pr.stateLock.Unlock()

		pr.state = &state

		// New PolicyRecommendationScope resource
		query := getPolicyRecommendationScopeQuery(scope.Name, scope, api.UpdateTypeKVNew)
		if _, err = pr.synchronizer.RunQuery(ctx, query); err != nil {
			log.Error(err)
			return err
		}

		// Create go routine for engine
		go pr.continuousRecommend(ctx)

		return nil
	}

	if !resources.DeepEqual(scope.Spec, pr.state.object.Spec) {
		// The PolicyRecommendationScope has been updated, update the state as to not reset the
		// recommendation engine

		ctx, cancel := context.WithCancel(context.Background())

		state := policyRecommendationScopeState{
			object: *scope,
			cancel: cancel,
		}

		if pr.state.object.Spec.Interval != scope.Spec.Interval {
			// Signify a new ticker interval for the recommendation engine
			pr.state.object.Spec.Interval = scope.Spec.Interval

			duration := getDurationUntilNextIteration(pr.state.object.Spec.Interval.Duration)
			pr.ticker = time.NewTicker(duration)
		}

		pr.stateLock.Lock()
		defer pr.stateLock.Unlock()

		// Updates the state with the new policy recommendation scope
		pr.state = &state

		// Update the PolicyRecommendationScope resources
		query := getPolicyRecommendationScopeQuery(scope.Name, scope, api.UpdateTypeKVUpdated)
		if _, err = pr.synchronizer.RunQuery(ctx, query); err != nil {
			log.Error(err)
			return err
		}
	}

	return nil
}

func (pr *policyRecommendationReconciler) Close() {
	// Close the engine
	pr.cancelRecommendationRoutine()
}

func (pr *policyRecommendationReconciler) cancelRecommendationRoutine() {
	pr.stateLock.Lock()
	defer pr.stateLock.Unlock()

	if pr.state == nil {
		return
	}

	pr.state.cancel()
	pr.state = nil
}

// continuousRecommend runs the engine that will generate new recommendation rules for each Snp.
// Updates the ticker if policy recommendation scope interval is updated.
func (pr *policyRecommendationReconciler) continuousRecommend(ctx context.Context) {
	if pr.state == nil {
		// Break once policy recommendation has been disabled or is not tracked anymore
		return
	}

	// Set a new ticker duration for the engine run cycle
	duration := getDurationUntilNextIteration(pr.state.object.Spec.Interval.Duration)
	pr.ticker = time.NewTicker(duration)

	currEnabled := pr.state.object.Spec.NamespaceSpec.RecStatus == v3.Enabled
	prevEnabled := !currEnabled
	go func() {
		for {
			pr.stateLock.Lock()
			select {
			case <-pr.ticker.C:
				{
					if currEnabled != prevEnabled {
						// Print the status once, after the state changes
						logRecommendationStatus(currEnabled)
						prevEnabled = currEnabled
					}

					if pr.state != nil {
						currEnabled = pr.state.object.Spec.NamespaceSpec.RecStatus == v3.Enabled

						if currEnabled {
							// Generate new recommendations, and sync with datastore
							snps := pr.caches.StagedNetworkPolicies.GetAll()
							for _, snp := range snps {
								pr.RecommendSnp(ctx, pr.clock, snp)
							}

						}
					}
				}
			case <-ctx.Done():
				pr.ticker.Stop()
				return
			}
			// Do not use a deferral, as this is in an infinite for loop and we want to unlock the state
			// per for-loop cycle
			pr.stateLock.Unlock()
		}
	}()
}

// getDurationUntilNextIteration returns the duration until the next engine reconciliation. If the
// value is less than the allowed minimum, will return the minimum value (min: 30s).
func getDurationUntilNextIteration(interval time.Duration) time.Duration {
	retInterval := interval
	if retInterval < minimumInterval {
		// return MinimumInterval instead of a duration less than 30s to guarantee that we would never
		// have a tight loop that burns through our pod resources.
		retInterval = minimumInterval
	}

	log.Infof("Polling interval set to: %s (min 30s)", retInterval.String())
	return retInterval
}

// GetNetworkSets returns the set of network sets currently in the cache
func (pr *policyRecommendationReconciler) GetNetworkSets() set.Set[engine.NetworkSet] {
	if pr.caches == nil || pr.caches.Namespaces == nil {
		return set.Empty[engine.NetworkSet]()
	}

	// Get the list of synched-up network sets, and define the set of network sets
	netSets := pr.caches.Namespaces.GetAll()
	netSetNames := []engine.NetworkSet{}
	for _, ns := range netSets {
		netSetNames = append(netSetNames, engine.NetworkSet{Name: ns.Name, Namespace: ns.Namespace})
	}

	return set.FromArray(netSetNames)
}

// getLookback returns the InitialLookback period if the policy is new and has not previously
// been updated, otherwise use twice the engine-run interval (Default: 2.5min).
func (pr *policyRecommendationReconciler) getLookback(snp v3.StagedNetworkPolicy) time.Duration {
	_, ok := snp.Annotations["policyrecommendation.tigera.io/lastUpdated"]
	if !ok {
		// First time run will use the initial lookback
		return pr.state.object.Spec.InitialLookback.Duration
	}
	// Twice the engine-run interval
	lookback := pr.state.object.Spec.Interval.Duration * 2

	return lookback
}

// RecommendSnp consolidates an snp rules into the engine's for a new run, and updates the datastore
// if a change occurs.
func (pr *policyRecommendationReconciler) RecommendSnp(ctx context.Context, clock engine.Clock, snp *v3.StagedNetworkPolicy) {
	tierOrderPtr := ptrFloat64(tierOrder)

	// Run the engine to update the snp's rules from new flows
	engine.RunEngine(
		ctx,
		pr.calico,
		pr.linseedClient,
		pr.getLookback(*snp),
		tierOrderPtr,
		pr.cluster,
		pr.serviceNameSuffix,
		clock,
		pr.state.object.Spec.Interval.Duration,
		pr.state.object.Spec.StabilizationPeriod.Duration,
		getRecommendationScopeOwner(&pr.state.object),
		snp,
	)

	if err := calicores.MaybeCreateTier(ctx, pr.calico, prtypes.PolicyRecommendationTier, tierOrderPtr); err != nil {
		// Failed to create the tier, cannot proceed further. Will give it a go in the next cycle
		return
	}

	prevStatus, currStatus := pr.updateStatus(snp)
	if currStatus == calicores.StableStatus && prevStatus != calicores.StableStatus {
		// Once stable, replace the policy with a new one, containing a different name suffix
		pr.replaceRecommendation(ctx, snp)
	} else {
		// Update the snp on the datastore
		_ = pr.syncToDatastore(ctx, snp.Name, snp.Namespace, snp)
	}
}

// getPolicyCopyWithNewSuffix returns a copy of the staged network policy with a new hash suffix in
// the name.
func (pr *policyRecommendationReconciler) getPolicyCopyWithNewSuffix(snp v3.StagedNetworkPolicy) v3.StagedNetworkPolicy {
	// Create a copy with a new suffix in the name
	snp.Name = utils.GetPolicyName(snp.Spec.Tier, snp.Namespace, *pr.suffixGenerator)

	return snp
}

// reconcileCacheMeta reconciles the staged action metadata of the cache, if the datastore values
// have been updated. This should have been done by the StagedNetworkPolicy reconciler, making sure
// in case that hasn't occurred.
func (pr *policyRecommendationReconciler) reconcileCacheMeta(cache, store *v3.StagedNetworkPolicy) {
	if cache == nil || store == nil {
		return
	}

	sa := store.Spec.StagedAction
	if cache.Spec.StagedAction != sa || cache.Labels[calicores.StagedActionKey] != string(sa) {
		log.WithField("key", cache).Debugf("Reconciling cached staged action to: %s", sa)
		cache.Spec.StagedAction = sa
		cache.Labels[calicores.StagedActionKey] = string(sa)
	}
}

// isEnforced returns true if there is a policy within the same namespace that is enforced within
// the same tier.
func (pr *policyRecommendationReconciler) isEnforced(ctx context.Context, namespace string, clog *log.Entry) bool {
	if nps, err := pr.calico.NetworkPolicies(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", calicores.TierKey, pr.state.object.Spec.NamespaceSpec.TierName),
	}); err == nil {
		if len(nps.Items) > 0 {
			return true
		}
	} else {
		clog.WithError(err).Error("failed to get the network policy")
	}

	return false
}

// isLearning returns true if the recommendation is in 'Learning' status. The recommendation has to
// be owned by the v3.PolicyRecommendationScope resource to be learning.
func (pr *policyRecommendationReconciler) isLearning(snp *v3.StagedNetworkPolicy) bool {
	if snp == nil {
		return false
	}

	owner := getRecommendationScopeOwner(&pr.state.object)
	owners := []metav1.OwnerReference{*owner}
	if snp.OwnerReferences == nil || !reflect.DeepEqual(snp.OwnerReferences, owners) {
		// If the recommendation is not owned by policy recommendation, it isn't learning
		return false
	}
	if snp.Spec.StagedAction != v3.StagedActionLearn {
		return false
	}

	return true
}

// replaceRecommendation replaces the cache and store entries with a new policy. The new policy is a copy of the old one, with a new
// The approach meant to address denied-bytes associated with long lived connections.
func (pr *policyRecommendationReconciler) replaceRecommendation(ctx context.Context, snp *v3.StagedNetworkPolicy) {
	namespace := snp.Namespace
	// Delete the processing recommendation from the store
	oldKey := snp.Name
	if err := pr.syncToDatastore(ctx, oldKey, namespace, nil); err != nil {
		log.WithField("key", oldKey).Debug("Could not delete policy")
		return
	}
	// Add the new recommendation to the store. This will be the stable version of the old policy
	newSnp := pr.getPolicyCopyWithNewSuffix(*snp)
	newKey := newSnp.Name
	if err := pr.syncToDatastore(ctx, newKey, namespace, &newSnp); err != nil {
		log.WithField("key", newKey).Debug("Could not create policy")
		return
	}
	// Update the cache after a successful replacement
	pr.caches.StagedNetworkPolicies.Set(namespace, &newSnp)
	log.WithField("key", newSnp.Name).Debug("Added item to cache")

	log.Infof("Replaced %s with stable recommendation %s", oldKey, newKey)
}

// syncToDatastore syncs staged network policy item in the cache with the Calico datastore. Updates
// the datastore only if there are rules associated with the cached policy, otherwise returns
// without error.
func (pr *policyRecommendationReconciler) syncToDatastore(ctx context.Context, cacheKey, namespace string, cacheItem *v3.StagedNetworkPolicy) error {
	clog := log.WithField("key", cacheKey)
	key := cacheKey
	clog.Debug("Synching to datastore")

	if cacheItem == nil {
		datastoreSnp, err := pr.calico.StagedNetworkPolicies(namespace).Get(ctx, key, metav1.GetOptions{})
		if err == nil {
			if datastoreSnp != nil && pr.isLearning(datastoreSnp) {
				if err := pr.calico.StagedNetworkPolicies(namespace).Delete(ctx, cacheKey, metav1.DeleteOptions{}); err != nil {
					if !k8serrors.IsNotFound(err) {
						clog.WithError(err).Error("failed to delete StagedNetworkPolicy")
					}

					return err
				}
				clog.Info("Deleted StagedNetworkPolicy from datastore")
				return nil
			}

			return errors.New("cannot delete an active policy")
		}

		return err
	}

	if skipUpdate(cacheItem) {
		// Cached item does not contain rules, return without error
		clog.Debugf("Skipping StagedNetworkPolicy update, cached item doesn't contain rules")
		return nil
	}

	datastoreSnp, err := pr.calico.StagedNetworkPolicies(namespace).Get(ctx, key, metav1.GetOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			clog.WithError(err).Error("failed to create StagedNetworkPolicy in datastore")
			// We hit an error other than "not found"
			return err
		}

		if pr.isEnforced(ctx, namespace, clog) {
			// An enforced network policy exists in the tier and namespace
			clog.Debug("Skipping datastore creation, enforced policy exists")
			return nil
		}

		// Item does not exist in the datastore - create a new StagedNetworkPolicy
		if _, err = pr.calico.StagedNetworkPolicies(namespace).Create(ctx, cacheItem, metav1.CreateOptions{}); err != nil {
			clog.WithError(err).Error("failed to create StagedNetworkPolicy in datastore")
			return err
		}
		clog.Info("Created StagedNetworkPolicy in datastore")

		return nil
	}

	// Updates to the store value's stagedAction should be synced with the cache item
	pr.reconcileCacheMeta(cacheItem, datastoreSnp)

	if pr.isLearning(datastoreSnp) && !equalSnps(datastoreSnp, cacheItem) {
		// Update a store to reflect the update to the cache
		utils.CopyStagedNetworkPolicy(datastoreSnp, *cacheItem)

		if _, err = pr.calico.StagedNetworkPolicies(namespace).Update(ctx, datastoreSnp, metav1.UpdateOptions{}); err != nil {
			clog.WithError(err).Error("failed to update StagedNetworkPolicy in datastore")
			return err
		}
		clog.WithField("stagedNetworkPolicy", key).Info("Updated StagedNetworkPolicy in datastore")
	}

	return nil
}

// updateStatus updates the recommendation's status, and returns true if the status has become
// stable
func (pr *policyRecommendationReconciler) updateStatus(snp *v3.StagedNetworkPolicy) (prev, curr string) {
	prev = snp.Annotations[calicores.StatusKey]
	// Update the status annotation, if necessary
	pr.updateStatusAnnotation(snp)
	curr = snp.Annotations[calicores.StatusKey]

	return
}

// updateStatusAnnotation updates the learning annotation of a staged network policy given
// the time since the last update.
//
//   - Learning
//     Policy rule was updated <= 2 x recommendation interval ago
//   - Stabilizing
//     Policy was updated > 2 x recommendation interval ago. The flows contain policy matches that
//     match the expected policy hits for the recommended policy, and may still contain some logs
//     that do not. The flows that do not match are fully covered by the existing rules in the
//     recommended policy (i.e. no further changes are required to the policy)
//   - Stable
//     Policy was updated > stabilization period ago. The flows all contain the expected
//     recommended policy hits
func (pr *policyRecommendationReconciler) updateStatusAnnotation(snp *v3.StagedNetworkPolicy) {
	if len(snp.Spec.Egress) == 0 && len(snp.Spec.Ingress) == 0 {
		// No update to status annotation necessary
		return
	}

	lastUpdateStr, ok := snp.Annotations[calicores.LastUpdatedKey]
	if !ok {
		// Fist time creating the last update key
		snp.Annotations[calicores.StatusKey] = calicores.LearningStatus
		return
	}
	snpLastUpdateTime, err := time.Parse(time.RFC3339, lastUpdateStr)
	if err != nil {
		log.WithError(err).Debugf("Failed to parse snp last update time using the RFC3339 format")
		return
	}
	nowTime, err := time.Parse(time.RFC3339, pr.clock.NowRFC3339())
	if err != nil {
		log.WithError(err).Debugf("Failed to parse the time now using the RFC3339 format")
		return
	}
	durationSinceLastUpdate := nowTime.Sub(snpLastUpdateTime)

	learningPeriod := 2 * pr.state.object.Spec.Interval.Duration
	stabilizingPeriod := pr.state.object.Spec.StabilizationPeriod.Duration

	// Update status
	switch {
	case durationSinceLastUpdate >= 0 && durationSinceLastUpdate <= learningPeriod:
		if snp.Annotations[calicores.StatusKey] != calicores.LearningStatus {
			log.WithField("key", snp.Name).Debugf("Learning. Duration since last update %f <= %f", durationSinceLastUpdate.Seconds(), learningPeriod.Seconds())
			snp.Annotations[calicores.StatusKey] = calicores.LearningStatus
		}
	case durationSinceLastUpdate <= stabilizingPeriod:
		if snp.Annotations[calicores.StatusKey] != calicores.StabilizingStatus {
			log.WithField("key", snp.Name).Debugf("Stablizing. Duration since last update %f > %f (learning period) and %f <= %f (stable period)",
				durationSinceLastUpdate.Seconds(), learningPeriod.Seconds(), durationSinceLastUpdate.Seconds(), stabilizingPeriod.Seconds())
			snp.Annotations[calicores.StatusKey] = calicores.StabilizingStatus
		}
	case durationSinceLastUpdate > stabilizingPeriod:
		if snp.Annotations[calicores.StatusKey] != calicores.StableStatus {
			log.WithField("key", snp.Name).Debugf("Stable. Duration since last update %f > %f", durationSinceLastUpdate.Seconds(), stabilizingPeriod.Seconds())
			snp.Annotations[calicores.StatusKey] = calicores.StableStatus
		}
	default:
		log.Warnf("Invalid status")
		snp.Annotations[calicores.StatusKey] = calicores.NoDataStatus
	}
}

// Utilities

// equalSnps returns true if two compared staged network policies are equal by name, namespace,
// spec.selector, owner references, annotations, labels and rules to determine their equality.
func equalSnps(left, right *v3.StagedNetworkPolicy) bool {
	if left.Name != right.Name {
		log.Infof("Name %s differs from %s", left.Name, right.Name)
		return false
	}
	if left.Namespace != right.Namespace {
		log.Infof("Namespace %s differs from %s", left.Namespace, right.Namespace)
		return false
	}
	if left.Spec.Selector != right.Spec.Selector {
		log.Infof("Selector %s differs from %s", left.Spec.Selector, right.Spec.Selector)
		return false
	}
	if !reflect.DeepEqual(left.OwnerReferences, right.OwnerReferences) {
		log.Infof("OwnerReferences %v differs from %v", left.OwnerReferences, right.OwnerReferences)
		return false
	}
	if !reflect.DeepEqual(left.Annotations, right.Annotations) {
		log.Infof("Annotations %v differs from %v", left.Annotations, right.Annotations)
		return false
	}
	if !reflect.DeepEqual(left.Labels, right.Labels) {
		log.Infof("Labels %v differs from %v", left.Labels, right.Labels)
		return false
	}
	if !reflect.DeepEqual(left.Spec.Types, right.Spec.Types) {
		log.Infof("Types %v differs from %v", left.Spec.Types, right.Spec.Types)
		return false
	}
	if !(len(left.Spec.Egress) == 0 && len(right.Spec.Egress) == 0) && !reflect.DeepEqual(left.Spec.Egress, right.Spec.Egress) {
		log.Infof("Egress %v differs from %v", left.Spec.Egress, right.Spec.Egress)
		return false
	}
	if !(len(left.Spec.Ingress) == 0 && len(right.Spec.Ingress) == 0) && !reflect.DeepEqual(left.Spec.Ingress, right.Spec.Ingress) {
		log.Infof("Ingress %v differs from %v", left.Spec.Ingress, right.Spec.Ingress)
		return false
	}

	return true
}

// logRecommendationStatus logs the Policy Recommendation enabled status change.
func logRecommendationStatus(enabled bool) {
	if enabled {
		log.Info("Policy Recommendation enabled, start polling...")
	} else {
		log.Info("Policy Recommendation disabled...")
	}
}

// getPolicyRecommendationScopeQuery returns a syncer.PolicyRecommendationScopeQuery query.
func getPolicyRecommendationScopeQuery(
	name string, scope *v3.PolicyRecommendationScope, updateType api.UpdateType,
) syncer.PolicyRecommendationScopeQuery {
	// Update type is 'Delete'
	if updateType == api.UpdateTypeKVDeleted {
		return syncer.PolicyRecommendationScopeQuery{
			MetaSelectors: syncer.MetaSelectors{
				Source: &api.Update{
					UpdateType: api.UpdateTypeKVDeleted,
					KVPair: model.KVPair{
						Key: model.ResourceKey{
							Name: name,
							Kind: v3.KindPolicyRecommendationScope,
						},
					},
				},
			},
		}
	}

	// Update Type is 'New' or 'Update'
	return syncer.PolicyRecommendationScopeQuery{
		MetaSelectors: syncer.MetaSelectors{
			Source: &api.Update{
				UpdateType: updateType,
				KVPair: model.KVPair{
					Key: model.ResourceKey{
						Name: name,
						Kind: v3.KindPolicyRecommendationScope,
					},
					Value: scope,
				},
			},
			Labels: scope.Labels,
		},
	}
}

// getRecommendationScopeOwner returns policy recommendation scope resource as an owner reference
// resource.
func getRecommendationScopeOwner(scope *v3.PolicyRecommendationScope) *metav1.OwnerReference {
	ctrl := true
	blockOwnerDelete := false

	log.Debugf("Owner - apiversion: %s, kind: %s, name: %s, uid: %s", scope.APIVersion, scope.Kind, scope.Name, scope.UID)

	return &metav1.OwnerReference{
		APIVersion:         "projectcalico.org/v3",
		Kind:               "PolicyRecommendationScope",
		Name:               scope.Name,
		UID:                scope.UID,
		Controller:         &ctrl,
		BlockOwnerDeletion: &blockOwnerDelete,
	}
}

// ptrFloat64 returns a pointer to the float64 parameter passed in as input.
func ptrFloat64(fl float64) *float64 {
	return &fl
}

// setPolicyRecommendationScopeDefaults sets the default values of policy recommendations scope
// parameters, if not defined in the resource definition. The recStatus and selector are assumed
// to have been set.
func setPolicyRecommendationScopeDefaults(scope *v3.PolicyRecommendationScope) {
	// InitialLookback. Default: 24h
	if scope.Spec.InitialLookback == nil {
		scope.Spec.InitialLookback = &metav1.Duration{
			Duration: defaultPolicyRecEngineInitialLookback,
		}
	}
	// Interval. Default: 2m30s
	if scope.Spec.Interval == nil {
		scope.Spec.Interval = &metav1.Duration{Duration: defaultPolicyRecEngineInterval}
	}
	// StabilizationPeriod. Default: 10m
	if scope.Spec.StabilizationPeriod == nil {
		scope.Spec.StabilizationPeriod = &metav1.Duration{
			Duration: defaultPolicyRecEngineStabilizationPeriod,
		}
	}
	// MaxRules. Default: 20
	if scope.Spec.MaxRules == nil {
		maxRules := defaultPolicyRecEngineMaxRules
		scope.Spec.MaxRules = &maxRules
	}
	// PoliciesLearningCutOff. Default: 20
	if scope.Spec.PoliciesLearningCutOff == nil {
		policiesLearningCutOff := defaultPolicyRecEnginePoliciesLearningCutOff
		scope.Spec.PoliciesLearningCutOff = &policiesLearningCutOff
	}
	// TierName. Default: 'namespace-isolation'
	if scope.Spec.NamespaceSpec.TierName == "" {
		scope.Spec.NamespaceSpec.TierName = prtypes.PolicyRecommendationTier
	}
}

// skipUpdate returns true if the staged network policy's v3.PolicyType is empty, i.e. doesn't
// contain any rules.
func skipUpdate(snp *v3.StagedNetworkPolicy) bool {
	return len(snp.Spec.Types) == 0
}
