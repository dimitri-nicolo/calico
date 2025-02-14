// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package index

import (
	"time"

	"github.com/olivere/elastic/v7"
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
)

func SingleIndexThreatfeedsDomainNameSet() Helper {
	return domainSetIndexHelper{singleIndex: true}
}

func MultiIndexThreatfeedsDomainNameSet() Helper {
	return domainSetIndexHelper{}
}

// domainSetIndexHelper implements the Helper interface for flow logs.
type domainSetIndexHelper struct {
	singleIndex bool
}

func (h domainSetIndexHelper) BaseQuery(i bapi.ClusterInfo, params v1.Params) (*elastic.BoolQuery, error) {
	return defaultBaseQuery(i, h.singleIndex, params)
}

func (h domainSetIndexHelper) NewSelectorQuery(selector string) (elastic.Query, error) {
	return nil, nil
}

func (h domainSetIndexHelper) NewRBACQuery(resources []apiv3.AuthorizedResourceVerbs) (elastic.Query, error) {
	return nil, nil
}

func (h domainSetIndexHelper) NewTimeRangeQuery(r *lmav1.TimeRange) elastic.Query {
	timeField := GetTimeFieldForQuery(h, r)
	timeRangeQuery := elastic.NewRangeQuery(timeField)
	if timeField == "generated_time" {
		if !r.From.IsZero() {
			timeRangeQuery.Gt(r.From)
		}
		if !r.To.IsZero() {
			timeRangeQuery.Lte(r.To)
		}
		return timeRangeQuery
	}

	unset := time.Time{}
	if r.From != unset {
		timeRangeQuery.From(r.From)
	}
	if r.To != unset {
		timeRangeQuery.To(r.To)
	}
	return timeRangeQuery
}

func (h domainSetIndexHelper) GetTimeField() string {
	return "created_at"
}
