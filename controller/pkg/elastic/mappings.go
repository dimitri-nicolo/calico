// Copyright 2019 Tigera Inc. All rights reserved.

package elastic

const ipSetMapping = `{
  "mappings": {
    "_doc": {
      "properties": {
        "created_at": {
            "type": "date",
            "format": "strict_date_optional_time"
        },
        "ips": {
            "type": "ip_range"
        }
      }
    }
  }
}`

const domainNameSetMapping = `{
  "mappings": {
    "_doc": {
      "properties": {
        "created_at": {
            "type": "date",
            "format": "strict_date_optional_time"
        },
        "domains": {
            "type": "keyword"
        }
      }
    }
  }
}`

const eventMapping = `{
  "mappings": {
    "_doc": {
      "properties" : {
        "time": {
            "type": "date",
            "format": "epoch_second"
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
        "flow_log_index": {
            "type": "keyword"
        },
        "flow_log_id": {
            "type": "keyword"
        },
        "protocol": {
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
        "flow_action": {
            "type": "keyword"
        },
        "sources": {
            "type": "keyword"
        },
        "suspicious_prefix": {
            "type": "keyword"
        },
        "anomaly_record": {
            "type": "object"
        }
      }   
    }
  }
}`
