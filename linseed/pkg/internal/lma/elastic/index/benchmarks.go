// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package index

import (
	"github.com/olivere/elastic/v7"
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
)

// FlowLogs returns an instance of the flow logs index helper that uses a single index.
func SingleIndexBenchmarks() Helper {
	return benchmarksIndexHelper{singleIndex: true}
}

// LegacyFlowLogs returns an instance of the flow logs index helper.
func MultiIndexBenchmarks() Helper {
	return benchmarksIndexHelper{}
}

// benchmarksIndexHelper implements the Helper interface for flow logs.
type benchmarksIndexHelper struct {
	singleIndex bool
}

func (h benchmarksIndexHelper) BaseQuery(i bapi.ClusterInfo, params v1.Params) (*elastic.BoolQuery, error) {
	return defaultBaseQuery(i, h.singleIndex, params)
}

func (h benchmarksIndexHelper) NewSelectorQuery(selector string) (elastic.Query, error) {
	return nil, nil
}

func (h benchmarksIndexHelper) NewRBACQuery(resources []apiv3.AuthorizedResourceVerbs) (elastic.Query, error) {
	return nil, nil
}

func (h benchmarksIndexHelper) NewTimeRangeQuery(r *lmav1.TimeRange) elastic.Query {
	timeField := GetTimeFieldForQuery(h, r)
	timeRangeQuery := elastic.NewRangeQuery(timeField)
	switch timeField {
	case "generated_time":
		return processGeneratedField(r, timeRangeQuery)
	default:
		// Any query that targets the default time field will target value higher than the start
		// and lower than the end of the interval
		if !r.From.IsZero() {
			timeRangeQuery.From(r.From)
		}
		if !r.To.IsZero() {
			timeRangeQuery.To(r.To)
		}
		return timeRangeQuery
	}
}

func (h benchmarksIndexHelper) GetTimeField() string {
	return "timestamp"
}
