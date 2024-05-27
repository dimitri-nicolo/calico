# Elasticsearch-metrics

The goal of this repo is to create a Docker image based on the project 
github.com/prometheus-community/elasticsearch_exporter.

We do not want to use the original build process, because we need to build our go code 
using our own builder image for practical reasons and for compliance reasons.

# Code changes

We made the following code changes to the original project:
- Add TLS: the original project listened on http only.
- Enforce our cryptographic standards.

All code changes are isolated inside of `cmd/main.go`. 
Each change is enclosed by the following two comment lines:
- `// BEGIN TIGERA CHANGES`
- `// END TIGERA CHANGES`

If you want to update the pin of elasticsearch_exporter, you need to replace cmd/main.go
and apply the same changes.
Also, you need to verify that `cmd/logger.go` has not changed.
