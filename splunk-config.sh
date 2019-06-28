
##### Splunk #####
if [[ "${SPLUNK_AUDIT_LOG}" == "true" || "${SPLUNK_AUDIT_TSEE_LOG}" == "true" || "${SPLUNK_AUDIT_KUBE_LOG}" == "true" || "${SPLUNK_FLOW_LOG}" == "true" ]]; then
  # Optional Splunk audit log output
  if [ -z "${SPLUNK_AUDIT_INDEX}" ]; then
    sed -i 's|index .*||g' /fluentd/etc/outputs/out-splunk-audit.conf
  fi
  if [ -z "${SPLUNK_AUDIT_SOURCETYPE}" ]; then
    sed -i 's|sourcetype .*||g' /fluentd/etc/outputs/out-splunk-audit.conf
  fi
  if [ -z "${SPLUNK_AUDIT_SOURCE}" ]; then
    sed -i 's|source .*||g' /fluentd/etc/outputs/out-splunk-audit.conf
  fi
  if [ "${SPLUNK_AUDIT_LOG}" == "true" ]; then
    cp /fluentd/etc/outputs/out-splunk-audit.conf /fluentd/etc/output_tsee_audit/out-splunk-audit.conf
    cp /fluentd/etc/outputs/out-splunk-audit.conf /fluentd/etc/output_kube_audit/out-splunk-audit.conf
  elif [ "${SPLUNK_AUDIT_TSEE_LOG}" == "true" ]; then
    cp /fluentd/etc/outputs/out-splunk-audit.conf /fluentd/etc/output_tsee_audit/out-splunk-audit.conf
  elif [ "${SPLUNK_AUDIT_KUBE_LOG}" == "true" ]; then
    cp /fluentd/etc/outputs/out-splunk-audit.conf /fluentd/etc/output_kube_audit/out-splunk-audit.conf
  fi
  
  # Optional Splunk flow log output
  if [ -z "${SPLUNK_FLOW_INDEX}" ]; then
    sed -i 's|index .*||g' /fluentd/etc/outputs/out-splunk-flow.conf
  fi
  if [ -z "${SPLUNK_FLOW_SOURCETYPE}" ]; then
    sed -i 's|sourcetype .*||g' /fluentd/etc/outputs/out-splunk-flow.conf
  fi
  if [ -z "${SPLUNK_FLOW_SOURCE}" ]; then
    sed -i 's|source .*||g' /fluentd/etc/outputs/out-splunk-flow.conf
  fi
  if [ "${SPLUNK_FLOW_LOG}" == "true" ]; then
    cp /fluentd/etc/outputs/out-splunk-flow.conf /fluentd/etc/output_flows/out-splunk-flow.conf
  fi
fi

