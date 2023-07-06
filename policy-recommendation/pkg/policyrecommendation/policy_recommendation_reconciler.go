// Copyright (c) 2022-2023 Tigera Inc. All rights reserved.

package policyrecommendation

import (
	"context"
	"errors"
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
	calicoresources "github.com/projectcalico/calico/policy-recommendation/pkg/calico-resources"
	"github.com/projectcalico/calico/policy-recommendation/pkg/engine"
	"github.com/projectcalico/calico/policy-recommendation/pkg/resources"
	"github.com/projectcalico/calico/policy-recommendation/pkg/syncer"
	prtypes "github.com/projectcalico/calico/policy-recommendation/pkg/types"
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
	}
}

func NewPolicyRecommendationScopeState(
	object *v3.PolicyRecommendationScope,
	cancel context.CancelFunc,
) *policyRecommendationScopeState {
	return &policyRecommendationScopeState{
		object: *object,
		cancel: cancel,
	}
}

// Public

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

// Private

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

							// Get the list of synched-up policies
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
	// time to look back while querying flows
	lookback := pr.getLookback(*snp)

	order := float64(tierOrder)
	interval := pr.state.object.Spec.Interval.Duration
	stbl := pr.state.object.Spec.StabilizationPeriod.Duration

	owner := getRecommendationScopeOwner(&pr.state.object)

	// Run the engine to update the snp's rules from new flows
	engine.RunEngine(
		ctx, pr.calico, pr.linseedClient, lookback, &order, pr.cluster, pr.serviceNameSuffix, clock, interval, stbl, owner, snp,
	)

	// Create the Policy Recommendation tier, if it doesn't already exist
	tier := snp.Spec.Tier
	if err := calicoresources.MaybeCreateTier(ctx, pr.calico, tier, &order); err != nil {
		log.WithError(err).Errorf("failed to create tier: %s", tier)
		return
	}

	// Update the snp on the datastore
	if err := pr.syncToDatastore(ctx, snp); err != nil {
		log.WithError(err).Debugf("failed to write staged network policy for namespace: %s", snp.Name)
	}
}

// reconcileCacheMeta reconciles the staged action metadata of the cache, if the datastore values
// have been updated. This should have been done by the StagedNetworkPolicy reconciler, making sure
// in case that hasn't occurred.
func (pr *policyRecommendationReconciler) reconcileCacheMeta(cache, store *v3.StagedNetworkPolicy) {
	if cache == nil || store == nil {
		return
	}

	sa := store.Spec.StagedAction
	if cache.Spec.StagedAction != sa || cache.Labels[calicoresources.StagedActionKey] != string(sa) {
		log.WithField("key", cache).Debugf("Reconciling cached staged action to: %s", sa)
		cache.Spec.StagedAction = sa
		cache.Labels[calicoresources.StagedActionKey] = string(sa)
	}
}

// shouldSkipStoreUpdate returns true of the store is nil, but the cache isn't or if the store's
// staged action metadata differs from that of the cache's
func (pr *policyRecommendationReconciler) shouldSkipStoreUpdate(cache, store *v3.StagedNetworkPolicy) bool {
	if cache == nil {
		return true
	}

	owner := getRecommendationScopeOwner(&pr.state.object)
	owners := []metav1.OwnerReference{*owner}
	if cache.OwnerReferences == nil || !reflect.DeepEqual(cache.OwnerReferences, owners) {
		// If the recommendation is not owned by policy recommendation, don't update
		return true
	}

	if (store == nil && cache.Spec.StagedAction == v3.StagedActionLearn) ||
		(store != nil && store.Spec.StagedAction == v3.StagedActionLearn) {
		// An update should occur. The cache is in 'Learn' state, and the only source of truth, or the
		// store is in 'Learn' state
		return false
	}

	return true
}

// syncToDatastore syncs staged network policy item in the cache with the Calico datastore. Updates
// the datastore only if there are rules associated with the cached policy, otherwise returns
// without error.
func (pr *policyRecommendationReconciler) syncToDatastore(ctx context.Context, cacheItem *v3.StagedNetworkPolicy) error {
	if cacheItem == nil {
		log.Debug("Empty cache item, skipping sync to datastore")
		return nil
	}

	key := cacheItem.Name
	clog := log.WithField("key", key)
	clog.Debug("Synching to datastore")

	if len(cacheItem.Spec.Types) == 0 {
		// Cached item does not contain rules, return without error
		clog.WithField("stagedNetworkPolicy", key).Debugf("Skipping StagedNetworkPolicy update, cached item contains empty rules")
		return nil
	}

	namespace := cacheItem.Namespace

	// Lookup the item in the datastore
	datastoreSnp, err := pr.calico.StagedNetworkPolicies(namespace).Get(ctx, key, metav1.GetOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			clog.WithError(err).Info("Unexpected error querying StagedNetworkPolicies")
			// We hit an error other than "not found"
			return err
		}

		if pr.shouldSkipStoreUpdate(cacheItem, nil) {
			return nil
		}

		addCreationTimestamp(pr.clock, cacheItem)

		// Item does not exist in the datastore - create a new StagedNetworkPolicy
		clog.WithField("stagedNetworkPolicy", key).Info("Creating StagedNetworkPolicy in Calico datastore")
		if _, err = pr.calico.StagedNetworkPolicies(namespace).Create(ctx, cacheItem, metav1.CreateOptions{}); err != nil {
			clog.WithError(err).Infof("Error creating StagedNetworkPolicy in Calico datastore: %#v", cacheItem)
			return err
		}

		return nil
	}

	pr.reconcileCacheMeta(cacheItem, datastoreSnp)

	// Update the datastore to reflect the its latest state
	if !pr.shouldSkipStoreUpdate(cacheItem, datastoreSnp) && !equalSnps(datastoreSnp, cacheItem) {
		clog.WithField("stagedNetworkPolicy", key).Infof("Updating StagedNetworkPolicy in Calico datastore")
		// Copy over relevant items. This way we make sure that we don't update any unintended parameters
		copyStagedNetworkPolicy(datastoreSnp, *cacheItem)
		if _, err = pr.calico.StagedNetworkPolicies(namespace).Update(ctx, datastoreSnp, metav1.UpdateOptions{}); err != nil {
			clog.WithError(err).Infof("Error updating StagedNetworkPolicy in Calico datastore: %#v", datastoreSnp)
			return err
		}
	}

	return nil
}

// Utilities

func addCreationTimestamp(clock engine.Clock, snp *v3.StagedNetworkPolicy) {
	if clock == nil || snp == nil {
		log.Errorf("failed to update creation timestamp")
		return
	}
	snp.Annotations[calicoresources.CreationTimestampKey] = clock.NowRFC3339()
}

// copyStagedNetworkPolicy copies the StagedNetworkPolicy context that may be altered by the engine,
// from a source to a destination.
// Copy:
// - egress, ingress rules, and policy types
// - Name, and namespace
// - Labels, and annotations
func copyStagedNetworkPolicy(dest *v3.StagedNetworkPolicy, src v3.StagedNetworkPolicy) {
	// Copy egress, ingres and policy type over to the destination
	dest.Spec.Egress = make([]v3.Rule, len(src.Spec.Egress))
	copy(dest.Spec.Egress, src.Spec.Egress)
	dest.Spec.Ingress = make([]v3.Rule, len(src.Spec.Ingress))
	copy(dest.Spec.Ingress, src.Spec.Ingress)
	dest.Spec.Types = make([]v3.PolicyType, len(src.Spec.Types))
	copy(dest.Spec.Types, src.Spec.Types)

	// Copy ObjectMeta context. Context relevant to this controller is name, labels and annotation
	dest.ObjectMeta.Name = src.GetObjectMeta().GetName()
	dest.ObjectMeta.Namespace = src.GetObjectMeta().GetNamespace()

	dest.ObjectMeta.Labels = make(map[string]string)
	for key, label := range src.GetObjectMeta().GetLabels() {
		dest.ObjectMeta.Labels[key] = label
	}
	dest.ObjectMeta.Annotations = make(map[string]string)
	for key, annotation := range src.GetObjectMeta().GetAnnotations() {
		dest.ObjectMeta.Annotations[key] = annotation
	}
}

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
