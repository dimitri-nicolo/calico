#!/bin/sh

wget --no-check-certificate -q "https://localhost:8443/tigera_secure_ee_flows.${ELASTIC_INDEX_SUFFIX}.*/_search?size=0" -O /dev/null
