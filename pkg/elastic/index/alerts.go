// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package index

import (
	"fmt"
	"net/http"
	"time"

	"github.com/olivere/elastic/v7"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	lmaelastic "github.com/tigera/lma/pkg/elastic"
	"github.com/tigera/lma/pkg/httputils"

	"github.com/projectcalico/calico/libcalico-go/lib/validator/v3/query"
)

// alertsIndexHelper implements the Helper interface for events.
type alertsIndexHelper struct{}

// Alerts returns an instance of the alerts index helper.
func Alerts() Helper {
	return alertsIndexHelper{}
}

// NewAlertsConverter returns a converter instance defined for alerts.
func NewAlertsConverter() converter {
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
	converter := NewAlertsConverter()
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

// GetIndex returns generic name of events index that can be query from both older events index
// `tigera_secure_ee_events.<cluster>` used prior to CEv3.12 and newer events index name
// `tigera_secure_ee_events.<cluster>` used in CE >=v3.12
func (h alertsIndexHelper) GetIndex(cluster string) string {
	return fmt.Sprintf("%s.%s*", lmaelastic.EventsIndex, cluster)
}
