// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package logtools

import (
	"fmt"
	"strconv"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	lmaindex "github.com/projectcalico/calico/lma/pkg/elastic/index"
)

// BuildQuery builds an elastic log query using the given parameters.
func BuildQuery(h lmaindex.Helper, i bapi.ClusterInfo, opts v1.LogParams) (elastic.Query, error) {
	query := elastic.NewBoolQuery()

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

// StartFrom parses the given parameters to determine which log to start from in the ES query.
func StartFrom(opts v1.LogParams) (int, error) {
	if ak := opts.GetAfterKey(); ak != nil {
		if val, ok := ak["startFrom"]; ok {
			switch v := val.(type) {
			case string:
				if sf, err := strconv.Atoi(v); err != nil {
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
