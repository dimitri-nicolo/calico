// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package runtime

import (
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/handler"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
)

const (
	ReportsPath     = "/runtime/reports"
	ReportsPathBulk = "/runtime/reports/bulk"
)

type runtime struct {
	reports *handler.GenericHandler[v1.RuntimeReport, v1.RuntimeReportParams, v1.Report]
}

func New(b bapi.RuntimeBackend) *runtime {
	reports := handler.GenericHandler[v1.RuntimeReport, v1.RuntimeReportParams, v1.Report]{
		CreateFn: b.Create,
		ListFn:   b.List,
	}

	return &runtime{
		reports: &reports,
	}
}

func (h runtime) APIS() []handler.API {
	return []handler.API{
		{
			Method:  "POST",
			URL:     ReportsPathBulk,
			Handler: h.reports.Create(),
		},
		{
			Method:  "POST",
			URL:     ReportsPath,
			Handler: h.reports.List(),
		},
	}
}
