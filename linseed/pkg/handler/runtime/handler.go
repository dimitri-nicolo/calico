// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package runtime

import (
	authzv1 "k8s.io/api/authorization/v1"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/handler"
)

const (
	ReportsPath     = "/runtime/reports"
	ReportsPathBulk = "/runtime/reports/bulk"
)

type runtime struct {
	reports handler.RWHandler[v1.RuntimeReport, v1.RuntimeReportParams, v1.Report]
}

func New(b bapi.RuntimeBackend) *runtime {
	return &runtime{
		reports: handler.NewRWHandler[v1.RuntimeReport, v1.RuntimeReportParams, v1.Report](b.Create, b.List),
	}
}

func (h runtime) APIS() []handler.API {
	return []handler.API{
		{
			Method:          "POST",
			URL:             ReportsPathBulk,
			Handler:         h.reports.Create(),
			AuthzAttributes: &authzv1.ResourceAttributes{Verb: handler.Create, Group: handler.APIGroup, Resource: "runtimereports"},
		},
		{
			Method:          "POST",
			URL:             ReportsPath,
			Handler:         h.reports.List(),
			AuthzAttributes: &authzv1.ResourceAttributes{Verb: handler.Get, Group: handler.APIGroup, Resource: "runtimereports"},
		},
	}
}
