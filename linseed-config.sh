# Append additional output matcher config (for IDS events) when SYSLOG forwarding is turned on
if [ "${LINSEED_ENABLED}" == "true" ]; then
  cp ${ROOT_DIR}/fluentd/etc/outputs/out-linseed.conf ${ROOT_DIR}/fluentd/etc/output_flows/out-linseed.conf
fi

