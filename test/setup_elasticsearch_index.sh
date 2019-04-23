#!/bin/bash
# Copyright (c) 2019 Tigera, Inc. All rights reserved.
set -xe

ELASTIC_SCHEME=${ELASTIC_SCHEME:-"http"}
ELASTIC_HOST="127.0.0.1"
CLUSTER_NAME=${CLUSTER_NAME:-"cluster"}
FLOW_INDEX=${FLOW_INDEX:-"tigera_secure_ee_flows"}
AUDIT_EE_INDEX=${AUDIT_EE_INDEX:-"tigera_secure_ee_audit_ee"}
AUDIT_KUBE_INDEX=${AUDIT_KUBE_INDEX:-"tigera_secure_ee_audit_kube"}
EVENTS_INDEX=${EVENTS_INDEX:-"tigera_secure_ee_events"}

if [ -n ${ELASTIC_PASSWORD} ]; then
  EXTRA_CURL_ARGS="-u elastic:${ELASTIC_PASSWORD}"
fi

RETRY_TIMES=3

function create_index()
{
  local INDEX_NAME=$1

  curl --insecure -f --retry ${RETRY_TIMES} -X PUT "${ELASTIC_SCHEME}://${ELASTIC_HOST}:9200/${INDEX_NAME}.${CLUSTER_NAME}.${DATE_SUFFIX}" ${EXTRA_CURL_ARGS}
}

create_index ${FLOW_INDEX}
create_index ${AUDIT_KUBE_INDEX}
create_index ${AUDIT_EE_INDEX}
create_index ${EVENTS_INDEX}
