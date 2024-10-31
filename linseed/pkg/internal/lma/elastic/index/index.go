// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package index

import (
	"github.com/olivere/elastic/v7"
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
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
	BaseQuery(i bapi.ClusterInfo) *elastic.BoolQuery
}

func GetTimeFieldForQuery(h Helper, r *lmav1.TimeRange) string {
	if r != nil && r.Field != lmav1.FieldDefault {
		return string(r.Field)
	}
	return h.GetTimeField()
}
