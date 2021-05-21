// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package elastic

import (
	"github.com/olivere/elastic/v7"

	api "github.com/tigera/es-proxy/pkg/apis/v1"
)

func GetEndTimeRangeQuery(tr api.TimeRange) elastic.Query {
	return elastic.NewRangeQuery("end_time").Gt(tr.From.Unix()).Lte(tr.To.Unix())
}

func GetTimeRangeQuery(tr api.TimeRange) elastic.Query {
	return elastic.NewRangeQuery("time").Gt(tr.From.Unix()).Lte(tr.To.Unix())
}
