package elastic

const ipSetMapping = `{
  "mappings": {
    "_doc": {
      "properties": {
        "ips": {
            "type": "ip_range"
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
        "feeds": {
                /* This is an array of keywords. It is not necessary to declare this as an array. Elastic will automatically accept a list of strings here */
                "type": "nested",
                "properties": {
                        "labels": {"type": "keyword"}
                }
        },
        "suspicious_prefix": {
            "type": "keyword"
        },
      }   
    }
  }
}`
