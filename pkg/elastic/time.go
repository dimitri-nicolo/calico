// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package elastic

import (
	"time"

	"github.com/olivere/elastic/v7"
)

func GetEndTimeRangeEpochSecondQuery(from, to time.Time) elastic.Query {
	return elastic.NewRangeQuery("end_time").Gt(from.Unix()).Lte(to.Unix())
}

func GetEndTimeRangeQuery(from, to time.Time) elastic.Query {
	return elastic.NewRangeQuery("end_time").Gt(from).Lte(to)
}

func GetTimeRangeQuery(from, to time.Time) elastic.Query {
	return elastic.NewRangeQuery("time").Gt(from.Unix()).Lte(to.Unix())
}
