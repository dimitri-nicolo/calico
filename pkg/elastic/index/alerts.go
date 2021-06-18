// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package index

import (
	"fmt"
	"net/http"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/tigera/lma/pkg/httputils"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/validator/v3/query"
)

const esAlertsIndexPrefix = "tigera_secure_ee_events"

// alertsIndexHelper implements the Helper interface for events.
type alertsIndexHelper struct{}

// Alerts returns an instance of the alerts index helper.
func Alerts() Helper {
	return alertsIndexHelper{}
}

// NewAuditLogsConverter returns a converter instance defined for alerts.
func NewAuditLogsConverter() converter {
	return converter{basicAtomToElastic}
}

// Helper.

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
	converter := NewAuditLogsConverter()
	return JsonObjectElasticQuery(converter.Convert(q)), nil
}

func (h alertsIndexHelper) NewRBACQuery(
	resources []apiv3.AuthorizedResourceVerbs,
) (elastic.Query, error) {
	return nil, nil
}

func (h alertsIndexHelper) NewTimeRangeQuery(from, to time.Time) elastic.Query {
	return elastic.NewRangeQuery("time").Gt(from.Unix()).Lte(to.Unix())
}

func (h alertsIndexHelper) GetTimeField() string {
	return "time"
}

func (h alertsIndexHelper) GetIndex(cluster string) string {
	return fmt.Sprintf("%s.%s", esAlertsIndexPrefix, cluster)
}
