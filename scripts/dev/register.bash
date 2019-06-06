#!/usr/bin/env bash

curl "http://localhost:${VOLTRON_PORT:-5555}/targets" -X PUT -d 'name=cluster1' -d "target=https://${IP_CLUSTER1:-kubernetes.default}"
curl "http://localhost:${VOLTRON_PORT:-5555}/targets" -X PUT -d 'name=cluster2' -d "target=http://${IP_CLUSTER2:-kubernetes.default}"

