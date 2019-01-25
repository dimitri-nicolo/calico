#!/usr/bin/dumb-init /bin/sh
set -e

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


# Check if s3 external storage is required and add the appropriate output sections.
if [ "${S3_STORAGE}" == "true" ]; then
  cat /fluentd/etc/fluent_output_s3.conf >> /fluentd/etc/fluent.conf
else
  cat /fluentd/etc/fluent_output.conf >> /fluentd/etc/fluent.conf
fi

# Check if we should strip out the secure settings from the configuration file.
if [ -z ${FLUENTD_ES_SECURE} ] || [ "${FLUENTD_ES_SECURE}" = "false" ]; then
  sed -i 's|scheme .*||g' /fluentd/etc/fluent.conf
  sed -i 's|user .*||g' /fluentd/etc/fluent.conf
  sed -i 's|password .*||g' /fluentd/etc/fluent.conf
  sed -i 's|ca_file .*||g' /fluentd/etc/fluent.conf
  sed -i 's|ssl_verify .*||g' /fluentd/etc/fluent.conf
fi

# Run fluentd
"$@"