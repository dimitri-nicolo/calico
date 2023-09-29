// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package index

import (
	"time"

	"github.com/olivere/elastic/v7"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
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

func (h ipsetIndexHelper) BaseQuery(i bapi.ClusterInfo) *elastic.BoolQuery {
	q := elastic.NewBoolQuery()
	if h.singleIndex {
		q.Must(elastic.NewTermQuery("cluster", i.Cluster))
		if i.Tenant != "" {
			q.Must(elastic.NewTermQuery("tenant", i.Tenant))
		}
	}
	return q
}

func (h ipsetIndexHelper) NewSelectorQuery(selector string) (elastic.Query, error) {
	return nil, nil
}

func (h ipsetIndexHelper) NewRBACQuery(resources []apiv3.AuthorizedResourceVerbs) (elastic.Query, error) {
	return nil, nil
}

func (h ipsetIndexHelper) NewTimeRangeQuery(from, to time.Time) elastic.Query {
	unset := time.Time{}
	tr := elastic.NewRangeQuery("created_at")
	if from != unset {
		tr.From(from)
	}
	if to != unset {
		tr.To(to)
	}
	return tr
}

func (h ipsetIndexHelper) GetTimeField() string {
	return "created_at"
}
