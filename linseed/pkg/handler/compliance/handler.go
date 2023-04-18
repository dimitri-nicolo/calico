// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package compliance

import (
	authzv1 "k8s.io/api/authorization/v1"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/handler"
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
	benchmarks handler.RWHandler[v1.Benchmarks, v1.BenchmarksParams, v1.Benchmarks]
	snapshots  handler.RWHandler[v1.Snapshot, v1.SnapshotParams, v1.Snapshot]
	reports    handler.RWHandler[v1.ReportData, v1.ReportDataParams, v1.ReportData]
}

func New(b bapi.BenchmarksBackend, s bapi.SnapshotsBackend, r bapi.ReportsBackend) *compliance {

	return &compliance{
		benchmarks: handler.NewRWHandler[v1.Benchmarks, v1.BenchmarksParams, v1.Benchmarks](b.Create, b.List),
		snapshots:  handler.NewRWHandler[v1.Snapshot, v1.SnapshotParams, v1.Snapshot](s.Create, s.List),
		reports:    handler.NewRWHandler[v1.ReportData, v1.ReportDataParams, v1.ReportData](r.Create, r.List),
	}
}

func (h compliance) APIS() []handler.API {
	return []handler.API{
		// Reports
		{
			Method:          "POST",
			URL:             ReportsPath,
			Handler:         h.reports.List(),
			AuthzAttributes: &authzv1.ResourceAttributes{Verb: handler.Get, Group: handler.APIGroup, Resource: "compliancereports"},
		},
		{
			Method:          "POST",
			URL:             ReportsPathBulk,
			Handler:         h.reports.Create(),
			AuthzAttributes: &authzv1.ResourceAttributes{Verb: handler.Create, Group: handler.APIGroup, Resource: "compliancereports"},
		},

		// Benchmarks
		{
			Method:          "POST",
			URL:             BenchmarksPath,
			Handler:         h.benchmarks.List(),
			AuthzAttributes: &authzv1.ResourceAttributes{Verb: handler.Get, Group: handler.APIGroup, Resource: "benchmarks"},
		},
		{
			Method:          "POST",
			URL:             BenchmarksPathBulk,
			Handler:         h.benchmarks.Create(),
			AuthzAttributes: &authzv1.ResourceAttributes{Verb: handler.Create, Group: handler.APIGroup, Resource: "benchmarks"},
		},

		// Snapshots
		{
			Method:          "POST",
			URL:             SnapshotsPath,
			Handler:         h.snapshots.List(),
			AuthzAttributes: &authzv1.ResourceAttributes{Verb: handler.Get, Group: handler.APIGroup, Resource: "snapshots"},
		},
		{
			Method:          "POST",
			URL:             SnapshotsPathBulk,
			Handler:         h.snapshots.Create(),
			AuthzAttributes: &authzv1.ResourceAttributes{Verb: handler.Create, Group: handler.APIGroup, Resource: "snapshots"},
		},
	}
}
