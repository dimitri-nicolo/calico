// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package index

import (
	"fmt"

	"github.com/olivere/elastic/v7"
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	"github.com/projectcalico/calico/lma/pkg/httputils"
)

// Helper provides a set of functions to provide access to index-specific data. This hides
// the fact that the different index mappings are subtly different.
type Helper interface {
	// NewSelectorQuery creates an elasticsearch query from a selector string.
	NewSelectorQuery(selector string) (elastic.Query, error)

	// NewRBACQuery creates an elasticsearch query from an RBAC authorization matrix.
	NewRBACQuery(resources []apiv3.AuthorizedResourceVerbs) (elastic.Query, error)

	// NewTimeQuery creates an elasticsearch timerange query using the appropriate time field.
	NewTimeRangeQuery(r *lmav1.TimeRange) elastic.Query

	// GetTimeField returns the time field used for the query.
	GetTimeField() string

	// BaseQuery returns the base query for the index, to which additional query clauses can be added.
	BaseQuery(i bapi.ClusterInfo, logParams v1.Params) (*elastic.BoolQuery, error)
}

func GetTimeFieldForQuery(h Helper, r *lmav1.TimeRange) string {
	if r != nil && r.Field != lmav1.FieldDefault {
		return string(r.Field)
	}
	return h.GetTimeField()
}

func defaultBaseQuery(i bapi.ClusterInfo, singleIndex bool, params v1.Params) (*elastic.BoolQuery, error) {

	q := elastic.NewBoolQuery()
	if singleIndex && i.Tenant != "" {
		q.Must(elastic.NewTermQuery("tenant", i.Tenant))
	}

	if i.IsQueryMultipleClusters() {
		// one or more clusters expected in the request parameters
		if params == nil {
			return nil, httputils.NewHttpStatusErrorBadRequest("No parameters specified in request", nil)
		}

		if params.IsAllClusters() {
			// all clusters => no cluster filtering

		} else if clusters := params.GetClusters(); len(clusters) > 0 {
			for _, c := range clusters {
				if err := validateClusterValue(c); err != nil {
					return nil, err
				}
			}
			q.Must(elastic.NewTermsQueryFromStrings("cluster", clusters...))

		} else {
			return nil, httputils.NewHttpStatusErrorBadRequest("No clusters specified in request", nil)
		}

	} else {
		// single cluster expected in the x-cluster-id header
		if err := validateClusterValue(i.Cluster); err != nil {
			return nil, err
		}
		q.Must(elastic.NewTermQuery("cluster", i.Cluster))
	}

	return q, nil
}

func validateClusterValue(cluster string) error {
	if cluster == "" || cluster == "*" {
		return httputils.NewHttpStatusErrorBadRequest(fmt.Sprintf("invalid cluster specified in request: '%s'", cluster), nil)
	}
	return nil
}

func processGeneratedField(r *lmav1.TimeRange, timeRangeQuery *elastic.RangeQuery) elastic.Query {
	// Any query that targets generated_time will optionally
	// pass the start and/or the end of the interval without
	// any processing on the value provided. This query will
	// target any value that is higher that the start, but lower or
	// equal to the end of the interval
	if !r.From.IsZero() {
		timeRangeQuery.Gt(r.From)
	}
	if !r.To.IsZero() {
		timeRangeQuery.Lte(r.To)
	}
	return timeRangeQuery
}
