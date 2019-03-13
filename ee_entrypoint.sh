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

# Check if we should strip out the secure settings from the configuration file.
if [ -z ${FLUENTD_ES_SECURE} ] || [ "${FLUENTD_ES_SECURE}" = "false" ]; then
  sed -i 's|scheme .*||g' /fluentd/etc/fluent_output_es.conf.sh
  sed -i 's|user .*||g' /fluentd/etc/fluent_output_es.conf.sh
  sed -i 's|password .*||g' /fluentd/etc/fluent_output_es.conf.sh
  sed -i 's|ca_file .*||g' /fluentd/etc/fluent_output_es.conf.sh
  sed -i 's|ssl_verify .*||g' /fluentd/etc/fluent_output_es.conf.sh
fi

source /fluentd/etc/fluent_output_es.conf.sh
if [ "${S3_STORAGE}" == "true" ]; then
  source /fluentd/etc/fluent_output_s3.conf.sh
  NEED_COPY=true
fi

# If we are outputing to 2 then the root type is copy and each sub block is
# wrapped in <store>...</store>. The ES block does not include the store wrap
# but the others do.
# If there is only the ES output then the envs will be unset/empty and will not
# add anything to the template.
if [ "${NEED_COPY}" == "true" ]; then
  COPY="  @type copy"
  START_STORE="  <store>"
  END_STORE="  </store>"
fi

template=$(cat /fluentd/etc/fluent_output.conf)
eval "echo \"${template}\"" >> /fluentd/etc/fluent.conf

# Run fluentd
"$@"
