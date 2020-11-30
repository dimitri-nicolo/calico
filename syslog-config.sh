if [ -z "${SYSLOG_VERIFY_MODE}" ]; then
  sed -i 's|verify_mode .*||g' /fluentd/etc/outputs/out-syslog.conf
fi
if [ ! -f /etc/fluentd/syslog/ca.pem ]; then
  sed -i 's|ca_file .*||g' /fluentd/etc/outputs/out-syslog.conf
fi
if [ "${SYSLOG_FLOW_LOG}" == "true" ]; then
  cp /fluentd/etc/outputs/out-syslog.conf /fluentd/etc/output_flows/out-syslog.conf
fi
if [ "${SYSLOG_DNS_LOG}" == "true" ]; then
  cp /fluentd/etc/outputs/out-syslog.conf /fluentd/etc/output_dns/out-syslog.conf
fi
if [ "${SYSLOG_AUDIT_EE_LOG}" == "true" ]; then
  cp /fluentd/etc/outputs/out-syslog.conf /fluentd/etc/output_tsee_audit/out-syslog.conf
fi
if [ "${SYSLOG_AUDIT_KUBE_LOG}" == "true" ]; then
  cp /fluentd/etc/outputs/out-syslog.conf /fluentd/etc/output_kube_audit/out-syslog.conf
fi
if [ "${SYSLOG_IDS_EVENT_LOG}" == "true" ]; then
  cp /fluentd/etc/outputs/out-syslog.conf /fluentd/etc/output_ids_events/out-syslog.conf
fi

# Append additional output matcher config (for IDS events) when SYSLOG forwarding is turned on
if [ "${SYSLOG_IDS_EVENT_LOG}" == "true" ]; then
  cat /fluentd/etc/output_match/ids-events.conf >> /fluentd/etc/fluent.conf
fi
