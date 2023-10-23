// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package index

import (
	"time"

	"github.com/olivere/elastic/v7"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

func SingleIndexComplianceReports() Helper {
	return complianceReportsIndexHelper{singleIndex: true}
}

func MultiIndexComplianceReports() Helper {
	return complianceReportsIndexHelper{}
}

type complianceReportsIndexHelper struct {
	singleIndex bool
}

func (h complianceReportsIndexHelper) BaseQuery(i bapi.ClusterInfo) *elastic.BoolQuery {
	q := elastic.NewBoolQuery()
	if h.singleIndex {
		q.Must(elastic.NewTermQuery("cluster", i.Cluster))
		if i.Tenant != "" {
			q.Must(elastic.NewTermQuery("tenant", i.Tenant))
		}
	}
	return q
}

func (h complianceReportsIndexHelper) NewSelectorQuery(selector string) (elastic.Query, error) {
	return nil, nil
}

func (h complianceReportsIndexHelper) NewRBACQuery(resources []apiv3.AuthorizedResourceVerbs) (elastic.Query, error) {
	return nil, nil
}

func (h complianceReportsIndexHelper) NewTimeRangeQuery(from, to time.Time) elastic.Query {
	unset := time.Time{}
	if from != unset && to != unset {
		return elastic.NewBoolQuery().Should(
			elastic.NewRangeQuery("startTime").From(from).To(to),
			elastic.NewRangeQuery("endTime").From(from).To(to),
		)
	} else if from != unset && to == unset {
		return elastic.NewRangeQuery("endTime").From(from)
	} else if from == unset && to != unset {
		return elastic.NewRangeQuery("startTime").To(to)
	}
	return nil
}

func (h complianceReportsIndexHelper) GetTimeField() string {
	return ""
}
