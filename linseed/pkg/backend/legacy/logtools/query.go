// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package logtools

import (
	"fmt"
	"strconv"
	"time"

	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	lmaindex "github.com/projectcalico/calico/linseed/pkg/internal/lma/elastic/index"
)

// BuildQuery builds an elastic log query using the given parameters.
func BuildQuery(h lmaindex.Helper, i bapi.ClusterInfo, opts v1.LogParams, start time.Time, end time.Time) (*elastic.BoolQuery, error) {
	query := elastic.NewBoolQuery()

	// Parse times from the request. We default to a time-range query
	// if no other search parameters are given.
	query.Filter(h.NewTimeRangeQuery(start, end))

	// If RBAC constraints were given, add them in.
	if perms := opts.GetPermissions(); len(perms) > 0 {
		rbacQuery, err := h.NewRBACQuery(perms)
		if err != nil {
			return nil, err
		}
		if rbacQuery != nil {
			query.Filter(rbacQuery)
		}
	}

	// If a selector was provided, parse it and add it in.
	if sel := opts.GetSelector(); len(sel) > 0 {
		selQuery, err := h.NewSelectorQuery(sel)
		if err != nil {
			return nil, err
		}
		if selQuery != nil {
			query.Must(selQuery)
		}
	}

	return query, nil
}

func ExtractTimeRange(timeRange *lmav1.TimeRange) (time.Time, time.Time) {
	// Parse times from the request. We default to a time-range query
	// if no other search parameters are given.
	var start, end time.Time
	if timeRange != nil {
		start = timeRange.From
		end = timeRange.To
	} else {
		// Default to the start of the timeline
		start = time.Time{}
		end = time.Now()
	}
	return start, end
}

// StartFrom parses the given parameters to determine which log to start from in the ES query.
func StartFrom(opts v1.Params) (int, error) {
	if ak := opts.GetAfterKey(); ak != nil {
		if val, ok := ak["startFrom"]; ok {
			switch v := val.(type) {
			case string:
				if sf, err := strconv.Atoi(v); err == nil {
					return sf, nil
				} else {
					return 0, fmt.Errorf("Could not parse startFrom (%s) as an integer", v)
				}
			case float64:
				logrus.WithField("val", val).Trace("Handling float64 startFrom")
				return int(v), nil
			case int:
				logrus.WithField("val", val).Trace("Handling int startFrom")
				return v, nil
			default:
				logrus.WithField("val", val).Warnf("Unexpected type (%T) for startFrom, will not perform paging", val)
			}
		}
	}
	logrus.Trace("Starting query from 0")
	return 0, nil
}

// NextStartFromAfterKey generates an AfterKey to use for log queries that use startFrom to pass
// the document index from which to start the next page of results.
func NextStartFromAfterKey(opts v1.Params, numHits, prevStartFrom int, totalHits int64) map[string]interface{} {
	var ak map[string]interface{}

	// Calculate the next starting point using the value received in the request
	// and the current hits returned on the query
	nextStartFrom := prevStartFrom + numHits

	if numHits < opts.GetMaxPageSize() || nextStartFrom >= int(totalHits) {
		// We fully satisfied the request, no afterkey.
		ak = nil
	} else {
		// There are more hits, return an afterKey the client can use for pagination.
		// We add the number of hits to the start from provided on the request, if any.
		ak = map[string]interface{}{
			"startFrom": nextStartFrom,
		}
	}
	return ak
}
