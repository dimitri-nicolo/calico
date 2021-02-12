package servicegraph

import (
	"fmt"

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
		return fmt.Sprintf("Flow %s", t.Edge)
	}
	return fmt.Sprintf("Flow %s (%s)", t.Edge, t.AggregatedProtoPorts)
}

type FlowData struct {
	TimeIntervals []v1.TimeRange
	FilteredFlows []TimeSeriesFlow
	ServiceGroups ServiceGroups
}

type FlowCache interface {
	GetFilteredFlowData(index string, tr v1.TimeRange, timeSeries bool, filter RBACFilter) (*FlowData, error)
}

func NewFlowCache(client lmaelastic.Client) FlowCache {
	return &flowCache{
		client: client,
	}
}

type flowCache struct {
	client lmaelastic.Client
}

func (fc *flowCache) GetFilteredFlowData(index string, tr v1.TimeRange, timeSeries bool, filter RBACFilter) (*FlowData, error) {
	// At the moment there is no cache and only a single data point in the flow.
	raw, err := GetRawFlowData(fc.client, index, tr)
	if err != nil {
		return nil, err
	}

	fd := &FlowData{
		TimeIntervals: []v1.TimeRange{tr},
		ServiceGroups: NewServiceGroups(),
	}

	// Filter the flows based on RBAC. All other graph content is removed through graph pruning.
	for _, rf := range raw {
		if !filter.IncludeFlow(rf.Edge) {
			continue
		}
		if rf.Edge.ServicePort != nil {
			fd.ServiceGroups.AddMapping(*rf.Edge.ServicePort, rf.Edge.Dest)
		}
		fd.FilteredFlows = append(fd.FilteredFlows, TimeSeriesFlow{
			Edge:                 rf.Edge,
			AggregatedProtoPorts: rf.AggregatedProtoPorts,
			TrafficStats: []v1.GraphTrafficStats{
				rf.Stats,
			},
		})
	}
	fd.ServiceGroups.FinishMappings()

	return fd, nil
}
