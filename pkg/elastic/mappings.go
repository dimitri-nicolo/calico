// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package elastic

const (
	IndexTimeFormat  = "20060102"
	SnapshotsIndex   = "tigera_secure_ee_snapshots"
	snapshotsMapping = `{
	"properties": {
		"kind": { "type": "keyword" },
		"apiVersion": { "type": "keyword" },
		"items": { "type": "object", "enabled": false },
		"metadata": { "type": "object" },
		"requestStartedTimestamp": { "type": "date" },
		"requestCompletedTimestamp": { "type": "date" }
	}
}`

	ReportsIndex   = "tigera_secure_ee_compliance_reports"
	reportsMapping = `{
	"properties": {
		"reportName": { "type": "keyword" },
		"reportTypeName": { "type": "keyword" },
		"reportSpec": { "type": "object", "enabled": false },
		"reportTypeSpec": { "type": "object", "enabled": false },
		"startTime": { "type": "date" },
		"endTime": { "type": "date" },
		"generationTime": { "type": "date" },
		"uiSummary": { "type": "text" },
		"endpoints": { "type": "object", "enabled": false },
		"namespaces": { "type": "object", "enabled": false },
		"services": { "type": "object", "enabled": false },
		"auditEvents": { "type": "object", "enabled": false }
	}
}`

	BenchmarksIndex   = "tigera_secure_ee_benchmark_results"
	benchmarksMapping = `{
	"properties": {
		"version": { "type": "keyword" },
		"kubernetesVersion": { "type": "keyword" },
		"type": { "type": "keyword" },
		"node_name": { "type": "keyword" },
		"timestamp": { "type": "date" },
		"error": { "type": "text" },
		"tests": {
			"type": "nested",
			"properties": {
				"section": { "type": "keyword" },
				"section_desc": { "type": "text" },
				"test_number": { "type": "keyword" },
				"test_desc": { "type": "text" },
				"test_info": { "type": "text" },
				"status": { "type": "text" },
				"scored": { "type": "boolean" }
			}
		}
	}
}`
	EventsIndex   = "tigera_secure_ee_events"
	eventsMapping = `{
	"dynamic": false,
	"properties": {
		"time": {
			"type": "date",
 			"format": "strict_date_optional_time||epoch_second"
		},
		"type": {
			"type": "keyword"
		},
		"description": {
			"type": "keyword"
		},
		"severity": {
			"type": "long"
		},
		"origin": {
			"type": "keyword"
		},
		"source_ip": {
			"type": "ip",
			"null_value": "0.0.0.0"
		},
		"source_port": {
			"type": "long",
			"null_value": "0"
		},
		"source_namespace": {
			"type": "keyword"
		},
		"source_name": {
			"type": "keyword"
		},
		"source_name_aggr": {
			"type": "keyword"
		},
		"dest_ip": {
			"type": "ip",
			"null_value": "0.0.0.0"
		},
		"dest_port": {
			"type": "long",
			"null_value": "0"
		},
		"dest_namespace": {
			"type": "keyword"
		},
		"dest_name": {
			"type": "keyword"
		},
		"dest_name_aggr": {
			"type": "keyword"
		},
		"host": {
			"type": "keyword"
		},
		"record": {
			"type": "object"
		}
	}
}`
)
