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
func SingleIndexBGPLogs() Helper {
	return bgpLogsIndexHelper{singleIndex: true}
}

// LegacyFlowLogs returns an instance of the flow logs index helper.
func MultiIndexBGPLogs() Helper {
	return bgpLogsIndexHelper{}
}

// bgpLogsIndexHelper implements the Helper interface for flow logs.
type bgpLogsIndexHelper struct {
	singleIndex bool
}

func (h bgpLogsIndexHelper) BaseQuery(i bapi.ClusterInfo, params v1.Params) (*elastic.BoolQuery, error) {
	return defaultBaseQuery(i, h.singleIndex, params)
}

func (h bgpLogsIndexHelper) NewSelectorQuery(selector string) (elastic.Query, error) {
	return nil, nil
}

func (h bgpLogsIndexHelper) NewRBACQuery(resources []apiv3.AuthorizedResourceVerbs) (elastic.Query, error) {
	return nil, nil
}

func (h bgpLogsIndexHelper) NewTimeRangeQuery(r *lmav1.TimeRange) elastic.Query {
	timeField := GetTimeFieldForQuery(h, r)
	timeRangeQuery := elastic.NewRangeQuery(timeField)
	switch timeField {
	case "generated_time":
		return processGeneratedField(r, timeRangeQuery)
	default:
		// Any query that targets the default field requires further processing
		// and assumes we have defaults for both start and end of the interval.
		// This query will target any value that is higher that the start, but lower or
		// equal to the end of the interval
		from := r.From.Format(v1.BGPLogTimeFormat)
		to := r.To.Format(v1.BGPLogTimeFormat)
		return timeRangeQuery.Gt(from).Lte(to)
	}
}

func (h bgpLogsIndexHelper) GetTimeField() string {
	return "logtime"
}
