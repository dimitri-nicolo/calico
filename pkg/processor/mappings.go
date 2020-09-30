// Copyright 2019 Tigera Inc. All rights reserved.

package processor

/*
const ipSetMapping = `{
    "properties": {
        "created_at": {
              "type": "date",
              "format": "strict_date_optional_time"
        },
        "ips": {
            "type": "ip_range"
        }
    }
}`

const domainNameSetMapping = `{
    "properties": {
        "created_at": {
            "type": "date",
            "format": "strict_date_optional_time"
        },
        "domains": {
            "type": "keyword"
        }
    }
}`
*/
const EventMapping = `{
    "properties" : {
        "alert" : {
          "type" : "text",
          "fields" : {
            "keyword" : {
              "type" : "keyword",
              "ignore_above" : 256
            }
          }
        },
        "anomaly_record" : {
          "type" : "object"
        },
        "description" : {
          "type" : "keyword"
        },
        "dest_ip" : {
          "type" : "ip",
          "null_value" : "0.0.0.0"
        },
        "dest_name" : {
          "type" : "keyword"
        },
        "dest_namespace" : {
          "type" : "keyword"
        },
        "dest_port" : {
          "type" : "long",
          "null_value" : 0
        },
        "dns_log_id" : {
          "type" : "keyword"
        },
        "dns_log_index" : {
          "type" : "keyword"
        },
        "flow_action" : {
          "type" : "keyword"
        },
        "flow_log_id" : {
          "type" : "keyword"
        },
        "flow_log_index" : {
          "type" : "keyword"
        },
        "protocol" : {
          "type" : "keyword"
        },
        "record" : {
          "properties" : {
            "count" : {
              "type" : "long"
            },
            "dest_name_aggr" : {
              "type" : "text",
              "fields" : {
                "keyword" : {
                  "type" : "keyword",
                  "ignore_above" : 256
                }
              }
            },
            "dest_namespace" : {
              "type" : "text",
              "fields" : {
                "keyword" : {
                  "type" : "keyword",
                  "ignore_above" : 256
                }
              }
            },
            "host" : {
              "properties" : {
                "keyword" : {
                  "type" : "text",
                  "fields" : {
                    "keyword" : {
                      "type" : "keyword",
                      "ignore_above" : 256
                    }
                  }
                }
              }
            },
            "snort" : {
              "properties" : {
                "Category" : {
                  "type" : "text",
                  "fields" : {
                    "keyword" : {
                      "type" : "keyword",
                      "ignore_above" : 256
                    }
                  }
                },
                "Descripton" : {
                  "type" : "text",
                  "fields" : {
                    "keyword" : {
                      "type" : "keyword",
                      "ignore_above" : 256
                    }
                  }
                },
                "Flags" : {
                  "type" : "text",
                  "fields" : {
                    "keyword" : {
                      "type" : "keyword",
                      "ignore_above" : 256
                    }
                  }
                },
                "Occurance" : {
                  "type" : "text",
                  "fields" : {
                    "keyword" : {
                      "type" : "keyword",
                      "ignore_above" : 256
                    }
                  }
                },
                "Other" : {
                  "type" : "text",
                  "fields" : {
                    "keyword" : {
                      "type" : "keyword",
                      "ignore_above" : 256
                    }
                  }
                }
              }
            },
            "source_name_aggr" : {
              "type" : "text",
              "fields" : {
                "keyword" : {
                  "type" : "keyword",
                  "ignore_above" : 256
                }
              }
            },
            "source_namespace" : {
              "type" : "text",
              "fields" : {
                "keyword" : {
                  "type" : "keyword",
                  "ignore_above" : 256
                }
              }
            }
          }
        },
        "severity" : {
          "type" : "long"
        },
        "source_ip" : {
          "type" : "ip",
          "null_value" : "0.0.0.0"
        },
        "source_name" : {
          "type" : "keyword"
        },
        "source_namespace" : {
          "type" : "keyword"
        },
        "source_port" : {
          "type" : "long",
          "null_value" : 0
        },
        "sources" : {
          "type" : "keyword"
        },
        "suspicious_domains" : {
          "type" : "keyword"
        },
        "suspicious_prefix" : {
          "type" : "keyword"
        },
        "time" : {
          "type" : "date"
        },
        "type" : {
          "type" : "keyword"
        }
      }    
}`
