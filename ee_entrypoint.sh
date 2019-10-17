#!/bin/sh
set -e

setup_secure_es_conf() {
  sed -i 's|scheme .*||g' /fluentd/etc/output_${1}/out-es.conf
  sed -i 's|user .*||g' /fluentd/etc/output_${1}/out-es.conf
  sed -i 's|password .*||g' /fluentd/etc/output_${1}/out-es.conf
  sed -i 's|ca_file .*||g' /fluentd/etc/output_${1}/out-es.conf
  sed -i 's|ssl_verify .*||g' /fluentd/etc/output_${1}/out-es.conf
}

if [ "${MANAGED_K8S}" == "true" ]; then
  # start from scratch
  > /fluentd/etc/fluent.conf

  # source
  if [ "${K8S_PLATFORM}" == "eks" ]; then
    export EKS_CLOUDWATCH_LOG_STREAM_PREFIX=${EKS_CLOUDWATCH_LOG_STREAM_PREFIX:-"kube-apiserver-audit-"}
    cat /fluentd/etc/inputs/in-eks.conf >> /fluentd/etc/fluent.conf
  fi

  # filter
  cat /fluentd/etc/filters/filter-eks-audit.conf >> /fluentd/etc/fluent.conf
  echo >> /fluentd/etc/fluent.conf

  # match
  cp /fluentd/etc/outputs/out-es-kube-audit.conf /fluentd/etc/output_kube_audit/out-es.conf
  if [ -z ${FLUENTD_ES_SECURE} ] || [ "${FLUENTD_ES_SECURE}" == "false" ]; then
    setup_secure_es_conf kube_audit
  fi
  if [ "${S3_STORAGE}" == "true" ]; then
    cp /fluentd/etc/outputs/out-s3-kube-audit.conf /fluentd/etc/output_kube_audit/out-s3.conf
  fi
  source /bin/syslog-environment.sh
  source /bin/syslog-config.sh
  source /bin/splunk-environment.sh
  source /bin/splunk-config.sh
  source /bin/sumo-environment.sh
  source /bin/sumo-config.sh
  cat /fluentd/etc/outputs/out-eks-audit-es.conf >> /fluentd/etc/fluent.conf
  echo >> /fluentd/etc/fluent.conf

  # Run fluentd
  "$@"

  # bail earlier
  exit $?
fi

# Set the number of shards for index tigera_secure_ee_flows
sed -i 's|"number_of_shards": \d*|"number_of_shards": '"$ELASTIC_FLOWS_INDEX_SHARDS"'|g' /fluentd/etc/elastic_mapping_flows.template

# Set the number of shards for index tigera_secure_ee_dns
sed -i 's|"number_of_shards": \d*|"number_of_shards": '"$ELASTIC_DNS_INDEX_SHARDS"'|g' /fluentd/etc/elastic_mapping_dns.template

# Build the fluentd configuration file bit by bit, because order is important.
# Add the sources.
cat /fluentd/etc/fluent_sources.conf >> /fluentd/etc/fluent.conf
echo >> /fluentd/etc/fluent.conf

# Append additional filter blocks to the fluentd config if provided.
if [ "${FLUENTD_FLOW_FILTERS}" == "true" ]; then
  cat /etc/fluentd/flow-filters.conf >> /fluentd/etc/fluent.conf
  echo >> /fluentd/etc/fluent.conf
fi

# Append additional filter blocks to the fluentd config if provided.
if [ "${FLUENTD_DNS_FILTERS}" == "true" ]; then
  cat /etc/fluentd/dns-filters.conf >> /fluentd/etc/fluent.conf
  echo >> /fluentd/etc/fluent.conf
fi

# Record transformations to add additional identifiers.
cat /fluentd/etc/fluent_transforms.conf >> /fluentd/etc/fluent.conf
echo >> /fluentd/etc/fluent.conf

cp /fluentd/etc/outputs/out-es-flows.conf /fluentd/etc/output_flows/out-es.conf
cp /fluentd/etc/outputs/out-es-dns.conf /fluentd/etc/output_dns/out-es.conf
cp /fluentd/etc/outputs/out-es-tsee-audit.conf /fluentd/etc/output_tsee_audit/out-es.conf
cp /fluentd/etc/outputs/out-es-kube-audit.conf /fluentd/etc/output_kube_audit/out-es.conf

# Check if we should strip out the secure settings from the configuration file.
if [ -z ${FLUENTD_ES_SECURE} ] || [ "${FLUENTD_ES_SECURE}" == "false" ]; then
  for x in flows dns tsee_audit kube_audit; do
    setup_secure_es_conf $x
  done
fi

if [ "${S3_STORAGE}" == "true" ]; then
  cp /fluentd/etc/outputs/out-s3-flows.conf /fluentd/etc/output_flows/out-s3.conf
  cp /fluentd/etc/outputs/out-s3-dns.conf /fluentd/etc/output_dns/out-s3.conf
  cp /fluentd/etc/outputs/out-s3-tsee-audit.conf /fluentd/etc/output_tsee_audit/out-s3.conf
  cp /fluentd/etc/outputs/out-s3-kube-audit.conf /fluentd/etc/output_kube_audit/out-s3.conf
  cp /fluentd/etc/outputs/out-s3-compliance-reports.conf /fluentd/etc/output_compliance_reports/out-s3.conf
fi

source /bin/syslog-environment.sh
source /bin/syslog-config.sh

source /bin/splunk-environment.sh
source /bin/splunk-config.sh

source /bin/sumo-environment.sh
source /bin/sumo-config.sh

cat /fluentd/etc/fluent_output.conf >> /fluentd/etc/fluent.conf
# Append additional output config (for Compliance reports) when S3 archiving is turned on
if [ "${S3_STORAGE}" == "true" ]; then
  cat /fluentd/etc/fluent_output_optional.conf >> /fluentd/etc/fluent.conf
fi
echo >> /fluentd/etc/fluent.conf

# Run fluentd
"$@"
