// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package index

import (
	"github.com/olivere/elastic/v7"
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
)

func SingleIndexThreatfeedsIPSet() Helper {
	return ipsetIndexHelper{singleIndex: true}
}

func MultiIndexThreatfeedsIPSet() Helper {
	return ipsetIndexHelper{}
}

type ipsetIndexHelper struct {
	singleIndex bool
}

func (h ipsetIndexHelper) BaseQuery(i bapi.ClusterInfo, params v1.Params) (*elastic.BoolQuery, error) {
	return defaultBaseQuery(i, h.singleIndex, params)
}

func (h ipsetIndexHelper) NewSelectorQuery(selector string) (elastic.Query, error) {
	return nil, nil
}

func (h ipsetIndexHelper) NewRBACQuery(resources []apiv3.AuthorizedResourceVerbs) (elastic.Query, error) {
	return nil, nil
}

func (h ipsetIndexHelper) NewTimeRangeQuery(r *lmav1.TimeRange) elastic.Query {
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

func (h ipsetIndexHelper) GetTimeField() string {
	return "created_at"
}
