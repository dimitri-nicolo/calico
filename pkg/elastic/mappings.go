// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package elastic

const (
	IndexTimeFormat  = "20060102"
	SnapshotsIndex   = "tigera_secure_ee_snapshots"
	snapshotsMapping = `{
  "mappings": {
		"properties": {
			"kind": { "type": "keyword" },
			"apiVersion": { "type": "keyword" },
			"items": { "type": "object", "enabled": false },
			"metadata": { "type": "object" },
			"requestStartedTimestamp": { "type": "date" },
			"requestCompletedTimestamp": { "type": "date" }
		}
  }
}`

	ReportsIndex   = "tigera_secure_ee_compliance_reports"
	reportsMapping = `{
  "mappings": {
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
  }
}`

	BenchmarksIndex   = "tigera_secure_ee_benchmark_results"
	benchmarksMapping = `{
  "mappings": {
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
  }
}`
)
