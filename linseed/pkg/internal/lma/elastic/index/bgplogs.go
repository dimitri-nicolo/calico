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

func (h bgpLogsIndexHelper) BaseQuery(i bapi.ClusterInfo) *elastic.BoolQuery {
	q := elastic.NewBoolQuery()
	if h.singleIndex {
		q.Must(elastic.NewTermQuery("cluster", i.Cluster))
		if i.Tenant != "" {
			q.Must(elastic.NewTermQuery("tenant", i.Tenant))
		}
	}
	return q
}

func (h bgpLogsIndexHelper) NewSelectorQuery(selector string) (elastic.Query, error) {
	return nil, nil
}

func (h bgpLogsIndexHelper) NewRBACQuery(resources []apiv3.AuthorizedResourceVerbs) (elastic.Query, error) {
	return nil, nil
}

func (h bgpLogsIndexHelper) NewTimeRangeQuery(r *lmav1.TimeRange) elastic.Query {
	return elastic.NewRangeQuery(GetTimeFieldForQuery(h, r)).
		Gt(r.From.Format(v1.BGPLogTimeFormat)).
		Lte(r.To.Format(v1.BGPLogTimeFormat))
}

func (h bgpLogsIndexHelper) GetTimeField() string {
	return "logtime"
}
