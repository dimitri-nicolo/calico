// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package flows

import (
	"github.com/olivere/elastic/v7"

	api "github.com/tigera/es-proxy/pkg/apis/v1"
)

func GetTimeRangeQuery(tr api.TimeRange) elastic.Query {
	query := elastic.NewRangeQuery("end_time")
	if tr.From != nil {
		query = query.Gt(tr.From.Unix())
	}
	if tr.To != nil {
		query = query.Lte(tr.To.Unix())
	}
	return query
}
