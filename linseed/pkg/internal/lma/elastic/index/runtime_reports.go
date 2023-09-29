// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package index

import (
	"fmt"
	"time"

	"github.com/olivere/elastic/v7"
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
)

// runtimeReportsIndexHelper implements the Helper interface.
type runtimeReportsIndexHelper struct {
	singleIndex bool
}

func MultiIndexRuntimeReports() Helper {
	return runtimeReportsIndexHelper{}
}

func SingleIndexRuntimeReports() Helper {
	return runtimeReportsIndexHelper{
		singleIndex: true,
	}
}

func (h runtimeReportsIndexHelper) BaseQuery(i bapi.ClusterInfo) *elastic.BoolQuery {
	q := elastic.NewBoolQuery()
	if h.singleIndex {
		if i.Tenant != "" {
			// Query is meant for a specific tenant - filter on tenant.
			q.Must(elastic.NewTermQuery("tenant", i.Tenant))
		}

		// This is a request from a single-tenant system. Return all clusters regardless of the x-cluster-id provided.
		// Note that this is different from how most other data types work, but is the expected behavior for
		// runtime reports.
	}
	return q
}

func (h runtimeReportsIndexHelper) NewSelectorQuery(selector string) (elastic.Query, error) {
	return nil, fmt.Errorf("not implemented")
}

func (h runtimeReportsIndexHelper) NewRBACQuery(resources []apiv3.AuthorizedResourceVerbs) (elastic.Query, error) {
	return nil, fmt.Errorf("not implemented")
}

func (h runtimeReportsIndexHelper) NewTimeRangeQuery(from, to time.Time) elastic.Query {
	return nil
}

func (h runtimeReportsIndexHelper) GetTimeField() string {
	return ""
}
