// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package compliance

import (
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/handler"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
)

const (
	ReportsPath        = "/compliance/reports"
	ReportsPathBulk    = "/compliance/reports/bulk"
	BenchmarksPath     = "/compliance/benchmarks"
	BenchmarksPathBulk = "/compliance/benchmarks/bulk"
	SnapshotsPath      = "/compliance/snapshots"
	SnapshotsPathBulk  = "/compliance/snapshots/bulk"
)

type compliance struct {
	benchmarks *handler.GenericHandler[v1.Benchmarks, v1.BenchmarksParams, v1.Benchmarks]
	snapshots  *handler.GenericHandler[v1.Snapshot, v1.SnapshotParams, v1.Snapshot]
	reports    *handler.GenericHandler[v1.ReportData, v1.ReportDataParams, v1.ReportData]
}

func New(b bapi.BenchmarksBackend, s bapi.SnapshotsBackend, r bapi.ReportsBackend) *compliance {
	benchmarks := &handler.GenericHandler[v1.Benchmarks, v1.BenchmarksParams, v1.Benchmarks]{
		CreateFn: b.Create,
		ListFn:   b.List,
	}
	snapshots := &handler.GenericHandler[v1.Snapshot, v1.SnapshotParams, v1.Snapshot]{
		CreateFn: s.Create,
		ListFn:   s.List,
	}
	reports := &handler.GenericHandler[v1.ReportData, v1.ReportDataParams, v1.ReportData]{
		CreateFn: r.Create,
		ListFn:   r.List,
	}

	return &compliance{
		benchmarks: benchmarks,
		snapshots:  snapshots,
		reports:    reports,
	}
}

func (h compliance) APIS() []handler.API {
	return []handler.API{
		// Reports
		{
			Method:  "POST",
			URL:     ReportsPath,
			Handler: h.reports.List(),
		},
		{
			Method:  "POST",
			URL:     ReportsPathBulk,
			Handler: h.reports.Create(),
		},

		// Benchmarks
		{
			Method:  "POST",
			URL:     BenchmarksPath,
			Handler: h.benchmarks.List(),
		},
		{
			Method:  "POST",
			URL:     BenchmarksPathBulk,
			Handler: h.benchmarks.Create(),
		},

		// Snapshots
		{
			Method:  "POST",
			URL:     SnapshotsPath,
			Handler: h.snapshots.List(),
		},
		{
			Method:  "POST",
			URL:     SnapshotsPathBulk,
			Handler: h.snapshots.Create(),
		},
	}
}
