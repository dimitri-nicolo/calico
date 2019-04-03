// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package elastic

const (
	snapshotsIndex   = "tigera_secure_ee_snapshots"
	snapshotsMapping = `{
  "mappings": {
    "_doc": {
      "properties": {
        "apiVersion": { "type": "keyword" },
        "kind": { "type": "text" },
        "items": { "type": "object", "enabled": false },
        "metadata": { "type": "object" },
        "requestStartedTimestamp": { "type": "date" },
        "requestCompletedTimestamp": { "type": "date" }
      }
    }
  }
}`
)
