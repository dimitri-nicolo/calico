package servicegraph

import (
	"fmt"
	"sync"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	lmaelastic "github.com/tigera/lma/pkg/elastic"
)

// This file provides a cache interface for pre-correlated flows (those returned by the machinery in flow.go), filtered
// based on the users RBAC.
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
	TrafficStats         []v1.GraphTrafficStats
}

func (t TimeSeriesFlow) String() string {
	if t.AggregatedProtoPorts == nil {
		return fmt.Sprintf("L3Flow %s", t.Edge)
	}
	return fmt.Sprintf("L3Flow %s (%s)", t.Edge, t.AggregatedProtoPorts)
}

type FlowData struct {
	TimeIntervals []v1.TimeRange
	FilteredFlows []TimeSeriesFlow
	ServiceGroups ServiceGroups
}

type FlowCache interface {
	GetFilteredFlowData(indexL3, indexL7 string, tr v1.TimeRange, filter RBACFilter) (*FlowData, error)
}

func NewFlowCache(client lmaelastic.Client) FlowCache {
	return &flowCache{
		client: client,
	}
}

type flowCache struct {
	client lmaelastic.Client
}

func (fc *flowCache) GetFilteredFlowData(indexL3, indexL7 string, tr v1.TimeRange, filter RBACFilter) (*FlowData, error) {
	// At the moment there is no cache and only a single data point in the flow. Kick off the L3 and L7 queries at the
	// same time.
	wg := sync.WaitGroup{}
	var rawL3 []L3Flow
	var rawL7 []L7Flow
	var errL3, errL7 error

	wg.Add(2)
	go func() {
		rawL3, errL3 = GetRawL3FlowData(fc.client, indexL3, tr)
		wg.Done()
	}()
	go func() {
		rawL7, errL7 = GetRawL7FlowData(fc.client, indexL7, tr)
		wg.Done()
	}()
	wg.Wait()
	if errL3 != nil {
		return nil, errL3
	}
	if errL7 != nil {
		return nil, errL7
	}

	fd := &FlowData{
		TimeIntervals: []v1.TimeRange{tr},
		ServiceGroups: NewServiceGroups(),
	}

	// Filter the L3 flows based on RBAC. All other graph content is removed through graph pruning.
	for _, rf := range rawL3 {
		if !filter.IncludeFlow(rf.Edge) {
			continue
		}
		if rf.Edge.ServicePort != nil {
			fd.ServiceGroups.AddMapping(*rf.Edge.ServicePort, rf.Edge.Dest)
		}
		fd.FilteredFlows = append(fd.FilteredFlows, TimeSeriesFlow{
			Edge:                 rf.Edge,
			AggregatedProtoPorts: rf.AggregatedProtoPorts,
			TrafficStats: []v1.GraphTrafficStats{{
				L3: &rf.Stats,
			}},
		})
	}
	fd.ServiceGroups.FinishMappings()

	// Filter the L7 flows based on RBAC. All other graph content is removed through graph pruning.
	for _, rf := range rawL7 {
		if !filter.IncludeFlow(rf.Edge) {
			continue
		}
		fd.FilteredFlows = append(fd.FilteredFlows, TimeSeriesFlow{
			Edge: rf.Edge,
			TrafficStats: []v1.GraphTrafficStats{{
				L7: &rf.Stats,
			}},
		})
	}
	fd.ServiceGroups.FinishMappings()

	return fd, nil
}
