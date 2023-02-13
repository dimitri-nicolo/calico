// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package logtools

import (
	"strconv"
	"time"

	"github.com/olivere/elastic/v7"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	lmaindex "github.com/projectcalico/calico/lma/pkg/elastic/index"
	"github.com/sirupsen/logrus"
)

// BuildQuery builds an elastic log query using the given parameters.
func BuildQuery(h lmaindex.Helper, i bapi.ClusterInfo, opts v1.LogParams) (elastic.Query, error) {
	// Parse times from the request. We default to a time-range query
	// if no other search parameters are given.
	var start, end time.Time
	if tr := opts.GetTimeRange(); tr != nil {
		start = tr.From
		end = tr.To
	} else {
		// Default to the latest 5 minute window.
		start = time.Now().Add(-5 * time.Minute)
		end = time.Now()
	}
	constraints := []elastic.Query{
		h.NewTimeRangeQuery(start, end),
	}

	// If RBAC constraints were given, add them in.
	if perms := opts.GetPermissions(); len(perms) > 0 {
		rbacQuery, err := h.NewRBACQuery(perms)
		if err != nil {
			return nil, err
		}
		constraints = append(constraints, rbacQuery)
	}

	// If a selector was provided, parse it and add it in.
	if sel := opts.GetSelector(); len(sel) > 0 {
		selQuery, err := h.NewSelectorQuery(sel)
		if err != nil {
			return nil, err
		}
		constraints = append(constraints, selQuery)
	}

	if len(constraints) == 1 {
		// This is just a time-range query. We don't need to join multiple
		// constraints together.
		return constraints[0], nil
	}

	// We need to perform a boolean query with multiple constraints.
	return elastic.NewBoolQuery().Filter(constraints...), nil
}

// StartFrom parses the given parameters to determine which log to start from in the ES query.
func StartFrom(opts v1.LogParams) int {
	if ak := opts.GetAfterKey(); ak != nil {
		if val, ok := ak["startFrom"]; ok {
			switch v := val.(type) {
			case string:
				if sf, err := strconv.Atoi(v); err != nil {
					return sf
				} else {
					logrus.WithField("val", v).Warn("Could not parse startFrom as an integer")
				}
			case float64:
				logrus.WithField("val", val).Debug("Handling float64 startFrom")
				return int(v)
			case int:
				logrus.WithField("val", val).Debug("Handling int startFrom")
				return v
			default:
				logrus.WithField("val", val).Warnf("Unexpected type (%T) for startFrom, will not perform paging", val)
			}
		}
	}
	logrus.Debug("Starting query from 0")
	return 0
}
