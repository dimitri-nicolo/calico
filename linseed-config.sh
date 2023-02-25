# Append additional output matcher config (for IDS events) when SYSLOG forwarding is turned on
if [ "${LINSEED_ENABLED}" == "true" ]; then
  cp ${ROOT_DIR}/fluentd/etc/outputs/out-linseed-flows.conf ${ROOT_DIR}/fluentd/etc/output_flows/out-linseed-flows.conf
  cp ${ROOT_DIR}/fluentd/etc/outputs/out-linseed-l7.conf ${ROOT_DIR}/fluentd/etc/output_l7/out-linseed-l7.conf
  cp ${ROOT_DIR}/fluentd/etc/outputs/out-linseed-dns.conf ${ROOT_DIR}/fluentd/etc/output_dns/out-linseed-dns.conf
  cp ${ROOT_DIR}/fluentd/etc/outputs/out-linseed-kube-audit.conf ${ROOT_DIR}/fluentd/etc/output_kube_audit/out-linseed-kube-audit.conf
  cp ${ROOT_DIR}/fluentd/etc/outputs/out-linseed-ee-audit.conf ${ROOT_DIR}/fluentd/etc/output_tsee_audit/out-linseed-ee-audit.conf
  cp ${ROOT_DIR}/fluentd/etc/outputs/out-linseed-bgp.conf ${ROOT_DIR}/fluentd/etc/output_bgp/out-linseed-bgp.conf
fi

