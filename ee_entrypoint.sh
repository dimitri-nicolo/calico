#!/usr/bin/dumb-init /bin/sh
set -e

# Set the number of shards for index tigera_secure_ee_flows
sed -i 's|"number_of_shards": \d*|"number_of_shards": '"$ELASTIC_FLOWS_INDEX_SHARDS"'|g' /fluentd/etc/elastic_mapping_flows.template

# Build the fluentd configuration file bit by bit, because order is important.
# Add the sources.
cat /fluentd/etc/fluent_sources.conf >> /fluentd/etc/fluent.conf
echo >> /fluentd/etc/fluent.conf

# Append additional filter blocks to the fluentd config if provided.
if [ "${FLUENTD_FLOW_FILTERS}" == "true" ]; then
  cat /etc/fluentd/flow-filters.conf >> /fluentd/etc/fluent.conf
  echo >> /fluentd/etc/fluent.conf
fi

# Record transformations to add additional identifiers.
cat /fluentd/etc/fluent_transforms.conf >> /fluentd/etc/fluent.conf
echo >> /fluentd/etc/fluent.conf

cp /fluentd/etc/outputs/out-es-flows.conf /fluentd/etc/output_flows/out-es.conf
cp /fluentd/etc/outputs/out-es-tsee-audit.conf /fluentd/etc/output_tsee_audit/out-es.conf
cp /fluentd/etc/outputs/out-es-kube-audit.conf /fluentd/etc/output_kube_audit/out-es.conf
if [ "${S3_STORAGE}" == "true" ]; then
  cp /fluentd/etc/outputs/out-s3-flows.conf /fluentd/etc/output_flows/out-s3.conf
  cp /fluentd/etc/outputs/out-s3-tsee-audit.conf /fluentd/etc/output_tsee_audit/out-s3.conf
  cp /fluentd/etc/outputs/out-s3-kube-audit.conf /fluentd/etc/output_kube_audit/out-s3.conf
fi

# Check if we should strip out the secure settings from the configuration file.
if [ -z ${FLUENTD_ES_SECURE} ] || [ "${FLUENTD_ES_SECURE}" == "false" ]; then
  for x in flows tsee_audit kube_audit; do
    sed -i 's|scheme .*||g' /fluentd/etc/output_${x}/out-es.conf
    sed -i 's|user .*||g' /fluentd/etc/output_${x}/out-es.conf
    sed -i 's|password .*||g' /fluentd/etc/output_${x}/out-es.conf
    sed -i 's|ca_file .*||g' /fluentd/etc/output_${x}/out-es.conf
    sed -i 's|ssl_verify .*||g' /fluentd/etc/output_${x}/out-es.conf
  done
fi

cat /fluentd/etc/fluent_output.conf >> /fluentd/etc/fluent.conf

# Run fluentd
"$@"
