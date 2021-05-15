// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"context"
	"fmt"
	"sync"

	lmaelastic "github.com/tigera/lma/pkg/elastic"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	"github.com/tigera/es-proxy/pkg/middleware/k8s"
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
	TimeIntervals []v1.TimeRange
	FilteredFlows []TimeSeriesFlow
	ServiceGroups ServiceGroups
	EventIDs      []EventID
}

type ServiceGraphCache interface {
	GetFilteredServiceGraphData(
		ctx context.Context, cluster string, sgr *v1.ServiceGraphRequest, rbacFilter RBACFilter,
	) (*ServiceGraphData, error)
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
func (fc *serviceGraphCache) GetFilteredServiceGraphData(
	ctx context.Context, cluster string, sgr *v1.ServiceGraphRequest, rbacFilter RBACFilter,
) (*ServiceGraphData, error) {
	// At the moment there is no cache and only a single data point in the flow. Kick off the L3 and L7 queries at the
	// same time.
	wg := sync.WaitGroup{}
	var flowConfig *FlowConfig
	var rawL3 []L3Flow
	var rawL7 []L7Flow
	var rawEvents []EventID
	var nameFormatter *NameFormatter
	var errFlowConfig, errL3, errL7, errEvents, errNameFormatter error

	// Extract the cluster specific k8s client.
	k8sClient := k8s.GetClientSetApplicationFromContext(ctx)

	flowConfig, errFlowConfig = GetFlowConfig(ctx, k8sClient)
	if errFlowConfig != nil {
		return nil, errFlowConfig
	}

	wg.Add(1)
	go func() {
		rawL3, errL3 = GetL3FlowData(ctx, fc.elasticClient, cluster, sgr.TimeRange, flowConfig)
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		rawL7, errL7 = GetL7FlowData(ctx, fc.elasticClient, cluster, sgr.TimeRange)
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		rawEvents, errEvents = GetEventIDs(ctx, fc.elasticClient, cluster, sgr.TimeRange)
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		// The name formatter is used to adjust names based on user supplied configuration.  We do this here rather
		// than updating the cached data.
		nameFormatter, errNameFormatter = GetNameFormatter(ctx, k8sClient, sgr.SelectedView)
		wg.Done()
	}()
	wg.Wait()
	if errL3 != nil {
		return nil, errL3
	}
	if errL7 != nil {
		return nil, errL7
	}
	if errEvents != nil {
		return nil, errEvents
	}
	if errNameFormatter != nil {
		return nil, errNameFormatter
	}

	fd := &ServiceGraphData{
		TimeIntervals: []v1.TimeRange{sgr.TimeRange},
		ServiceGroups: NewServiceGroups(),
	}

	// Filter the L3 flows based on RBAC. All other graph content is removed through graph pruning.
	for _, rf := range rawL3 {
		if !rbacFilter.IncludeFlow(rf.Edge) {
			continue
		}

		// Update the names in the flow (if required).
		nameFormatter.UpdateL3Flow(&rf)

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
		if !rbacFilter.IncludeFlow(rf.Edge) {
			continue
		}

		// Update the names in the flow (if required).
		nameFormatter.UpdateL7Flow(&rf)

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
		nameFormatter.UpdateEvent(&ev)

		fd.EventIDs = append(fd.EventIDs, ev)
	}

	return fd, nil
}
