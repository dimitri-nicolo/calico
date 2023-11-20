// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package index

import (
	"fmt"
	"net/http"
	"time"

	"github.com/olivere/elastic/v7"
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/libcalico-go/lib/validator/v3/query"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/lma/pkg/httputils"
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
	q, err := query.ParseQuery(selector)
	if err != nil {
		return nil, &httputils.HttpStatusError{
			Status: http.StatusBadRequest,
			Err:    err,
			Msg:    fmt.Sprintf("Invalid selector (%s) in request: %v", selector, err),
		}
	} else if err := query.Validate(q, IsValidRuntimeAtom); err != nil {
		return nil, &httputils.HttpStatusError{
			Status: http.StatusBadRequest,
			Err:    err,
			Msg:    fmt.Sprintf("Invalid selector (%s) in request: %v", selector, err),
		}
	}
	converter := converter{basicAtomToElastic}
	return JsonObjectElasticQuery(converter.Convert(q)), nil
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

func IsValidRuntimeAtom(*query.Atom) error {
	// We don't need any detailed validation here because the querying of runtime reports via
	// selector is not exposed to our users.  It is only used by the Sasha code for retrieving
	// historical reports.
	return nil
}
