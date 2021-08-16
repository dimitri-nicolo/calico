[![Build Status](https://tigera.semaphoreci.com/badges/es-gateway.svg?key=3c01c819-532b-4ccc-8305-5dd45c10bf93)](https://tigera.semaphoreci.com/projects/es-gateway)

# es-gateway
Component for gate-keeping all requests to Elasticsearch & Kibana.

## Configuration

Name | Default | Description
--- | --- | ---
ES_GATEWAY_LOG_LEVEL | INFO | Log level for ES gateway.
ES_GATEWAY_PORT | 5554 | Listen port for ES gateway.
ES_GATEWAY_HTTPS_CERT | /certs/https/cert | Path to cert for ES gateway to serve HTTPS requests.
ES_GATEWAY_HTTPS_KEY | /certs/https/key | Path to key for ES gateway to serve HTTPS requests.
ES_GATEWAY_K8S_CONFIG_PATH | | Path to Kubeconfig file.
ES_GATEWAY_ELASTIC_ENDPOINT | https://tigera-secure-es-http.tigera-elasticsearch.svc:9200 | Target endpoint (host and port) for connecting to Elasticsearch API.
ES_GATEWAY_ELASTIC_CATCH_ALL_ROUTE | / |
ES_GATEWAY_ELASTIC_CA_BUNDLE_PATH | /certs/elasticsearch/tls.crt | Path to CA cert for connecting to Elasticsearch.
ES_GATEWAY_ELASTIC_CLIENT_CERT_PATH | /certs/elasticsearch/client.crt | Path to client cert for connecting to Elasticsearch using mTLS.
ES_GATEWAY_ELASTIC_CLIENT_KEY_PATH | /certs/elasticsearch/client.key | Path to client key for connecting to Elasticsearch using mTLS.
ES_GATEWAY_ENABLE_ELASTIC_MUTUAL_TLS | false | Flag for enabling mTLS with Elasticsearch.
ES_GATEWAY_ELASTIC_USERNAME | | Username of Elasticsearch user for ES gateway to make API calls to Elasticsearch.
ES_GATEWAY_ELASTIC_PASSWORD | | Password of Elasticsearch user for ES gateway to make API calls to Elasticsearch.
ES_GATEWAY_KIBANA_ENDPOINT | https://tigera-secure-kb-http.tigera-kibana.svc:5601 | Target endpoint (host and port) for connecting to Kibana API.
ES_GATEWAY_KIBANA_CATCH_ALL_ROUTE | /tigera-kibana |
ES_GATEWAY_KIBANA_CA_BUNDLE_PATH | /certs/kibana/tls.crt | Path to CA cert for connecting to Kibana.
ES_GATEWAY_KIBANA_CLIENT_CERT_PATH | /certs/kibana/client.crt | Path to client cert for connecting to Kibana using mTLS.
ES_GATEWAY_KIBANA_CLIENT_KEY_PATH | /certs/kibana/client.key | Path to client key for connecting to Kibana using mTLS.
ES_GATEWAY_ENABLE_KIBANA_MUTUAL_TLS | false | Flag for enabling mTLS with Kibana.

## Build tags

There are two variants of es-gateway Cloud and Enterprise. To build or deploy the Cloud version preprend `TESLA=true` to make commands.

To add code that only targets one of the variants include the following at the top of the .go file:

Cloud only:
```
// +build tesla
```

Enterprise only:
```
// +build !tesla
```

Note that this only works at the file-level, meaning you can only include or exclude entire files.

An example of using build tags to write variant specific code can be found in:
- [pkg/version/version.go](pkg/version/version.go)
- [pkg/version/cloud.go](pkg/version/cloud.go)
- [pkg/version/enterprise.go](pkg/version/enterprise.go)
