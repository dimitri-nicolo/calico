// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package index

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/olivere/elastic/v7"
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/libcalico-go/lib/validator/v3/query"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	"github.com/projectcalico/calico/lma/pkg/httputils"
)

// wafLogsIndexHelper implements the Helper interface for waf logs.
type wafLogsIndexHelper struct {
	singleIndex bool
}

// MultiIndexWAFLogs returns an instance of the waf logs index helper.
func MultiIndexWAFLogs() Helper {
	return wafLogsIndexHelper{}
}

// SingleIndexWAFLogs returns an instance of the waf logs index helper.
func SingleIndexWAFLogs() Helper {
	return wafLogsIndexHelper{singleIndex: true}
}

// NewWAFLogsConverter returns a Converter instance defined for waf logs.
func NewWAFLogsConverter() converter {
	return converter{wafAtomToElastic}
}

// wafAtomToElastic returns a waf log atom as an elastic JsonObject.
func wafAtomToElastic(a *query.Atom) JsonObject {
	switch a.Key {
	case "rules.id", "rules.message", "rules.severity", "rules.file", "rules.disruptive", "rules.line":

		path := a.Key[:strings.Index(a.Key, ".")]
		return JsonObject{
			"nested": JsonObject{
				"path":  path,
				"query": basicAtomToElastic(a),
			},
		}
	}

	return basicAtomToElastic(a)
}

func (h wafLogsIndexHelper) BaseQuery(i bapi.ClusterInfo, params v1.Params) (*elastic.BoolQuery, error) {
	return defaultBaseQuery(i, h.singleIndex, params)
}

func (h wafLogsIndexHelper) NewSelectorQuery(selector string) (elastic.Query, error) {
	q, err := query.ParseQuery(selector)
	if err != nil {
		return nil, &httputils.HttpStatusError{
			Status: http.StatusBadRequest,
			Err:    err,
			Msg:    fmt.Sprintf("Invalid selector (%s) in request: %v", selector, err),
		}
	} else if err := query.Validate(q, query.IsValidWAFAtom); err != nil {
		return nil, &httputils.HttpStatusError{
			Status: http.StatusBadRequest,
			Err:    err,
			Msg:    fmt.Sprintf("Invalid selector (%s) in request: %v", selector, err),
		}
	}
	converter := NewWAFLogsConverter()
	return JsonObjectElasticQuery(converter.Convert(q)), nil
}

func (h wafLogsIndexHelper) NewRBACQuery(
	resources []apiv3.AuthorizedResourceVerbs,
) (elastic.Query, error) {
	return nil, fmt.Errorf("not implemented")
}

func (h wafLogsIndexHelper) NewTimeRangeQuery(r *lmav1.TimeRange) elastic.Query {
	timeField := GetTimeFieldForQuery(h, r)
	timeRangeQuery := elastic.NewRangeQuery(timeField)
	if timeField == "generated_time" {
		if !r.From.IsZero() {
			timeRangeQuery.Gt(r.From)
		}
		if !r.To.IsZero() {
			timeRangeQuery.Lte(r.To)
		}
		return timeRangeQuery
	}

	return timeRangeQuery.Gt(r.From).Lte(r.To)
}

func (h wafLogsIndexHelper) GetTimeField() string {
	return "@timestamp"
}
