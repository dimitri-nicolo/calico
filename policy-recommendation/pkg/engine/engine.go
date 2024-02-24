// Copyright (c) 2024 Tigera Inc. All rights reserved.
package engine

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/types"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"

	rcache "github.com/projectcalico/calico/kube-controllers/pkg/cache"
	libcselector "github.com/projectcalico/calico/libcalico-go/lib/selector"
	"github.com/projectcalico/calico/libcalico-go/lib/set"
	linseed "github.com/projectcalico/calico/linseed/pkg/client"
	calicores "github.com/projectcalico/calico/policy-recommendation/pkg/calico-resources"
	"github.com/projectcalico/calico/policy-recommendation/pkg/flows"
	poltypes "github.com/projectcalico/calico/policy-recommendation/pkg/types"
	"github.com/projectcalico/calico/policy-recommendation/utils"
)

const (
	// DefaultStabilizationPeriod is the default stabilization period.
	defaultStabilizationPeriod = 5 * time.Minute

	// DefaultInterval is the default recommendation interval.
	defaultInterval = 10 * time.Minute

	// DefaultLookback is the default lookback period.
	defaultLookback = 24 * time.Hour

	// DefaultSelector is the default namespace selector.
	defaultSelector = `(!projectcalico.org/name starts with "tigera-" && ` +
		`!projectcalico.org/name starts with "calico-" && ` +
		`!projectcalico.org/name starts with "kube-" && ` +
		`!projectcalico.org/name starts with "openshift-")`
)

// Clock is an interface added for testing purposes.
type Clock interface {
	NowRFC3339() string
}

type recommendationScope struct {
	// initialLookback is the flow log query lookback period for the first run of the engine.
	initialLookback time.Duration

	// interval is the engine run interval.
	interval time.Duration

	// stabilization is the period used to determine if a recommendation is stable.
	stabilization time.Duration

	// selector is the logical expression used to select namespaces for processing.
	selector libcselector.Selector

	// passIntraNamespaceTraffic is a flag to allow/pass intra-namespace traffic.
	passIntraNamespaceTraffic bool

	// Metadata
	uid types.UID
}

type RecommendationEngine struct {
	// Cache for storing the recommendations (SNPs)
	cache rcache.ResourceCache

	// ProcessedNamespaces to process and recommend policies for
	ProcessedNamespaces set.Set[string]

	// Channel for receiving PolicyRecommendationScope updates
	UpdateChannel chan v3.PolicyRecommendationScope

	// Context for the engine
	ctx context.Context

	// Calico client
	calico calicoclient.ProjectcalicoV3Interface

	// Linseed client
	linseedClient linseed.Client

	// Cluster name
	cluster string

	// Engine scope
	scope *recommendationScope

	// Clock for setting the latest update timestamp
	clock Clock

	// Cluster domain
	clusterDomain string

	// Query for querying flows logs
	query flows.PolicyRecommendationQuery

	// Logger
	clog *log.Entry
}

// NewRecommendationEngine returns a new RecommendationEngine struct.
func NewRecommendationEngine(
	ctx context.Context,
	clusterID string,
	calico calicoclient.ProjectcalicoV3Interface,
	linseedClient linseed.Client,
	namespaces set.Set[string],
	query flows.PolicyRecommendationQuery,
	cache rcache.ResourceCache,
	scope *v3.PolicyRecommendationScope,
	clock Clock,
) *RecommendationEngine {
	logEntry := log.WithField("cluster", clusterID)
	if clusterID == "cluster" {
		logEntry = log.WithField("cluster", "management")
	}
	logEntry.Info("Creating engine")

	clusterDomain, err := utils.GetClusterDomain(utils.DefaultResolveConfPath)
	if err != nil {
		clusterDomain = utils.DefaultClusterDomain
		log.WithError(err).Warningf("Defaulting cluster domain to %s", clusterDomain)
	}

	// Create a new scope with the default values.
	parsedSelector, _ := libcselector.Parse(defaultSelector)
	newScope := &recommendationScope{
		initialLookback:           defaultLookback,            // 24h0m0s
		interval:                  defaultInterval,            // 10m0s
		stabilization:             defaultStabilizationPeriod, // 5m0s
		selector:                  parsedSelector,             // Exclude Tigera and Kubernetes and Calico namespaces
		passIntraNamespaceTraffic: false,                      // Allow intra-namespace traffic
	}
	// Update the scope with the values from the incoming PolicyRecommendationScope.
	updateScope(logEntry, newScope, *scope)

	return &RecommendationEngine{
		ctx:                 ctx,
		calico:              calico,
		linseedClient:       linseedClient,
		cache:               cache,
		ProcessedNamespaces: namespaces,
		cluster:             clusterID,
		scope:               newScope,
		UpdateChannel:       make(chan v3.PolicyRecommendationScope),
		clock:               clock,
		clusterDomain:       clusterDomain,
		query:               query,
		clog:                logEntry,
	}
}

// Run starts the engine. It runs the engine loop and processes the recommendations. It also updates
// the engine scope with the latest PolicyRecommendationScopeSpec. It stops the engine when the
// stopChan is closed.
func (e *RecommendationEngine) Run(stopChan chan struct{}) {
	ticker := time.NewTicker(e.scope.interval)
	defer ticker.Stop()

	for {
		select {
		case update, ok := <-e.UpdateChannel:
			if !ok {
				continue // Channel closed, exit the loop
			}
			interval := e.scope.interval
			if ticker == nil {
				// Start the ticker with the default interval.
				ticker = time.NewTicker(interval)
			}
			e.clog.Debugf("[Consumer] Received scope update: %+v", update)
			// Update the engine scope with the new PolicyRecommendationScopeSpec.
			updateScope(e.clog, e.scope, update)
			if interval != e.scope.interval {
				// The interval has changed, update the ticker with the new interval.
				// Stop the previous ticker and start a new one with the updated interval.
				ticker.Stop()
				ticker.C = time.NewTicker(e.scope.interval).C
				e.clog.Debugf("Updated ticker with new interval %s", e.scope.interval.String())
			}
		case <-ticker.C:
			e.clog.Debug("Running engine")

			if e.cache == nil {
				e.clog.Warn("Cache is not set, avoiding engine run")
				continue
			}

			// Iterate through the namespaces and process the recommendations (SNPs). Add each
			// new/updated recommendation to the cache for reconciliation with datastore.
			e.ProcessedNamespaces.Iter(func(namespace string) error {
				e.clog.WithField("namespace", namespace).Debug("Processing")
				var recommendation *v3.StagedNetworkPolicy
				if item, found := e.cache.Get(namespace); found {
					var ok bool
					value, ok := item.(v3.StagedNetworkPolicy)
					if !ok {
						e.clog.Warnf("unexpected item in cache: %+v", item)
						return nil
					}
					recommendation = &value
				} else {
					// Create a new recommendation for this namespace. The recommendation is a
					// StagedNetworkPolicy. This will only be used if there are new rules to add. Otherwise,
					// the recommendation will be discarded.
					recommendation = calicores.NewStagedNetworkPolicy(
						utils.GenerateRecommendationName(poltypes.PolicyRecommendationTierName, namespace, utils.SuffixGenerator),
						namespace,
						poltypes.PolicyRecommendationTierName,
						e.scope.uid,
					)
				}
				if e.update(recommendation) {
					// The recommendation contains new rules, or status metadata has been updated so add to
					// cache for syncing.
					e.cache.Set(namespace, *recommendation)
					e.clog.WithField("name", recommendation.Name).Debug("Add/Update cache item")
				}

				return nil
			})
		case <-stopChan:
			e.clog.Info("Received stop signal, stopping engine")
			return
		}
	}
}

// GetScope returns the engine scope.
func (e *RecommendationEngine) GetScope() *recommendationScope {
	return e.scope
}

// update processes the flows logs into new rules and adds them to the recommendation. Returns true
// if there is an update to recommendation (SNP).
func (e *RecommendationEngine) update(snp *v3.StagedNetworkPolicy) bool {
	if snp == nil {
		e.clog.Debug("Empty staged network policy")
		return false
	}
	if snp.Spec.StagedAction != v3.StagedActionLearn {
		// Skip this recommendation, the engine only processes "Learn" recommendations.
		e.clog.WithField("recommendation", snp.Name).Debug("Ignoring recommendation, staged action is not learning")
		return false
	}
	// Query flows logs for the namespace
	params := flows.NewRecommendationFlowLogQueryParams(e.getLookback(*snp), snp.Namespace, e.cluster)
	flows, err := e.query.QueryFlows(params)
	if err != nil {
		e.clog.WithError(err).WithField("params", params).Warning("Failed to query flows logs")
		return false
	}
	// New flow logs were found, process and sort them into the existing rules in the policy.
	// If the rules have changed, update the recommendation. If the rules have not changed, then there
	// still may be a status update to process.
	rec := newRecommendation(
		e.cluster,
		snp.Namespace,
		e.scope.interval,
		e.scope.stabilization,
		e.scope.passIntraNamespaceTraffic,
		utils.GetServiceNameSuffix(e.clusterDomain),
		snp,
		e.clock,
	)

	if len(flows) == 0 {
		e.clog.WithField("params", params).Debug("No matching flows logs found")
		// No matching flows found, however we may still want to update the status
		return rec.updateStatus(snp)
	}

	// Return true if the recommendation has been updated, by adding new rules or updating the status.
	return rec.update(flows, snp)
}

// getLookback returns the InitialLookback period if the policy is new and has not previously
// been updated, otherwise use twice the engine-run interval (Default: 2.5min).
func (e *RecommendationEngine) getLookback(snp v3.StagedNetworkPolicy) time.Duration {
	initialLookback := defaultLookback
	interval := defaultInterval
	if e.scope != nil {
		if e.scope.initialLookback != 0 {
			initialLookback = e.scope.initialLookback
		}
		if e.scope.interval != 0 {
			interval = e.scope.interval
		}
	}

	_, ok := snp.Annotations[calicores.LastUpdatedKey]
	if !ok {
		// First time run will use the initial lookback
		return initialLookback
	}
	// Twice the engine-run interval
	lookback := interval * 2

	return lookback
}

// updateScope updates the engine scope with the new PolicyRecommendationScopeSpec.
func updateScope(clog *log.Entry, scope *recommendationScope, update v3.PolicyRecommendationScope) {
	if update.Spec.Interval != nil && scope.interval != update.Spec.Interval.Duration {
		scope.interval = update.Spec.Interval.Duration
		clog.Infof("[Consumer] Setting new interval to: %s", scope.interval.String())
	}

	if update.Spec.InitialLookback != nil && scope.initialLookback != update.Spec.InitialLookback.Duration {
		scope.initialLookback = update.Spec.InitialLookback.Duration
		clog.Infof("[Consumer] Setting new initial lookback to: %s", scope.initialLookback.String())
	}

	if update.Spec.StabilizationPeriod != nil && scope.stabilization != update.Spec.StabilizationPeriod.Duration {
		scope.stabilization = update.Spec.StabilizationPeriod.Duration
		clog.Infof("[Consumer] Setting new stabilization to: %s", scope.stabilization.String())
	}

	if scope.passIntraNamespaceTraffic != update.Spec.NamespaceSpec.IntraNamespacePassThroughTraffic {
		scope.passIntraNamespaceTraffic = update.Spec.NamespaceSpec.IntraNamespacePassThroughTraffic
		clog.Infof("[Consumer] Setting passIntraNamespaceTraffic to: %t", scope.passIntraNamespaceTraffic)
	}

	if update.Spec.NamespaceSpec.Selector != "" && scope.selector.String() != update.Spec.NamespaceSpec.Selector {
		parsedSelector, err := libcselector.Parse(update.Spec.NamespaceSpec.Selector)
		if err != nil {
			clog.WithError(err).Errorf("failed to parse selector: %s", update.Spec.NamespaceSpec.Selector)
		} else {
			if scope.selector.String() != parsedSelector.String() {
				scope.selector = parsedSelector
				clog.Infof("[Consumer] Setting new namespace selector to: %s", parsedSelector.String())
			}
		}
	}

	if update.UID != scope.uid {
		scope.uid = update.UID
		clog.Infof("[Consumer] Setting recommendation owner UID to: %s", scope.uid)
	}
}

// Defined for testing purposes.

// GetInterval returns the engine interval.
func (s *recommendationScope) GetInterval() time.Duration {
	return s.interval
}

// GetInitialLookback returns the engine initial lookback.
func (s *recommendationScope) GetInitialLookback() time.Duration {
	return s.initialLookback
}

// GetStabilization returns the engine stabilization period.
func (s *recommendationScope) GetStabilization() time.Duration {
	return s.stabilization
}

// GetSelector returns the engine namespace selector.
func (s *recommendationScope) GetSelector() libcselector.Selector {
	return s.selector
}

// GetPassIntraNamespaceTraffic returns the engine passIntraNamespaceTraffic flag.
func (s *recommendationScope) GetPassIntraNamespaceTraffic() bool {
	return s.passIntraNamespaceTraffic
}
