// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package index

import (
	"time"

	"github.com/olivere/elastic/v7"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

func SingleIndexComplianceSnapshots() Helper {
	return complianceSnapshotsIndexHelper{singleIndex: true}
}

func MultiIndexComplianceSnapshots() Helper {
	return complianceSnapshotsIndexHelper{}
}

// complianceSnapshotsIndexHelper implements the Helper interface for flow logs.
type complianceSnapshotsIndexHelper struct {
	singleIndex bool
}

func (h complianceSnapshotsIndexHelper) BaseQuery(i bapi.ClusterInfo) *elastic.BoolQuery {
	q := elastic.NewBoolQuery()
	if h.singleIndex {
		q.Must(elastic.NewTermQuery("cluster", i.Cluster))
		if i.Tenant != "" {
			q.Must(elastic.NewTermQuery("tenant", i.Tenant))
		}
	}
	return q
}

func (h complianceSnapshotsIndexHelper) NewSelectorQuery(selector string) (elastic.Query, error) {
	return nil, nil
}

func (h complianceSnapshotsIndexHelper) NewRBACQuery(resources []apiv3.AuthorizedResourceVerbs) (elastic.Query, error) {
	return nil, nil
}

func (h complianceSnapshotsIndexHelper) NewTimeRangeQuery(from, to time.Time) elastic.Query {
	unset := time.Time{}
	tr := elastic.NewRangeQuery(h.GetTimeField())
	if from != unset {
		tr.From(from)
	}
	if to != unset {
		tr.To(to)
	}
	return tr
}

func (h complianceSnapshotsIndexHelper) GetTimeField() string {
	return "requestCompletedTimestamp"
}
