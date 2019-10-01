#!/bin/bash
# Copyright (c) 2019 Tigera, Inc. All rights reserved.
set -xe

FV_ES_PROXY_TEST_IMAGE=${FV_ES_PROXY_TEST_IMAGE:-"tigera/es-proxy:latest"}
FV_ELASTICSEARCH_IMAGE=${FV_ELASTICSEARCH_IMAGE:-"docker.elastic.co/elasticsearch/elasticsearch:7.3.0"}
FV_GINKGO_ARGS=${FV_GINKGO_ARGS:-""}
PACKAGE_ROOT=${PACKAGE_ROOT:-$(pwd)/..}

GO_BUILD_IMAGE=${GO_BUILD_IMAGE:-"calico/go-build:v0.21"}
PROXY_LISTEN_HOST="127.0.0.1"
PROXY_LISTEN_PORT="8000"
ELASTIC_ACCESS_MODE=${ELASTIC_ACCESS_MODE:-"insecure"}
ELASTIC_SCHEME=${ELASTIC_SCHEME:-"http"}
ELASTIC_HOST=${ELASTIC_HOST:-"127.0.0.1"}
ELASTIC_PORT=${ELASTIC_PORT:-"9200"}
KUBERNETES_SERVICE_HOST=${KUBERNETES_SERVICE_HOST:-"127.0.0.1:6443"}
KUBERNETES_SERVICE_PORT=${KUBERNETES_SERVICE_PORT:-"6443"}

TEST_CONTAINER_NAME="fv-proxy-test"
ELASTICSEARCH_CONTAINER_NAME="fv-elasticsearch"
# LC_CTYPE required to work on macOS.
BOOTSTRAP_PASSWORD=$(cat /dev/urandom | LC_CTYPE=C tr -dc A-Za-z0-9 | head -c16)

function run_fvs()
{
	local GINKGO_ARGS=$1
	# Run test - if this fails output the logs from the proxy container. Running ginkgo with the--failFast flag
	# is useful if you are debugging issues since we do not correlate tests with output from the proxy container.
	docker run \
		--rm \
		--net=host \
		-v ${PACKAGE_ROOT}/.go-pkg-cache:/go/pkg:rw \
		-v ${PACKAGE_ROOT}/.go-build-cache:/home/user/.cache/go-build:rw \
		-v ${PACKAGE_ROOT}:/${PACKAGE_NAME}:rw \
		-v ${PACKAGE_ROOT}/report:/report:rw \
		-e LOCAL_USER_ID=$(id -u) \
		-w /${PACKAGE_NAME} \
		${GO_BUILD_IMAGE} \
		sh -c "ginkgo ${GINKGO_ARGS} ./test/" || (docker logs ${TEST_CONTAINER_NAME} && false)
}

function run_elasticsearch()
{
	local ELASTIC_SCHEME=$1
	local ELASTIC_ACCESS_MODE=$2

	if [ ${ELASTIC_ACCESS_MODE} != "insecure"  ]; then
		echo "BOOTSTRAP_PASSWORD is: ${BOOTSTRAP_PASSWORD}"
		ELASTICSEARCH_RUN_SECURITY_ARGS="-e xpack.security.enabled=true -e ELASTIC_PASSWORD=${BOOTSTRAP_PASSWORD}"
		EXTRA_CURL_ARGS="-u elastic:${BOOTSTRAP_PASSWORD}"
		ELASTICSEARCH_EXEC_SECURITY_ARGS="-e BOOTSTRAP_PASSWORD=${BOOTSTRAP_PASSWORD} -e ELASTIC_PASSWORD=${BOOTSTRAP_PASSWORD}"
	fi

	docker rm -f ${ELASTICSEARCH_CONTAINER_NAME} || true

	echo "Starting elasticsearch"
	docker run \
		--name ${ELASTICSEARCH_CONTAINER_NAME} \
		--detach \
		-p 9200:9200 \
		-p 9300:9300 \
		-e "discovery.type=single-node" \
		${ELASTICSEARCH_RUN_SECURITY_ARGS} \
		-v ${PACKAGE_ROOT}/test:/test:ro \
		${FV_ELASTICSEARCH_IMAGE}

	until docker exec ${ELASTICSEARCH_CONTAINER_NAME} curl http://127.0.0.1:9200 ${EXTRA_CURL_ARGS} 2> /dev/null;
	do
		echo "Waiting for Elasticsearch to come up..."; \
		sleep 1
	done

	docker exec ${ELASTICSEARCH_EXEC_SECURITY_ARGS} ${ELASTICSEARCH_CONTAINER_NAME} /test/setup_elasticsearch_index.sh

	# Enable XPack
	if [ ${ELASTIC_ACCESS_MODE} != "insecure"  ]; then
		RETRY_TIMES=3
		curl --insecure -f --retry ${RETRY_TIMES} -X POST "${ELASTIC_SCHEME}://${ELASTIC_HOST}:9200/_xpack/license/start_trial?acknowledge=true" -u "elastic:${BOOTSTRAP_PASSWORD}"
	fi

}

function run_proxy()
{
	local ELASTIC_SCHEME=$1
	local ELASTIC_ACCESS_MODE=$2
	local ELASTIC_HOST=$3
	local ELASTIC_PORT=$4
	local ELASTIC_USERNAME=$5
	local ELASTIC_PASSWORD=$6

	docker rm -f ${TEST_CONTAINER_NAME} || true

	# Start test image
	docker run \
		--net=host \
		--detach \
		-v ${PACKAGE_ROOT}/test:/test:ro \
		-e LOG_LEVEL=debug \
		-e LISTEN_ADDR="${PROXY_LISTEN_HOST}:${PROXY_LISTEN_PORT}" \
		-e ELASTIC_ACCESS_MODE=${ELASTIC_ACCESS_MODE} \
		-e ELASTIC_SCHEME=${ELASTIC_SCHEME} \
		-e ELASTIC_HOST=${ELASTIC_HOST} \
		-e ELASTIC_PORT=${ELASTIC_PORT} \
		-e ELASTIC_USERNAME=${ELASTIC_USERNAME} \
		-e ELASTIC_PASSWORD=${ELASTIC_PASSWORD} \
		-e ELASTIC_ENABLE_TRACE=true \
		-e KUBECONFIG=/test/test-apiserver-kubeconfig.conf \
		-e TIGERA_INTERNAL_RUNNING_FUNCTIONAL_VERIFICATION=true \
		--name ${TEST_CONTAINER_NAME} \
		${FV_ES_PROXY_TEST_IMAGE}
}

# Run a batch of tests that don't require elasticsearch first.
run_fvs "-skip Elasticsearch"

# Setup elasticsearch and run a second batch with insecure access.
run_elasticsearch "http" "insecure"
run_proxy "http" "insecure" "127.0.0.1" "9200"
run_fvs "-focus Elasticsearch"

# Setup elasticsearch this time with basic auth and run a third batch with insecure access.
run_elasticsearch "http" "serviceuser"
run_proxy "http" "serviceuser" "127.0.0.1" "9200" "elastic" ${BOOTSTRAP_PASSWORD}
run_fvs "-focus Elasticsearch"
# TODO(doublek): Enable TLS for TLS backend tests.

