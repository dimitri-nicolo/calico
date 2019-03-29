#!/bin/bash

ELASTIC_URL=localhost:9200
curl $ELASTIC_URL/tigera_secure_ee_snapshots/_search?size=1000 > lists.json
curl $ELASTIC_URL/tigera_secure_ee_audit_kube*/_search?size=1000 -HContent-type:application/json -d '{"query":{"match":{"objectRef.namespace": "compliance-testing"}}}' > kube_events.json
curl $ELASTIC_URL/tigera_secure_ee_audit_ee*/_search?size=1000 -HContent-type:application/json > ee_events.json
