[![Build Status](https://tigera.semaphoreci.com/badges/es-gateway.svg?key=3c01c819-532b-4ccc-8305-5dd45c10bf93)](https://tigera.semaphoreci.com/projects/es-gateway)

# es-gateway
Component for gate-keeping all requests to Elasticsearch & Kibana.

## Configuration

Name | Type | Default
--- | --- | ---
ES_GATEWAY_LOG_LEVEL | Environment | INFO
ES_GATEWAY_PORT | Environment | 5554
ES_GATEWAY_HOST | Environment | any
ES_GATEWAY_HTTPS_CERT | Environment | /certs/https/cert
ES_GATEWAY_HTTPS_KEY | Environment | /certs/https/key
ES_GATEWAY_ELASTIC_ENDPOINT | Environment | https://tigera-secure-es-http.tigera-elasticsearch.svc:9200
ES_GATEWAY_ELASTIC_PATH_PREFIXES | Environment | /
ES_GATEWAY_ELASTIC_CA_BUNDLE_PATH | Environment | /certs/elasticsearch/tls.crt
ES_GATEWAY_KIBANA_ENDPOINT | Environment | https://tigera-secure-kb-http.tigera-kibana.svc:5601
ES_GATEWAY_KIBANA_PATH_PREFIXES | Environment | /tigera-kibana
ES_GATEWAY_KIBANA_CA_BUNDLE_PATH | Environment | /certs/kibana/tls.crt
