// Copyright (c) 2019-2022 Tigera, Inc. All rights reserved.
package elastic

import (
	_ "embed"
)

const (
	IndexTimeFormat = "20060102"

	BenchmarksIndex = "tigera_secure_ee_benchmark_results"
	EventsIndex     = "tigera_secure_ee_events"
	ReportsIndex    = "tigera_secure_ee_compliance_reports"
	SnapshotsIndex  = "tigera_secure_ee_snapshots"
)

var (
	//go:embed mappings/snapshots.json
	snapshotsMapping string

	//go:embed mappings/reports.json
	reportsMapping string

	//go:embed mappings/benchmark.json
	benchmarksMapping string

	// TODO CASEY
	// This has moved to Linseed. Leaving this here for now so the code still builds, but we'll
	// eventually need to clean this up once all components that write events
	// have been moved over to Linseed.
	//go:embed mappings/events.json
	eventsMapping string

	// Old events index mapping up to Calico Enterprise v3.11.
	// Security events written to this old events index are readonly from Calico Enterprise v3.12,
	// with one exception for events dismissal and deletion introduced in Calico Enterprise v3.14.
	// This old mapping is taken from: intrusion-detection/blob/master/pkg/elastic/mappings.go.
	// In Calico Enterprise v3.14, A new "dismissed" field is added to this old mapping.
	//go:embed mappings/old_events.json
	oldEventsMapping string
)
