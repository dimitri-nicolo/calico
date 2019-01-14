#!/usr/bin/dumb-init /bin/sh
set -e

# Check if s3 external storage is required and set the configuration file
if [ "${S3_STORAGE}" == "true" ]; then
  mv /fluentd/etc/fluent_s3.conf /fluentd/etc/fluent.conf
fi

# Check if we should strip out the setting from the configuration file.
if [ -z ${FLUENTD_ES_SECURE} ] || [ "${FLUENTD_ES_SECURE}" = "false" ]; then
  sed -i 's|scheme .*||g' /fluentd/etc/fluent.conf
  sed -i 's|user .*||g' /fluentd/etc/fluent.conf
  sed -i 's|password .*||g' /fluentd/etc/fluent.conf
  sed -i 's|ca_file .*||g' /fluentd/etc/fluent.conf
  sed -i 's|ssl_verify .*||g' /fluentd/etc/fluent.conf
fi
"$@"
