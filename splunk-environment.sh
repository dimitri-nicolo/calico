
##### Splunk #####
if [[ "${SPLUNK_AUDIT_LOG}" == "true" || "${SPLUNK_AUDIT_TSEE_LOG}" == "true" || "${SPLUNK_AUDIT_KUBE_LOG}" == "true" || "${SPLUNK_FLOW_LOG}" == "true" || "${SPLUNK_DNS_LOG}" == "true" ]]; then
  # Splunk defaults 
  export SPLUNK_AUDIT_LOG=${SPLUNK_AUDIT_LOG:-"false"}
  export SPLUNK_AUDIT_TSEE_LOG=${SPLUNK_AUDIT_TSEE_LOG:-"false"}
  export SPLUNK_AUDIT_KUBE_LOG=${SPLUNK_AUDIT_KUBE_LOG:-"false"}
  export SPLUNK_FLOW_LOG=${SPLUNK_FLOW_LOG:-"false"}
  export SPLUNK_DNS_LOG=${SPLUNK_DNS_LOG:-"false"}
  export SPLUNK_HEC_HOST=${SPLUNK_HEC_HOST:-"splunk-hec-host-not-set.com"}
  export SPLUNK_HEC_PORT=${SPLUNK_HEC_PORT:-"8088"}
  export SPLUNK_HEC_TOKEN=${SPLUNK_HEC_TOKEN:-"splunk-hec-token-not-set"}
  export SPLUNK_PROTOCOL=${SPLUNK_PROTOCOL:-"https"}
  export SPLUNK_FLUSH_INTERVAL=${SPLUNK_FLUSH_INTERVAL:-"5s"}
  export SPLUNK_FLUSH_THREAD_COUNT=${SPLUNK_FLUSH_THREAD_COUNT:-"2"}
  export SPLUNK_CA_FILE=${SPLUNK_CA_FILE:-"nil"}
  
  # Optional Splunk audit log output
  export SPLUNK_AUDIT_HEC_HOST=${SPLUNK_AUDIT_HEC_HOST:-$SPLUNK_HEC_HOST}
  export SPLUNK_AUDIT_HEC_PORT=${SPLUNK_AUDIT_HEC_PORT:-$SPLUNK_HEC_PORT}
  export SPLUNK_AUDIT_HEC_TOKEN=${SPLUNK_AUDIT_HEC_TOKEN:-$SPLUNK_HEC_TOKEN}
  export SPLUNK_AUDIT_PROTOCOL=${SPLUNK_AUDIT_PROTOCOL:-$SPLUNK_PROTOCOL}
  export SPLUNK_AUDIT_FLUSH_INTERVAL=${SPLUNK_AUDIT_FLUSH_INTERVAL:-$SPLUNK_FLUSH_INTERVAL}
  export SPLUNK_AUDIT_FLUSH_THREAD_COUNT=${SPLUNK_AUDIT_FLUSH_THREAD_COUNT:-$SPLUNK_FLUSH_THREAD_COUNT}
  export SPLUNK_AUDIT_INDEX=${SPLUNK_AUDIT_INDEX:-$SPLUNK_INDEX}
  export SPLUNK_AUDIT_SOURCETYPE=${SPLUNK_AUDIT_SOURCETYPE:-$SPLUNK_SOURCETYPE}
  export SPLUNK_AUDIT_SOURCE=${SPLUNK_AUDIT_SOURCE:-$SPLUNK_SOURCE}
  
  # Optional Splunk flow log output
  export SPLUNK_FLOW_HEC_HOST=${SPLUNK_FLOW_HEC_HOST:-$SPLUNK_HEC_HOST}
  export SPLUNK_FLOW_HEC_PORT=${SPLUNK_FLOW_HEC_PORT:-$SPLUNK_HEC_PORT}
  export SPLUNK_FLOW_HEC_TOKEN=${SPLUNK_FLOW_HEC_TOKEN:-$SPLUNK_HEC_TOKEN}
  export SPLUNK_FLOW_PROTOCOL=${SPLUNK_FLOW_PROTOCOL:-$SPLUNK_PROTOCOL}
  export SPLUNK_FLOW_FLUSH_INTERVAL=${SPLUNK_FLOW_FLUSH_INTERVAL:-$SPLUNK_FLUSH_INTERVAL}
  export SPLUNK_FLOW_FLUSH_THREAD_COUNT=${SPLUNK_FLOW_FLUSH_THREAD_COUNT:-$SPLUNK_FLUSH_THREAD_COUNT}
  export SPLUNK_FLOW_INDEX=${SPLUNK_FLOW_INDEX:-$SPLUNK_INDEX}
  export SPLUNK_FLOW_SOURCETYPE=${SPLUNK_FLOW_SOURCETYPE:-$SPLUNK_SOURCETYPE}
  export SPLUNK_FLOW_SOURCE=${SPLUNK_FLOW_SOURCE:-$SPLUNK_SOURCE}
  
  # Optional Splunk dns log output
  export SPLUNK_DNS_HEC_HOST=${SPLUNK_DNS_HEC_HOST:-$SPLUNK_HEC_HOST}
  export SPLUNK_DNS_HEC_PORT=${SPLUNK_DNS_HEC_PORT:-$SPLUNK_HEC_PORT}
  export SPLUNK_DNS_HEC_TOKEN=${SPLUNK_DNS_HEC_TOKEN:-$SPLUNK_HEC_TOKEN}
  export SPLUNK_DNS_PROTOCOL=${SPLUNK_DNS_PROTOCOL:-$SPLUNK_PROTOCOL}
  export SPLUNK_DNS_FLUSH_INTERVAL=${SPLUNK_DNS_FLUSH_INTERVAL:-$SPLUNK_FLUSH_INTERVAL}
  export SPLUNK_DNS_FLUSH_THREAD_COUNT=${SPLUNK_DNS_FLUSH_THREAD_COUNT:-$SPLUNK_FLUSH_THREAD_COUNT}
  export SPLUNK_DNS_INDEX=${SPLUNK_DNS_INDEX:-$SPLUNK_INDEX}
  export SPLUNK_DNS_SOURCETYPE=${SPLUNK_DNS_SOURCETYPE:-$SPLUNK_SOURCETYPE}
  export SPLUNK_DNS_SOURCE=${SPLUNK_DNS_SOURCE:-$SPLUNK_SOURCE}
fi

