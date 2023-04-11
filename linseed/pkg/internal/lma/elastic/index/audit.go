// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package index

import (
	"fmt"
	"net/http"
	"time"

	"github.com/olivere/elastic/v7"
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/libcalico-go/lib/validator/v3/query"
	"github.com/projectcalico/calico/lma/pkg/httputils"
)

// auditLogsIndexHelper implements the Helper interface for audit logs.
type auditLogsIndexHelper struct{}

// AuditLogs returns an instance of the audit logs index helper.
func AuditLogs() Helper {
	return auditLogsIndexHelper{}
}

func NewAuditLogsConverter() converter {
	return converter{basicAtomToElastic}
}

func (h auditLogsIndexHelper) NewSelectorQuery(selector string) (elastic.Query, error) {
	q, err := query.ParseQuery(selector)
	if err != nil {
		return nil, &httputils.HttpStatusError{
			Status: http.StatusBadRequest,
			Err:    err,
			Msg:    fmt.Sprintf("Invalid selector (%s) in request: %v", selector, err),
		}
	} else if err := query.Validate(q, query.IsValidAuditAtom); err != nil {
		return nil, &httputils.HttpStatusError{
			Status: http.StatusBadRequest,
			Err:    err,
			Msg:    fmt.Sprintf("Invalid selector (%s) in request: %v", selector, err),
		}
	}
	converter := NewAuditLogsConverter()
	return JsonObjectElasticQuery(converter.Convert(q)), nil
}

func (h auditLogsIndexHelper) NewRBACQuery(
	resources []apiv3.AuthorizedResourceVerbs,
) (elastic.Query, error) {
	return nil, fmt.Errorf("not implemented")
}

func (h auditLogsIndexHelper) NewTimeRangeQuery(from, to time.Time) elastic.Query {
	return elastic.NewRangeQuery("requestReceivedTimestamp").Gt(from).Lte(to)
}

func (h auditLogsIndexHelper) GetTimeField() string {
	return "requestReceivedTimestamp"
}

func (h auditLogsIndexHelper) GetIndex(cluster string) string {
	panic("Not implemented!")
}
