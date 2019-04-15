// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package elastic

const (
	snapshotsIndex   = "tigera_secure_ee_snapshots"
	snapshotsMapping = `{
  "mappings": {
    "_doc": {
      "properties": {
        "kind": { "type": "keyword" },
        "apiVersion": { "type": "keyword" },
        "items": { "type": "object", "enabled": false },
        "metadata": { "type": "object" },
        "requestStartedTimestamp": { "type": "date" },
        "requestCompletedTimestamp": { "type": "date" }
      }
    }
  }
}`

	reportsIndex   = "tigera_secure_ee_compliance_reports"
	reportsMapping = `{
  "mappings": {
    "_doc": {
      "properties": {
        "reportType": { "type": "text" },
        "reportSpec": { "type": "object", "enabled": false },
        "startTime": { "type": "date" },
        "endTime": { "type": "date" },
        "uiSummary": { "type": "text" },
        "endpoints": { "type": "object", "enabled": false },
        "namespaces": { "type": "object", "enabled": false },
        "services": { "type": "object", "enabled": false },
        "auditEvents": { "type": "object", "enabled": false }
      }
    }
  }
}`
)
