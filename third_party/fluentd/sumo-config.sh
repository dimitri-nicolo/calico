if [[ "${SUMO_AUDIT_LOG}" == "true" || "${SUMO_AUDIT_TSEE_LOG}" == "true" || "${SUMO_AUDIT_KUBE_LOG}" == "true" || "${SUMO_FLOW_LOG}" == "true" ]]; then
  # Optional SumoLogic audit log output
  if [ -z "${SUMO_AUDIT_SOURCE_CATEGORY}" ]; then
    sed -i 's|source_category .*||g' ${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-audit.conf
  fi
  if [ -z "${SUMO_AUDIT_SOURCE_NAME}" ]; then
    sed -i 's|source_name .*||g' ${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-audit.conf
  fi
  if [ -z "${SUMO_AUDIT_SOURCE_HOST}" ]; then
    sed -i 's|source_host .*||g' ${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-audit.conf
  fi
  if [ "${SUMO_AUDIT_LOG}" == "true" ]; then
    cp "${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-audit.conf" "${ROOT_DIR}/fluentd/etc/output_tsee_audit/out-sumologic-audit.conf"
    cp "${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-audit.conf" "${ROOT_DIR}/fluentd/etc/output_kube_audit/out-sumologic-audit.conf"
  elif [ "${SUMO_AUDIT_TSEE_LOG}" == "true" ]; then
    cp "${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-audit.conf" "${ROOT_DIR}/fluentd/etc/output_tsee_audit/out-sumologic-audit.conf"
  elif [ "${SUMO_AUDIT_KUBE_LOG}" == "true" ]; then
    cp "${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-audit.conf" "${ROOT_DIR}/fluentd/etc/output_kube_audit/out-sumologic-audit.conf"
  fi

  # Optional SumoLogic flow log output
  if [ -z "${SUMO_FLOW_SOURCE_CATEGORY}" ]; then
    sed -i 's|source_category .*||g' "${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-flow.conf"
  fi
  if [ -z "${SUMO_FLOW_SOURCE_NAME}" ]; then
    sed -i 's|source_name .*||g' "${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-flow.conf"
  fi
  if [ -z "${SUMO_FLOW_SOURCE_HOST}" ]; then
    sed -i 's|source_host .*||g' "${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-flow.conf"
  fi
  if [ "${SUMO_FLOW_LOG}" == "true" ]; then
    cp "${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-flow.conf" "${ROOT_DIR}/fluentd/etc/output_flows/out-sumologic-flow.conf"
  fi

  # Optional SumoLogic dns log output
  if [ -z "${SUMO_DNS_SOURCE_CATEGORY}" ]; then
    sed -i 's|source_category .*||g' "${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-dns.conf"
  fi
  if [ -z "${SUMO_DNS_SOURCE_NAME}" ]; then
    sed -i 's|source_name .*||g' "${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-dns.conf"
  fi
  if [ -z "${SUMO_DNS_SOURCE_HOST}" ]; then
    sed -i 's|source_host .*||g' "${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-dns.conf"
  fi
  if [ "${SUMO_DNS_LOG}" == "true" ]; then
    cp "${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-dns.conf" "${ROOT_DIR}/fluentd/etc/output_dnss/out-sumologic-dns.conf"
  fi

  # Optional SumoLogic l7 log output
  if [ -z "${SUMO_L7_SOURCE_CATEGORY}" ]; then
    sed -i 's|source_category .*||g' "${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-l7.conf"
  fi
  if [ -z "${SUMO_L7_SOURCE_NAME}" ]; then
    sed -i 's|source_name .*||g' "${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-l7.conf"
  fi
  if [ -z "${SUMO_L7_SOURCE_HOST}" ]; then
    sed -i 's|source_host .*||g' "${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-l7.conf"
  fi
  if [ "${SUMO_L7_LOG}" == "true" ]; then
    cp "${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-l7.conf" "${ROOT_DIR}/fluentd/etc/output_l7/out-sumologic-l7.conf"
  fi

  # Optional SumoLogic WAF log output
  if [ -z "${SUMO_WAF_SOURCE_CATEGORY}" ]; then
    sed -i 's|source_category .*||g' "${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-waf.conf"
  fi
  if [ -z "${SUMO_WAF_SOURCE_NAME}" ]; then
    sed -i 's|source_name .*||g' "${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-waf.conf"
  fi
  if [ -z "${SUMO_WAF_SOURCE_HOST}" ]; then
    sed -i 's|source_host .*||g' "${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-waf.conf"
  fi
  if [ "${SUMO_WAF_LOG}" == "true" ]; then
    cp "${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-waf.conf" "${ROOT_DIR}/fluentd/etc/output_waf/out-sumologic-waf.conf"
  fi

  # Optional SumoLogic runtime-security report log output
  if [ -z "${SUMO_RUNTIME_SOURCE_CATEGORY}" ]; then
    sed -i 's|source_category .*||g' "${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-runtime.conf"
  fi
  if [ -z "${SUMO_RUNTIME_SOURCE_NAME}" ]; then
    sed -i 's|source_name .*||g' "${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-runtime.conf"
  fi
  if [ -z "${SUMO_RUNTIME_SOURCE_HOST}" ]; then
    sed -i 's|source_host .*||g' "${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-runtime.conf"
  fi
  if [ "${SUMO_RUNTIME_LOG}" == "true" ]; then
    cp "${ROOT_DIR}/fluentd/etc/outputs/out-sumologic-runtime.conf" "${ROOT_DIR}/fluentd/etc/output_runtime/out-sumologic-runtime.conf"
  fi
fi
