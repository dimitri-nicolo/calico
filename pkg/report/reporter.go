// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package report

import (
	"context"

	"github.com/sirupsen/logrus"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/compliance/pkg/event"
	"github.com/tigera/compliance/pkg/list"
	"github.com/tigera/compliance/pkg/replay"
	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
	"github.com/tigera/compliance/pkg/xrefcache"
)

const (
	// A zero-trust exposure is indicated when any of these flags are *set* in the endpoint cache entry.
	ZeroTrustWhenEndpointFlagsSet = xrefcache.CacheEntryInternetExposedIngress |
		xrefcache.CacheEntryInternetExposedEgress |
		xrefcache.CacheEntryOtherNamespaceExposedIngress |
		xrefcache.CacheEntryOtherNamespaceExposedEgress

	// A zero-trust exposure is indicated when any of these flags are *unset* in the endpoint cache entry.
	ZeroTrustWhenEndpointFlagsUnset = xrefcache.CacheEntryProtectedIngress |
		xrefcache.CacheEntryProtectedEgress |
		xrefcache.CacheEntryEnvoyEnabled

	// The full set of zero-trust flags for an endpoint.
	ZeroTrustFlags = ZeroTrustWhenEndpointFlagsSet | ZeroTrustWhenEndpointFlagsUnset
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
	ctx      context.Context
	cfg      *Config
	clog     *logrus.Entry
	listDest list.Destination
	eventer  event.Fetcher
	xc       xrefcache.XrefCache
	replayer syncer.Starter

	// Consolidate the tracked in-scope endpoint events into a local cache, which will get converted and copied into
	// the report data structure.
	inScopeEndpoints map[apiv3.ResourceID]*reportEndpoint
	services         map[apiv3.ResourceID]xrefcache.CacheEntryFlags
	namespaces       map[string]xrefcache.CacheEntryFlags
	data             *apiv3.ReportData
}

type reportEndpoint struct {
	zeroTrustFlags xrefcache.CacheEntryFlags
	policies       resources.Set
	services       resources.Set
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
	if update.Type&xrefcache.EventResourceDeleted != 0 {
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
	//           update flags are returned, so let's not trust to luck.
	ep.zeroTrustFlags |= zeroTrustFlags
	ep.policies.AddSet(x.AppliedPolicies)
	ep.services.AddSet(x.Services)

	// Loop through and update the flags on the services.
	ep.services.Iter(func(item apiv3.ResourceID) error {
		r.services[item] |= zeroTrustFlags
		return nil
	})

	// Update the namespace flags.
	r.namespaces[update.ResourceID.Namespace] |= zeroTrustFlags
}

func (r *reporter) getEndpoint(id apiv3.ResourceID) *reportEndpoint {
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
	// Create the endpoints slice up-front
	r.data.Endpoints = make([]apiv3.EndpointsReportEndpoint, len(r.inScopeEndpoints))

	// Transfer the aggregated data to the ReportData structure. Start with endpoints.
	for id, ep := range r.inScopeEndpoints {
		r.data.Endpoints = append(r.data.Endpoints, apiv3.EndpointsReportEndpoint{
			ID:                        id,
			IngressProtected:          ep.zeroTrustFlags&xrefcache.CacheEntryProtectedIngress == 0, // We reversed this for zero-trust
			EgressProtected:           ep.zeroTrustFlags&xrefcache.CacheEntryProtectedEgress == 0,  // We reversed this for zero-trust
			IngressFromInternet:       ep.zeroTrustFlags&xrefcache.CacheEntryInternetExposedIngress != 0,
			EgressToInternet:          ep.zeroTrustFlags&xrefcache.CacheEntryInternetExposedEgress != 0,
			IngressFromOtherNamespace: ep.zeroTrustFlags&xrefcache.CacheEntryOtherNamespaceExposedIngress != 0,
			EgressToOtherNamespace:    ep.zeroTrustFlags&xrefcache.CacheEntryOtherNamespaceExposedEgress != 0,
			EnvoyEnabled:              ep.zeroTrustFlags&xrefcache.CacheEntryEnvoyEnabled == 0, // We reversed this for zero-trust
			AppliedPolicies:           ep.policies.ToSlice(),
			Services:                  ep.services.ToSlice(),
		})

		// Delete from our dictionary now.
		delete(r.inScopeEndpoints, id)
	}

	// We can delete the dictionary now.
	r.inScopeEndpoints = nil

	// Now handle namespaces.
	for name, zeroTrustFlags := range r.namespaces {
		r.data.Namespaces = append(r.data.Namespaces, apiv3.EndpointsReportNamespace{
			Namespace: apiv3.ResourceID{
				TypeMeta: resources.TypeK8sNamespaces,
				Name:     name,
			},
			IngressProtected:          zeroTrustFlags&xrefcache.CacheEntryProtectedIngress == 0, // We reversed this for zero-trust
			EgressProtected:           zeroTrustFlags&xrefcache.CacheEntryProtectedEgress == 0,  // We reversed this for zero-trust
			IngressFromInternet:       zeroTrustFlags&xrefcache.CacheEntryInternetExposedIngress != 0,
			EgressToInternet:          zeroTrustFlags&xrefcache.CacheEntryInternetExposedEgress != 0,
			IngressFromOtherNamespace: zeroTrustFlags&xrefcache.CacheEntryOtherNamespaceExposedIngress != 0,
			EgressToOtherNamespace:    zeroTrustFlags&xrefcache.CacheEntryOtherNamespaceExposedEgress != 0,
			EnvoyEnabled:              zeroTrustFlags&xrefcache.CacheEntryEnvoyEnabled == 0, // We reversed this for zero-trust
		})

		// Delete from our dictionary now.
		delete(r.namespaces, name)
	}

	// We can delete the dictionary now.
	r.namespaces = nil

	// Now handle services.
	for id, zeroTrustFlags := range r.services {
		r.data.Services = append(r.data.Services, apiv3.EndpointsReportService{
			Service:                   id,
			IngressProtected:          zeroTrustFlags&xrefcache.CacheEntryProtectedIngress == 0, // We reversed this for zero-trust
			IngressFromInternet:       zeroTrustFlags&xrefcache.CacheEntryInternetExposedIngress != 0,
			IngressFromOtherNamespace: zeroTrustFlags&xrefcache.CacheEntryOtherNamespaceExposedIngress != 0,
			EnvoyEnabled:              zeroTrustFlags&xrefcache.CacheEntryEnvoyEnabled == 0, // We reversed this for zero-trust
		})

		// Delete from our dictionary now.
		delete(r.services, id)
	}

	// We can delete the dictionary now.
	r.services = nil
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
	if updateType&xrefcache.EventResourceAdded != 0 {
		changedFlags = ZeroTrustFlags
	}

	// Calculate the corresponding set of flags that indicate a zero-trust exposure.
	zeroTrust := (flags ^ ZeroTrustWhenEndpointFlagsUnset) & ZeroTrustFlags

	return zeroTrust, zeroTrust & changedFlags
}
