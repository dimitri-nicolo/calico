#!/usr/bin/env bash

USERNAME=$(kubectl -n tigera-intrusion-detection get secret tigera-ee-installer-elasticsearch-access -o go-template='{{.data.username | base64decode}}')
PASSWORD=$(kubectl -n tigera-intrusion-detection get secret tigera-ee-installer-elasticsearch-access -o go-template='{{.data.password | base64decode}}')

DASHBOARDS="api-kibana-dashboard.json tor-vpn-dashboard.json honeypod-dashboard.json dns-dashboard.json kubernetes-api-dashboard.json"
for FILE in $DASHBOARDS; do
  CODE=1
  while [ $CODE != 0 ]; do
    RESPONSE=$(curl --insecure --max-time 2 -XPOST "https://${USERNAME}:${PASSWORD}@localhost:9443/tigera-kibana/api/saved_objects/_bulk_get" --header 'kbn-xsrf: reporting' --header 'Content-Type: application/json' --data-raw "$(jq -c '. | map({id: .id, type: .type})' cmd/data/${FILE})")
    CODE=$?
  done

  echo $RESPONSE | jq '.saved_objects | map(del(.namespaces) | del(.updated_at))' > cmd/data/${FILE}
done
