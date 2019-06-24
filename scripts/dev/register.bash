#!/usr/bin/env bash

VOLTRON_URL=${1-"localhost:5555"}

curl -X PUT "https://${VOLTRON_URL}/voltron/api/clusters" -H "Content-type: application/json" -d '{"id":"1", "displayName":"Cluster 1"}' --insecure -o guardian1.yaml
curl -X PUT "https://${VOLTRON_URL}/voltron/api/clusters" -H "Content-type: application/json" -d '{"id":"2", "displayName":"Cluster 2"}' --insecure -o guardian2.yaml

