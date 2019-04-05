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
)
