if [ -z "${SYSLOG_VERIFY_MODE}" ]; then
  sed -i 's|verify_mode .*||g' /fluentd/etc/outputs/out-syslog.conf
fi
if [ ! -f /etc/fluentd/syslog/ca.pem ]; then
  sed -i 's|ca_file .*||g' /fluentd/etc/outputs/out-syslog.conf
fi
if [ "${SYSLOG_FLOW_LOG}" == "true" ]; then
  cp /fluentd/etc/outputs/out-syslog.conf /fluentd/etc/output_flows/out-syslog.conf
  cp /fluentd/etc/outputs/out-syslog.conf /fluentd/etc/output_dns/out-syslog.conf
fi
if [ "${SYSLOG_AUDIT_LOG}" == "true" ]; then
  cp /fluentd/etc/outputs/out-syslog.conf /fluentd/etc/output_tsee_audit/out-syslog.conf
  cp /fluentd/etc/outputs/out-syslog.conf /fluentd/etc/output_kube_audit/out-syslog.conf
fi
