package elastic

const ipSetMapping = `{
  "mappings": {
    "_doc": {
      "properties": {
        "ips": {
            "type": "ip"
        }
      }
    }
  }
}`

const eventMapping = `{
  "mappings": {
    "_doc": {
      "properties" : {
        "start_time": {
            "type": "date",
            "format": "epoch_second"
        },
        "end_time": {
            "type": "date",
            "format": "epoch_second"
        },
        "action": {
            "type": "keyword"
        },
        "bytes_in": {
            "type": "long"
        },
        "bytes_out": {
            "type": "long"
        },
        "dest_ip": {
            "type": "ip",
            "null_value": "0.0.0.0"
        },
        "dest_name": {
            "type": "keyword"
        },
        "dest_name_aggr": {
            "type": "keyword"
        },
        "dest_namespace": {
            "type": "keyword"
        },
        "dest_port": {
            "type": "long",
            "null_value": "0"
        },
        "dest_type": {
            "type": "keyword"
        },
        "dest_labels": {
                /* This is an array of keywords. It is not necessary to declare this as an array. Elastic will automatically accept a list of strings here */
                "type": "nested",
                "properties": {
                        "labels": {"type": "keyword"}
                }
        },
        "reporter": {
            "type": "keyword"
        },
        "num_flows": {
            "type": "long"
        },
        "num_flows_completed": {
            "type": "long"
        },
        "num_flows_started": {
            "type": "long"
        },
        "packets_in": {
            "type": "long"
        },
        "packets_out": {
            "type": "long"
        },
        "proto": {
            "type": "keyword"
        },
        "policies": {
                /* This is an array of keywords. It is not necessary to declare this as an array. Elastic will automatically accept a list of strings here */
                "type": "nested",
                "properties": {
                        "all_policies": {"type": "keyword"}
                }
        },
        "source_ip": {
            "type": "ip",
            "null_value": "0.0.0.0"
        },
        "source_name": {
            "type": "keyword"
        },
        "source_name_aggr": {
            "type": "keyword"
        },
        "source_namespace": {
            "type": "keyword"
        },
        "source_port": {
            "type": "long",
            "null_value": "0"
        },
        "source_type": {
            "type": "keyword"
        },
        "source_labels": {
                /* This is an array of keywords. It is not necessary to declare this as an array. Elastic will automatically accept a list of strings here */
                "type": "nested",
                "properties": {
                        "labels": {"type": "keyword"}
                }
        }
      }   
    }
  }
}`

