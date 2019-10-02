#!/bin/bash

# This script will generate fluentd configs using the image
# tigera/fluentd:${IMAGETAG} based off the environment variables configurations
# below and then compare to previously captured configurations to ensure
# only expected changes have happened.

DEBUG="false"
TEST_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
mkdir -p $TEST_DIR/tmp

ADDITIONAL_MOUNT=""

function generateAndCollectConfig() {
  ENV_FILE=$1
  OUT_FILE=$2

  docker run -d --rm --name generate-fluentd-config $ADDITIONAL_MOUNT --hostname config.generator --env-file $ENV_FILE tigera/fluentd:${IMAGETAG} >/dev/null
  if [ $? -ne 0 ]; then echo "Running fluentd container failed"; exit 1; fi
  sleep 2

  docker logs generate-fluentd-config | sed -n '/<ROOT>/,/<\/ROOT>/p' | sed -e 's|^.*<ROOT>|<ROOT>|' > $OUT_FILE
  if [ $? -ne 0 ]; then echo "Grabbing config from fluentd container failed"; exit 1; fi

  docker stop generate-fluentd-config >/dev/null
  if [ $? -ne 0 ]; then echo "Stopping fluentd container failed"; exit 1; fi
  unset ADDITIONAL_MOUNT
}

function checkConfiguration() {
  ENV_FILE=$1
  CFG_NAME=$2
  READABLE_NAME=$3

  EXPECTED=$TEST_DIR/$CFG_NAME.cfg
  UUT=$TEST_DIR/tmp/$CFG_NAME.cfg

  echo "#### Testing configuration of $READABLE_NAME"

  generateAndCollectConfig $ENV_FILE $UUT

  diff $EXPECTED $UUT &> /dev/null
  if [ $? -eq 0 ]; then
    echo "  ## configuration is correct"
  else
    echo " XXX configuration is not correct"
    $DEBUG && diff $EXPECTED $UUT
  fi
}


STANDARD_ENV_VARS=$(cat << EOM
ELASTIC_INDEX_SUFFIX=test-cluster-name
ELASTIC_FLOWS_INDEX_SHARDS=5
ELASTIC_DNS_INDEX_SHARDS=5
FLUENTD_FLOW_FILTERS=# not a real filter
FLOW_LOG_FILE=/var/log/calico/flowlogs/flows.log
DNS_LOG_FILE=/var/log/calico/dnslogs/dns.log
ELASTIC_HOST=elasticsearch-tigera-elasticsearch.calico-monitoring.svc.cluster.local
ELASTIC_PORT=9200
EOM
)

ES_SECURE_VARS=$(cat <<EOM
ELASTIC_SSL_VERIFY=true
ELASTIC_USER=es-user
ELASTIC_PASSWORD=es-password
EOM
)

S3_VARS=$(cat <<EOM
AWS_KEY_ID=aws-key-id-value
AWS_SECRET_KEY=aws-secret-key-value
S3_STORAGE=true
S3_BUCKET_NAME=dummy-bucket
AWS_REGION=not-real-region
S3_BUCKET_PATH=not-a-bucket
S3_FLUSH_INTERVAL=30
EOM
)

SYSLOG_NO_TLS_VARS=$(cat <<EOM
SYSLOG_FLOW_LOG=true
SYSLOG_HOST=169.254.254.254
SYSLOG_PORT=3665
SYSLOG_PROTOCOL=udp
SYSLOG_HOSTNAME=nodename
SYSLOG_FLUSH_INTERVAL=17s
EOM
)

SYSLOG_TLS_VARS=$(cat <<EOM
SYSLOG_FLOW_LOG=true
SYSLOG_AUDIT_LOG=true
SYSLOG_HOST=169.254.254.254
SYSLOG_PORT=3665
SYSLOG_PROTOCOL=tcp
SYSLOG_TLS=true
SYSLOG_VERIFY_MODE=\${OPENSSL::SSL::VERIFY_NONE}
SYSLOG_HOSTNAME=nodename
EOM
)

EKS_VARS=$(cat <<EOM
MANAGED_K8S=true
K8S_PLATFORM=eks
EKS_CLOUDWATCH_LOG_GROUP=/aws/eks/eks-audit-test/cluster/
EOM
)

# Test with ES not secure
cat > $TEST_DIR/tmp/es-no-secure.env <<EOM
$STANDARD_ENV_VARS
FLUENTD_ES_SECURE=false
EOM

checkConfiguration $TEST_DIR/tmp/es-no-secure.env es-no-secure "ES secure false"

# Test with ES secure
cat > $TEST_DIR/tmp/es-secure.env << EOM
$STANDARD_ENV_VARS
FLUENTD_ES_SECURE=true
$ES_SECURE_VARS
EOM

checkConfiguration $TEST_DIR/tmp/es-secure.env es-secure "ES secure"

# Test with S3 and ES secure
cat > $TEST_DIR/tmp/es-secure-with-s3.env << EOM
$STANDARD_ENV_VARS
FLUENTD_ES_SECURE=true
$ES_SECURE_VARS
$S3_VARS
EOM

checkConfiguration $TEST_DIR/tmp/es-secure-with-s3.env es-secure-with-s3 "ES secure with S3"

# Test with S3 and ES not secure
cat > $TEST_DIR/tmp/es-no-secure-with-s3.env << EOM
$STANDARD_ENV_VARS
FLUENTD_ES_SECURE=false
$S3_VARS
EOM

checkConfiguration $TEST_DIR/tmp/es-no-secure-with-s3.env es-no-secure-with-s3 "ES secure false with S3"

# Test with ES not secure and syslog w/no tls
cat > $TEST_DIR/tmp/es-no-secure-with-syslog-no-tls.env << EOM
$STANDARD_ENV_VARS
FLUENTD_ES_SECURE=false
$SYSLOG_NO_TLS_VARS
EOM

checkConfiguration $TEST_DIR/tmp/es-no-secure-with-syslog-no-tls.env es-no-secure-with-syslog-no-tls "ES secure false with syslog without TLS"

# Test with ES secure and syslog with tls
cat > $TEST_DIR/tmp/es-secure-with-syslog-with-tls.env << EOM
$STANDARD_ENV_VARS
FLUENTD_ES_SECURE=true
$ES_SECURE_VARS
$SYSLOG_TLS_VARS
EOM

TMP=$(tempfile)
ADDITIONAL_MOUNT="-v $TMP:/etc/fluentd/syslog/ca.pem"
checkConfiguration $TEST_DIR/tmp/es-secure-with-syslog-with-tls.env es-secure-with-syslog-with-tls "ES secure with syslog with TLS"

# Test with ES secure and syslog with tls
cat > $TEST_DIR/tmp/es-secure-with-syslog-and-s3.env << EOM
$STANDARD_ENV_VARS
FLUENTD_ES_SECURE=true
$ES_SECURE_VARS
$SYSLOG_TLS_VARS
$S3_VARS
EOM

checkConfiguration $TEST_DIR/tmp/es-secure-with-syslog-and-s3.env es-secure-with-syslog-and-s3 "ES secure with syslog and S3"

# Test with EKS
cat > $TEST_DIR/tmp/eks.env <<EOM
$EKS_VARS
EOM
checkConfiguration $TEST_DIR/tmp/eks.env eks "EKS"

# Test with EKS, Log Stream Prefix overwritten
cat > $TEST_DIR/tmp/eks-log-stream-pfx.env <<EOM
$EKS_VARS
EKS_CLOUDWATCH_LOG_STREAM_PREFIX=kube-apiserver-audit-overwritten-
EOM
checkConfiguration $TEST_DIR/tmp/eks-log-stream-pfx.env eks-log-stream-pfx "EKS - Log Stream Prefix overwritten"

rm -f $TMP
