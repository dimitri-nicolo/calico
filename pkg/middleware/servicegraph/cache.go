// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"context"
	"fmt"
	"sync"

	"github.com/tigera/es-proxy/pkg/authorization"

	lmaelastic "github.com/tigera/lma/pkg/elastic"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
)

// This file provides a cache interface for pre-correlated flows, logs and events (those returned by the machinery in
// flowl3.go, flowl7.go and events.go). Flow data is filtered based on the users RBAC. Events data is not yet filtered
// by RBAC - instead we only overlay events onto correlated nodes that are in the final service graph response.
//
// The cache is not yet implemented, and so queries are made on demand to elastic.
//
// The tentative plan is to have the pre-correlated flows cached at 15 minute intervals so that the API can provide fast
// access times for this data. For more detailed data, the user would issue separate requests that would not be cached.
// An alternative cache approach might be to temporarily cache user-specific requests.  For anything beyond 5hrs (say)
// we could roll up the data at hourly intervals.  For anything beyond 1 day (say) we could roll up the data at daily
// intervals.  We probably want to have a mixture of cache mechanisms - fixed time intervals and user specific.  There
// will always be scenarios where the user will want to look at a very specific window - although arguably we could
// just return the time series covering that window, quantized into appropriate intervals.

type TimeSeriesFlow struct {
	Edge                 FlowEdge
	AggregatedProtoPorts *v1.AggregatedProtoPorts
	Stats                []v1.GraphStats
}

func (t TimeSeriesFlow) String() string {
	if t.AggregatedProtoPorts == nil {
		return fmt.Sprintf("L3Flow %s", t.Edge)
	}
	return fmt.Sprintf("L3Flow %s (%s)", t.Edge, t.AggregatedProtoPorts)
}

type ServiceGraphData struct {
	TimeIntervals  []v1.TimeRange
	FilteredFlows  []TimeSeriesFlow
	ServiceGroups  ServiceGroups
	Events         []Event
	HostnameHelper HostnameHelper
}

type ServiceGraphCache interface {
	GetFilteredServiceGraphData(ctx context.Context, rd *RequestData) (*ServiceGraphData, error)
}

func NewServiceGraphCache(elastic lmaelastic.Client) ServiceGraphCache {
	return &serviceGraphCache{
		elasticClient: elastic,
	}
}

type serviceGraphCache struct {
	elasticClient lmaelastic.Client
}

// GetFilteredServiceGraphData returns RBAC filtered service graph data:
// -  correlated (source/dest) flow logs and flow stats
// -  service groups calculated from flows
// -  event IDs correlated to endpoints
// TODO(rlb): The events are not RBAC filtered, instead events are overlaid onto the filtered graph view - so the
//            presence of a graph node or not is used to determine whether or not an event is included. This will likely
//            need to be revisited when we refine RBAC control of events.
func (fc *serviceGraphCache) GetFilteredServiceGraphData(ctx context.Context, rd *RequestData) (*ServiceGraphData, error) {
	// At the moment there is no cache and only a single data point in the flow. Kick off the L3 and L7 queries at the
	// same time.
	wg := sync.WaitGroup{}
	var flowConfig *FlowConfig
	var rawL3 []L3Flow
	var rawL7 []L7Flow
	var rawEvents []Event
	var hostnameHelper HostnameHelper
	var rbac RBACFilter
	var errFlowConfig, errL3, errL7, errEvents, errHostnameHelper, errRBAC error

	// We user the flow config as part of the L3 flow loading so get this before kicking off the various go
	// routines.
	flowConfig, errFlowConfig = GetFlowConfig(ctx, rd)
	if errFlowConfig != nil {
		return nil, errFlowConfig
	}

	// The other queries we can run in parallel:
	// - Get the RBAC filter
	// - Get the host name mapping helper
	// - Get the L3 logs
	// - Get the L7 logs
	// - Get the events
	wg.Add(1)
	go func() {
		defer wg.Done()
		rbac, errRBAC = GetRBACFilter(ctx, rd)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		// The name formatter is used to adjust names based on user supplied configuration.  We do this here rather
		// than updating the cached data.
		hostnameHelper, errHostnameHelper = GetHostnameHelper(ctx, rd)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		rawL3, errL3 = GetL3FlowData(ctx, fc.elasticClient, rd, flowConfig)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		rawL7, errL7 = GetL7FlowData(ctx, fc.elasticClient, rd)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		rawEvents, errEvents = GetEvents(ctx, fc.elasticClient, rd)
	}()
	wg.Wait()
	if errRBAC != nil {
		return nil, errRBAC
	} else if errHostnameHelper != nil {
		return nil, errHostnameHelper
	} else if errL3 != nil {
		return nil, errL3
	} else if errL7 != nil {
		return nil, errL7
	} else if errEvents != nil {
		return nil, errEvents
	}

	fd := &ServiceGraphData{
		TimeIntervals:  []v1.TimeRange{rd.request.TimeRange},
		ServiceGroups:  NewServiceGroups(),
		HostnameHelper: hostnameHelper,
	}

	// Filter the L3 flows based on RBAC. All other graph content is removed through graph pruning.
	for _, rf := range rawL3 {
		if !rbac.IncludeFlow(rf.Edge) {
			continue
		}

		// Update the names in the flow (if required).
		hostnameHelper.ProcessL3Flow(&rf)

		if rf.Edge.ServicePort != nil {
			fd.ServiceGroups.AddMapping(*rf.Edge.ServicePort, rf.Edge.Dest)
		}
		stats := rf.Stats

		fd.FilteredFlows = append(fd.FilteredFlows, TimeSeriesFlow{
			Edge:                 rf.Edge,
			AggregatedProtoPorts: rf.AggregatedProtoPorts,
			Stats: []v1.GraphStats{{
				L3:        &stats,
				Processes: rf.Processes,
			}},
		})
	}
	fd.ServiceGroups.FinishMappings()

	// Filter the L7 flows based on RBAC. All other graph content is removed through graph pruning.
	for _, rf := range rawL7 {
		if !rbac.IncludeFlow(rf.Edge) {
			continue
		}

		// Update the names in the flow (if required).
		hostnameHelper.ProcessL7Flow(&rf)

		stats := rf.Stats
		fd.FilteredFlows = append(fd.FilteredFlows, TimeSeriesFlow{
			Edge: rf.Edge,
			Stats: []v1.GraphStats{{
				L7: &stats,
			}},
		})
	}

	// Filter the events.
	for _, ev := range rawEvents {
		// Update the names in the events (if required).
		hostnameHelper.ProcessEvent(&ev)

		fd.Events = append(fd.Events, ev)
	}

	return fd, nil
}

// GetRBACFilter performs an authorization review and uses the response to construct an RBAC filter.
func GetRBACFilter(ctx context.Context, rd *RequestData) (RBACFilter, error) {
	verbs, err := authorization.PerformAuthorizationReview(ctx, rd.userCluster, authReviewAttrListEndpoints)
	if err != nil {
		return nil, err
	}
	return NewRBACFilterFromAuth(verbs), nil
}
