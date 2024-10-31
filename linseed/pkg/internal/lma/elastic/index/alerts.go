// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package index

import (
	"fmt"
	"net/http"

	"github.com/olivere/elastic/v7"
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/libcalico-go/lib/validator/v3/query"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	"github.com/projectcalico/calico/lma/pkg/httputils"
)

// alertsIndexHelper implements the Helper interface for events.
type alertsIndexHelper struct {
	singleIndex bool
}

func MultiIndexAlerts() Helper {
	return alertsIndexHelper{}
}

func SingleIndexAlerts() Helper {
	return alertsIndexHelper{singleIndex: true}
}

// NewAlertsConverter returns a Converter instance defined for alerts.
func NewAlertsConverter() converter {
	return converter{basicAtomToElastic}
}

// Helper.

func (h alertsIndexHelper) BaseQuery(i bapi.ClusterInfo) *elastic.BoolQuery {
	q := elastic.NewBoolQuery()
	if h.singleIndex {
		q.Must(elastic.NewTermQuery("cluster", i.Cluster))
		if i.Tenant != "" {
			q.Must(elastic.NewTermQuery("tenant", i.Tenant))
		}
	}
	return q
}

func (h alertsIndexHelper) NewSelectorQuery(selector string) (elastic.Query, error) {
	q, err := query.ParseQuery(selector)
	if err != nil {
		return nil, &httputils.HttpStatusError{
			Status: http.StatusBadRequest,
			Err:    err,
			Msg:    fmt.Sprintf("Invalid selector (%s) in request: %v", selector, err),
		}
	} else if err := query.Validate(q, query.IsValidEventsKeysAtom); err != nil {
		return nil, &httputils.HttpStatusError{
			Status: http.StatusBadRequest,
			Err:    err,
			Msg:    fmt.Sprintf("Invalid selector (%s) in request: %v", selector, err),
		}
	}
	converter := NewAlertsConverter()
	return JsonObjectElasticQuery(converter.Convert(q)), nil
}

func (h alertsIndexHelper) NewRBACQuery(resources []apiv3.AuthorizedResourceVerbs) (elastic.Query, error) {
	return nil, nil
}

func (h alertsIndexHelper) NewTimeRangeQuery(r *lmav1.TimeRange) elastic.Query {
	return elastic.NewRangeQuery(GetTimeFieldForQuery(h, r)).Gt(r.From.Unix()).Lte(r.To.Unix())
}

func (h alertsIndexHelper) GetTimeField() string {
	return "time"
}
