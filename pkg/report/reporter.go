// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package report

import (
	"context"

	"github.com/sirupsen/logrus"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/compliance/pkg/event"
	"github.com/tigera/compliance/pkg/list"
	"github.com/tigera/compliance/pkg/replay"
	"github.com/tigera/compliance/pkg/syncer"
	"github.com/tigera/compliance/pkg/xrefcache"
	"github.com/tigera/compliance/pkg/resources"
)

const (
	// A zero-trust exposure is indicated when any of these flags are *set* in the endpoint cache entry.
	ZeroTrustWhenEndpointFlagsSet =  xrefcache.CacheEntryInternetExposedIngress |
		xrefcache.CacheEntryInternetExposedEgress |
		xrefcache.CacheEntryOtherNamespaceExposedIngress |
		xrefcache.CacheEntryOtherNamespaceExposedEgress

	// A zero-trust exposure is indicated when any of these flags are *unset* in the endpoint cache entry.
	ZeroTrustWhenEndpointFlagsUnset = xrefcache.CacheEntryProtectedIngress |
		xrefcache.CacheEntryProtectedIngress |
		xrefcache.CacheEntryEnvoyEnabled

	// The full set of zero-trust flags for an endpoint.
	ZeroTrustFlags = ZeroTrustWhenEndpointFlagsSet | ZeroTrustWhenEndpointFlagsUnset



	// Valid for Policies, pods and host endpoints
	EventProtectedIngress syncer.UpdateType = 1 << iota
	EventProtectedEgress
	// Valid for network sets
	EventInternetExposed
	// Valid for Policies, pods and host endpoints
	EventInternetExposedIngress
	EventInternetExposedEgress
	// Valid for Policies, pods and host endpoints
	EventOtherNamespaceExposedIngress
	EventOtherNamespaceExposedEgress
	// Valid for pods
	EventEnvoyEnabled

	// ----- Non boolean configuration values -----
	// The following event flags do not have equivalent CacheEntry flags.
	EventPolicyRuleSelectorMatchStarted
	EventPolicyRuleSelectorMatchStopped
	EventPolicyMatchStarted
	EventPolicyMatchStopped
	EventNetsetMatchStarted
	EventNetsetMatchStopped
	EventServiceAdded
	EventServiceDeleted

	// ----- Added by the generic cache processing -----
	EventResourceAdded
	EventResourceModified
	EventResourceDeleted
	EventInScope

)

// Run is the entrypoint to start running the reporter.
func Run(
	ctx context.Context, cfg *Config,
	lister list.Destination,
	eventer event.Fetcher,
) error {

	// Create the cross-reference cache that we use to monitor for changes in the relevant data.
	xc := xrefcache.NewXrefCache()
	replayer := replay.New(cfg.Start, cfg.End, lister, eventer, xc)

	r := &reporter{
		ctx: ctx,
		cfg: cfg,
		clog: logrus.WithFields(logrus.Fields{
			"name":  cfg.Name,
			"type":  cfg.ReportType,
			"start": cfg.Start,
			"end":   cfg.End,
		}),
		eventer:  eventer,
		xc:       xc,
		replayer: replayer,
	}
	return r.run()
}

type reporter struct {
	ctx              context.Context
	cfg              *Config
	clog             *logrus.Entry
	listDest         list.Destination
	eventer          event.Fetcher
	xc               xrefcache.XrefCache
	replayer         syncer.Starter
	data             *apiv3.ReportData

	//TODO(rlb): Urgh this is truly terrible. We have different definitions of ResourceID between compliance and
	//TODO       the API.
	inScopeEndpoints map[resources.ResourceID]*reportEndpoint
	services         map[resources.ResourceID]xrefcache.CacheEntryFlags
	namespaces       map[string]xrefcache.CacheEntryFlags
}

type reportEndpoint struct {
	zeroTrustFlags xrefcache.CacheEntryFlags
	policies resources.Set
	services resources.Set
}

type reportService struct {
	zeroTrustFlags xrefcache.CacheEntryFlags
}

type reportNamespace struct {
	zeroTrustFlags xrefcache.CacheEntryFlags
}

func (r *reporter) run() error {
	if r.cfg.ReportType.Spec.IncludeEndpointData {
		// We need to include endpoint data in the report.
		r.clog.Debug("Including endpoint data in report")

		// Register the endpoint selectors to specify which endpoints we will receive notification for.
		if err := r.xc.RegisterInScopeEndpoints(r.cfg.Report.Spec.EndpointsSelection); err != nil {
			r.clog.WithError(err).Debug("Unable to register inscope endpoints selection")
			return nil
		}

		// Configure the x-ref cache to spit out the events that we care about (which is basically all the endpoints
		// flagged as "in-scope".
		for _, k := range xrefcache.KindsEndpoint {
			r.xc.RegisterOnUpdateHandler(k, xrefcache.EventInScope, r.onUpdate)
		}

		// Populate the report data from the replayer.
		r.replayer.Start(r.ctx)

		// Create the initial ReportData structure
		r.transferAggregatedData()

		if r.cfg.ReportType.Spec.IncludeEndpointFlowLogData {
			// We also need to include flow logs data for the in-scope endpoints.
			r.clog.Debug("Including flow log data in report")
		}
	}

	if r.cfg.ReportType.Spec.AuditEventsSelection != nil {
		// We need to include audit log data in the report.
		r.clog.Debug("Including audit event data in report")
	}

	// Store report data

	return nil
}

func (r *reporter) onUpdate(update syncer.Update) {
	if update.Type & xrefcache.EventResourceDeleted != 0 {
		// We don't need to track deleted endpoints because we are getting the superset of resources managed within the
		// timeframe.
		return
	}
	x := update.Resource.(*xrefcache.CacheEntryEndpoint)
	ep := r.getEndpoint(update.ResourceID)
	zeroTrustFlags, _ := zeroTrustFlags(update.Type, x.Flags)

	// Update the endpoint and namespaces policies and services
	//TODO(rlb): Performance improvement here - we only need to update what has actually changed in particular no
	//           need to update policies or services set if not changed. However, I have not UTd that the correct
	//           flags are returned, so let's not trust to luck.
	ep.zeroTrustFlags |= zeroTrustFlags
	ep.policies.AddSet(x.AppliedPolicies)
	ep.services.AddSet(x.Services)

	// Loop through and update the flags on the services.
	ep.services.Iter(func(item resources.ResourceID) error {
		r.services[item] |= zeroTrustFlags
		return nil
	})

	// Update the namespace flags.
	r.namespaces[update.ResourceID.Namespace] |= zeroTrustFlags
}

func (r *reporter) getEndpoint(id resources.ResourceID) *reportEndpoint {
	re := r.inScopeEndpoints[id]
	if re == nil {
		re = &reportEndpoint{
			policies: resources.NewSet(),
			services: resources.NewSet(),
		}
		r.inScopeEndpoints[id] = re
	}
	return re
}

func (r *reporter) transferAggregatedData() {
	// Transfer the aggregated data to the ReportData structure.
	for id, ep := range r.inScopeEndpoints {
		epd := apiv3.EndpointsReportEndpoint{
			ID: apiv3.ResourceID{
				TypeMeta:,
			},
			IngressProtected: ep.zeroTrustFlags & xrefcache.CacheEntryProtectedIngress == 0, // We reversed this for zero-trust
			EgressProtected: ep.zeroTrustFlags & xrefcache.CacheEntryProtectedEgress == 0, // We reversed this for zero-trust
			IngressFromInternet: ep.zeroTrustFlags & xrefcache.CacheEntryInternetExposedIngress != 0,
			EgressToInternet: ep.zeroTrustFlags & xrefcache.CacheEntryInternetExposedEgress != 0,
			IngressFromOtherNamespace: ep.zeroTrustFlags & xrefcache.CacheEntryOtherNamespaceExposedIngress != 0,
			EgressToOtherNamespace: ep.zeroTrustFlags & xrefcache.CacheEntryOtherNamespaceExposedEgress != 0,
			EnvoyEnabled: ep.zeroTrustFlags & xrefcache.CacheEntryEnvoyEnabled == 0, // We reversed this for zero-trust
			AppliedPolicies: ep.policies.ToSlice(),
			Services: ep.services.ToSlice(),
		}
	}
}

// zeroTrustFlags converts the flags and updates into a set of zero-trust flags and changed zero-trust flags.
// The zero trust flags map on to the cache entry flags, but reverse the bit for the flags whose "unset" value indicates
// zero trust.
func zeroTrustFlags(updateType syncer.UpdateType, flags xrefcache.CacheEntryFlags) (allZeroTrust, changedZeroTrust xrefcache.CacheEntryFlags) {
	// The updateType flags correspond directly with the cache entry flags so we can perform bitwise manipulation on them
	// to check for updates.

	// Get the set of changed flags (update masked with the flags of interest). One alteration though, for an add we need
	// to treat as if all fields changed.
	changedFlags := xrefcache.CacheEntryFlags(updateType) & ZeroTrustFlags
	if updateType & xrefcache.EventResourceAdded != 0 {
		changedFlags = ZeroTrustFlags
	}

	// Calculate the corresponding set of flags that indicate a zero-trust exposure.
	zeroTrust := (flags ^ ZeroTrustWhenEndpointFlagsUnset) & ZeroTrustFlags

	return zeroTrust, zeroTrust & changedFlags
}

