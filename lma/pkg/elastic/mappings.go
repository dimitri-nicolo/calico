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

	// New events index mapping starting from Calico Enterprise v3.12.
	// Security events are written in this new format and a read/write alias is created for this new events index.
	// See design doc: https://docs.google.com/document/d/1W-qpjI1KWnLYJ0Rc13CkVVdZEIlviKTJ20N-YbF70hA/edit?usp=sharing
	// In Calico Enterprise v3.14, A new "dismissed" field is added to this new mapping.
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
