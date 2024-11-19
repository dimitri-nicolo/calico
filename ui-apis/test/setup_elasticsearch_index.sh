#!/bin/bash
# Copyright (c) 2019 Tigera, Inc. All rights reserved.
set -xe

ELASTIC_SCHEME=${ELASTIC_SCHEME:-"https"}
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
  local NO_DATE=$2
  # Create the index template first if we have one. Do this before we create the corresponding index.
  if [ -f "/go/src/${PACKAGE_NAME}/test/es-templates/$INDEX_NAME" ]; then
    TEMPLATE=$(cat "/go/src/${PACKAGE_NAME}/test/es-templates/$INDEX_NAME")
    curl --insecure -f --retry ${RETRY_TIMES} -X PUT \
      "${ELASTIC_SCHEME}://${ELASTIC_HOST}:9200/_template/${INDEX_NAME}.cluster" ${EXTRA_CURL_ARGS} \
      -H 'Content-Type: application/json' -d "$TEMPLATE"
  fi


  if [[ $NO_DATE == 1 ]]
  then
    URL="${ELASTIC_SCHEME}://${ELASTIC_HOST}:9200/${INDEX_NAME}.${CLUSTER_NAME}"
  else
    URL="${ELASTIC_SCHEME}://${ELASTIC_HOST}:9200/${INDEX_NAME}.${CLUSTER_NAME}.${DATE_SUFFIX}"
  fi

  # Create the index.
  curl --insecure -f --retry ${RETRY_TIMES} -X PUT \
    "$URL" ${EXTRA_CURL_ARGS}

  # Create a document if there is one.
  if [ -f "/go/src/${PACKAGE_NAME}/test/es-samples/$INDEX_NAME" ]; then
    SAMPLE=$(cat "/go/src/${PACKAGE_NAME}/test/es-samples/$INDEX_NAME")
    curl --insecure -f --retry ${RETRY_TIMES} -X POST \
      "${ELASTIC_SCHEME}://${ELASTIC_HOST}:9200/${INDEX_NAME}.${CLUSTER_NAME}.${DATE_SUFFIX}/_doc" ${EXTRA_CURL_ARGS} \
      -H 'Content-Type: application/json' -d "$SAMPLE"
  fi
}

function create_index_pattern()
{
    local INDEX_NAME=$1
	
	curl --insecure -XPOST "${ELASTIC_SCHEME}://${ELASTIC_HOST}:9200/.kibana/doc/index-pattern:$INDEX_NAME"  -H 'Content-Type: application/json' -d " {  \"type\" : \"index-pattern\",  \"index-pattern\" : {    \"title\": \"${INDEX_NAME}*\"  }}" ${EXTRA_CURL_ARGS}
}


create_index ${FLOW_INDEX}
create_index ${AUDIT_KUBE_INDEX}
create_index ${AUDIT_EE_INDEX}
create_index ${EVENTS_INDEX} 1

create_index_pattern ${FLOW_INDEX}
create_index_pattern ${AUDIT_EE_INDEX}
create_index_pattern ${EVENTS_INDEX}



